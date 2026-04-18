# Sobe Postgres + Redis + Evolution + API Go (rede interna com webhook http://api:8080).
# Uso (na raiz do repo):
#   powershell -File .\scripts\docker-dev-up.ps1
#   powershell -File .\scripts\docker-dev-up.ps1 -Rebuild   # após mudanças no código Go (equivale a npm run docker:dev:api:rebuild)
# Depois: na app web → Instâncias → sincronizar webhook.

param(
    [switch]$Rebuild
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root

$compose = Join-Path $Root "infra\docker-compose.dev.yml"
if (-not (Test-Path $compose)) {
    Write-Error "Ficheiro não encontrado: $compose"
}

if ($Rebuild) {
    Write-Host ">> build --no-cache api + up --force-recreate api" -ForegroundColor Cyan
    docker compose -f infra/docker-compose.dev.yml --profile api build --no-cache api
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    docker compose -f infra/docker-compose.dev.yml --profile api up -d --force-recreate api
} else {
    Write-Host ">> docker compose -f infra/docker-compose.dev.yml --profile api up -d --build" -ForegroundColor Cyan
    docker compose -f infra/docker-compose.dev.yml --profile api up -d --build
}

if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

Write-Host ""
Write-Host "Stack no ar. Próximos passos:" -ForegroundColor Green
Write-Host "  1. API (Docker): http://127.0.0.1:8080/health — ou a porta em API_HTTP_PORT no infra/.env"
Write-Host "  2. Evolution: http://127.0.0.1:8081 (ou EVOLUTION_HTTP_PORT no infra/.env)"
Write-Host "  3. Se a API corre no PC com go run: npm run docker:dev (sem perfil api) + npm run dev:stack; depois sincronizar webhook"
Write-Host "  4. Se a API é só no Docker: npm run dev:web e apontar VITE_API_BASE_URL para a porta da API"
Write-Host "  5. Na app: Instâncias → sincronizar webhook (obrigatório após mudar URL/rede)"
Write-Host ""
