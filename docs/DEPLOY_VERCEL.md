# Deploy do frontend no Vercel (API no Hetzner / Coolify)

O backend continua na **Coolify** (Hetzner). O Vercel serve só o **build estático** de [`apps/web`](../apps/web).

## Configuração do projeto Vercel

1. **Import Git Repository** → escolhe o mesmo repo que ligas ao Coolify.
2. Em **Root Directory**, define **`apps/web`** (obrigatório neste monorepo).
3. **Framework Preset:** Vite (detetado automaticamente na maior parte dos casos).
4. **Build / Output:** o ficheiro [`apps/web/vercel.json`](../apps/web/vercel.json) fixa `npm run build` e pasta `dist`.

Se o dashboard mostrar comando de instalação errado, confirma **Install Command** = `npm ci` na pasta `apps/web`.

## Variáveis de ambiente (build-time)

Definir no Vercel → **Settings → Environment Variables** (marcar **Production** e, se usares previews, **Preview**).

| Variável | Exemplo (ajusta ao teu domínio da API) | Notas |
|----------|----------------------------------------|--------|
| `VITE_API_BASE_URL` | `https://api.teudominio.com/api/v1` | URL pública da API Go; tem de incluir `/api/v1`. |
| `VITE_WS_URL` | `wss://api.teudominio.com/ws` | Se a API for **HTTPS**, usa **`wss://`**. Se for HTTP, `ws://`. |

Variáveis `VITE_*` são embutidas no bundle no **build**; após mudar, faz **Redeploy**.

Referência local: [`apps/web/.env.example`](../apps/web/.env.example).

## Matriz: origem da UI vs API

| Ambiente | Onde corre o front | Origem (`Origin` no browser) | `VITE_API_BASE_URL` / `VITE_WS_URL` apontam para |
|----------|--------------------|------------------------------|--------------------------------------------------|
| **Dev** | PC (`npm run dev` em `apps/web`) | `http://localhost:5173` | API local **ou** API Coolify (conforme `apps/web/.env`) |
| **Preview** (Vercel) | URL tipo `https://*.vercel.app` | esse host | Staging ou prod API (decide uma política) |
| **Production** (Vercel) | `https://teu-app.vercel.app` ou domínio próprio | esse host | URL pública **de produção** da API |

## CORS na API (Coolify)

O backend lê **`CORS_ALLOW_ORIGINS`** (lista separada por vírgulas). Para cada **origem** onde o front corre, tem de existir uma entrada **exacta** (incluindo `https://` e porta se aplicável).

**Checklist após primeiro deploy no Vercel**

1. Copia a URL **Production** do site (ex. `https://chatzap.vercel.app`).
2. No Coolify → serviço da API → variáveis: acrescenta essa origem a `CORS_ALLOW_ORIGINS` (mantém `http://localhost:5173` se ainda desenvolves no PC).
3. **Redeploy** da API.

**Previews Vercel** (`*.vercel.app`): o CORS é por string **exacta** — cada preview tem URL diferente. Opções:

- **Restrito:** só adicionas origens de previews que precisares (manual ou script).
- **Aberto para previews:** incluir um padrão que o teu middleware aceite **não** é suportado pelo Fiber/CORS actual (lista literal). Alternativa: API de **staging** com `CORS_ALLOW_ORIGINS` mais largo e `VITE_*` nos previews a apontar para staging; produção mantém CORS fechado.

Ver também: [HETZNER_CLOUD_SETUP.md](./HETZNER_CLOUD_SETUP.md) (front local + API na cloud).

## Segurança

- Não coloques **API keys secretas** em variáveis `VITE_*` — ficam visíveis no JavaScript do browser.
- Segredos de servidor ficam só nas variáveis da **Coolify** (API).
