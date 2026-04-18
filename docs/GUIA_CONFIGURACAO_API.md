# Guia único: o que você precisa fazer (API, Evolution, IA — Gemini ou GPT)

Este arquivo reúne **tudo que depende de você** para o backend Go funcionar no seu computador ou servidor. Está em **passos numerados** e em linguagem simples.

> **Ideia geral:** o WhatsApp fala com o **Evolution Go**. O Evolution avisa o nosso **backend** (webhook). O backend grava no **PostgreSQL**, usa **Redis** para checagens, e pode mandar o texto para o **Google Gemini** (padrão) ou para a **OpenAI (GPT)** e responder de volta pelo Evolution.

---

## Antes de começar — o que instalar

| Ferramenta | Para quê |
|------------|----------|
| **Docker Desktop** (Windows/Mac) ou Docker + Compose no Linux | Subir **PostgreSQL** e **Redis** sem complicação |
| **Go 1.22+** ([https://go.dev/dl/](https://go.dev/dl/)) | Compilar e rodar o backend (`go run`) |
| **Evolution Go** já rodando | Ponte com o WhatsApp (QR code, instância criada) |
| Chave **Google AI (Gemini)** ou **OpenAI** | Só obrigatória se `AUTO_REPLY_ENABLED=true` — por padrão usa **Gemini** |

Se você **não** quiser IA ainda, use `AUTO_REPLY_ENABLED=false` (não precisa de chave nenhuma).

---

## Passo 1 — Subir banco de dados e Redis

1. Abra o terminal **na pasta raiz do projeto** (onde está a pasta `infra`).
2. Execute:

```bash
docker compose -f infra/docker-compose.dev.yml up -d
```

3. Confirme que os containers estão no ar (no Docker Desktop, deve aparecer `wa-saas-postgres` e `wa-saas-redis`).

**Valores padrão** (se você não mudou nada):

- Utilizador Postgres: `wa_saas`
- Palavra-passe: `wa_saas_dev_change_me`
- Base de dados: `wa_saas`
- Porta Postgres: `5432`
- Porta Redis: `6379`

> Troque a palavra-passe se expuser a máquina na internet. Para só testar em casa, o padrão costuma bastar.

---

## Passo 2 — Criar o ficheiro de configuração do backend

1. Vá à pasta `backend`.
2. Copie o exemplo:
   - **Windows (PowerShell):** `copy .env.example .env`
   - **Mac/Linux:** `cp .env.example .env`
3. Abra o ficheiro `backend/.env` com um editor de texto e preencha **conforme a tabela abaixo**.

---

## Passo 3 — Entender cada variável (sem ser programador)

### Geral

| Variável | O que é | O que colocar |
|----------|---------|---------------|
| `ENV` | Ambiente | `development` enquanto testa |
| `HTTP_PORT` | Porta da API | `8080` (ou outra livre) |
| `LOG_LEVEL` | Detalhe dos logs | `info` ou `debug` |

### Base de dados e Redis

| Variável | O que é | Exemplo (padrão do compose) |
|----------|---------|-----------------------------|
| `DATABASE_URL` | Ligação ao Postgres | `postgres://wa_saas:wa_saas_dev_change_me@127.0.0.1:5432/wa_saas?sslmode=disable` |
| `REDIS_URL` | Ligação ao Redis | `redis://127.0.0.1:6379/0` |

Se um dia correr a **API dentro do Docker** com o `docker compose --profile api`, use no `.env` do host o que o guia do `infra/docker-compose.dev.yml` sobrescreve: o compose já define `DATABASE_URL` e `REDIS_URL` para os nomes `postgres` e `redis`. A **Evolution** nesse caso costuma estar no PC; em `EVOLUTION_BASE_URL` use `http://host.docker.internal:8081` (troque `8081` pela porta real da Evolution).

### WhatsApp — Evolution opcional (`WHATSAPP_PROVIDER`)

| Variável | Valores | Efeito |
|----------|---------|--------|
| `WHATSAPP_PROVIDER` | `evolution`, `baileys`, `none` | O `backend/.env.example` de desenvolvimento usa `none` para subir sem Evolution. Com `evolution`, `EVOLUTION_BASE_URL` e `EVOLUTION_API_KEY` são obrigatórios. Com `none` ou `baileys`, a API sobe **sem** cliente Evolution REST; webhooks e UI continuam úteis; o envio `POST /api/v1/internal/evolution/send` responde **501** até voltares a `evolution`. Se omitires a variável no `.env`, o código assume `evolution` (exige Evolution). |

Fluxo local sem Docker da Evolution: [DEV_LOCAL.md](DEV_LOCAL.md).

### Evolution — enviar mensagens pela API

Se usou o Docker deste projeto (`infra/docker-compose.dev.yml`), a Evolution já sobe na porta **8081** com a chave padrão documentada em `infra/README.md` (troque com `EVOLUTION_AUTH_API_KEY` em `infra/.env`).

| Variável | O que é | Onde achar |
|----------|---------|------------|
| `EVOLUTION_BASE_URL` | Endereço da Evolution **sem** barra no fim | Com compose local: `http://127.0.0.1:8081` |
| `EVOLUTION_API_KEY` | Chave global admin (header `apikey`) | Igual à `GLOBAL_API_KEY` / `EVOLUTION_AUTH_API_KEY` |
| `EVOLUTION_INSTANCE_NAME` | Fallback de token da instância (rotas internas) | Pode ficar vazio se enviar sempre `instance` no payload |

### Webhook — a Evolution avisar o seu backend

A Evolution envia eventos (mensagens recebidas, etc.) para o seu computador. Para **ninguém fingir** que é a Evolution, usamos segredos.

**Opção A — mais simples com Evolution:**  
Defina `EVOLUTION_WEBHOOK_API_KEY` com o mesmo valor do campo **`apikey`** que a Evolution manda no **corpo JSON** do webhook (muitas instalações enviam isso automaticamente).

**Opção B — header extra:**  
Defina `WEBHOOK_SHARED_SECRET` com uma frase longa e aleatória. Configure a Evolution (ou um proxy) para enviar o header HTTP **`X-Webhook-Secret`** com **exatamente** esse valor.  
Se **as duas** variáveis (`EVOLUTION_WEBHOOK_API_KEY` e `WEBHOOK_SHARED_SECRET`) estiverem preenchidas, a API exige **as duas** corretas.

**Só para teste rápido em casa:**  
Com `ENV=development`, pode usar `INSECURE_SKIP_WEBHOOK_AUTH=true` **(não use em produção)**.

### Chave interna (testes e ferramentas)

| Variável | O que é |
|----------|---------|
| `INTERNAL_API_KEY` | Senha para chamar a rota de envio manual. Envie no header **`X-Internal-API-Key`**. |

Invente uma string longa e guarde no gestor de palavras-passe.

### IA — Gemini (padrão) ou GPT (opcional pago)

| Variável | O que é |
|----------|---------|
| `LLM_PROVIDER` | `gemini` (padrão) ou `openai`. É só mudar esta linha no `.env` e reiniciar a API para trocar de fornecedor. |
| `AUTO_REPLY_ENABLED` | `true` = webhook gera resposta com IA e envia pelo WhatsApp. `false` = só grava eventos (sem chave de IA). |
| `GEMINI_API_KEY` | Chave da API Google AI Studio / Google Cloud (Gemini). Obrigatória se `LLM_PROVIDER=gemini` e auto-reply ligado. |
| `GEMINI_MODEL` | Modelo Gemini, ex.: `gemini-2.0-flash` (padrão no projeto). |
| `OPENAI_API_KEY` | Chave OpenAI (`sk-...`). Obrigatória **só** se `LLM_PROVIDER=openai` e auto-reply ligado. |
| `OPENAI_MODEL` | Modelo GPT, ex.: `gpt-4o-mini`. |
| `LLM_SYSTEM_PROMPT` | (Opcional) Personalidade do bot; tem prioridade sobre `OPENAI_SYSTEM_PROMPT` (mantido por compatibilidade). |

### Front-end (CORS)

| Variável | O que é |
|----------|---------|
| `CORS_ALLOW_ORIGINS` | Origem do dashboard Vite. Padrão: `http://localhost:5173`. Várias: separadas por vírgula. |

### Opcional — restringir instâncias

| Variável | O que é |
|----------|---------|
| `ALLOWED_INSTANCE_IDS` | Lista separada por vírgulas dos `instance_id` permitidos na URL do webhook. Vazio = aceita qualquer um. |

---

## Passo 4 — Ligar a API no computador

Na pasta `backend`:

```bash
go mod tidy
go run ./cmd/api
```

Se aparecer erro de módulos, volte a correr `go mod tidy` (faça isso também depois de um `git pull` que mude o backend).

Para validar o código sem subir a API:

```bash
go test ./...
```

**Teste rápido no navegador ou no terminal:**

- Saúde: [http://localhost:8080/health](http://localhost:8080/health)  
- Versão: [http://localhost:8080/api/v1/meta](http://localhost:8080/api/v1/meta)

---

## Passo 5 — Configurar o webhook na Evolution

O backend expõe **`POST /webhooks/whatsapp/:instance_id`** (sem prefixo `/api/v1`). O `:instance_id` deve ser o **nome técnico da instância** tal como está em `whatsapp_instances.evolution_instance_name` (o mesmo segmento que a app regista ao sincronizar o webhook).

**Forma recomendada:** no dashboard, **Instâncias → sincronizar webhook** (chama `POST /api/v1/instances/:id/sync-webhook`). Assim o URL e a chave ficam alinhados ao `PUBLIC_WEBHOOK_BASE_URL` do `backend/.env`.

1. Base do URL (substitua `PORTA` por `HTTP_PORT` do `.env`, ex. `8080` ou `8088`):

   `http://<alcance-pela-Evolution>:PORTA/webhooks/whatsapp/NOME_DA_INSTANCIA`

2. **Evolution no Docker e API no PC (Windows/Mac):** dentro do contentor, `127.0.0.1` é o próprio contentor — **não** é a tua API. Use **`host.docker.internal`** e a porta onde a API escuta, ex.:

   `http://host.docker.internal:8088/webhooks/whatsapp/minha-loja`

   Defina também `PUBLIC_WEBHOOK_BASE_URL=http://host.docker.internal:8088` no `backend/.env` (sem barra final). O `infra/docker-compose.dev.yml` já adiciona `extra_hosts` na Evolution para `host.docker.internal`.

3. **API no mesmo Docker Compose** (perfil `api`): o compose define por defeito `PUBLIC_WEBHOOK_BASE_URL=http://api:8080`; a Evolution chama a API pela rede interna. Volte a **sincronizar webhook** após subir a stack.

4. **Só no PC, Evolution noutro host na LAN:** `http://192.168.x.x:PORTA/webhooks/whatsapp/minha-loja` (IP do PC onde a API corre).

5. **Túnel (ngrok, Cloudflare Tunnel, etc.):** se a Evolution está na internet e o seu PC não, precisa de HTTPS público a apontar para a porta da API.

6. Se usou `WEBHOOK_SHARED_SECRET`, configure o envio desse header na Evolution ou no proxy.

---

## Passo 6 — Enviar uma mensagem de teste (sem WhatsApp à mão)

Com a API a correr, use **Postman**, **Insomnia** ou `curl`:

- **Método:** `POST`  
- **URL:** `http://localhost:8080/api/v1/internal/evolution/send`  
- **Header:** `X-Internal-API-Key: <o mesmo valor de INTERNAL_API_KEY do .env>`  
- **Header:** `Content-Type: application/json`  
- **Corpo:**

```json
{
  "number": "5511999999999",
  "text": "Teste a partir da API",
  "instance": "NOME_DA_INSTANCIA_OPCIONAL"
}
```

O número é só dígitos com código do país (Brasil: 55…). Em Evolution Go, o campo `instance` deve ser o **token da instância**; se omitir, usa `EVOLUTION_INSTANCE_NAME` do `.env` como fallback.

---

## Passo 7 — Ligar o site (dashboard) à API

Na pasta `apps/web`, copie `.env.example` para `.env` e confirme:

```env
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:8080/ws
```

(use a mesma porta que `HTTP_PORT` no backend, ex. `8088`). O WebSocket atualiza a Inbox em tempo real; sem `VITE_WS_URL`, a lista de mensagens da conversa aberta faz **poll** periódico. O webhook e o `/health` **não** levam o prefixo `/api/v1`.

---

## Passo 8 (opcional) — Rodar a API em container Docker

Com Docker a funcionar e o `backend/.env` preenchido:

```bash
docker compose -f infra/docker-compose.dev.yml --profile api up -d --build
```

Ajuste `EVOLUTION_BASE_URL` no `.env` para alcançar a Evolution no host (`http://host.docker.internal:PORTA` no Windows/Mac com Docker Desktop).

---

## Checklist rápido

- [ ] Docker com Postgres + Redis no ar  
- [ ] `backend/.env` criado e preenchido  
- [ ] `go run ./cmd/api` sem erros  
- [ ] `/health` e `/api/v1/meta` respondem  
- [ ] Webhook na Evolution aponta para `/webhooks/whatsapp/<instância>`  
- [ ] **Sincronizar webhook** na app (Instâncias) após mudar URL/porta ou Docker  
- [ ] Segredo do webhook configurado (ou modo inseguro **só** em dev)  
- [ ] Teste de envio com `X-Internal-API-Key` OK  
- [ ] Se quiser bot automático: `AUTO_REPLY_ENABLED=true` e chave certa (`GEMINI_API_KEY` ou `OPENAI_API_KEY` conforme `LLM_PROVIDER`)  

---

## Problemas frequentes

| Sintoma | O que verificar |
|---------|------------------|
| `connection refused` ao Postgres | Compose está `up`? Porta 5432 livre? `DATABASE_URL` igual ao utilizador/palavra-passe do compose? |
| Webhook 401 | `EVOLUTION_WEBHOOK_API_KEY` / `WEBHOOK_SHARED_SECRET` / header `X-Webhook-Secret` alinhados com o que a Evolution envia |
| Evolution não alcança o PC | Firewall, IP errado, ou falta de túnel se a Evolution está na nuvem |
| Consigo **enviar** WhatsApp mas **não vejo o que me mandam** | Webhook: URL alcançável pela Evolution (`host.docker.internal` + porta se a Evolution está no Docker), `PUBLIC_WEBHOOK_BASE_URL` certo, **sincronizar webhook** na app após mudar rede; logs da API (`webhook: inbound…`) se ainda falhar |
| Mensagens na UI atrasadas | Definir `VITE_WS_URL` alinhado à API; sem WS há poll ~12s na conversa aberta |
| IA dá erro (Gemini/GPT) | Chave ativa, modelo existe, `LLM_PROVIDER` alinhado à chave que preencheu; quotas na Google Cloud / OpenAI |
| CORS no browser | `CORS_ALLOW_ORIGINS` inclui `http://localhost:5173` |

---

## Próximo nível (quando for para produção)

- Tirar `INSECURE_SKIP_WEBHOOK_AUTH`.  
- HTTPS real (proxy reverso), palavras-passe fortes, segredos em cofre (não em ficheiros partilhados).  
- Substituir `INTERNAL_API_KEY` por login JWT e permissões (como no playbook).  
- Filas e workers para processar webhooks sem depender só do pedido HTTP.

---

*Última atualização: MVP backend (Fiber, webhook Evolution, resposta automática com Gemini por padrão ou OpenAI opcional).*
