# Backend API (Go + Fiber) — MVP

**Guia para configurar tudo do teu lado (passo a passo, linguagem simples):**  
[../docs/GUIA_CONFIGURACAO_API.md](../docs/GUIA_CONFIGURACAO_API.md)

## Resumo técnico

- **PostgreSQL** + **Redis** (Docker: `infra/docker-compose.dev.yml`)
- **Webhook:** `POST /webhooks/whatsapp/:instance_id` (Evolution v2)
- **Envio:** `POST /api/v1/internal/evolution/send` + header `X-Internal-API-Key`

### Checklist — mensagens recebidas (webhook)

1. **`PUBLIC_WEBHOOK_BASE_URL`** — URL que o **contentor da Evolution** consegue aceder (ex.: API no host: `http://host.docker.internal:8080`; túnel HTTPS em produção). O path final é `/webhooks/whatsapp/<nome_técnico_da_instância>`.
2. **Sincronizar webhook** — Na app, Instâncias → ação que chama `POST /api/v1/instances/:id/sync-webhook`, ou volta a criar/importar a instância para a Evolution gravar a URL.
3. **Autenticação** — O middleware aceita **`X-Webhook-Secret`** (se `WEBHOOK_SHARED_SECRET` estiver definido) e/ou **`apikey`** no header ou no JSON. A `apikey` é válida se coincidir com `EVOLUTION_WEBHOOK_API_KEY`, com **`EVOLUTION_API_KEY` (global)** ou com o **token da instância** (`evolution_instance_token` na BD, mesmo valor que a Evolution usa no webhook).
4. **`webhookBase64`** — Se a Evolution enviar o campo `data` como string Base64, a API descodifica antes de processar (`NormalizeWebhookData`).

### “Puxar” conversas / histórico antigo

- **`POST /api/v1/instances/:id/sync-contacts`** — Cria conversas na caixa a partir de `GET /user/contacts` na Evolution; **não traz texto** de mensagens antigas.
- **`POST /api/v1/instances/:id/sync-chats`** — Tenta importar mensagens via `findMessages` na Evolution Go; em muitas builds o endpoint **não existe (404)**. O histórico em tempo real depende dos **webhooks** (`MESSAGES_UPSERT` / equivalente) após o telemóvel ligado.
- **IA:** **Gemini** por defeito (`LLM_PROVIDER=gemini` + `GEMINI_API_KEY`); **OpenAI (GPT)** com `LLM_PROVIDER=openai` + `OPENAI_API_KEY`. Troca reiniciando a API.
- **Meta:** `GET /api/v1/meta` — versão do serviço
- **Saúde:** `GET /health` — Postgres + Redis

## Comandos

```bash
cp .env.example .env   # edita depois
go mod tidy
go run ./cmd/api
```

Ou `make tidy`, `make run`, `make test`.

## Docker (API)

```bash
docker build -f Dockerfile -t wa-saas-api .
```

Com compose (perfil `api`): na raiz do repo, `npm run docker:dev:api` ou `.\scripts\docker-dev-up.ps1` — ver `infra/docker-compose.dev.yml`.

**Atualizar a API no Docker** depois de alterar código: na raiz, `npm run docker:dev:api:rebuild` (imagem nova sem cache + contentor `wa-saas-api` recriado).

**Desenvolvimento normal (`dev:stack`):** API no PC + Evolution no Docker — `npm run docker:dev:stack` (ou `docker:dev`) **sem** `--profile api`; ver [../docs/DEV_LOCAL.md](../docs/DEV_LOCAL.md).

## Testes

```bash
go test ./...
```

### Teste manual: mensagem recebida na inbox

Com `DATABASE_URL` no `backend/.env`, Postgres no ar e a API a receber webhooks:

```bash
cd backend
go run ./cmd/waitinbound -phone 5569993378283 -text Cuiudu
```

Ou na raiz: `npm run wait:inbound -- -phone 5569993378283 -text Cuiudu`

O comando fica a consultar a BD até encontrar uma mensagem **inbound** cujo corpo contém o texto e a conversa é desse número (normalizado para JID, inclui `@lid` com o mesmo prefixo). Sai com código `0` se encontrar, `1` em timeout (~3 min por defeito), `-timeout 5m` para mudar.

**Diagnóstico** (webhooks gravados vs inbox): `go run ./cmd/waitinbound -diag` (opcional `-text Cuiudu` para procurar o texto em qualquer inbound).
