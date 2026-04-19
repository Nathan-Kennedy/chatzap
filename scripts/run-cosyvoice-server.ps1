# Arranca CosyVoice FastAPI (porta 50000). Modelo na pasta de pretrained de Get-ChatbotTtsCachePaths.
param(
    [string] $RepoRoot = "",
    [ValidateSet("CosyVoice2-0.5B", "Fun-CosyVoice3-0.5B", "CosyVoice-300M")]
    [string] $Model = "CosyVoice2-0.5B",
    [int] $Port = 50000
)

$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Split-Path -Parent $here
}

. (Join-Path $here "_tts_cache_paths.ps1")
$p = Get-ChatbotTtsCachePaths -RepoRoot $RepoRoot

$vendor = Join-Path $RepoRoot "services\tts-extras\vendor\CosyVoice"
$py = Join-Path $RepoRoot "services\tts-extras\venvs\cosyvoice\Scripts\python.exe"
$fastapiDir = Join-Path $vendor "runtime\python\fastapi"
$modelDir = Join-Path $p.CosyModels $Model

if (-not (Test-Path $py)) { Write-Error "Venv cosy em falta. install-tts-extras.ps1 -Components Cosy" }
if (-not (Test-Path $fastapiDir)) { Write-Error "CosyVoice clone em falta: $vendor" }
if (-not (Test-Path $modelDir)) {
    Write-Error "Modelo em falta: $modelDir. Corre npm run cosyvoice:download-model"
}

$modelDirAbs = (Resolve-Path $modelDir).Path
Push-Location $fastapiDir
try {
    & $py "server.py" "--port" $Port "--model_dir" $modelDirAbs
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Pop-Location
}
