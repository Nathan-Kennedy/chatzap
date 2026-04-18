# Skills — instruções para IAs (Cursor, Claude, Gemini)

Esta pasta contém **textos de contexto** que você pode anexar com `@skills/…` no Cursor ou colar em sessões no Claude/Gemini. Eles **não executam** sozinhos: servem para reduzir alucinação e padronizar segurança e arquitetura.

## Arquivos

| Arquivo | Uso |
|---------|-----|
| `seguranca-owasp-saas.md` | Checklist OWASP + hábitos para SaaS multi-tenant |
| `frontend-tanstack-auth.md` | Query, API client, auth, sem bypass de demo em prod |
| `backend-go-tenant-api.md` | Lembrete Fiber/GORM/tenant — detalhes no playbook |
| `review-checklist-pr.md` | Code review antes de merge |

## Integração com Cursor (regras do projeto)

O Cursor aplica automaticamente regras em **`.cursor/rules/`** (arquivos `.mdc` com frontmatter). Estes `.md` em `skills/` são portáteis; se quiser regra sempre ativa:

1. Crie `.cursor/rules/` na raiz do repo.
2. Copie o conteúdo relevante de um skill para um arquivo `.mdc` ou use `globs` para limitar a `apps/web/**` ou `backend/**`.

Documentação oficial: [Cursor Docs — Rules](https://cursor.com/docs) e [Skills](https://cursor.com/docs/context/rules).

## Fontes de referência

- [OWASP Top 10](https://owasp.org/Top10/)
- [OWASP Cheat Sheet Series](https://cheatsheetseries.owasp.org/)
- [TanStack Query](https://tanstack.com/query/latest)
- Playbook interno: `docs/saas-whatsapp-playbook.md`
