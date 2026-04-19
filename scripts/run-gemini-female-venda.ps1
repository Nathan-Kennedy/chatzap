# Gera WAV venda-casa (vozes femininas Gemini). Aceita argumentos extra do Python, ex.:
#   .\scripts\run-gemini-female-venda.ps1 --gemini-voices Kore
#   npm run voice-samples:paid:gemini-female-venda -- --gemini-voices Kore
$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$repo = Split-Path -Parent $here
Set-Location -LiteralPath $repo

$env:PYTHONUNBUFFERED = "1"
$script = Join-Path $here "generate-voice-samples-paid.py"

$runner = $null
if (Get-Command py -ErrorAction SilentlyContinue) {
    $null = & py -3 -c "import sys; sys.exit(0)" 2>$null
    if ($LASTEXITCODE -eq 0) { $runner = @{ Cmd = "py"; Args = @("-3", $script, "--gemini-female-venda-casa") } }
}
if (-not $runner -and (Get-Command python -ErrorAction SilentlyContinue)) {
    $runner = @{ Cmd = (Get-Command python).Source; Args = @($script, "--gemini-female-venda-casa") }
}
if (-not $runner) {
    Write-Error "Python nao encontrado. Instala Python 3 ou o launcher 'py' e tenta de novo."
}

$allArgs = $runner.Args + $args
Write-Host "Repo: $repo"
Write-Host "Comando: $($runner.Cmd) $($allArgs -join ' ')"
& $runner.Cmd @allArgs
exit $LASTEXITCODE
