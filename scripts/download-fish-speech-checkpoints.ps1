# Descarrega fishaudio/s2-pro (varios GB). Destino: ver CHATBOT_TTS_CACHE_ROOT em services/tts-extras/README.md
param(
    [string] $RepoRoot = ""
)

$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Split-Path -Parent $here
}

. (Join-Path $here "_tts_cache_paths.ps1")
$p = Get-ChatbotTtsCachePaths -RepoRoot $RepoRoot

$fish = Join-Path $RepoRoot "services\tts-extras\vendor\fish-speech"
if (-not (Test-Path (Join-Path $fish ".git"))) {
    Write-Error "Clone fish-speech em falta: $fish. Corre: powershell -File scripts\install-tts-extras.ps1 -Components Fish"
}

$dest = $p.FishS2Pro
New-Item -ItemType Directory -Force -Path $p.HfHome | Out-Null
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $dest) | Out-Null

$f5py = Join-Path $RepoRoot "services\tts-extras\venvs\f5-tts\Scripts\python.exe"
if (-not (Test-Path $f5py)) {
    Write-Error "Python em falta: $f5py. Corre install-tts-extras.ps1 -Components F5"
}

$pip = Join-Path $RepoRoot "services\tts-extras\venvs\f5-tts\Scripts\pip.exe"
& $pip install -q "huggingface_hub"

$env:HF_HOME = $p.HfHome
$env:HF_HUB_CACHE = $p.HfHome
Write-Host "HF_HOME=$($p.HfHome)"
Write-Host "Destino s2-pro=$dest"
Write-Host "A descarregar fishaudio/s2-pro (demorado, varios GB)..."
# sys.argv evita here-strings/-c com aspas que falham no Windows
& $f5py -c "import sys; from huggingface_hub import snapshot_download; d=sys.argv[1]; out=snapshot_download('fishaudio/s2-pro', local_dir=d); print('OK', out)" $dest
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "Checkpoints em: $dest"
Write-Host "Seguinte: npm run fish:server"
