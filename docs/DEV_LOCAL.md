# Desenvolvimento local integrado (sem Docker em tudo)

Objetivo: **Vite** + **API Go** + **Postgres** + **Redis** + opcionalmente **whatsapp-gateway** (Baileys), sem obrigar o contentor da Evolution.

## Fluxo recomendado: `dev:stack` + Evolution no Docker

1. **Não** subas a API no Docker (`--profile api` / `wa-saas-api`) — a API é o `go run` do `npm run dev:stack`. Se tiveres o contentor `wa-saas-api` a correr, para-o (`docker stop wa-saas-api`) para libertar a porta 8080 e evitar confusão com webhooks.
2. Sobe dados + Evolution: `npm run docker:dev:stack` (ou `npm run docker:dev` — é o mesmo compose **sem** perfil `api`).
3. `backend/.env`: `WHATSAPP_PROVIDER=evolution`, `EVOLUTION_BASE_URL=http://127.0.0.1:8081`, chaves alinhadas ao `GLOBAL_API_KEY` da Evolution, **`HTTP_PORT` igual à porta no `apps/web/.env`** (ex. 8088), e `PUBLIC_WEBHOOK_BASE_URL=http://host.docker.internal:8088` (troca `8088` se mudares a porta). **Envio de ficheiros/áudio na Inbox:** a Evolution precisa de fazer `GET` em URLs públicas (`GET /media/temp/:token`). Por defeito usa-se `PUBLIC_MEDIA_BASE_URL` = `PUBLIC_WEBHOOK_BASE_URL`; mantém o mesmo host/porta acessível a partir do contentor (`host.docker.internal`). Ficheiros temporários: `MEDIA_UPLOAD_DIR` (ex. `.tmp/media_uploads` no diretório de trabalho da API).
4. `apps/web/.env`: `VITE_API_BASE_URL` e `VITE_WS_URL` na **mesma** porta que o passo 3.
5. Na raiz: `npm run dev:stack` (ou tudo de uma vez: `npm run dev:stack:whatsapp`).
6. Na app: **Instâncias → sincronizar webhook** (regista o URL com o `PUBLIC_WEBHOOK_BASE_URL` actual).

Atalho único (Docker deps + API + Vite): `npm run dev:stack:whatsapp`.

## 1. Dependências de dados

**Opção A — só Postgres e Redis (recomendado):**

```powershell
npm run docker:deps
```

(Equivalente: `cd infra` e `docker compose -f docker-compose.deps.yml up -d`.)

**Opção B — stack completa** (Postgres + Redis + Evolution): `npm run docker:dev` na raiz. Ajusta `backend/.env` para `WHATSAPP_PROVIDER=evolution` e `EVOLUTION_BASE_URL=http://127.0.0.1:8081` (e as chaves alinhadas a `EVOLUTION_AUTH_API_KEY`).

**Opção C — tudo no Docker** (inclui API Go): `npm run docker:dev:api` ou `powershell -File .\scripts\docker-dev-up.ps1`. A API fica em `http://127.0.0.1:8080` (o compose força `HTTP_PORT=8080` e `PUBLIC_WEBHOOK_BASE_URL=http://api:8080`). Depois: Instâncias → sincronizar webhook.

Na **primeira** vez com Postgres novo, os scripts em `postgres-init/` criam as bases `evogo_auth`/`evogo_users` e as tabelas `wa_*`. Se o volume já existia antes do `02-wa-gateway.sql`:

```powershell
.\scripts\apply-wa-gateway-sql.ps1
```

## 2. Backend (API Go)

Copia [backend/.env.example](../backend/.env.example) para `backend/.env`. O exemplo já vem no **perfil rápido** (`WHATSAPP_PROVIDER=none`, `AUTO_REPLY_ENABLED=false`, `INSECURE_SKIP_WEBHOOK_AUTH=true`) para subir só com Postgres+Redis.

- `DATABASE_URL` / `REDIS_URL` → `127.0.0.1` com portas expostas pelo compose.
- **Sem Evolution local:** mantém `WHATSAPP_PROVIDER=none` (ou `baileys`). Com Evolution (ex.: `npm run docker:dev` na raiz): descomenta `EVOLUTION_*` no `.env.example` e usa `WHATSAPP_PROVIDER=evolution`.
- `INTERNAL_API_KEY` obrigatório.
- LLM: só precisas de chave Gemini/OpenAI se `AUTO_REPLY_ENABLED=true`.

```powershell
cd backend
go run ./cmd/api
```

## 3. Frontend

Copia `apps/web/.env.example` → `apps/web/.env`. A origem `http://localhost:5173` deve estar em `CORS_ALLOW_ORIGINS` no backend.

```powershell
cd apps/web
npm run dev
```

## 4. WhatsApp (gateway Baileys, opcional)

Copia `services/whatsapp-gateway/env.example` → `services/whatsapp-gateway/.env` (mesma `DATABASE_URL`/`REDIS_URL` lógica; Redis **DB 2** no URL).

```powershell
cd services/whatsapp-gateway
npm install
npm run pair
```

`npm run serve` expõe health em `:3090` (útil para readiness).

## 5. OmniVoice TTS (auto-resposta em voz)

