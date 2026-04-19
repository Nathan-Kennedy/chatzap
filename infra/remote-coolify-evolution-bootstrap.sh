#!/usr/bin/env bash
# Executar no VPS (não contém passwords): cria BD Evolution e sobe contentor na rede coolify.
# Uso: scp + ssh, ou colar no servidor. Ajusta PG_CONTAINER e EVOLUTION_API_KEY antes se quiseres.

set -euo pipefail
PG_CONTAINER="${PG_CONTAINER:-ylw5kcg46rpwon6ynq1f20v5}"
EVOLUTION_NAME="${EVOLUTION_NAME:-wa-saas-evolution}"
EVOLUTION_IMAGE="${EVOLUTION_IMAGE:-evoapicloud/evolution-go:latest}"
NETWORK="${NETWORK:-coolify}"
PG_HOST="${PG_HOST:-$PG_CONTAINER}"
# Porta no host para abrir o Manager no browser (licença Evolution Go) — http://IP_DO_VPS:8081/manager/login
EVOLUTION_HOST_PORT="${EVOLUTION_HOST_PORT:-8081}"

if ! docker ps --format '{{.Names}}' | grep -qx "$PG_CONTAINER"; then
  echo "Contentor Postgres não encontrado: $PG_CONTAINER" >&2
  echo "Lista: docker ps --format '{{.Names}}'" >&2
  exit 1
fi

docker exec "$PG_CONTAINER" sh -c 'export PGPASSWORD="$POSTGRES_PASSWORD"; psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE evogo_auth;"' 2>/dev/null || echo "evogo_auth já existe ou criada."
docker exec "$PG_CONTAINER" sh -c 'export PGPASSWORD="$POSTGRES_PASSWORD"; psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE evogo_users;"' 2>/dev/null || echo "evogo_users já existe ou criada."

if docker ps -a --format '{{.Names}}' | grep -qx "$EVOLUTION_NAME"; then
  echo "Remover contentor antigo $EVOLUTION_NAME..."
  docker rm -f "$EVOLUTION_NAME" >/dev/null
fi

if [[ -z "${EVOLUTION_GLOBAL_API_KEY:-}" ]]; then
  EVOLUTION_GLOBAL_API_KEY="$(openssl rand -hex 24)"
  echo "Gerado EVOLUTION_GLOBAL_API_KEY (guarda no Coolify da API): $EVOLUTION_GLOBAL_API_KEY"
else
  echo "A usar EVOLUTION_GLOBAL_API_KEY definido no ambiente."
fi

PW="$(docker exec "$PG_CONTAINER" printenv POSTGRES_PASSWORD)"
USER="$(docker exec "$PG_CONTAINER" printenv POSTGRES_USER)"
ENC_PW="$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=''))" "$PW")"
AUTH_URL="postgresql://${USER}:${ENC_PW}@${PG_HOST}:5432/evogo_auth?sslmode=disable"
USERS_URL="postgresql://${USER}:${ENC_PW}@${PG_HOST}:5432/evogo_users?sslmode=disable"

docker pull "$EVOLUTION_IMAGE"
docker run -d \
  --name "$EVOLUTION_NAME" \
  --restart unless-stopped \
  --network "$NETWORK" \
  -p "${EVOLUTION_HOST_PORT}:8080" \
  -e SERVER_PORT=8080 \
  -e "CLIENT_NAME=${EVOLUTION_CLIENT_NAME:-wa_saas_coolify}" \
  -e "GLOBAL_API_KEY=$EVOLUTION_GLOBAL_API_KEY" \
  -e "POSTGRES_AUTH_DB=$AUTH_URL" \
  -e "POSTGRES_USERS_DB=$USERS_URL" \
  -e DATABASE_SAVE_MESSAGES=false \
  -e WADEBUG=INFO \
  -e LOGTYPE=console \
  "$EVOLUTION_IMAGE"

if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
  echo "A permitir TCP ${EVOLUTION_HOST_PORT} no UFW (Manager Evolution)…"
  ufw allow "${EVOLUTION_HOST_PORT}/tcp" comment 'evolution-manager' >/dev/null || true
fi

echo "Evolution a correr. URL interna para a API: http://${EVOLUTION_NAME}:8080"
echo "Manager no browser (activar licença): http://<IP_DO_VPS>:${EVOLUTION_HOST_PORT}/manager/login"
echo "  API URL no manager: http://<IP_DO_VPS>:${EVOLUTION_HOST_PORT}  |  Token: GLOBAL_API_KEY (igual Coolify EVOLUTION_API_KEY)"
echo "No Coolify (serviço API): WHATSAPP_PROVIDER=evolution"
echo "EVOLUTION_BASE_URL=http://${EVOLUTION_NAME}:8080"
echo "EVOLUTION_API_KEY e EVOLUTION_WEBHOOK_API_KEY = o mesmo GLOBAL_API_KEY acima."
