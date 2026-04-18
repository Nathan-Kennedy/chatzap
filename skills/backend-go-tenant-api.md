# Skill: Backend Go (Fiber) — tenant, API e padrões

Resumo operacional; detalhes completos estão em **`docs/saas-whatsapp-playbook.md`** (seções 4 a 7).

## Estrutura

- `handler` → `service` → `repository`; domain sem importar infra.
- Rotas versionadas: prefixo `/api/v1`.

## Cada request autenticado

1. Validar JWT (RS256 no playbook).
2. Extrair `sub`, `workspace_id`, `role`.
3. Em queries/updates: **sempre** filtrar por `workspace_id` do token, nunca aceitar tenant só do body/query.

## Erros

- Retornar JSON padronizado `error.code`, `error.message`, `error.details`.
- Não expor stack em produção.

## Segurança mínima no stack Fiber

- Helmet / security headers.
- CORS com origem explícita em produção.
- Rate limit (login mais agressivo).
- Request ID + logs Zap sem senhas/tokens.

## Webhooks

- Rota sem JWT; validação **HMAC** + limite de tamanho.

## Testes

- `testify` para services; `httptest` para handlers críticos (auth, tenant isolation).

## Referências

- https://docs.gofiber.io/
- https://gorm.io/docs/
- Playbook interno: seções 6 e 7
