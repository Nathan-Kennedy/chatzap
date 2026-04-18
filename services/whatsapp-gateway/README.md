# WhatsApp Gateway (Baileys)

Serviço Node para parear WhatsApp com **QR no terminal**, com **Postgres** (metadados/auditoria) e **Redis** (pub/sub para a plataforma mostrar QR/estado sem polling pesado).

## Pré-requisitos

- Node 20+
- Postgres e Redis a correr (ex.: `infra/docker-compose.dev.yml` — só `postgres` + `redis`, ou stack completa)

## Base de dados

Na **primeira** criação do volume Postgres, `infra/postgres-init/02-wa-gateway.sql` cria as tabelas. Se o volume já existia antes deste ficheiro, aplica manualmente:

```powershell
Get-Content "infra\postgres-init\02-wa-gateway.sql" -Raw | docker exec -i wa-saas-postgres psql -U wa_saas -d wa_saas
```

## Uso local (recomendado para QR)

Copia `env.example` para `.env` e ajusta URLs se necessário.

```powershell
cd services\whatsapp-gateway
copy env.example .env
npm install
npm run pair
```

Variáveis úteis:

- `DATABASE_URL` — `postgres://wa_saas:...@127.0.0.1:5432/wa_saas?sslmode=disable`
- `REDIS_URL` — `redis://127.0.0.1:6379/2` (DB Redis `2` para não colidir com API `0` e Evolution `1`)
- `WA_INSTANCE_NAME` — nome lógico da instância (ex.: `default`, `tenant-1`)
- `WA_AUTH_DIR` — pasta onde ficam credenciais Baileys (por defeito `services/whatsapp-gateway/data/auth`)

`npm run clean` apaga a sessão da instância atual (equivalente a apagar a subpasta em `WA_AUTH_DIR`).

## HTTP (health)

```powershell
npm run serve
```

- `GET /health` — liveness
- `GET /ready` — Postgres + Redis (se configurados)

## Docker (só health + volume de auth)

```powershell
cd infra
docker compose -f docker-compose.dev.yml --profile gateway up -d --build wa-gateway
```

O pareamento **no terminal** costuma ser feito **no host** (Node local), com `DATABASE_URL`/`REDIS_URL` a apontar para `localhost`, para o QR aparecer na tua consola. O contentor `wa-gateway` mantém o HTTP e o volume `/data/auth` se quiseres correr o pair dentro do Docker mais tarde.

## Redis (plataforma)

Canais (prefixo `WA_REDIS_CHANNEL_PREFIX`, por defeito `wa:gw`):

- `wa:gw:<instance>:qr` — payload com campo `qr` (string)
- `wa:gw:<instance>:connection` — `{ state: 'open'|'close', ... }`
- `wa:gw:broadcast` — todos os eventos com `instance`

A UI pode subscrever com `SUBSCRIBE wa:gw:*` ou canal específico.

## Segurança

- Não commits da pasta de auth (`data/auth` / volume Docker).
- Trata ficheiros de sessão como passwords.

## Roadmap (ponte com a API Go)

- Publicar eventos normalizados (mesmo envelope que a Evolution envia para [backend/internal/handler/webhook.go](../../backend/internal/handler/webhook.go)) a partir deste serviço, **ou** o backend Go subscrever Redis nos canais `wa:gw:*`.
- Até lá: fluxo local descrito em [docs/DEV_LOCAL.md](../../docs/DEV_LOCAL.md) (`WHATSAPP_PROVIDER=none`, webhooks via `curl`).
