# Arranca Fish Speech API (porta 8080). CWD = clone em services/tts-extras/vendor/fish-speech.
# Checkpoints: pasta devolvida por Get-ChatbotTtsCachePaths (CHATBOT_TTS_CACHE_ROOT ou repo).
param(
    [string] $RepoRoot = "",
    [string] $Listen = "0.0.0.0:8080"
)

$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Split-Path -Parent $here
}

. (Join-Path $here "_tts_cache_paths.ps1")
$p = Get-ChatbotTtsCachePaths -RepoRoot $RepoRoot

$fish = Join-Path $RepoRoot "services\tts-extras\vendor\fish-speech"
$py = Join-Path $RepoRoot "services\tts-extras\venvs\fish-speech\Scripts\python.exe"
if (-not (Test-Path $py)) { Write-Error "Venv fish em falta. Corre install-tts-extras.ps1 -Components Fish" }
if (-not (Test-Path $fish)) { Write-Error "Clone fish-speech em falta: $fish" }

$s2 = $p.FishS2Pro
if (-not (Test-Path $s2)) {
    Write-Error "Checkpoints em falta: $s2. Corre: npm run fish:download-checkpoints"
}

$llamaAbs = (Resolve-Path $s2).Path
$codecAbs = Join-Path $llamaAbs "codec.pth"
if (-not (Test-Path $codecAbs)) {
    Write-Error "codec.pth em falta: $codecAbs. Completa npm run fish:download-checkpoints."
}
$codecAbs = (Resolve-Path $codecAbs).Path

# Sem CUDA, o Fish usa CPU com bfloat16 por defeito — em Windows costuma crashar (0xC0000005) ao carregar S2-Pro.
$useHalf = (& $py -c "import torch; print(0 if torch.cuda.is_available() else 1)" 2>$null).Trim() -eq "1"

Push-Location $fish
try {
    $args = @(
        "tools\api_server.py",
        "--llama-checkpoint-path", $llamaAbs,
        "--decoder-checkpoint-path", $codecAbs,
        "--listen", $Listen
    )
    if ($useHalf) {
        $args += "--half"
        Write-Host "CUDA indisponivel: arrancando com --half (float16 no CPU)."
    }
    & $py @args
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Pop-Location
}
