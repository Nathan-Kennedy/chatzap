# Arranca omnivoice-server na porta 8000 (alinhado a OMNIVOICE_DEFAULT_BASE_URL / agente).
# Usa run_patched.py (fix torch.isnan vs numpy no health-check do servidor upstream).
$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$Py = Join-Path $RepoRoot "services\omnivoice-server\venv\Scripts\python.exe"
$Runner = Join-Path $RepoRoot "services\omnivoice-server\run_patched.py"
if (-not (Test-Path $Py)) {
    Write-Error "Python venv não encontrado em services/omnivoice-server/venv. Segue o README nessa pasta."
    exit 1
}
if (-not (Test-Path $Runner)) {
    Write-Error "run_patched.py em falta em services/omnivoice-server/"
    exit 1
}
# Perfis de clone: coloca WAV em services/omnivoice-server/voice-profiles/<id>/ref_audio.wav
$ProfileDir = Join-Path $RepoRoot "services\omnivoice-server\voice-profiles"
# CFG mais baixo que o default (2.0): clone menos “preso” à prosódia da ref. (entonação mais neutra).
$env:OMNIVOICE_GUIDANCE_SCALE = "1.5"
# Por defeito CPU: evita falha no CUDA no Windows (erro 1455 / paging file / VRAM).
# GPU: $env:OMNIVOICE_DEVICE = "cuda" ou npm run omnivoice:server -- --device cuda
$Device = "cpu"
if ($env:OMNIVOICE_DEVICE -and $env:OMNIVOICE_DEVICE.Trim() -ne "") {
    $Device = $env:OMNIVOICE_DEVICE.Trim()
}
& $Py $Runner --host 127.0.0.1 --port 8000 --device $Device --profile-dir $ProfileDir @args