A API Go **não** inclui o motor TTS: precisas do pacote **[omnivoice-server](https://pypi.org/project/omnivoice-server/)** (HTTP compatível com OpenAI `POST /v1/audio/speech`). Neste repo: [services/omnivoice-server/README.md](../services/omnivoice-server/README.md) — venv Python, PyTorch **CUDA** (recomendado) ou CPU, primeira execução descarrega modelos.

1. Instala o venv em [services/omnivoice-server](../services/omnivoice-server/README.md). Arranca com **`npm run omnivoice:server`** (CUDA, porta **8000**) **no mesmo host** que o `go run` da API; CPU: `npm run omnivoice:server -- --device cpu`. Se o carregamento do modelo falhar (memória/paginação), vê a secção “Resolução de problemas” nesse README.
2. **Checklist `backend/.env`:** `PUBLIC_MEDIA_BASE_URL` (ou o mesmo host que `PUBLIC_WEBHOOK_BASE_URL`) acessível **pelo contentor Evolution** para `GET /media/temp/:token`; `OMNIVOICE_DEFAULT_BASE_URL` com o host que o **processo Go** usa para chamar o TTS (`http://127.0.0.1:8000` ou `http://host.docker.internal:8000` se a API estiver em Docker e o OmniVoice no host).
3. **No site → Agentes:** agente ativo, **Usar na auto-resposta WhatsApp**, **Responder em áudio (TTS)**, provedor **OmniVoice** (URL no campo ou só o fallback `OMNIVOICE_DEFAULT_BASE_URL`).
4. **Valida antes do WhatsApp** — o cliente HTTP da API é o processo Go; `127.0.0.1` só funciona se o TTS estiver na mesma máquina. Preferir corpo JSON em ficheiro no PowerShell (evita `json_invalid`): ver [scripts/omnivoice-smoke-body.json](../scripts/omnivoice-smoke-body.json) e `curl.exe ... --data-binary "@scripts\omnivoice-smoke-body.json"`. Esperado: HTTP **200** e WAV.

```powershell
curl.exe -s -o NUL -w "%{http_code}" -X POST "http://127.0.0.1:8000/v1/audio/speech" -H "Content-Type: application/json; charset=utf-8" --data-binary "@scripts\omnivoice-smoke-body.json"
```

Se vires `dial tcp ... connection refused` nos logs da API, **nada escuta** nesse host:porta — sobe o TTS ou corrige a URL (incluindo porta). Mudar o modelo LLM (ex. GPT mini) **não** substitui o OmniVoice; são serviços diferentes.

Respostas **longas** na auto-resposta em voz são enviadas como **vários** ficheiros de áudio em sequência (mesma lógica de divisão que as mensagens de texto).

## 6. Verificação rápida

| Serviço | URL |
|---------|-----|
| API | `GET http://127.0.0.1:8080/health` |
| Meta | `GET http://127.0.0.1:8080/api/v1/meta` (`whatsapp_provider`) |
| WhatsApp gateway | `GET http://127.0.0.1:3090/health` (se `npm run serve`) |
| Web | Vite (ex. `http://localhost:5173`) |

## 8. API na Hetzner + frontend no PC (checklist deploy)

Quando o **backend** corre em Docker na Hetzner (ou outro servidor público) e o **Vite** corre no teu PC, o browser faz pedidos **cross-origin**. Sem isto, vês erros de rede ou CORS no consola.

1. **`backend/.env` (servidor)**  
   - `CORS_ALLOW_ORIGINS`: lista CSV com **todas** as origens do browser que vão falar com a API. Inclui pelo menos `http://localhost:5173` (Vite default) e, se usares outra porta/host, essa URL completa (sem path). Se no futuro servires o build estático com domínio próprio, adiciona `https://teu-dominio.com`.  
   - `PUBLIC_WEBHOOK_BASE_URL`: URL **pública** HTTPS (ou HTTP só em testes) onde a Evolution (ou outro cliente) consegue fazer `POST /webhooks/whatsapp/:instance_id`. Deve apontar para o host onde a API está exposta (ex. `https://api.teudominio.com`).  
   - `PUBLIC_MEDIA_BASE_URL`: URL base onde a Evolution faz `GET` dos ficheiros temporários (`/media/temp/:token`). Por defeito pode ser igual a `PUBLIC_WEBHOOK_BASE_URL`; o importante é o contentor Evolution alcançar esse host (não `localhost` do teu PC).

2. **`apps/web/.env` (PC)**  
   - `VITE_API_BASE_URL`: URL completa da API **incluindo** o prefixo `/api/v1`, ex. `https://api.teudominio.com/api/v1`.  
   - `VITE_WS_URL` (se usas Inbox em tempo real): `wss://api.teudominio.com` ou o host onde o endpoint `/ws` está exposto (ajusta ao teu reverse proxy).

3. **Reverse proxy / TLS**  
   - Expõe a mesma origem para HTTP API e WebSocket, ou configura CORS + upgrade explícito conforme o teu Nginx/Caddy.

4. **Smoke test**  
   - No PC: `npm run dev` no `apps/web`, abre login. Se falhar com CORS, rever o passo 1.  
   - Na Evolution: após **Instâncias → sincronizar webhook**, o URL gravado deve ser o público do passo 1.

## 7. Roadmap (ponte Go ↔ gateway)

- Normalizar eventos do **whatsapp-gateway** (Redis `wa:gw:*` ou HTTP) para o mesmo envelope que [backend/internal/handler/webhook.go](../backend/internal/handler/webhook.go) espera da Evolution, ou consumir Redis no processo Go.
- Até lá: testar webhooks com `curl` POST em `/webhooks/whatsapp/:instance_id` e `WHATSAPP_PROVIDER=none` para não depender de envio REST.
