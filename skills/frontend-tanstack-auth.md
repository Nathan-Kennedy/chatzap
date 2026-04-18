# Skill: Frontend React — TanStack Query, API e auth profissional

Use ao gerar ou refatorar o SPA do WhatsApp SaaS. Alinhado ao playbook (API `/api/v1`, JWT curto, refresh).

## Estado de servidor

- **Sempre** `useQuery` / `useMutation` para dados remotos; evitar `useEffect` + `fetch` cru.
- Chaves de query estáveis: `['conversations', filters]`, `['conversation', id]`.
- Invalidar queries após mutations que alterem listas.

## Cliente HTTP (`api.ts`)

- `baseURL` via `import.meta.env.VITE_API_BASE_URL`.
- Interceptor: anexar `Authorization: Bearer <access_token>`.
- Em **401**: fluxo de **refresh único** (evitar tempestade); se falhar, limpar sessão e redirecionar a `/login`.
- Tratar envelope de erro `{ error: { code, message, details } }` e exibir toast amigável (sem vazar detalhes internos).

## Autenticação (proibições)

- **Não** usar `localStorage.getItem('token') || 'fake-token'`.
- **Não** simular login em produção sem `import.meta.env.DEV` explícito.
- Login real: `POST /api/v1/auth/login`; armazenar access conforme decisão de arquitetura; integrar `POST /api/v1/auth/refresh` e logout.

## Formulários

- **React Hook Form + Zod** em todos os forms; mensagens de erro em português.

## Conteúdo dinâmico

- Se renderizar HTML de mensagens: **DOMPurify** ou equivalente; evitar `dangerouslySetInnerHTML` sem sanitização.

## UI multi-tenant / RBAC

- Derivar `role` e `workspace_id` dos claims do JWT (apenas UX); **backend valida tudo**.
- Esconder rotas/ações com helper `can(user, 'campaigns:create')` centralizado.

## Referências

- https://tanstack.com/query/latest/docs/framework/react/overview
- https://cheatsheetseries.owasp.org/cheatsheets/DOM_based_XSS_Prevention_Cheat_Sheet.html
