# Skill: Checklist de code review (antes do merge)

Use em modo revisão (humano ou IA). Baseado no playbook (seção 10, debugging/review) e OWASP.

## Segurança

- [ ] Nenhuma query SQL/string montada com input do usuário sem placeholders.
- [ ] `workspace_id` validado em todo acesso a recurso (backend).
- [ ] Sem log de senha, JWT completo, API keys ou PII desnecessária.
- [ ] Webhooks com assinatura verificada.
- [ ] Novos endpoints com rate limit adequado quando expostos publicamente.

## Multi-tenant

- [ ] Impossível ler/alterar dados de outro workspace com token válido de um tenant (teste mental ou automatizado).

## Frontend

- [ ] Dados remotos via TanStack Query (não fetch solto em `useEffect`).
- [ ] Forms com Zod.
- [ ] Sem `fake-token` ou bypass de auth em código de feature.
- [ ] Nenhum secret em `VITE_*`.

## Qualidade

- [ ] Erros tratados em Go (`%w`); sem ignorar erro silenciosamente sem justificativa.
- [ ] Funções gigantes (> ~50 linhas) merecem split.
- [ ] Textos de UI em pt-BR.

## Operação

- [ ] Migration com `.up` e `.down` quando aplicável.
- [ ] README ou comentário de env atualizado se novas variáveis.
