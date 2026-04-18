# Pareamento WhatsApp (Baileys + Postgres/Redis opcional). Uso: .\scripts\whatsapp-pair.ps1 [nome-instancia]
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location (Join-Path $root "services\whatsapp-gateway")
if (-not (Test-Path "node_modules")) {
  npm install
}
if (Test-Path ".env") {
  # dotenv carrega .env no cli-pair
} else {
  Write-Host "Dica: copia env.example para .env com DATABASE_URL e REDIS_URL (opcional)." -ForegroundColor Yellow
}
$arg = $args[0]
if ($arg) {
  npm run pair -- $arg
} else {
  npm run pair
}
