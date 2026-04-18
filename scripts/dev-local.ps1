# Dev local: sobe Postgres+Redis (deps), opcionalmente aplica SQL wa_*, e indica como correr API + web.
# Uso:
#   .\scripts\dev-local.ps1              # docker deps + instruções
#   .\scripts\dev-local.ps1 -StartStack  # deps + npm run dev:stack (raiz do repo, após npm install)
#   .\scripts\dev-local.ps1 -SkipDocker  # só mensagens / stack
#   .\scripts\dev-local.ps1 -FullStack   # docker-compose.dev.yml (Postgres+Redis+Evolution) + instruções
param(
  [switch]$SkipDocker,
  [switch]$StartStack,
  [switch]$FullStack
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

if (-not $SkipDocker) {
  Push-Location (Join-Path $root "infra")
  if ($FullStack) {
    Write-Host "A subir Postgres + Redis + Evolution (docker-compose.dev.yml)..." -ForegroundColor Cyan
    docker compose -f docker-compose.dev.yml up -d
  } else {
    Write-Host "A subir Postgres + Redis (docker-compose.deps.yml)..." -ForegroundColor Cyan
    docker compose -f docker-compose.deps.yml up -d
  }
  Pop-Location
  Write-Host "Aguarda ~8s pelos healthchecks..." -ForegroundColor DarkGray
  Start-Sleep -Seconds 8
  Write-Host "Se o volume Postgres for antigo, corre: .\scripts\apply-wa-gateway-sql.ps1" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "1) Copia backend/.env.example -> backend/.env (perfil none ja permite API sem Evolution)" -ForegroundColor Green
Write-Host "   e apps/web/.env — ver docs/DEV_LOCAL.md" -ForegroundColor Green
Write-Host "2) Stack API + Vite (precisa Go + npm install na raiz):" -ForegroundColor Green
Write-Host "     cd `"$root`"" -ForegroundColor White
Write-Host "     npm install" -ForegroundColor White
Write-Host "     npm run dev:stack" -ForegroundColor White
Write-Host "3) WhatsApp QR (opcional): cd services\whatsapp-gateway ; npm run pair" -ForegroundColor Green
Write-Host ""

if ($StartStack) {
  Set-Location $root
  if (-not (Test-Path "node_modules\concurrently")) {
    npm install
  }
  npm run dev:stack
}
