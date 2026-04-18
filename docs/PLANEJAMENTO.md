# Planejamento — Prompt, repositório e caminho até produção

Documento de alinhamento entre o **`saas-whatsapp-playbook.md`**, o **`antigravity-prompt-v2.md`** e a organização do projeto **ChatBot**. Use este arquivo como contexto no **modo plano** do Claude/GPT antes de executar refatorações grandes.

---

## 1. O que o playbook já resolve (e o prompt v2 não)

O playbook cobre de forma profissional:

- **Multi-tenant** (`workspace_id` no JWT, isolamento, mídia por workspace).
- **Contrato de API** (`/api/v1`, paginação cursor-based, envelope `data` / `error`, datas ISO 8601 UTC).
- **Segurança backend**: JWT **RS256**, refresh opaco com rotação, rate limit, Helmet/CORS, HMAC em webhooks, audit log, bcrypt, criptografia de API keys de LLM.
- **Frontend**: menção a **DOMPurify**, CSP, não expor secrets em `VITE_*`, `npm audit`.
- **Infra**: Cloudflare, Caddy, Docker, CI/CD, backups, checklist de produção.

O **`antigravity-prompt-v2.md`** é excelente para **UI e escopo de telas**, mas ainda trata o app como **SPA + JWT no `localStorage`** sem amarrar ao contrato `/api/v1`, sem **RBAC na UI**, sem **workspace switcher**, sem **política de refresh**, sem **sanitização de mensagens** e sem **ambiente dev vs preview** explícitos.

**Conclusão:** dá para **melhorar muito** o prompt incorporando trechos do playbook (API, segurança, multi-tenant, LGPD mínimo). O resultado desejado é um **“v3”** = v2 (visual + rotas) + **contrato e segurança** do playbook.

---

## 2. Melhorias recomendadas ao prompt (segurança e profissionalismo)

Cole ou funda estas seções ao prompt do Antigravity / plano da IA (não substituir o design system nem a lista de páginas — **acrescentar**).

### 2.1 Contrato de API obrigatório

- Base: `import.meta.env.VITE_API_BASE_URL` deve apontar para `.../api/v1` (ou documentar que o backend prefixa `/api/v1` e o `api.ts` usa `baseURL` completo).
- Respostas: tratar `{ data, meta }` e erros `{ error: { code, message, details } }`.
- Paginação: cursor + `limit` conforme playbook; tipar `meta` nos hooks.
- Endpoints alinhados ao playbook (auth com **refresh**, não só login).

### 2.2 Autenticação (sem atalhos de demo em código que vá para produção)

- **Proibir** `fake-token`, `|| 'fake-token'` em `ProtectedRoute` e login simulado sem flag de ambiente.
- Access token: curta duração (15 min é o alvo do playbook); armazenar só **access** no cliente (ex.: `localStorage` ou memória; documentar trade-off XSS).
- **Refresh token**: nunca no frontend em texto claro em produção ideal; preferir rota `POST /api/v1/auth/refresh` com cookie **httpOnly Secure SameSite** definida pelo backend, ou fluxo BFF — se a stack for só SPA + API, documentar explicitamente o risco e usar rotação + revogação no servidor.
- Interceptor Axios: em `401`, tentar **uma vez** refresh antes de deslogar; fila simples ou `axios-auth-refresh` pattern.
- Incluir **logout** chamando API e limpando estado de Query.

### 2.3 Multi-tenant na UI

- Após login, o JWT traz `workspace_id` (e opcionalmente nome do workspace). Exibir **workspace atual** no shell; preparar hook `useWorkspace()` lendo claims do JWT decodificado (sem confiar só no cliente — backend é fonte da verdade).
- Todas as queries devem ser consistentes com o tenant (o backend valida; o front não envia `workspace_id` em query string se o playbook usar só JWT).

### 2.4 RBAC (papéis)

- Matriz mínima: Admin, Supervisor, Agent — **esconder rotas e ações** conforme role (ex.: Agent não vê “Gerenciar usuários”).
- Centralizar permissões em `src/lib/permissions.ts` (funções puras) para evitar `if` espalhado.

### 2.5 Segurança de conteúdo e dados

- Mensagens ricas / HTML vindas de usuário: sanitizar com **DOMPurify** antes de `dangerouslySetInnerHTML` (se existir); preferir texto + markdown seguro com biblioteca adequada.
- Não logar tokens nem PII no `console` em builds de produção.
- Variáveis `VITE_*`: apenas o que for **público** (URL da API, DSN do Sentry se aplicável). Chaves de LLM **somente backend**.

### 2.6 Tempo real

- `VITE_WS_URL`: conectar com token; **reconexão** com backoff; heartbeat — alinhado ao playbook (ping/pong, timeouts de proxy).
- Tipar eventos de socket (`conversation.updated`, `message.created`, etc.) em `src/types/ws.ts`.

### 2.7 LGPD (Brasil) — mínimo viável no produto

- Textos legais: link para política de privacidade e termos nas telas de cadastro/login (placeholder URLs).
- Campanhas: indicar no UI necessidade de **opt-in** e conformidade com políticas WhatsApp/Meta (o playbook já trata modelo de negócio; o front deve avisar o operador, não “resolver” jurídico).

### 2.8 Qualidade de engenharia no front

