# ChatBot — monorepo WhatsApp SaaS (UI + docs)

Frontend **React + Vite + TypeScript** para o dashboard descrito no playbook. Backend **Go 1.22+ / Fiber** em `backend/`.

## Estrutura

- `apps/web` — SPA (npm, Vite, Tailwind, shadcn, TanStack Query)
- `backend/` — API (webhook WhatsApp/Evolution, envio Evolution, IA Gemini/GPT, Postgres, Redis)
- `infra/` — Docker Compose (Postgres + Redis + **Evolution API** para dev)
- `docs/` — `saas-whatsapp-playbook.md`, `PLANEJAMENTO.md`, prompts
- `skills/` — instruções para IAs
- `directives/` / `execution/` — fluxo AGENTS.md (quando usar)

## Backend (API)

```bash
docker compose -f infra/docker-compose.dev.yml up -d
cd backend
cp .env.example .env
# Alinhar EVOLUTION_* com infra/README.md (chave padrão); preencher GEMINI_API_KEY se AUTO_REPLY=true; INTERNAL_API_KEY
go mod tidy
go run ./cmd/api
```

- `GET /health` — Postgres + Redis
- `POST /webhooks/whatsapp/:instance_id` — webhook (header `X-Webhook-Secret` e/ou `apikey` no JSON, conforme `.env.example`)
- `POST /api/v1/internal/evolution/send` — envio de texto (header `X-Internal-API-Key`)

**Configuração passo a passo (API, Evolution, Gemini/GPT, Docker):**  
[docs/GUIA_CONFIGURACAO_API.md](docs/GUIA_CONFIGURACAO_API.md)

Detalhes técnicos: [backend/README.md](backend/README.md).

## App web

```bash
cd apps/web
cp .env.example .env   # PowerShell: copy .env.example .env
npm install
npm run dev
```

- Dev: http://localhost:5173  
- API esperada: `VITE_API_BASE_URL` (ex.: `http://localhost:8080/api/v1`)

### Scripts (`apps/web`)

| Comando | Descrição |
|---------|-----------|
| `npm run dev` | Servidor de desenvolvimento |
| `npm run build` | Typecheck + build produção |
| `npm run test` | Vitest |
| `npm run lint` | ESLint |

### Auth sem backend (só UI)

No `.env` do app: `VITE_ENABLE_AUTH_MOCK=true` (apenas desenvolvimento).

### Contrato de API

Envelope JSON: sucesso `{ "data", "meta?" }`, erro `{ "error": { "code", "message", "details?" } }`. Ver playbook seção 6.

## Documentação

- [Playbook SaaS](docs/saas-whatsapp-playbook.md)
- [Planejamento / Prompt v3](docs/PLANEJAMENTO.md)

## CI

GitHub Actions em `.github/workflows/ci-web.yml` — lint, build e testes no `apps/web`.

Se ainda existir `node_modules` na **raiz** do repositório após a migração para `apps/web`, apague manualmente (pode falhar no Windows se algum processo estiver usando arquivos). Use apenas `apps/web/node_modules`.
