# Prompt — Dashboard SaaS WhatsApp + IA (v3)

Unifica o escopo visual do v2 (`antigravity-prompt-v2.md`) com contrato REST `/api/v1`, multi-tenant, RBAC e refresh descritos no `saas-whatsapp-playbook.md`.

**Uso:** cole no Antigravity / agente ao evoluir o código em `apps/web`.

## Stack

- Versões do `package.json` em `apps/web` (React, Router, Vite, TS).
- Tailwind + **shadcn/ui** em `src/components/ui`.
- TanStack Query para todo estado remoto; mocks com `placeholderData` / `queryFn` até o backend existir.
- Axios em `src/lib/api.ts` com `VITE_API_BASE_URL` apontando para **`/api/v1`**.
- RHF + Zod em todos os formulários.
- JWT access em `localStorage` (`access_token`); **proibido** `fake-token` fora de `VITE_ENABLE_AUTH_MOCK=true` em dev.

## Contrato

- Sucesso: `{ data, meta? }`. Erro: `{ error: { code, message, details? } }`.
- 401: uma tentativa de `POST /auth/refresh`; depois logout e redirect `/login`.

## UI

- Claims JWT: `workspace_id`, `role` (`admin` | `supervisor` | `agent`); `src/lib/permissions.ts` + shell com workspace.
- WebSocket: `VITE_WS_URL`, hook com reconexão exponencial.
- Mensagens: sanitizar com DOMPurify (`src/lib/sanitize.ts`).
- LGPD: links no login; aviso em campanhas.

## Proibições

- react-flow; SSR PHP; secrets em `VITE_*`.
