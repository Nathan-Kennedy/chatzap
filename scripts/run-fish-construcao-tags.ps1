# Gera voice-samples/fish-speech/construcao_pt_br_tags.wav a partir de INPUT_CONSTRUCAO_PT_BR_TAGS.txt
$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$repo = Split-Path -Parent $here
$f5py = Join-Path $repo "services\tts-extras\venvs\f5-tts\Scripts\python.exe"
if (-not (Test-Path $f5py)) {
    Write-Error "Venv F5 em falta (requests). Corre install-tts-extras.ps1 -Components F5"
}
$script = Join-Path $here "fish_speech_post_tts.py"
$txt = Join-Path $repo "voice-samples\fish-speech\INPUT_CONSTRUCAO_PT_BR_TAGS.txt"
$out = Join-Path $repo "voice-samples\fish-speech\construcao_pt_br_tags.wav"
& $f5py $script --url "http://127.0.0.1:8080/v1/tts" --text-file $txt --out $out
exit $LASTEXITCODE
