# Infraestrutura

## Desenvolvimento local — Postgres + Redis + Evolution Go

Um único comando sobe **PostgreSQL**, **Redis** e o **Evolution Go** (WhatsApp), na mesma rede Docker.

```bash
# Na raiz do repo (equivalente ao comando abaixo):
npm run docker:dev

docker compose -f infra/docker-compose.dev.yml up -d
```

- **Evolution no browser / API:** [http://localhost:8081](http://localhost:8081) (porta definida por `EVOLUTION_HTTP_PORT`, padrão `8081`).
- **Swagger do Evolution Go:** [http://localhost:8081/swagger/index.html](http://localhost:8081/swagger/index.html)
- **Manager (ativação/licença):** [http://localhost:8081/manager/login](http://localhost:8081/manager/login)
- **Chave global da Evolution** (header `apikey` admin): padrão `wa_saas_evolution_dev_key_change_me`. Para mudar, copie `infra/.env.example` → `infra/.env` e defina `EVOLUTION_AUTH_API_KEY`.

No **`backend/.env`**, use a **mesma** chave em:

- `EVOLUTION_API_KEY` (nesta integração, chave global para operações admin e fallback)
- `EVOLUTION_WEBHOOK_API_KEY`  
e `EVOLUTION_BASE_URL=http://127.0.0.1:8081` (API Go no PC) ou `http://evolution:8080` se a API Go também estiver no Docker (perfil `api` já sobrescreve isso no container).

### Bases do Evolution Go no Postgres

Na **primeira** vez que o volume do Postgres é criado, o script `postgres-init/` cria as bases `evogo_auth` e `evogo_users`.

Se já tinhas Postgres **antes** deste script e a Evolution falha a ligar às bases:

```bash
docker exec -it wa-saas-postgres psql -U wa_saas -d postgres -c "CREATE DATABASE evogo_auth;"
docker exec -it wa-saas-postgres psql -U wa_saas -d postgres -c "CREATE DATABASE evogo_users;"
```

### Variáveis opcionais (`infra/.env`)

| Variável | Padrão |
|----------|--------|
| `POSTGRES_USER` | `wa_saas` |
| `POSTGRES_PASSWORD` | `wa_saas_dev_change_me` |
| `POSTGRES_DB` | `wa_saas` |
| `POSTGRES_PORT` | `5432` |
| `REDIS_PORT` | `6379` |
| `EVOLUTION_HTTP_PORT` | `8081` |
| `EVOLUTION_AUTH_API_KEY` | `wa_saas_evolution_dev_key_change_me` |
| `EVOLUTION_AUTH_DB_NAME` | `evogo_auth` |
| `EVOLUTION_USERS_DB_NAME` | `evogo_users` |

### API Go em Docker (opcional)

```bash
docker compose -f infra/docker-compose.dev.yml --profile api up -d --build
```

Requer `backend/.env` válido. Guia geral: `docs/GUIA_CONFIGURACAO_API.md`.

## Só Postgres + Redis (sem Evolution)

Para desenvolver **API Go + Vite + whatsapp-gateway** no host, sem subir Evolution no Docker:

```bash
npm run docker:deps

docker compose -f infra/docker-compose.deps.yml up -d
```

- Mesmos volumes `wa_saas_pgdata` / `wa_saas_redisdata` que o compose principal (se já existirem, reutiliza).
- Guia de arranque: [docs/DEV_LOCAL.md](../docs/DEV_LOCAL.md).
- Tabelas `wa_*` (gateway): se o volume Postgres for antigo, aplica [scripts/apply-wa-gateway-sql.ps1](../scripts/apply-wa-gateway-sql.ps1).

No **`backend/.env`**, usa `WHATSAPP_PROVIDER=none` (ou `baileys`) para não exigir `EVOLUTION_BASE_URL` / `EVOLUTION_API_KEY`. Ver comentários em `backend/.env.example`.

## Script de arranque (Windows)

Na raiz do repositório:

```powershell
.\scripts\dev-local.ps1
```

Opcionalmente `.\scripts\dev-local.ps1 -StartStack` (deps + `npm run dev:stack`; requer `npm install` na raiz). Para subir também a Evolution no Docker: `.\scripts\dev-local.ps1 -FullStack`.
