# Kokoro-FastAPI no PC (sem Docker) - CPU via uv, porta 8880.
# Requisitos: Git, https://docs.astral.sh/uv/ no PATH.
# Opcional: eSpeak NG (https://github.com/espeak-ng/espeak-ng) - o start-cpu.ps1 upstream aponta para a DLL tipica.
#
# Pasta do clone (por defeito: pasta irma do repo ChatBot, ex. F:\Projetos Dev\Kokoro-FastAPI):
#   $env:KOKORO_FASTAPI_DIR = "D:\src\Kokoro-FastAPI"
#   npm run kokoro:server:native

$ErrorActionPreference = "Stop"

function Require-Cmd($name) {
    if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
        Write-Host ""
        Write-Host "Comando '$name' nao encontrado no PATH." -ForegroundColor Yellow
        exit 1
    }
}

function Resolve-UvExe {
    if ($env:UV_EXE -and (Test-Path -LiteralPath $env:UV_EXE)) { return $env:UV_EXE }
    $cmd = Get-Command uv -ErrorAction SilentlyContinue
    if ($cmd -and $cmd.Source) { return $cmd.Source }
    $home = $env:USERPROFILE
    if (-not $home) { return $null }
    $candidates = @(
        (Join-Path $home ".local\bin\uv.exe"),
        (Join-Path $home ".local\bin\uv.cmd"),
        (Join-Path $home ".cargo\bin\uv.exe"),
        (Join-Path $env:LOCALAPPDATA "Programs\uv\uv.exe")
    )
    foreach ($p in $candidates) {
        if ($p -and (Test-Path -LiteralPath $p)) { return $p }
    }
    return $null
}

Require-Cmd git

$uvExe = Resolve-UvExe
if (-not $uvExe) {
    Write-Host ""
    Write-Host "uv nao encontrado. Instala (PowerShell):" -ForegroundColor Yellow
    Write-Host '  powershell -ExecutionPolicy Bypass -c "irm https://astral.sh/uv/install.ps1 | iex"' -ForegroundColor White
    Write-Host "Depois fecha e reabre o terminal (ou acrescenta %USERPROFILE%\.local\bin ao PATH)." -ForegroundColor DarkGray
    Write-Host "Ou define UV_EXE ao caminho completo de uv.exe" -ForegroundColor DarkGray
    Write-Host ""
    exit 1
}
$uvDir = Split-Path $uvExe -Parent
if ($uvDir) {
    $env:PATH = "$uvDir;$env:PATH"
}
Write-Host "uv: $uvExe" -ForegroundColor DarkGray

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$DevRoot = Split-Path $RepoRoot -Parent
$DefaultDir = Join-Path $DevRoot "Kokoro-FastAPI"
$KokoroDir = if ($env:KOKORO_FASTAPI_DIR) { $env:KOKORO_FASTAPI_DIR.TrimEnd('\', '/') } else { $DefaultDir }

if (-not (Test-Path -LiteralPath $KokoroDir)) {
    Write-Host "Clonando remsky/Kokoro-FastAPI -> $KokoroDir" -ForegroundColor Cyan
    New-Item -ItemType Directory -Path (Split-Path $KokoroDir -Parent) -Force | Out-Null
    git clone https://github.com/remsky/Kokoro-FastAPI.git $KokoroDir
}

if (-not (Test-Path -LiteralPath (Join-Path $KokoroDir "pyproject.toml"))) {
    Write-Host "pyproject.toml nao encontrado em $KokoroDir (clone incompleto?)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Kokoro-FastAPI (nativo, CPU) em: $KokoroDir" -ForegroundColor Cyan
Write-Host "URL: http://127.0.0.1:8880  (docs: /docs , UI: /web)" -ForegroundColor Cyan
Write-Host "Primeira execucao: uv sync + modelo - pode demorar (torch/kokoro)." -ForegroundColor DarkGray
Write-Host ""

Push-Location $KokoroDir
try {
    if ($env:KOKORO_RESET_VENV -eq "1" -and (Test-Path -LiteralPath ".venv")) {
        Write-Host "KOKORO_RESET_VENV=1: a remover .venv ..." -ForegroundColor Yellow
        Remove-Item -LiteralPath ".venv" -Recurse -Force
    }

    # Mesmas variaveis que start-cpu.ps1 upstream (alinhado ao repo Kokoro-FastAPI).
    $env:PYTHONUTF8 = "1"
    $env:PROJECT_ROOT = $PWD.Path
    $env:USE_GPU = "false"
    $env:USE_ONNX = "false"
    $env:PYTHONPATH = "$($env:PROJECT_ROOT);$($env:PROJECT_ROOT)\api"
    $env:MODEL_DIR = "src/models"
    $env:VOICES_DIR = "src/voices/v1_0"
    $env:WEB_PLAYER_PATH = "$($env:PROJECT_ROOT)\web"
    $espeakDll = "C:\Program Files\eSpeak NG\libespeak-ng.dll"
    if (Test-Path -LiteralPath $espeakDll) {
        $env:PHONEMIZER_ESPEAK_LIBRARY = $espeakDll
    }

    # uv 0.11+: `uv pip install` sem venv falha; usar `uv sync` com extra cpu (pyproject do Kokoro-FastAPI).
    Write-Host "uv sync --extra cpu ..." -ForegroundColor DarkGray
    & $uvExe sync --extra cpu
    Write-Host "download_model.py ..." -ForegroundColor DarkGray
    & $uvExe run python docker/scripts/download_model.py --output api/src/models/v1_0
    Write-Host "uvicorn :8880 ..." -ForegroundColor DarkGray
    & $uvExe run uvicorn api.src.main:app --host 0.0.0.0 --port 8880
} finally {
    Pop-Location
}
