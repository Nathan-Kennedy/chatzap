# Arranca Kokoro-FastAPI (Docker) na porta 8880 — alinhado a KOKORO_DEFAULT_BASE_URL / agente.
# Imagem por defeito: CPU. Para GPU: $env:KOKORO_DOCKER_IMAGE="ghcr.io/remsky/kokoro-fastapi-gpu:latest"
$ErrorActionPreference = "Stop"
$Image = if ($env:KOKORO_DOCKER_IMAGE) { $env:KOKORO_DOCKER_IMAGE } else { "ghcr.io/remsky/kokoro-fastapi-cpu:latest" }
$Port = if ($env:KOKORO_HOST_PORT) { $env:KOKORO_HOST_PORT } else { "8880" }
& docker run --rm -p "${Port}:8880" $Image @args
