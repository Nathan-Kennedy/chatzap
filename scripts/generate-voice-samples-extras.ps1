# Gera WAV em voice-samples/f5-tts, fish-speech, cosyvoice (ver services/tts-extras/README.md).
param(
    [string] $RepoRoot = "",
    [switch] $SkipF5,
    [switch] $SkipFish,
    [switch] $SkipCosy,
    [string] $FishUrl = "http://127.0.0.1:8080",
    [string] $CosyUrl = "http://127.0.0.1:50000",
    [string] $F5Model = "",
    [string] $F5Device = ""
)

$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Split-Path -Parent $here
}

$f5py = Join-Path $RepoRoot "services\tts-extras\venvs\f5-tts\Scripts\python.exe"
$runner = $f5py
if (-not (Test-Path $runner)) {
    $pyCmd = Get-Command python -ErrorAction SilentlyContinue
    if ($pyCmd) { $runner = $pyCmd.Source }
}
if (-not (Test-Path $runner)) {
    Write-Error "Python em falta. Instala F5: scripts\install-tts-extras.ps1 -Components F5"
}

$scriptPy = Join-Path $here "tts_extras_generate_samples.py"
$pyArgs = @(
    $scriptPy,
    "--repo-root", $RepoRoot,
    "--fish-url", $FishUrl,
    "--cosy-url", $CosyUrl
)
if ($SkipF5) { $pyArgs += "--skip-f5" }
if ($SkipFish) { $pyArgs += "--skip-fish" }
if ($SkipCosy) { $pyArgs += "--skip-cosy" }
if (-not [string]::IsNullOrWhiteSpace($F5Model)) { $pyArgs += "--f5-model", $F5Model }
if (-not [string]::IsNullOrWhiteSpace($F5Device)) { $pyArgs += "--f5-device", $F5Device }

& $runner @pyArgs
exit $LASTEXITCODE
