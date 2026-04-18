# 🚀 WhatsApp AI SaaS — Playbook Completo
> **Versão:** 1.0 | **Autor:** Nathan | **Stack:** Go + React + PostgreSQL + Redis  
> *Documento vivo — atualize conforme o projeto evolui*

---

## 📋 Índice

1. [Visão do Produto](#1-visão-do-produto)
2. [Stack Técnica Completa](#2-stack-técnica-completa)
3. [Arquitetura do Sistema](#3-arquitetura-do-sistema)
4. [Estrutura de Diretórios](#4-estrutura-de-diretórios)
5. [Modelagem de Dados](#5-modelagem-de-dados)
6. [API Design & Contratos](#6-api-design--contratos)
7. [Segurança & Cybersecurity](#7-segurança--cybersecurity)
8. [Guia de Desenvolvimento Local](#8-guia-de-desenvolvimento-local)
9. [Variáveis de Ambiente](#9-variáveis-de-ambiente)
10. [Playbook de Prompts para IA (Cursor/Claude)](#10-playbook-de-prompts-para-ia-cursorclaude)
11. [Roadmap de Features](#11-roadmap-de-features)
12. [Deploy & Infraestrutura](#12-deploy--infraestrutura)
13. [Checklist de Produção](#13-checklist-de-produção)

---

## 1. Visão do Produto

### O que é
Plataforma SaaS **multi-tenant** de automação de atendimento via WhatsApp com IA, inspirada em Chatwoot + WhatsCRM. Permite que empresas conectem múltiplas instâncias do WhatsApp, criem agentes de IA personalizados, gerenciem conversas em equipe e automatizem fluxos de atendimento.

### Personas
| Persona | Dor | O que a plataforma resolve |
|---------|-----|---------------------------|
| Dono de PME | Perde vendas por não responder rápido | Bot 24/7 com handoff humano |
| Gestor de CS | Equipe sobrecarregada, sem visibilidade | Inbox multiagente + métricas |
| Agência de Marketing | Gerencia clientes diferentes | Multi-tenant com workspaces isolados |

### Modelo de Negócio
- **Freemium:** 1 instância, 500 mensagens/mês
- **Starter R$97/mês:** 3 instâncias, 5.000 mensagens, 2 agentes IA
- **Pro R$297/mês:** 10 instâncias, ilimitado, 10 agentes IA, RAG
- **Enterprise:** Custom, SLA, suporte dedicado

---

## 2. Stack Técnica Completa

### Backend
| Camada | Tecnologia | Justificativa |
|--------|-----------|---------------|
| Linguagem | **Go 1.23+** | Performance, concorrência nativa, binário único |
| Framework HTTP | **Fiber v2** | Fasthttp, API similar ao Express, middleware rico |
| ORM | **GORM v2** | Migrations, associations, hooks — maduro |
| Banco Principal | **PostgreSQL 16** | ACID, JSONB, pgvector para RAG |
| Cache / Filas | **Redis 7** | Session, pub/sub, rate limit, filas Asynq |
| Filas de Jobs | **Asynq** | Workers assíncronos sobre Redis |
| WebSocket | **Fiber WebSocket** | Real-time inbox, notificações |
| Validação | **go-playground/validator** | Tags de validação nas structs |
| JWT | **golang-jwt/jwt v5** | Access token (15min) + Refresh token (7d) |
| Migrations | **golang-migrate** | Versionamento de schema SQL |
| Logs | **Zap (Uber)** | Structured logging, performance |
| Métricas | **Prometheus + Grafana** | Observabilidade em produção |
| Tracing | **OpenTelemetry** | Distributed tracing |
| Testes | **testify + httptest** | Unit + integration tests |

### Frontend
| Camada | Tecnologia | Justificativa |
|--------|-----------|---------------|
| Framework | **React 18 + TypeScript** | Ecossistema, tipagem, DX |
| Build | **Vite 5** | HMR instantâneo, build otimizado |
| Estilização | **Tailwind CSS 3** | Utilitário, sem CSS custom |
| Componentes | **shadcn/ui** | Acessível, customizável, sem lock-in |
| Roteamento | **React Router v6** | SPA client-side |
| Server State | **TanStack Query v5** | Cache, revalidação, mutations |
| Forms | **React Hook Form + Zod** | Performance, validação tipada |
| Drag & Drop | **@hello-pangea/dnd** | Kanban board |
| Gráficos | **Recharts** | Composable, React-native |
| HTTP Client | **Axios** | Interceptors, instância centralizada |
| Ícones | **Lucide React** | Consistente, tree-shakeable |
| Datas | **date-fns** | Leve, funcional, i18n |
| Toasts | **Sonner** | Minimalista, acessível |
| Testes | **Vitest + Testing Library** | Rápido, compatível Vite |

### Integrações Externas
| Serviço | Uso |
|---------|-----|
| **Evolution API** | WhatsApp via QR Code (self-hosted) |
| **Meta Cloud API** | WhatsApp Business oficial |
| **Anthropic (Claude)** | LLM para agentes IA |
| **OpenAI (GPT-4o)** | LLM alternativo |
| **Groq (LLaMA 3)** | LLM rápido e barato |
| **Qdrant / pgvector** | Vector store para RAG |
| **Asaas / AbacatePay** | Pagamentos BR |
| **Resend / SendGrid** | E-mails transacionais |
| **Cloudflare R2** | Storage de mídia (S3-compatible) |
| **Sentry** | Error tracking frontend + backend |

### Infraestrutura
| Ambiente | Stack |
|----------|-------|
| **Local Dev** | Docker Compose (todos os serviços) |
| **Produção** | Hetzner VPS (CX31) + Coolify + Caddy |
| **CI/CD** | GitHub Actions |
| **Registros** | GitHub Container Registry (GHCR) |
| **DNS / CDN** | Cloudflare |
| **Certificados** | Let's Encrypt via Caddy (automático) |

---

## 3. Arquitetura do Sistema

```
┌─────────────────────────────────────────────────────────────┐
│                        INTERNET                              │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTPS / WSS
                    ┌──────▼───────┐
                    │   Cloudflare  │  WAF + DDoS + CDN
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │    Caddy      │  Reverse Proxy + TLS
                    └──┬───────┬───┘
                       │       │
          ┌────────────▼──┐ ┌──▼────────────┐
          │  React SPA    │ │   Go API       │
          │  (static)     │ │   (Fiber)      │
          └───────────────┘ └──┬────────┬───┘
                               │        │
               ┌───────────────▼─┐  ┌───▼──────────────┐
               │   PostgreSQL 16  │  │    Redis 7        │
               │   + pgvector     │  │  Cache + Pub/Sub  │
               └─────────────────┘  └───────────────────┘
                                             │
                                    ┌────────▼──────────┐
                                    │   Asynq Workers    │
                                    │  (jobs assíncronos)│
                                    └────────┬──────────┘
                                             │
                              ┌──────────────┼──────────────┐
                              │              │              │
                    ┌─────────▼──┐  ┌────────▼──┐  ┌───────▼─────┐
                    │ Evolution  │  │  LLM APIs  │  │  Cloudflare │
                    │    API     │  │ Claude/GPT │  │     R2      │
                    └────────────┘  └───────────┘  └─────────────┘
```

### Padrão Multi-Tenant
- Cada **Workspace** é um tenant isolado
- Row-Level Security no PostgreSQL via `workspace_id` em todas as tabelas
- Middleware Go verifica `workspace_id` do JWT em **todo request**
- Dados de mídia separados por `workspaces/{id}/` no R2

### Fluxo de Mensagem WhatsApp (Inbound)
```
WhatsApp → Evolution API → Webhook POST /webhooks/whatsapp
    → Go API valida HMAC signature
    → Salva mensagem no PostgreSQL
    → Publica evento no Redis Pub/Sub
    → Worker verifica se há agente IA ativo
        → Se sim: envia para LLM → retorna resposta → Evolution API → WhatsApp
        → Se não: notifica agentes humanos via WebSocket
    → Frontend atualiza inbox em real-time
```

---

## 4. Estrutura de Diretórios

### Backend (Go)
```
backend/
├── cmd/
│   ├── api/
│   │   └── main.go              # Entry point da API
│   └── worker/
│       └── main.go              # Entry point dos workers Asynq
├── internal/
│   ├── config/
│   │   └── config.go            # Carrega .env, valida obrigatórios
│   ├── database/
│   │   ├── postgres.go          # Conexão + pool PostgreSQL
│   │   └── redis.go             # Conexão Redis
│   ├── middleware/
│   │   ├── auth.go              # JWT validation + workspace injection
│   │   ├── ratelimit.go         # Rate limiting por IP e por user
│   │   ├── cors.go              # CORS configurado por env
│   │   ├── logger.go            # Request logging com Zap
│   │   ├── recover.go           # Panic recovery
│   │   └── tenant.go            # Multi-tenant workspace guard
│   ├── domain/                  # Entidades de negócio (sem dependências)
│   │   ├── workspace.go
│   │   ├── user.go
│   │   ├── conversation.go
│   │   ├── message.go
│   │   ├── contact.go
│   │   ├── agent.go
│   │   ├── instance.go
│   │   ├── campaign.go
│   │   └── flow.go
│   ├── repository/              # Acesso ao banco (interfaces + implementações)
│   │   ├── interfaces.go        # Contratos (facilita mock em testes)
│   │   ├── workspace_repo.go
│   │   ├── user_repo.go
│   │   ├── conversation_repo.go
│   │   ├── message_repo.go
│   │   ├── contact_repo.go
│   │   ├── agent_repo.go
│   │   └── instance_repo.go
│   ├── service/                 # Lógica de negócio
│   │   ├── auth_service.go
│   │   ├── conversation_service.go
│   │   ├── ai_service.go        # Orquestra chamadas LLM
│   │   ├── whatsapp_service.go  # Abstração Evolution/Meta API
│   │   ├── campaign_service.go
│   │   └── rag_service.go       # RAG com pgvector
│   ├── handler/                 # HTTP handlers (controllers)
│   │   ├── auth_handler.go
│   │   ├── conversation_handler.go
│   │   ├── contact_handler.go
│   │   ├── agent_handler.go
│   │   ├── instance_handler.go
│   │   ├── campaign_handler.go
│   │   ├── analytics_handler.go
│   │   ├── webhook_handler.go   # Recebe eventos WhatsApp
│   │   └── ws_handler.go        # WebSocket handler
│   ├── worker/                  # Asynq task handlers
│   │   ├── tasks.go             # Constantes de nomes de tasks
│   │   ├── ai_reply_worker.go   # Processa respostas IA
│   │   ├── campaign_worker.go   # Dispara campanhas
│   │   └── cleanup_worker.go    # Tarefas de manutenção
│   └── router/
│       └── router.go            # Definição de todas as rotas
├── migrations/                  # SQL migrations numeradas
│   ├── 000001_create_workspaces.up.sql
│   ├── 000001_create_workspaces.down.sql
│   └── ...
├── pkg/                         # Pacotes reutilizáveis (sem deps internas)
│   ├── crypto/                  # Hashing, encryption helpers
│   ├── pagination/              # Cursor-based pagination helper
│   ├── validator/               # Custom validators
│   └── response/                # Padronização de respostas JSON
├── docker/
│   ├── Dockerfile               # Multi-stage build
│   └── docker-compose.yml       # Dev environment completo
├── .env.example
├── go.mod
├── go.sum
└── Makefile                     # Comandos: make dev, make test, make migrate
```

### Frontend (React)
```
frontend/
├── src/
│   ├── components/
│   │   ├── layout/
│   │   │   ├── AppShell.tsx     # Layout principal com sidebar
│   │   │   ├── Sidebar.tsx
│   │   │   ├── Topbar.tsx
│   │   │   └── MobileNav.tsx
│   │   ├── shared/
│   │   │   ├── ConversationCard.tsx
│   │   │   ├── MessageBubble.tsx
│   │   │   ├── StatusBadge.tsx
│   │   │   ├── EmptyState.tsx
│   │   │   ├── LoadingSkeleton.tsx
│   │   │   ├── ConfirmDialog.tsx
│   │   │   └── CommandPalette.tsx  # Ctrl+K
│   │   └── ui/                  # shadcn/ui components
│   ├── pages/
│   │   ├── LoginPage.tsx
│   │   ├── InboxPage.tsx
│   │   ├── ContactsPage.tsx
│   │   ├── ContactDetailPage.tsx
│   │   ├── KanbanPage.tsx
│   │   ├── CampaignsPage.tsx
│   │   ├── AgentsPage.tsx
│   │   ├── InstancesPage.tsx
│   │   ├── FlowsPage.tsx
│   │   ├── FlowEditorPage.tsx
│   │   ├── AnalyticsPage.tsx
│   │   ├── SettingsPage.tsx
│   │   └── NotFoundPage.tsx
│   ├── hooks/
│   │   ├── useAuth.ts           # Auth state + logout
│   │   ├── useWebSocket.ts      # Conexão WS + reconexão automática
│   │   ├── useConversations.ts  # Queries de conversas
│   │   └── useDebounce.ts
│   ├── lib/
│   │   ├── api.ts               # Axios instance + interceptors
│   │   └── queryClient.ts       # TanStack Query config
│   ├── types/
│   │   └── index.ts             # Todas as interfaces TypeScript
│   ├── utils/
│   │   ├── formatDate.ts
│   │   ├── formatPhone.ts
│   │   └── cn.ts                # Tailwind class merge helper
│   ├── store/
│   │   └── authStore.ts         # Zustand (token, user, workspace)
│   ├── App.tsx                  # Router setup + ProtectedRoute
│   └── main.tsx
├── public/
├── index.html
├── vite.config.ts
├── tailwind.config.ts
├── tsconfig.json
├── .env.example
└── package.json
```

---

## 5. Modelagem de Dados

### Schema Principal (PostgreSQL)

```sql
-- WORKSPACES (multi-tenant root)
CREATE TABLE workspaces (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name        VARCHAR(255) NOT NULL,
  slug        VARCHAR(100) UNIQUE NOT NULL,
  plan        VARCHAR(50) NOT NULL DEFAULT 'free',
  logo_url    TEXT,
  settings    JSONB DEFAULT '{}',
  created_at  TIMESTAMPTZ DEFAULT NOW(),
  updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- USERS
CREATE TABLE users (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  name            VARCHAR(255) NOT NULL,
  email           VARCHAR(255) NOT NULL,
  password_hash   TEXT NOT NULL,
  role            VARCHAR(50) NOT NULL DEFAULT 'agent', -- admin | supervisor | agent
  avatar_url      TEXT,
  is_active       BOOLEAN DEFAULT TRUE,
  last_seen_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(workspace_id, email)
);

-- REFRESH TOKENS
CREATE TABLE refresh_tokens (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash   TEXT NOT NULL UNIQUE,
  expires_at   TIMESTAMPTZ NOT NULL,
  created_at   TIMESTAMPTZ DEFAULT NOW()
);

-- WHATSAPP INSTANCES
CREATE TABLE instances (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  name            VARCHAR(255) NOT NULL,
  phone_number    VARCHAR(50),
  provider        VARCHAR(50) NOT NULL DEFAULT 'evolution', -- evolution | meta
  external_id     TEXT,                -- ID na Evolution API ou Meta
  status          VARCHAR(50) DEFAULT 'disconnected',
  agent_id        UUID REFERENCES ai_agents(id) ON DELETE SET NULL,
  settings        JSONB DEFAULT '{}',
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- CONTACTS
CREATE TABLE contacts (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  name            VARCHAR(255),
  phone           VARCHAR(50) NOT NULL,
  email           VARCHAR(255),
  company         VARCHAR(255),
  avatar_url      TEXT,
  tags            TEXT[] DEFAULT '{}',
  custom_attrs    JSONB DEFAULT '{}',
  pipeline_stage  VARCHAR(100),
  assigned_to     UUID REFERENCES users(id) ON DELETE SET NULL,
  blocked         BOOLEAN DEFAULT FALSE,
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  updated_at      TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(workspace_id, phone)
);

-- CONVERSATIONS
CREATE TABLE conversations (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  contact_id      UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  instance_id     UUID REFERENCES instances(id) ON DELETE SET NULL,
  assigned_to     UUID REFERENCES users(id) ON DELETE SET NULL,
  status          VARCHAR(50) DEFAULT 'open', -- open | pending | resolved | snoozed
  channel         VARCHAR(50) DEFAULT 'whatsapp',
  labels          TEXT[] DEFAULT '{}',
  unread_count    INT DEFAULT 0,
  last_message_at TIMESTAMPTZ,
  snoozed_until   TIMESTAMPTZ,
  meta            JSONB DEFAULT '{}',
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- MESSAGES
CREATE TABLE messages (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  sender_type     VARCHAR(50) NOT NULL, -- contact | user | bot | system
  sender_id       UUID,                 -- NULL para bot/system
  content         TEXT,
  content_type    VARCHAR(50) DEFAULT 'text', -- text | image | audio | video | document | template
  media_url       TEXT,
  media_meta      JSONB DEFAULT '{}',
  is_private      BOOLEAN DEFAULT FALSE, -- notas privadas
  external_id     TEXT,                  -- ID na Evolution/Meta API
  status          VARCHAR(50) DEFAULT 'sent', -- sent | delivered | read | failed
  ai_generated    BOOLEAN DEFAULT FALSE,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- AI AGENTS
CREATE TABLE ai_agents (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  name            VARCHAR(255) NOT NULL,
  description     TEXT,
  model           VARCHAR(100) NOT NULL, -- claude-sonnet-4-5 | gpt-4o | groq-llama3
  system_prompt   TEXT NOT NULL,
  temperature     DECIMAL(3,2) DEFAULT 0.7,
  max_tokens      INT DEFAULT 1000,
  use_rag         BOOLEAN DEFAULT FALSE,
  use_memory      BOOLEAN DEFAULT TRUE,
  handoff_enabled BOOLEAN DEFAULT TRUE,
  handoff_keyword VARCHAR(100) DEFAULT 'humano',
  is_active       BOOLEAN DEFAULT TRUE,
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- RAG KNOWLEDGE BASE
CREATE TABLE knowledge_items (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  agent_id        UUID REFERENCES ai_agents(id) ON DELETE CASCADE,
  title           TEXT NOT NULL,
  content         TEXT NOT NULL,
  embedding       vector(1536),          -- pgvector
  source_type     VARCHAR(50),           -- manual | url | pdf
  source_url      TEXT,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- CAMPAIGNS
CREATE TABLE campaigns (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  name            VARCHAR(255) NOT NULL,
  instance_id     UUID REFERENCES instances(id),
  message_template TEXT NOT NULL,
  status          VARCHAR(50) DEFAULT 'draft', -- draft | scheduled | sending | completed | failed
  total_contacts  INT DEFAULT 0,
  sent_count      INT DEFAULT 0,
  delivered_count INT DEFAULT 0,
  read_count      INT DEFAULT 0,
  failed_count    INT DEFAULT 0,
  scheduled_at    TIMESTAMPTZ,
  completed_at    TIMESTAMPTZ,
  created_by      UUID REFERENCES users(id),
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- FLOWS (chatbot)
CREATE TABLE flows (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  name            VARCHAR(255) NOT NULL,
  description     TEXT,
  nodes           JSONB NOT NULL DEFAULT '[]',
  edges           JSONB NOT NULL DEFAULT '[]',
  agent_id        UUID REFERENCES ai_agents(id),
  is_active       BOOLEAN DEFAULT FALSE,
  trigger_type    VARCHAR(50) DEFAULT 'keyword', -- keyword | all | first_message
  trigger_value   TEXT,
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- CANNED RESPONSES
CREATE TABLE canned_responses (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  short_code      VARCHAR(100) NOT NULL,
  content         TEXT NOT NULL,
  created_by      UUID REFERENCES users(id),
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(workspace_id, short_code)
);

-- AUDIT LOG (segurança)
CREATE TABLE audit_logs (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id    UUID REFERENCES workspaces(id) ON DELETE SET NULL,
  user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
  action          VARCHAR(100) NOT NULL, -- user.login | contact.delete | etc
  resource_type   VARCHAR(100),
  resource_id     UUID,
  ip_address      INET,
  user_agent      TEXT,
  meta            JSONB DEFAULT '{}',
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- INDEXES CRÍTICOS
CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX idx_conversations_workspace_status ON conversations(workspace_id, status);
CREATE INDEX idx_conversations_assigned_to ON conversations(assigned_to);
CREATE INDEX idx_contacts_workspace_phone ON contacts(workspace_id, phone);
CREATE INDEX idx_knowledge_items_embedding ON knowledge_items USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_audit_logs_workspace_created ON audit_logs(workspace_id, created_at DESC);
```

---

## 6. API Design & Contratos

### Convenções REST
- Base URL: `/api/v1`
- Autenticação: `Authorization: Bearer <access_token>`
- Content-Type: `application/json`
- Paginação: cursor-based com `?cursor=<uuid>&limit=50`
- Datas: sempre **ISO 8601 UTC**
- Erros: estrutura padronizada

### Estrutura de Resposta Padrão
```json
// Sucesso
{
  "data": { ... },
  "meta": { "cursor": "uuid", "has_more": true }
}

// Erro
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Campo obrigatório ausente",
    "details": [
      { "field": "phone", "message": "Número de telefone inválido" }
    ]
  }
}
```

### Endpoints Principais
```
# AUTH
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
POST   /api/v1/auth/forgot-password
POST   /api/v1/auth/reset-password

# WORKSPACES
GET    /api/v1/workspace
PATCH  /api/v1/workspace
POST   /api/v1/workspace/avatar

# USERS
GET    /api/v1/users
POST   /api/v1/users
GET    /api/v1/users/:id
PATCH  /api/v1/users/:id
DELETE /api/v1/users/:id

# CONVERSATIONS
GET    /api/v1/conversations          ?status=open&assigned_to=me&cursor=&limit=50
POST   /api/v1/conversations
GET    /api/v1/conversations/:id
PATCH  /api/v1/conversations/:id      (status, assigned_to, labels)
GET    /api/v1/conversations/:id/messages
POST   /api/v1/conversations/:id/messages
GET    /api/v1/conversations/counts   (unread, open, pending por tab)

# CONTACTS
GET    /api/v1/contacts
POST   /api/v1/contacts
GET    /api/v1/contacts/:id
PATCH  /api/v1/contacts/:id
DELETE /api/v1/contacts/:id
POST   /api/v1/contacts/import        (CSV upload)

# AI AGENTS
GET    /api/v1/agents
POST   /api/v1/agents
GET    /api/v1/agents/:id
PATCH  /api/v1/agents/:id
DELETE /api/v1/agents/:id
POST   /api/v1/agents/:id/test        (testa mensagem)

# INSTANCES
GET    /api/v1/instances
POST   /api/v1/instances
GET    /api/v1/instances/:id
GET    /api/v1/instances/:id/status
GET    /api/v1/instances/:id/qrcode
DELETE /api/v1/instances/:id
DELETE /api/v1/instances/:id/session  (desconectar)

# CAMPAIGNS
GET    /api/v1/campaigns
POST   /api/v1/campaigns
GET    /api/v1/campaigns/:id
PATCH  /api/v1/campaigns/:id
DELETE /api/v1/campaigns/:id
POST   /api/v1/campaigns/:id/send

# FLOWS
GET    /api/v1/flows
POST   /api/v1/flows
GET    /api/v1/flows/:id
PATCH  /api/v1/flows/:id
DELETE /api/v1/flows/:id
POST   /api/v1/flows/:id/publish

# ANALYTICS
GET    /api/v1/analytics/overview     ?from=&to=
GET    /api/v1/analytics/messages     ?from=&to=
GET    /api/v1/analytics/agents       ?from=&to=

# WEBHOOKS (sem auth JWT, validado por HMAC)
POST   /webhooks/whatsapp/:instance_id

# WEBSOCKET
WS     /ws?token=<jwt>

# SETTINGS
GET    /api/v1/settings
PATCH  /api/v1/settings
GET    /api/v1/settings/api-keys
POST   /api/v1/settings/api-keys
DELETE /api/v1/settings/api-keys/:id
```

---

## 7. Segurança & Cybersecurity

### 🔐 Autenticação & Autorização

```
Access Token:  JWT RS256 (não HS256!), expira em 15 minutos
Refresh Token: Opaque token (UUID), armazenado em banco (hash bcrypt), expira em 7 dias
Rotação:       Refresh token rotacionado a cada uso (Refresh Token Rotation)
Revogação:     Blacklist de tokens no Redis para logout imediato
```

**Claims do JWT:**
```json
{
  "sub": "user-uuid",
  "workspace_id": "workspace-uuid",
  "role": "admin",
  "iat": 1712345678,
  "exp": 1712346578
}
```

**RBAC (Role-Based Access Control):**
| Recurso | Admin | Supervisor | Agent |
|---------|-------|-----------|-------|
| Gerenciar usuários | ✅ | ❌ | ❌ |
| Criar agentes IA | ✅ | ✅ | ❌ |
| Criar instâncias | ✅ | ❌ | ❌ |
| Ver todas conversas | ✅ | ✅ | ❌ |
| Ver próprias conversas | ✅ | ✅ | ✅ |
| Criar campanhas | ✅ | ✅ | ❌ |
| Ver analytics | ✅ | ✅ | ❌ |
| Configurações workspace | ✅ | ❌ | ❌ |

### 🛡️ Proteções HTTP

```go
// Todos obrigatórios no middleware stack do Fiber:

// 1. Helmet — Security headers
app.Use(helmet.New(helmet.Config{
    XSSProtection:         "1; mode=block",
    ContentTypeNosniff:    "nosniff",
    XFrameOptions:         "DENY",
    HSTSMaxAge:            31536000,
    ContentSecurityPolicy: "default-src 'self'",
    ReferrerPolicy:        "strict-origin-when-cross-origin",
}))

// 2. CORS — Apenas origens permitidas
app.Use(cors.New(cors.Config{
    AllowOrigins:     os.Getenv("ALLOWED_ORIGINS"), // nunca "*" em produção
    AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
    AllowHeaders:     "Authorization,Content-Type,X-Request-ID",
    AllowCredentials: false,
    MaxAge:           86400,
}))

// 3. Rate Limiting — por IP e por usuário autenticado
// - Geral: 100 req/min por IP
// - Auth endpoints: 5 req/min por IP (força bruta)
// - Webhooks: 500 req/min por instance_id

// 4. Request ID — rastreabilidade
app.Use(requestid.New())

// 5. Body size limit
app.Use(func(c *fiber.Ctx) error {
    if c.Request().Header.ContentLength() > 10*1024*1024 { // 10MB
        return fiber.ErrRequestEntityTooLarge
    }
    return c.Next()
})
```

### 🔒 Proteção de Dados

**Senhas:**
```go
// Sempre bcrypt com custo 12
hash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)

// NUNCA armazenar senha em texto simples
// NUNCA logar senhas ou tokens
```

**Dados sensíveis:**
- API keys de LLMs: criptografadas com AES-256-GCM no banco
- Webhooks secrets: armazenados criptografados
- Logs: nunca incluir `Authorization` header, body de auth, ou PII

**SQL Injection:**
```go
// SEMPRE usar parameterized queries — NUNCA concatenar strings
// Com GORM isso é automático, mas em raw queries:
db.Raw("SELECT * FROM users WHERE workspace_id = ? AND email = ?", workspaceID, email)
// NUNCA: fmt.Sprintf("... WHERE email = '%s'", email)
```

**Validação de Input:**
```go
// Toda struct de request deve ter tags de validação
type CreateContactRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=255"`
    Phone string `json:"phone" validate:"required,e164"` // formato E.164: +5511999999999
    Email string `json:"email" validate:"omitempty,email"`
}
```

### 🔑 Webhook Security (Evolution API)
```go
// Validar HMAC-SHA256 signature em todo webhook
func ValidateWebhookSignature(body []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
// Se assinatura inválida: retorna 401 e registra no audit log
```

### 🚨 Audit Log
Todo evento sensível deve ser registrado:
```go
// Eventos a auditar obrigatoriamente:
// - auth: login, logout, failed_login, password_change
// - user: created, updated, deleted, role_changed
// - instance: created, connected, disconnected, deleted
// - contact: bulk_deleted, blocked, exported
// - campaign: created, sent, deleted
// - settings: api_key_created, api_key_revoked
// - workspace: plan_changed
```

### 🛡️ Frontend Security

```typescript
// 1. Nunca armazenar dados sensíveis além do JWT
// O token fica em localStorage — aceitável para SPA, mas:
// - Token expira em 15min (curto)
// - Refresh token NUNCA vai para o frontend

// 2. Sanitizar todo conteúdo de mensagens antes de renderizar
// Usar DOMPurify se renderizar HTML de usuário
import DOMPurify from 'dompurify'
const safeHTML = DOMPurify.sanitize(message.content)

// 3. CSP no Vite build (via plugin vite-plugin-csp)
// Bloqueia execução de scripts inline injetados

// 4. Nunca expor variáveis de ambiente sensíveis
// VITE_API_BASE_URL = OK (público)
// API keys de terceiros = NUNCA no frontend, sempre no backend

// 5. Dependências — auditar regularmente
// npm audit fix
// Usar Dependabot ou Renovate para atualizações automáticas
```

### 🔥 Proteção contra Ataques Comuns

| Ataque | Proteção |
|--------|----------|
| Brute Force | Rate limit 5 req/min em /auth/login por IP |
| SQL Injection | Parameterized queries (GORM) |
| XSS | CSP headers + DOMPurify no frontend |
| CSRF | SPA + JWT (sem cookies) = imune por design |
| IDOR | Middleware tenant verifica workspace_id em todo request |
| DDoS | Cloudflare WAF (camada 7) + rate limit no backend |
| Path Traversal | Nunca concatenar input de usuário em file paths |
| Mass Assignment | Whitelist explícita de campos atualizáveis |
| Sensitive Data Exposure | TLS everywhere, HSTS, sem HTTP |

---

## 8. Guia de Desenvolvimento Local

### Pré-requisitos
```bash
- Go 1.23+
- Node.js 20+
- Docker Desktop
- make
- golang-migrate CLI
```

### Setup inicial
```bash
# 1. Clone o repositório
git clone https://github.com/seu-usuario/whatsapp-saas
cd whatsapp-saas

# 2. Copie os arquivos de env
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env

# 3. Suba a infraestrutura local
docker compose up -d postgres redis

# 4. Rode as migrations
cd backend
make migrate-up

# 5. Seed de dados de desenvolvimento (opcional)
make seed

# 6. Inicie o backend
make dev   # air para hot reload

# 7. Inicie o frontend (outro terminal)
cd frontend
npm install
npm run dev

# API disponível em:  http://localhost:8080
# Frontend em:        http://localhost:5173
# Adminer (DB UI):    http://localhost:8081
# Redis Commander:    http://localhost:8082
```

### docker-compose.yml (desenvolvimento)
```yaml
version: '3.9'
services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: whatsapp_saas
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports: ["5432:5432"]
    volumes: [postgres_data:/var/lib/postgresql/data]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    command: redis-server --appendonly yes
    volumes: [redis_data:/data]

  adminer:
    image: adminer
    ports: ["8081:8080"]

  redis-commander:
    image: rediscommander/redis-commander
    environment:
      REDIS_HOSTS: local:redis:6379
    ports: ["8082:8081"]

volumes:
  postgres_data:
  redis_data:
```

### Makefile
```makefile
.PHONY: dev test migrate-up migrate-down seed lint build

dev:
	air -c .air.toml

test:
	go test ./... -v -cover

test-integration:
	go test ./... -v -tags=integration

migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

seed:
	go run ./cmd/seed/main.go

lint:
	golangci-lint run ./...

build:
	go build -ldflags="-s -w" -o bin/api ./cmd/api
	go build -ldflags="-s -w" -o bin/worker ./cmd/worker

docker-build:
	docker build -t whatsapp-saas-api:latest .
```

---

## 9. Variáveis de Ambiente

### Backend (`backend/.env`)
```bash
# Servidor
APP_ENV=development          # development | production
APP_PORT=8080
ALLOWED_ORIGINS=http://localhost:5173

# Banco de dados
DATABASE_URL=postgres://postgres:postgres@localhost:5432/whatsapp_saas?sslmode=disable
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5

# Redis
REDIS_URL=redis://localhost:6379/0

# JWT (GERE CHAVES SEGURAS EM PRODUÇÃO)
JWT_PRIVATE_KEY=<RS256 private key PEM base64>
JWT_PUBLIC_KEY=<RS256 public key PEM base64>
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h     # 7 dias

# Criptografia para API keys armazenadas
ENCRYPTION_KEY=<32 bytes hex — gere com: openssl rand -hex 32>

# Evolution API
EVOLUTION_API_URL=http://localhost:8000
EVOLUTION_API_KEY=seu_api_key_evolution

# LLMs (criptografados, inseridos via settings UI)
ANTHROPIC_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-...
GROQ_API_KEY=gsk_...

# Storage (Cloudflare R2 — compatível S3)
S3_ENDPOINT=https://<account>.r2.cloudflarestorage.com
S3_ACCESS_KEY=<access key>
S3_SECRET_KEY=<secret key>
S3_BUCKET=whatsapp-saas-media
S3_PUBLIC_URL=https://media.seudominio.com.br

# Email
RESEND_API_KEY=re_...
EMAIL_FROM=noreply@seudominio.com.br

# Pagamentos
ASAAS_API_KEY=<chave asaas>
ASAAS_WEBHOOK_SECRET=<secret>

# Sentry
SENTRY_DSN=https://...@sentry.io/...

# Worker
WORKER_CONCURRENCY=10
```

### Frontend (`frontend/.env`)
```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/ws
VITE_SENTRY_DSN=https://...@sentry.io/...
VITE_APP_VERSION=0.1.0
```

---

## 10. Playbook de Prompts para IA (Cursor/Claude)

> Use estes prompts no Cursor (Agent mode) ou Claude Code.  
> **Regra de ouro:** sempre forneça contexto → tarefa → restrições → output esperado.

---

### 🏗️ Prompt Base (cole no `.cursorrules` ou como System Prompt no Cursor)

```
Você é um engenheiro de software sênior com especialização em:
- Backend Go com Fiber, GORM, PostgreSQL e Redis
- Frontend React 18 + TypeScript + Tailwind CSS + shadcn/ui
- Arquitetura multi-tenant SaaS
- Segurança de aplicações web (OWASP Top 10)
- Clean Architecture e princípios SOLID

CONTEXTO DO PROJETO:
- SaaS de automação WhatsApp com IA
- Multi-tenant: cada request tem workspace_id no JWT
- Backend: Go + Fiber + GORM + PostgreSQL + Redis + Asynq
- Frontend: React + TypeScript + Vite + Tailwind + shadcn/ui + TanStack Query
- Todos os textos da UI em português brasileiro

REGRAS OBRIGATÓRIAS:
1. Nunca use fmt.Sprintf para montar queries SQL — sempre parameterized
2. Sempre valide workspace_id do JWT contra o recurso acessado
3. Toda struct de request Go deve ter tags `validate:` 
4. Toda função Go de serviço deve retornar (resultado, error)
5. Erros Go: wrap com fmt.Errorf("contexto: %w", err)
6. Frontend: toda chamada de API via TanStack Query (nunca useEffect+fetch direto)
7. Zod schema para todo form do frontend
8. Nunca exponha stack traces em respostas de erro da API em produção
9. Logs: nunca logar senhas, tokens ou dados sensíveis de usuário
10. Testes: toda função de service deve ter ao menos um teste unitário
```

---

### 📝 Prompts por Tarefa

#### Criar um novo endpoint REST

```
TAREFA: Crie o endpoint PATCH /api/v1/conversations/:id/assign

CONTEXTO:
- Permite atribuir (ou desatribuir) uma conversa a um agente
- Apenas Admin e Supervisor podem atribuir para qualquer agente
- Agent pode apenas se auto-atribuir

RESTRIÇÕES:
- Validar que o agente pertence ao mesmo workspace
- Registrar no audit_log com action "conversation.assigned"
- Emitir evento WebSocket para o novo agente notificando a atribuição
- Retornar a conversa atualizada com os dados do agente

ESTRUTURA ESPERADA:
1. Request struct com validação
2. Handler em internal/handler/conversation_handler.go
3. Service method em internal/service/conversation_service.go
4. Repository method em internal/repository/conversation_repo.go
5. Rota registrada em internal/router/router.go
6. Teste unitário do service
```

#### Criar componente React

```
TAREFA: Crie o componente ConversationCard.tsx

CONTEXTO: Card na lista de conversas do inbox (coluna esquerda)
ARQUIVO: src/components/shared/ConversationCard.tsx

DEVE EXIBIR:
- Avatar do contato (iniciais com cor baseada no nome via hash)
- Nome do contato + número de telefone formatado (BR)
- Ícone do canal (WhatsApp verde, etc.)
- Preview da última mensagem (truncado em 60 chars)
- Badge de mensagens não lidas (violeta, some quando = 0)
- Timestamp relativo com date-fns (pt-BR)
- Avatar pequeno do agente atribuído (canto inferior direito)
- Badges de labels como tags coloridas

ESTADO:
- Prop `isActive: boolean` → highlight com borda esquerda violeta
- Prop `onClick: () => void`

RESTRIÇÕES:
- TypeScript estrito (sem any)
- Tailwind only (sem CSS inline)
- Componente memoizado (React.memo)
- Acessível (role="button", tabIndex, onKeyDown para Enter/Space)
```

#### Implementar worker assíncrono

```
TAREFA: Implemente o worker de resposta automática com IA

ARQUIVO: internal/worker/ai_reply_worker.go

FLUXO:
1. Task enfileirada quando chega mensagem e a instância tem agente IA ativo
2. Worker busca as últimas 20 mensagens da conversa (contexto)
3. Monta o prompt com: system_prompt do agente + histórico + nova mensagem
4. Chama o LLM configurado no agente (Claude/GPT/Groq)
5. Salva a resposta como mensagem do tipo "bot" no banco
6. Envia a resposta via Evolution API
7. Emite evento WebSocket para o frontend atualizar a conversa
8. Em caso de erro: retenta até 3x com backoff exponencial

RESTRIÇÕES:
- Timeout de 30s por chamada LLM
- Se o agente tiver handoff_enabled e a mensagem contiver handoff_keyword:
  → NÃO responde com IA
  → Muda status da conversa para "pending"
  → Notifica agentes humanos via WebSocket
- Logar tempo de resposta do LLM como métrica Prometheus
```

#### Criar migration SQL

```
TAREFA: Crie a migration para adicionar a tabela pipeline_stages

ARQUIVO: migrations/000012_create_pipeline_stages.up.sql

SCHEMA:
- id UUID PK
- workspace_id UUID FK → workspaces (cascade delete)
- pipeline_id UUID FK → pipelines (cascade delete)
- name VARCHAR(100) NOT NULL
- color VARCHAR(7) (hex color, ex: #7C3AED)
- position INTEGER NOT NULL (ordem das colunas)
- created_at TIMESTAMPTZ

RESTRIÇÕES:
- Index em (workspace_id, pipeline_id)
- Unique em (pipeline_id, position)
- Também crie o arquivo .down.sql correspondente
```

#### Debugging com contexto

```
TAREFA: Debug — WebSocket desconecta após ~30 segundos no frontend

CONTEXTO:
- Hook: src/hooks/useWebSocket.ts
- Backend: internal/handler/ws_handler.go (Fiber WebSocket)
- Ocorre apenas em produção (Hetzner + Caddy)
- Em desenvolvimento funciona normalmente
- Browser console mostra: "WebSocket closed: code 1001"

INVESTIGUE:
1. Verifique se o Caddy está configurado com proxy_read_timeout adequado para WebSocket
2. Verifique se o Fiber tem configurado ReadTimeout/WriteTimeout muito curtos
3. Verifique se o frontend está enviando ping/pong (heartbeat) para manter a conexão viva
4. Verifique se o Cloudflare está com timeout de WebSocket padrão (100s)

IMPLEMENTE A SOLUÇÃO:
- Backend: ping/pong handler no ws_handler.go (30s interval)
- Frontend: reconnect automático com exponential backoff no useWebSocket.ts
- Caddy config: adicionar snippet para WebSocket com timeout adequado
```

#### Code Review Checklist

```
TAREFA: Faça code review do seguinte PR antes de fazer merge

CRITÉRIOS DE REVISÃO:
1. SEGURANÇA
   - Há alguma query SQL sem parameterização?
   - Há dados sensíveis sendo logados?
   - O workspace_id está sendo validado em todo acesso a recursos?
   - Há possibilidade de IDOR (acesso a recurso de outro tenant)?

2. PERFORMANCE
   - Há N+1 queries (loop com query dentro)?
   - Os índices necessários foram criados na migration?
   - Há chamadas de API síncronas que deveriam ser assíncronas?

3. QUALIDADE
   - Todos os erros estão sendo tratados (sem `_` em erros Go)?
   - Há testes para a lógica de negócio?
   - As funções têm mais de 50 linhas? (sinal de necessidade de refatoração)

4. CONVENÇÕES DO PROJETO
   - Seguiu a estrutura handler → service → repository?
   - Textos da UI em português brasileiro?
   - TanStack Query no frontend (não useEffect+fetch)?

APONTE cada problema com: arquivo, linha, severidade (crítico/médio/sugestão) e solução.
```

---

### 🎯 Prompts de Arquitetura

#### Planejar nova feature

```
TAREFA: Planeje a implementação da feature "Campanha com disparo programado"

ME FORNEÇA:
1. Lista de mudanças necessárias no banco (nova migration ou alteração em tabelas existentes)
2. Novos endpoints de API necessários
3. Jobs Asynq necessários (workers assíncronos)
4. Componentes React novos ou alterados
5. Considerações de segurança específicas desta feature
6. Possíveis gargalos de performance com alto volume (ex: 10.000 contatos)
7. Estimativa de complexidade (P (horas), M (1-2 dias), G (3-5 dias))

RESTRIÇÕES DO PROJETO:
- Multi-tenant: tudo isolado por workspace_id
- Workers via Asynq sobre Redis
- Rate limit de envio: 1 mensagem/segundo por instância (limite WhatsApp)
```

---

## 11. Roadmap de Features

### MVP (Local Dev — Fase 1)
- [x] Estrutura base do projeto (Go + React)
- [ ] Auth (login, refresh, logout)
- [ ] Multi-tenant workspaces
- [ ] Instâncias WhatsApp (Evolution API + QR Code)
- [ ] Inbox básico (receber e enviar mensagens)
- [ ] Contatos (CRUD)
- [ ] Agentes IA (Claude/GPT/Groq)
- [ ] Resposta automática com IA
- [ ] Handoff IA → Humano

### V1.0 (Produção Inicial)
- [ ] Campanhas de broadcast
- [ ] Kanban CRM
- [ ] Flow Builder visual
- [ ] Notas privadas e menções
- [ ] Canned responses
- [ ] Analytics básico
- [ ] Billing (Asaas — planos e limites)
- [ ] Onboarding wizard

### V1.5 (Crescimento)
- [ ] RAG (base de conhecimento para agentes)
- [ ] Multi-canal (Instagram, Email)
- [ ] Meta Cloud API (oficial)
- [ ] CSAT (pesquisa de satisfação)
- [ ] Relatórios exportáveis (PDF/CSV)
- [ ] White-label (logo customizável por workspace)

### V2.0 (Escala)
- [ ] Sub-contas (agências gerenciando clientes)
- [ ] Marketplace de templates de fluxo
- [ ] API pública documentada (Swagger)
- [ ] Webhooks de saída configuráveis
- [ ] Integração nativa n8n / Zapier
- [ ] App mobile (React Native)

---

## 12. Deploy & Infraestrutura

### Ambiente de Produção (Hetzner + Coolify)

Notas de consola (SSH, Volumes, avisos de billing): `docs/HETZNER_CLOUD_SETUP.md`.

```
Servidor: Hetzner CX31 (2 vCPU, 8GB RAM, 80GB SSD) — ~€12/mês
CDN/WAF:  Cloudflare (free tier) — DNS + DDoS + SSL
Painel:   Coolify (self-hosted PaaS via Docker)
Proxy:    Caddy (gerenciado pelo Coolify, TLS automático)
```

### Caddyfile (produção)
```
api.seudominio.com.br {
    reverse_proxy localhost:8080 {
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
    }
    # WebSocket support
    @websocket {
        header Connection *Upgrade*
        header Upgrade websocket
    }
    handle @websocket {
        reverse_proxy localhost:8080
    }
}

app.seudominio.com.br {
    root * /var/www/frontend/dist
    file_server
    try_files {path} /index.html    # SPA fallback
}
```

### Dockerfile (backend — multi-stage)
```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o bin/api ./cmd/api

# Production stage
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /app/bin/api .
COPY --from=builder /app/migrations ./migrations
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["./api"]
```

### GitHub Actions (CI/CD)
```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - run: go test ./... -cover

  build-and-deploy:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build Docker image
        run: |
          docker build -t ghcr.io/${{ github.repository }}/api:${{ github.sha }} .
          docker push ghcr.io/${{ github.repository }}/api:${{ github.sha }}
      - name: Deploy via Coolify webhook
        run: |
          curl -X POST "${{ secrets.COOLIFY_WEBHOOK_URL }}" \
            -H "Authorization: Bearer ${{ secrets.COOLIFY_TOKEN }}"
```

### Backups
```bash
# Backup diário do PostgreSQL (cron no servidor)
# /etc/cron.d/postgres-backup
0 3 * * * postgres pg_dump $DATABASE_URL | gzip | \
  aws s3 cp - s3://backups/postgres/$(date +%Y%m%d).sql.gz \
  --endpoint-url $S3_ENDPOINT

# Retenção: 30 dias locais, 90 dias no R2
# Testar restore mensalmente
```

---

## 13. Checklist de Produção

### Antes do primeiro deploy

#### Segurança
- [ ] JWT usando RS256 (não HS256) com chaves geradas de forma segura
- [ ] Todas as variáveis de ambiente de produção definidas (nunca commitar `.env`)
- [ ] Rate limiting configurado e testado
- [ ] CORS restrito apenas ao domínio do frontend
- [ ] Headers de segurança ativos (Helmet)
- [ ] HTTPS forçado, HSTS habilitado
- [ ] Webhook HMAC validation implementado e testado
- [ ] Audit log funcionando para eventos sensíveis
- [ ] Secrets do GitHub Actions configurados (nunca no código)

#### Performance
- [ ] Índices do banco criados e verificados com EXPLAIN ANALYZE
- [ ] Connection pool configurado adequadamente
- [ ] Redis cache para dados frequentes (ex: conversas recentes)
- [ ] Imagens/mídia servidas pelo R2/CDN, não pelo backend Go
- [ ] Build do frontend com `npm run build` (não dev server)
- [ ] Gzip habilitado no Caddy

#### Observabilidade
- [ ] Sentry configurado (frontend + backend)
- [ ] Logs estruturados com Zap em formato JSON
- [ ] Health check endpoint: `GET /health` → `{"status": "ok"}`
- [ ] Métricas Prometheus expostas em `/metrics` (restrito por IP)
- [ ] Alertas básicos configurados (uso de disco, CPU, erros 5xx)

#### Operacional
- [ ] Backups automáticos do PostgreSQL configurados e testados
- [ ] Estratégia de rollback definida (Coolify permite rollback de imagem)
- [ ] Runbook de incidentes escrito
- [ ] Domínio apontado para Cloudflare
- [ ] Registros MX configurados para emails transacionais (SPF, DKIM, DMARC)
- [ ] Limite de planos implementado e testado (free tier não ultrapassa limites)

#### Qualidade
- [ ] Testes passando (go test ./... verde)
- [ ] Sem `console.log` de debug no frontend
- [ ] Sem credenciais hardcoded em nenhum arquivo
- [ ] `npm audit` sem vulnerabilidades críticas
- [ ] TypeScript sem erros (`tsc --noEmit`)
- [ ] Migrations testadas com migrate up e migrate down

---

## 📚 Referências e Recursos

- [Go Fiber Docs](https://docs.gofiber.io)
- [GORM Docs](https://gorm.io/docs)
- [TanStack Query](https://tanstack.com/query/v5)
- [shadcn/ui](https://ui.shadcn.com)
- [Evolution API](https://doc.evolution-api.com)
- [Anthropic API](https://docs.anthropic.com)
- [pgvector](https://github.com/pgvector/pgvector)
- [Asynq](https://github.com/hibiken/asynq)
- [OWASP Top 10](https://owasp.org/Top10/)
- [golang-migrate](https://github.com/golang-migrate/migrate)

---

> **Última atualização:** Abril 2026  
> *Mantenha este documento atualizado conforme decisões arquiteturais forem tomadas.*
