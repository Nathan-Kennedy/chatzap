# Aplica infra/postgres-init/02-wa-gateway.sql ao Postgres do compose (volume já existente).
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$sql = Join-Path $root "infra\postgres-init\02-wa-gateway.sql"
Get-Content $sql -Raw | docker exec -i wa-saas-postgres psql -U wa_saas -d wa_saas
Write-Host "OK: tabelas wa_* aplicadas (ou já existiam)."
