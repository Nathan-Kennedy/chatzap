# Skill: Segurança OWASP + SaaS multi-tenant

Use ao implementar autenticação, webhooks, API ou revisar PRs. Complementa o `docs/saas-whatsapp-playbook.md` (seção 7).

## Princípios

1. **Confiar no servidor:** validar `workspace_id` (tenant) e `role` em **todo** handler; nunca confiar só no frontend.
2. **Menos superfície:** desabilitar endpoints de debug em produção; não retornar stack trace ao cliente.
3. **Secrets:** variáveis sensíveis só no backend e em gestores de secrets; nunca `VITE_*` para chaves privadas.

## OWASP Top 10 — ações concretas

| Risco | Ação |
|-------|------|
| **A01 Broken Access Control** | Checar posse do recurso + tenant; testes de IDOR entre workspaces |
| **A02 Cryptographic Failures** | TLS em trânsito; bcrypt para senhas; AES-GCM para tokens/API keys em repouso |
| **A03 Injection** | Queries parametrizadas (GORM); validar input com tags `validate:` |
| **A04 Insecure Design** | Rate limit em login; refresh rotation; audit log em ações sensíveis |
| **A05 Security Misconfiguration** | CORS explícito; headers Helmet; desativar defaults inseguros |
| **A06 Vulnerable Components** | `npm audit`, Dependabot/Renovate, imagens base atualizadas |
| **A07 Auth Failures** | Bloqueio progressivo; 2FA futuro; sessões revogáveis |
| **A08 Data Integrity** | HMAC em webhooks; assinar payloads críticos |
| **A09 Logging/Monitoring** | Logs estruturados sem PII/secrets; alertas 5xx; Sentry |
| **A10 SSRF** | Validar URLs em webhooks/nós de fluxo; allowlist quando possível |

## Webhooks (WhatsApp / Evolution)

- Validar **HMAC** com comparação em tempo constante (`hmac.Equal`).
- Rejeitar body acima do limite; idempotência com `message_id` quando aplicável.

## Referência

- https://owasp.org/Top10/
- https://cheatsheetseries.owasp.org/cheatsheets/Multitenant_Architecture_Cheat_Sheet.html
