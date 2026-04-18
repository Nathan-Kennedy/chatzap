# Smoke test POST /v1/audio/speech (omnivoice-server).
# Evita json_invalid no PowerShell: corpo vem de ficheiro UTF-8 ou bytes UTF-8 explícitos.
param(
    [string] $BaseUrl = "http://127.0.0.1:8000",
    [string] $OutPath = ""
)

$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$bodyPath = Join-Path $here "omnivoice-smoke-body.json"
if (-not (Test-Path $bodyPath)) {
    Write-Error "Ficheiro em falta: $bodyPath"
}
$uri = ($BaseUrl.TrimEnd("/")) + "/v1/audio/speech"

# 1) curl com corpo a partir de ficheiro (recomendado no Windows)
$code = curl.exe -s -o NUL -w "%{http_code}" -X POST $uri -H "Content-Type: application/json; charset=utf-8" --data-binary "@$bodyPath"
if ($code -ne "200") {
    Write-Host "curl falhou (HTTP $code). Detalhe:"
    curl.exe -s -D - -X POST $uri -H "Content-Type: application/json; charset=utf-8" --data-binary "@$bodyPath"
    exit 1
}

if ($OutPath -ne "") {
    curl.exe -s -o $OutPath -X POST $uri -H "Content-Type: application/json; charset=utf-8" --data-binary "@$bodyPath"
    Write-Host "OK HTTP 200 -> $OutPath"
} else {
    Write-Host "OK HTTP 200 (curl + ficheiro JSON)"
}
