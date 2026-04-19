# Descarrega um modelo CosyVoice (Hugging Face). Pasta base: CHATBOT_TTS_CACHE_ROOT (ver README tts-extras).
param(
    [ValidateSet("CosyVoice2-0.5B", "Fun-CosyVoice3-0.5B", "CosyVoice-300M")]
    [string] $Model = "CosyVoice2-0.5B"
)

$ErrorActionPreference = "Stop"
function Ensure-Dir([string] $Path) {
    if (-not (Test-Path $Path)) { New-Item -ItemType Directory -Force -Path $Path | Out-Null }
}
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $here
. (Join-Path $here "_tts_cache_paths.ps1")
$p = Get-ChatbotTtsCachePaths -RepoRoot $repoRoot

$venv = Join-Path $repoRoot "services\tts-extras\venvs\cosyvoice"
$py = Join-Path $venv "Scripts\python.exe"
if (-not (Test-Path $py)) {
    Write-Error "Venv Cosy em falta. Corre primeiro: scripts\install-tts-extras.ps1 -Components Cosy"
}

$hfId = switch ($Model) {
    "CosyVoice2-0.5B" { "FunAudioLLM/CosyVoice2-0.5B" }
    "Fun-CosyVoice3-0.5B" { "FunAudioLLM/Fun-CosyVoice3-0.5B-2512" }
    "CosyVoice-300M" { "FunAudioLLM/CosyVoice-300M" }
}
$dest = Join-Path $p.CosyModels $Model
Ensure-Dir (Split-Path -Parent $dest)
Ensure-Dir $p.HfHome

$env:HF_HOME = $p.HfHome
$env:HF_HUB_CACHE = $p.HfHome

# transformers/tokenizers do Cosy exigem huggingface_hub < 1.0
& (Join-Path $venv "Scripts\pip.exe") install -U "huggingface_hub>=0.23,<1.0"

& $py -c "import sys; from huggingface_hub import snapshot_download; snapshot_download(sys.argv[1], local_dir=sys.argv[2]); print('OK', sys.argv[2])" $hfId $dest
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "Modelo em: $dest"
Write-Host "Arranque: npm run cosyvoice:server"
