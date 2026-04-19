#!/usr/bin/env bash
# Dispara redeploy no Coolify (pull + build conforme o projecto).
# 1) No painel Coolify: Settings → Keys & Tokens → API Token (permissão de deploy).
# 2) Copia o UUID do recurso (Applications → a tua app → identificador / uuid).
# 3) Copia este ficheiro para coolify-trigger-deploy.sh, preenche as variáveis e corre no PC ou no VPS.
#
# Exemplo no VPS (Coolify na porta 8000):
#   export COOLIFY_URL='http://127.0.0.1:8000'
#   export COOLIFY_TOKEN='ct_...'
#   export COOLIFY_APP_UUID='sti15flcufforbh39nc0dbul'
#   bash infra/coolify-trigger-deploy.sh

set -euo pipefail
: "${COOLIFY_URL:?defina COOLIFY_URL (ex.: http://127.0.0.1:8000)}"
: "${COOLIFY_TOKEN:?defina COOLIFY_TOKEN (API token do Coolify)}"
: "${COOLIFY_APP_UUID:?defina COOLIFY_APP_UUID (uuid do recurso)}"

BASE="${COOLIFY_URL%/}"
URL="${BASE}/api/v1/deploy?uuid=${COOLIFY_APP_UUID}&force=false"

echo "POST deploy: ${URL}"
curl -fsS -X GET -H "Authorization: Bearer ${COOLIFY_TOKEN}" -H "Accept: application/json" "${URL}"
echo
echo "OK — vê o progresso no painel Coolify (Deployments)."
