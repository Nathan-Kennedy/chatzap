# Arranca Kokoro-FastAPI (Docker) na porta 8880 — alinhado a KOKORO_DEFAULT_BASE_URL / agente.
# Imagem por defeito: CPU. Para GPU: $env:KOKORO_DOCKER_IMAGE="ghcr.io/remsky/kokoro-fastapi-gpu:latest"
$ErrorActionPreference = "Stop"
$Image = if ($env:KOKORO_DOCKER_IMAGE) { $env:KOKORO_DOCKER_IMAGE } else { "ghcr.io/remsky/kokoro-fastapi-cpu:latest" }
$Port = if ($env:KOKORO_HOST_PORT) { $env:KOKORO_HOST_PORT } else { "8880" }

function Resolve-DockerExe {
    if ($env:DOCKER_EXE -and (Test-Path -LiteralPath $env:DOCKER_EXE)) { return $env:DOCKER_EXE }
    $cmd = Get-Command docker -ErrorAction SilentlyContinue
    if ($cmd -and $cmd.Source) { return $cmd.Source }
    $candidates = @(
        (Join-Path $env:ProgramFiles "Docker\Docker\resources\bin\docker.exe"),
        (Join-Path ${env:ProgramFiles(x86)} "Docker\Docker\resources\bin\docker.exe")
    )
    foreach ($p in $candidates) {
        if ($p -and (Test-Path -LiteralPath $p)) { return $p }
    }
    return $null
}

$dockerExe = Resolve-DockerExe
if (-not $dockerExe) {
    Write-Host ""
    Write-Host "Docker nao foi encontrado no PATH nem em Program Files\Docker\..." -ForegroundColor Yellow
    Write-Host "  1) Instala o Docker Desktop para Windows: https://docs.docker.com/desktop/install/windows-install/"
    Write-Host "  2) Abre o Docker Desktop e espera ficar a correr (icone verde)."
    Write-Host "  3) Fecha e volta a abrir este terminal, ou define DOCKER_EXE ao caminho completo de docker.exe"
    Write-Host ""
    exit 1
}

Write-Host "Kokoro-FastAPI: imagem=$Image porta host=$Port -> http://127.0.0.1:${Port}" -ForegroundColor Cyan
& $dockerExe run --rm -p "${Port}:8880" $Image @args