- **TanStack Query** para **todo** estado de servidor; `placeholderData` para mocks — proibir `useEffect` + `fetch` solto.
- **React Hook Form + Zod** em **todos** os formulários (login, contato, agente, campanha, settings).
- **shadcn/ui**: instalar componentes usados; não deixar só `components.json` sem `src/components/ui`.
- Testes: mencionar **Vitest + Testing Library** para componentes críticos (login, `api.ts`, um hook de query).
- **README**: substituir template Vite por overview, scripts, env, estrutura, como rodar contra API local.

### 2.9 Alinhamento de versões

- Playbook cita React 18 e Router v6; o projeto atual pode estar em React 19 / Router 7. No prompt: **“use as versões já fixadas no `package.json` do repositório”** para evitar churn, ou fixar explicitamente versões alvo.

---

## 3. Organização da pasta ChatBot (engenharia limpa + caminho para produção)

Hoje o repositório mistura **raiz do monorepo** com **app Vite** (`src/`, `package.json` na raiz), documentação solta e possivelmente `node_modules`/`dist` na raiz. Para ficar **profissional** e fácil de levar a um servidor “de verdade”:

### 3.1 Estrutura alvo (monorepo simples)

```
ChatBot/
├── apps/
│   └── web/                 # SPA React (package.json, src/, vite.config, etc.)
├── backend/                 # Go + Fiber (quando existir) — espelha playbook seção 4
├── docs/
│   ├── saas-whatsapp-playbook.md   # ou symlink / cópia versionada
│   ├── PLANEJAMENTO.md
│   └── prompts/
│       └── antigravity-prompt-v2.md
├── infra/                   # docker-compose.dev.yml, exemplos Caddy, scripts deploy
├── .github/workflows/       # CI: lint, test, build web, test go
├── .env.example             # nunca secrets reais
├── .gitignore               # .env, dist, coverage, credenciais
├── AGENTS.md                # já existe — regras da orquestração
├── directives/              # SOPs (se usar arquitetura 3 camadas)
├── execution/               # scripts Python determinísticos (se usar)
├── skills/                  # instruções para IAs (este repo)
└── README.md                # visão do monorepo + como rodar web + backend
```

**Benefícios:** separação clara **app / API / docs / infra**; deploy da API e do static `dist` independentes; menos risco de commitar `dist/` ou `.env` por engano.

### 3.2 O que fazer com o código atual

- Mover o frontend para **`apps/web/`** (ou `frontend/` como no playbook) e ajustar paths no CI.
- Manter **um** `package.json` por app; na raiz, opcionalmente **npm/pnpm workspaces** ou apenas documentação “entre em `apps/web` e rode `npm run dev`”.

### 3.3 Cybersegurança no PC de desenvolvimento

- **Nunca** commitar `.env`, `credentials.json`, chaves JWT, secrets de webhook.
- Usar **`.env.example`** com chaves vazias e comentários.
- Rodar **`npm audit`** / **`go vet`** / **`golangci-lint`** no CI.
- Para clonar em produção: **secrets** só em variáveis do provedor (Coolify, GitHub Actions, etc.), nunca no repositório.
- **Backups** e rotação de chaves conforme playbook seções 12 e 13.

### 3.4 Entrega em produção (resumo)

- Frontend: `npm run build` → artefatos estáticos em CDN ou `file_server` atrás de HTTPS.
- Backend: imagem Docker distroless, TLS na borda (Caddy/Cloudflare), WAF, rate limit, healthcheck, métricas, Sentry.

---

## 4. Modo plano — prompt curto para colar no agente

Use após anexar **`saas-whatsapp-playbook.md`**, **`antigravity-prompt-v2.md`** e este **`PLANEJAMENTO.md`**:

```
Modo PLANO (não escreva código ainda).

1) Compare antigravity-prompt-v2.md com saas-whatsapp-playbook.md e PLANEJAMENTO.md.
2) Produza um documento "Prompt v3" em seções: (A) stack UI inalterada, (B) contrato API /api/v1,
   (C) auth+refresh+RBAC+tenant na UI, (D) segurança front (DOMPurify, sem secrets VITE, sem fake-token),
   (E) WebSocket, (F) LGPD mínimo, (G) testes e README.
3) Liste a migração de pastas proposta (apps/web, docs, backend) com ordem de commits sugerida.
4) Liste riscos e dependências (backend antes do front em quais telas).
Priorize segurança e aderência ao playbook; seja específico e acionável.
```

---

## 5. Skills na pasta `skills/`

Foram adicionados arquivos reutilizáveis (ver `skills/README.md`). Eles condensam práticas alinhadas a:

- [OWASP Top 10](https://owasp.org/Top10/)
- [TanStack Query](https://tanstack.com/query/latest)
- Documentação de regras do Cursor: [Cursor — Skills / customization](https://cursor.com/docs)

**Como usar no Cursor:** referencie com `@skills/nome-do-arquivo.md` no chat ou copie trechos para **Project Rules** em `.cursor/rules/*.mdc` (formato recomendado pelo Cursor para regras sempre ativas).

---

## 6. Próximo passo seu

1. Rodar o **modo plano** com o prompt da seção 4.  
2. Aprovar o “Prompt v3” e a estrutura de pastas.  
3. Só então **modo agente** para mover arquivos e implementar.

---

*Última atualização: alinhado ao playbook v1.0 e ao estado do repositório ChatBot (frontend na raiz, documentação em evolução).*
