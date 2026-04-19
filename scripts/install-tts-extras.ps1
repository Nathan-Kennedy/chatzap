# Instala Fish Speech, CosyVoice e/ou F5-TTS sob services/tts-extras/ (venvs + vendor).
# Pesos grandes: CosyVoice/Fish requerem download manual ou scripts auxiliares.
param(
    [ValidateSet("F5", "Fish", "Cosy", "All")]
    [string[]] $Components = @("All")
)

$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $here
$base = Join-Path $repoRoot "services\tts-extras"
$assets = Join-Path $base "assets"
. (Join-Path $here "_tts_cache_paths.ps1")
$cachePaths = Get-ChatbotTtsCachePaths -RepoRoot $repoRoot

function Test-HasComponent {
    param([string] $Name)
    if ($Components -contains "All") { return $true }
    return $Components -contains $Name
}

function Ensure-Dir([string] $Path) {
    if (-not (Test-Path $Path)) { New-Item -ItemType Directory -Force -Path $Path | Out-Null }
}

function Get-PyLauncher {
    $pyCmd = Get-Command py -ErrorAction SilentlyContinue
    if ($pyCmd) {
        foreach ($ver in @("3.12", "3.11", "3.10")) {
            $code = & py "-$ver" -c "import sys; print(sys.executable)" 2>$null
            if ($LASTEXITCODE -eq 0 -and $code) { return @{ Launcher = "py"; Version = $ver; Executable = $code.Trim() } }
        }
    }
    $ex = (Get-Command python -ErrorAction SilentlyContinue).Source
    if ($ex) { return @{ Launcher = $null; Version = $null; Executable = $ex } }
    return $null
}

Ensure-Dir $base
Ensure-Dir $assets
Ensure-Dir $cachePaths.HfHome

$pyInfo = Get-PyLauncher
if (-not $pyInfo) {
    Write-Error "Python 3.10+ nao encontrado (instala Python ou o launcher 'py')."
}

Write-Host "Python: $($pyInfo.Executable)"

# --- Assets partilhados (Cosy cross_lingual / zero-shot) ---
$cross = Join-Path $assets "cross_lingual_prompt.wav"
if (-not (Test-Path $cross)) {
    Write-Host "A descarregar cross_lingual_prompt.wav (CosyVoice)..."
    curl.exe -sL -o $cross "https://raw.githubusercontent.com/FunAudioLLM/CosyVoice/main/asset/cross_lingual_prompt.wav"
    if (-not (Test-Path $cross) -or (Get-Item $cross).Length -lt 1000) {
        Write-Warning "Download do prompt Cosy falhou; copia manualmente de CosyVoice/asset/"
    }
}

if (Test-HasComponent "F5") {
    $venv = Join-Path $base "venvs\f5-tts"
    Ensure-Dir (Split-Path -Parent $venv)
    if (-not (Test-Path (Join-Path $venv "Scripts\python.exe"))) {
        if ($pyInfo.Launcher) {
            & py "-$($pyInfo.Version)" -m venv $venv
        }
        else {
            & python -m venv $venv
        }
    }
    $pip = Join-Path $venv "Scripts\pip.exe"
    & $pip install --upgrade pip
    Write-Host "pip install f5-tts (pode demorar)..."
    & $pip install "f5-tts"
    # torchaudio 2.9+ usa TorchCodec no load(); no Windows exige FFmpeg "full-shared" + DLLs.
    if ($IsWindows) {
        Write-Host "Windows: fixar torch/torchaudio 2.8.x para carregar WAV sem TorchCodec; remover torchcodec se existir..."
        & $pip install --force-reinstall "torch==2.8.0" "torchaudio==2.8.0"
        $null = & $pip uninstall -y torchcodec 2>$null
    }
    Write-Host "F5-TTS OK -> $venv"
}

if (Test-HasComponent "Fish") {
    $vendor = Join-Path $base "vendor\fish-speech"
    if (-not (Test-Path (Join-Path $vendor ".git"))) {
        Write-Host "git clone fish-speech (shallow)..."
        Ensure-Dir (Split-Path -Parent $vendor)
        git clone --depth 1 "https://github.com/fishaudio/fish-speech.git" $vendor
    }
    $venv = Join-Path $base "venvs\fish-speech"
    if (-not (Test-Path (Join-Path $venv "Scripts\python.exe"))) {
        $fishPy = "3.12"
        if ($pyInfo.Launcher) {
            $null = & py "-$fishPy" -c "1" 2>$null
            if ($LASTEXITCODE -ne 0) { $fishPy = $pyInfo.Version }
            & py "-$fishPy" -m venv $venv
        }
        else {
            & python -m venv $venv
        }
    }
    $pip = Join-Path $venv "Scripts\pip.exe"
    & $pip install --upgrade pip
    Write-Host 'Fish Speech: tentativa pip install -e .[cpu] (Linux/WSL recomendado; Windows pode falhar)...'
    Push-Location $vendor
    try {
        & $pip install -e '.[cpu]'
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "pip Fish saiu com codigo $LASTEXITCODE. Tenta Docker/WSL - services/tts-extras/README.md"
        }
    }
    finally { Pop-Location }
    Write-Host "Fish clone -> $vendor ; venv -> $venv"
    Write-Host 'AVISO: checkpoints s2-pro - npm run fish:download-checkpoints (CHATBOT_TTS_CACHE_ROOT ou vendor\fish-speech\checkpoints; ver services/tts-extras/README.md)'
}

if (Test-HasComponent "Cosy") {
    $vendor = Join-Path $base "vendor\CosyVoice"
    if (-not (Test-Path (Join-Path $vendor ".git"))) {
        Write-Host 'git clone CosyVoice --recursive (demorado; submodulos necessarios)...'
        Ensure-Dir (Split-Path -Parent $vendor)
        git clone --recursive "https://github.com/FunAudioLLM/CosyVoice.git" $vendor
    }
    else {
        Write-Host "CosyVoice ja existe; git submodule update..."
        Push-Location $vendor
        try { git submodule update --init --recursive } finally { Pop-Location }
    }

    $venv = Join-Path $base "venvs\cosyvoice"
    if (-not (Test-Path (Join-Path $venv "Scripts\python.exe"))) {
        $cvPy = if ($pyInfo.Version -and [version]$pyInfo.Version -ge [version]"3.10") { $pyInfo.Version } else { "3.10" }
        if ($pyInfo.Launcher) {
            & py "-$cvPy" -m venv $venv
        }
        else {
            & python -m venv $venv
        }
    }
    $pip = Join-Path $venv "Scripts\pip.exe"
    $venvPy = Join-Path $venv "Scripts\python.exe"
    & $venvPy -m pip install --upgrade pip wheel
    # openai-whisper (Cosy): build isolado com setuptools 80+ falha (pkg_resources).
    & $venvPy -m pip install "setuptools==69.5.1"
    & $venvPy -m pip install "openai-whisper==20231117" --no-build-isolation
    Write-Host 'pip install -r CosyVoice/requirements.txt (muito grande; pode falhar no Windows)...'
    $req = Join-Path $vendor "requirements.txt"
    & $pip install -r $req
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "CosyVoice pip saiu com codigo $LASTEXITCODE. Tenta WSL/Docker."
    }
    Write-Host "CosyVoice clone -> $vendor"
    Write-Host "Seguinte: powershell -File scripts\download-cosyvoice-model.ps1"
}

Write-Host "Concluido. Le services/tts-extras/README.md"
