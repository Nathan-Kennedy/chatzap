# CRM / fluxos / mídia — levantamento (épico)

Este documento fecha a fase D do plano “voz + resumo, Kanban, interativos e CRM/fluxos”: o que já existe no repositório e o que fica para iterações futuras.

## Já entregue (MVP)

- **Perfil por conversa:** tabela `contact_profiles` (`workspace_id` + `conversation_id` únicos) com `facts_json` (objeto JSON livre).
- **API:** `GET` e `PATCH` `/api/v1/conversations/:id/contact-profile` — leitura e gravação de `facts` (merge substitui o objeto completo no PATCH).

## Próximas iterações sugeridas

1. **UI na inbox:** painel “Dados do contacto” que edita `facts` (campos fixos + JSON avançado).
2. **Extração LLM:** job ou passo pós-mensagem que preenche/atualiza `facts` com schema validado (Zod no front, JSON Schema no back).
3. **Biblioteca de mídia:** entidade `media_assets` por workspace, URLs estáveis e reutilização em fluxos.
4. **Motor de fluxo:** nós (pergunta, condição, envio de botões/lista Evolution, atualização de Kanban) — reutilizar `Flow` existente ou evoluir modelo.
5. **Orçamentos automáticos:** regras + templates + PDF — depende de perfil estruturado e de anexos.

## Integração com Kanban e interativos

- **Kanban:** regras automáticas por palavra-chave já movem `pipeline_stage` no webhook.
- **Interativos:** `SendButtons` na Evolution + parse de `buttonsResponseMessage` no inbound; UI de construção de fluxos é o passo seguinte.
