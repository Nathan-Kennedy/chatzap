# Envio de teste: POST /api/v1/internal/evolution/send (Evolution Go — token da instância no parâmetro -Instance).
# Uso (na raiz do repo):
#   .\scripts\send-internal-evolution.ps1
#   .\scripts\send-internal-evolution.ps1 -Number "556992658179" -Text "Olá" -Instance "uuid-token-da-instancia"
#   .\scripts\send-internal-evolution.ps1 -ApiBase "http://localhost:8088"

param(
    [string]$Number = "556992658179",
    [string]$Text = "Teste Evolution Go",
    [string]$Instance = "",
    [string]$ApiBase = ""
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $root "backend\.env"

if (-not (Test-Path $envFile)) {
    Write-Error "Ficheiro não encontrado: $envFile"
}

$lines = Get-Content $envFile
$internalKey = ($lines | Where-Object { $_ -match '^\s*INTERNAL_API_KEY=' }) -replace '^\s*INTERNAL_API_KEY=', '' | ForEach-Object { $_.Trim() }
if ([string]::IsNullOrWhiteSpace($internalKey)) {
    Write-Error "INTERNAL_API_KEY não definido em backend/.env"
}

$httpPort = "8088"
$portLine = $lines | Where-Object { $_ -match '^\s*HTTP_PORT=' }
if ($portLine) {
    $httpPort = ($portLine -replace '^\s*HTTP_PORT=', '').Trim()
}

if ([string]::IsNullOrWhiteSpace($ApiBase)) {
    $ApiBase = "http://localhost:$httpPort"
}

$instanceToken = $Instance.Trim()
if ([string]::IsNullOrWhiteSpace($instanceToken)) {
    $nameLine = $lines | Where-Object { $_ -match '^\s*EVOLUTION_INSTANCE_NAME=' }
    if ($nameLine) {
        $instanceToken = ($nameLine -replace '^\s*EVOLUTION_INSTANCE_NAME=', '').Trim()
    }
}
if ([string]::IsNullOrWhiteSpace($instanceToken)) {
    Write-Error "Defina -Instance ou EVOLUTION_INSTANCE_NAME em backend/.env (token UUID da instância no Evolution Go)."
}

$uri = "$ApiBase/api/v1/internal/evolution/send"
$headers = @{
    "Content-Type"        = "application/json"
    "X-Internal-API-Key" = $internalKey
}
$bodyObj = @{
    number   = $Number
    text     = $Text
    instance = $instanceToken
}
$body = $bodyObj | ConvertTo-Json -Compress

try {
    $r = Invoke-WebRequest -Method POST -Uri $uri -Headers $headers -Body $body -UseBasicParsing
    Write-Host "Status: $($r.StatusCode)" -ForegroundColor Green
    Write-Host $r.Content
} catch {
    $err = $_.ErrorDetails.Message
    if ($err) { Write-Host $err -ForegroundColor Red }
    throw
}
