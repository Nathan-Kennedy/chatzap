# Prompt — WhatsApp AI Automation SaaS Dashboard (v2)
> Cole este prompt completo no Google AI Studio (Antigravity).
> Inspirado em: Chatwoot, WhatsCRM, WhatsSaaS, WhatsMarkSaaS, Maven Labs.

---

## PROMPT

Build a complete, modern, production-ready **WhatsApp AI Automation SaaS Dashboard** in **React 18 + TypeScript + Vite**. The UI must look and feel like a polished, premium product — inspired by Chatwoot's inbox UX, WhatsCRM's flow builder, and modern SaaS tools like Linear and Notion. Dark theme by default with a clean, dense layout optimized for power users.

---

### TECH STACK (mandatory — do not deviate)

- **React 18 + TypeScript + Vite**
- **Tailwind CSS** (utility-first, no custom CSS files)
- **shadcn/ui** — use: Card, Button, Badge, Avatar, Table, Dialog, Sheet, Tabs, Input, Textarea, Select, Switch, Dropdown, Tooltip, Skeleton, Sonner (toasts), ScrollArea, Separator, Progress, Popover
- **React Router v6** (client-side SPA routing)
- **TanStack Query v5** (`useQuery` / `useMutation` for all server state)
- **Axios** — centralized in `/src/lib/api.ts` with base URL from `import.meta.env.VITE_API_BASE_URL`
- **React Hook Form + Zod** — all forms must be validated
- **Lucide React** — icons only (no other icon libraries)
- **Recharts** — all charts and analytics
- **@hello-pangea/dnd** — drag and drop for Kanban board
- **date-fns** — date formatting throughout the app
- JWT auth: store in `localStorage` as `"token"`, attach via Axios interceptor as `Authorization: Bearer <token>`

---

### DESIGN SYSTEM

```
Background:     #0A0A0F  (app bg)
Sidebar:        #0F0F17  (nav bg)
Card/Surface:   #15151F  (card bg)
Surface Hover:  #1C1C28  (hover states)
Border:         #ffffff0f (subtle borders)
Primary:        #7C3AED  (purple — brand accent)
Primary Hover:  #6D28D9
Cyan Accent:    #06B6D4  (online status, highlights)
Success:        #10B981
Warning:        #F59E0B
Danger:         #EF4444
Text Primary:   #F1F5F9
Text Secondary: #94A3B8
Text Muted:     #475569
```

- Font: Inter from Google Fonts (weights: 400, 500, 600, 700)
- Cards: `rounded-xl border border-white/[0.06] bg-[#15151F]`
- Buttons (primary): `bg-violet-600 hover:bg-violet-700 text-white`
- Sidebar nav items: icon + label, active state with `bg-violet-600/10 text-violet-400 border-l-2 border-violet-500`
- Conversation bubbles (incoming): `bg-[#1C1C28] text-slate-200 rounded-2xl rounded-tl-sm`
- Conversation bubbles (outgoing/bot): `bg-violet-600/20 text-violet-100 rounded-2xl rounded-tr-sm`
- Glassmorphism modals: `bg-[#15151F]/90 backdrop-blur-xl border border-white/10`
- Status badges: green dot (online/connected), yellow (pending/waiting), red (offline/error), gray (resolved/closed)
- All pages fully responsive — mobile sidebar becomes a bottom tab bar

---

### APP STRUCTURE

```
/src
  /components
    /layout       → AppShell, Sidebar, Topbar, MobileNav
    /ui           → shadcn components (auto-generated)
    /shared       → ConversationCard, MessageBubble, StatusBadge, EmptyState, LoadingSkeleton
  /pages          → one file per route
  /lib
    api.ts        → Axios instance + interceptors
    queryClient.ts → TanStack Query client config
  /types          → TypeScript interfaces for all API entities
  /hooks          → custom hooks (useAuth, useConversations, useRealtime)
  /utils          → formatDate, formatPhone, truncate helpers
```

---

### CENTRALIZED API CLIENT

`/src/lib/api.ts`:
```ts
import axios from 'axios'

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)
```

---

### PAGES & ROUTES

---

#### `/login` — Authentication
- Centered card layout with gradient background (subtle purple radial gradient)
- Product logo (WhatsApp-style icon in purple + product name)
- Email + password inputs with validation
- "Entrar" primary button — on submit: `POST /auth/login` → save token → redirect `/`
- "Esqueceu a senha?" link below button
- Loading spinner on submit button while pending

---

#### `/` → redirect to `/inbox`

---

#### `/inbox` — Omnichannel Inbox (main screen — most complex page)

**Layout:** 3-column split view (like Chatwoot)

**Column 1 — Conversation List (320px fixed)**
- Header: "Caixa de Entrada" + filter icon + compose button
- Tabs: `Minhas | Não atribuídas | Todas` (fetch counts from `/conversations/counts`)
- Search bar (debounced, 300ms) — `GET /conversations?search=`
- Filter dropdown: por canal (WhatsApp/Email/Instagram), por agente, por label, por status
- Scrollable conversation list — each card shows:
  - Contact avatar (initials circle, color based on name hash)
  - Contact name + phone (small, muted)
  - Channel icon (WhatsApp green icon, Instagram purple, etc.)
  - Last message preview (truncated 60 chars)
  - Unread count badge (violet)
  - Timestamp (relative: "2min", "1h", "ontem")
  - Assigned agent avatar (small, bottom right)
- Active conversation highlighted with left border + surface highlight
- Infinite scroll pagination (`GET /conversations?page=N`)

**Column 2 — Chat View (flex-1)**
- Header bar:
  - Contact avatar + name + phone number
  - Status badge (Aberto / Pendente / Resolvido)
  - Assigned agent dropdown (click to reassign)
  - Labels dropdown (add/remove conversation labels as colored badges)
  - Action buttons: Resolver (green), Reabrir (if resolved), Transferir, Snooze (clock icon)
  - More options menu: Bloquear contato, Exportar conversa, Ver no CRM
- Message timeline:
  - Date separators ("Hoje", "Ontem", "12 Jan")
  - Incoming messages: avatar + bubble (left-aligned)
  - Outgoing/bot messages: bubble (right-aligned, purple tint)
  - System events (italic, centered): "Conversa atribuída a João", "Conversa resolvida"
  - Private notes: yellow background, 🔒 lock icon, visible only to agents
  - Message status icons: sent ✓, delivered ✓✓, read ✓✓ (blue)
  - Attachment previews: image thumbnail, audio waveform bar, document card
  - AI-generated message indicator: small ✨ sparkle icon + "IA" label
- Reply box (bottom):
  - Tab toggle: "Responder | Nota Privada"
  - Rich textarea with placeholder "Digite uma mensagem..."
  - Toolbar: emoji picker, attach file, audio record button
  - Canned responses trigger: type `/` to open dropdown of saved replies (`GET /canned-responses?search=`)
  - Send button (violet) — `POST /conversations/:id/messages`
  - Character count for WhatsApp templates
- Fetch messages: `GET /conversations/:id/messages` (paginated, load more on scroll up)

**Column 3 — Contact Details Panel (280px)**
- Toggle visible/hidden with sidebar icon
- Contact info card: avatar, name, phone, email, created_at
- "Iniciar nova conversa" button
- Conversation attributes (custom fields): editable inline
- Conversation labels: add/remove tags
- Previous conversations list (last 5): clickable to switch
- Contact timeline (notes + activity log)
- "Ver perfil completo" link → `/contacts/:id`

---

#### `/contacts` — Contact Management

- Header: "Contatos" + search bar + "Importar CSV" button + "Novo Contato" button
- Filter bar: por tag, por agente responsável, por data de criação
- Data table with columns: Avatar+Nome, Telefone, Email, Tags (colored badges), Última interação (relative date), Agente, Ações
- Row actions: Ver conversa, Editar, Excluir
- Pagination (50 per page)
- **"Novo/Editar Contato" Sheet (right slide-in):**
  - Fields: Nome*, Telefone* (masked), Email, Empresa, Tags (multi-select creatable), Notas (textarea)
  - Zod validation
  - `POST /contacts` / `PATCH /contacts/:id`
- **Contact Detail Page `/contacts/:id`:**
  - Full profile with all fields
  - Conversation history timeline
  - Notes section (add/edit/delete notes)
  - Custom attributes section

---

#### `/kanban` — CRM Pipeline (Kanban Board)

- Header: "Pipeline CRM" + pipeline selector dropdown (multiple pipelines) + "+ Nova Coluna"
- Full-width horizontal drag-and-drop board using `@hello-pangea/dnd`
- Columns represent stages (e.g.: Novo Lead → Qualificado → Proposta → Fechado → Perdido)
- Each card shows:
  - Contact name + avatar
  - Phone number
  - Value/deal amount (if set) in BRL format
  - Assigned agent avatar
  - Labels/tags
  - Last activity date
  - WhatsApp icon button (quick open chat)
- Add card button at bottom of each column
- Drag card between columns → `PATCH /contacts/:id/stage`
- Column header: stage name + count badge + total value sum
- "+ Nova Coluna" → inline editable column at end
- Card click → opens Contact Detail Sheet (same as /contacts)
- Fetch: `GET /pipeline/:id/contacts` grouped by stage

---

#### `/campaigns` — Broadcasts & Campaigns

- Tabs: `Campanhas | Disparos Rápidos | Agendados`
- Campaign list table: Nome, Canal, Status (Rascunho/Enviando/Concluída/Falha), Enviadas, Entregues, Lidas, Data
- Status badge with progress bar for active campaigns
- "Nova Campanha" button → multi-step Dialog (Wizard):
  - Step 1: Nome + canal (WhatsApp instance selector)
  - Step 2: Selecionar contatos (table with checkboxes + filter por tag/agente) OR upload CSV
  - Step 3: Mensagem (textarea with variable placeholders: `{{nome}}`, `{{empresa}}`) OR select approved template
  - Step 4: Agendar (imediato ou data/hora específica)
  - Step 5: Revisão + Confirmar
- `POST /campaigns`
- Campaign detail page: delivery stats + recipient list + error log

---

#### `/agents` — AI Agents

- Grid (3 cols): agent cards showing name, model badge (GPT-4o / Claude / Groq / Gemini), status toggle, instances count, last active
- "Novo Agente" button → Sheet with form:
  - Nome*, Descrição
  - Modelo (Select): `gpt-4o`, `claude-sonnet-4-5`, `groq-llama-3.3-70b`, `gemini-2.0-flash`
  - System Prompt (large textarea with character count)
  - Temperature (slider 0–2, step 0.1)
  - Max Tokens (number input)
  - Toggles: Usar RAG (knowledge base), Usar memória da conversa, Handoff para humano quando não souber
  - Handoff keyword (text input, only if handoff enabled)
  - Connect to instances (multi-select checkboxes)
- CRUD: `POST /agents`, `GET /agents`, `PATCH /agents/:id`, `DELETE /agents/:id`
- Agent card: click to expand test chat panel (send test message, see raw response)

---

#### `/instances` — WhatsApp Instances

- Table: Nome, Número, Status (Conectado🟢 / QR Pendente🟡 / Desconectado🔴), Agente vinculado, Mensagens hoje, Última atividade, Ações
- "Nova Instância" button → Dialog:
  - Name input + submit → `POST /instances`
  - Show QR code returned (base64 `<img>`) in a centered modal with a phone mockup frame around it
  - Auto-polling every 3s: `GET /instances/:id/status` → when `status === "connected"`, show success state + confetti animation
  - QR expires in 60s — countdown timer shown
- Per-row actions: Desconectar, Reconectar, Ver logs, Excluir
- Disconnect: `DELETE /instances/:id/session`
- Stats mini-chart per instance (messages last 7 days — small sparkline)

---

#### `/flows` — Flow Builder

- Header: "Construtor de Fluxos" + "Novo Fluxo" button + search
- Flow list: cards with flow name, description, agent linked, status toggle, last edited, actions (edit, duplicate, delete)
- "Novo Fluxo" → opens full-screen canvas (separate route `/flows/:id/edit`):
  - **Visual node-based editor** using a simple custom canvas (SVG-based or div-based — NO external node editor library needed, build a minimal one):
    - Nodes are draggable boxes connected by bezier curves (SVG lines)
    - Node types (each visually distinct by color/icon):
      - 🟢 **Início** — trigger node (start of flow)
      - 💬 **Mensagem** — send text message (textarea)
      - ❓ **Pergunta** — ask user and wait for input (stores in variable)
      - 🔀 **Condição** — if/else branching based on variable or keyword
      - 🤖 **Agente IA** — hand off to AI agent
      - 👤 **Humano** — transfer to human agent
      - 🔗 **Webhook** — HTTP request (URL, method, body)
      - ⏱️ **Esperar** — delay node (seconds/minutes)
      - 🏷️ **Tag** — add/remove tag to contact
      - ✅ **Encerrar** — close conversation
    - Click node → right panel opens with node config form
    - Connect nodes by dragging from output port to input port
    - Toolbar: Salvar, Publicar, Undo/Redo, Zoom in/out, Auto-layout button
  - `GET /flows/:id` to load, `PATCH /flows/:id` to save, `POST /flows/:id/publish`

---

#### `/analytics` — Reports & Analytics

- Date range picker at top (preset: Hoje / 7 dias / 30 dias / 90 dias / Personalizado)
- KPI cards row (6 cards): Total Conversas, Mensagens Enviadas, Tempo Médio Resposta, Taxa Resolução, CSAT Score, Conversas com IA
- Charts section (2 columns):
  - Line chart: conversas abertas vs resolvidas por dia
  - Bar chart: volume de mensagens por hora do dia (heatmap style)
  - Bar chart: performance por agente (conversas + tempo médio)
  - Donut chart: distribuição por canal (WhatsApp, Email, Instagram)
- Table: Top 10 contatos por volume de mensagens
- Fetch: `GET /analytics/overview?from=&to=`

---

#### `/settings` — Settings

- Tabs: `Conta | Workspace | Planos | Integrações | API Keys | Notificações`

**Conta:** profile form (name, email, avatar upload via `POST /settings/avatar`), change password (current + new + confirm)

**Workspace:** workspace name, logo upload, timezone selector, default language, business hours (week schedule with on/off toggles + time range pickers)

**Integrações:** Cards for available integrations — Evolution API (connected status + config), OpenAI (API key input), Anthropic (API key), Groq (API key), n8n webhook URL. Each shows connected/disconnected badge + configure button.

**API Keys:** table of API keys with name, prefix (e.g. `sk-...abc`), created date, last used. Show/hide toggle, copy button, revoke button. "Gerar nova chave" → Dialog.

**Notificações:** toggle switches for: nova conversa, mensagem não lida, conversa atribuída, resolução, menção em nota privada. Email + in-app toggles per event.

---

### LAYOUT & NAVIGATION (AppShell)

**Sidebar (240px, collapsible to 64px icon-only):**
```
Logo (top)
─────────────
📥 Caixa de Entrada   /inbox      [unread badge]
👥 Contatos           /contacts
🗂️ Kanban             /kanban
📣 Campanhas          /campaigns
🤖 Agentes IA         /agents
📱 Instâncias         /instances
🔀 Fluxos             /flows
📊 Analytics          /analytics
─────────────
⚙️ Configurações      /settings
─────────────
[Avatar] Nome do usuário
[Plan badge: Pro/Starter/Enterprise]
```

**Top navbar:**
- Breadcrumb trail
- 🔔 Notification bell (badge with count) → dropdown with last 5 notifications
- 🌙 Dark/light toggle (store in localStorage)
- User avatar → dropdown: Meu Perfil, Configurações, Sair

**Mobile (< 768px):**
- Sidebar hidden by default, hamburger menu opens Sheet overlay
- Bottom tab bar: Inbox, Contatos, Kanban, Agentes, Mais

---

### GLOBAL UX REQUIREMENTS

- **Protected routes:** wrap all routes with `<ProtectedRoute>` — redirect to `/login` if no token
- **Loading skeletons:** every data-fetching page shows `<Skeleton>` while loading (match the shape of the real content)
- **Empty states:** illustrated empty state component (icon + title + subtitle + action button) for all empty lists
- **Optimistic updates:** mark conversation as read immediately on click; revert on error
- **Toast notifications:** Sonner toasts for all mutation success/error states
- **Confirmation dialogs:** before any DELETE or destructive PATCH action
- **Keyboard shortcuts:** `Ctrl+K` opens command palette (list of routes + recent conversations)
- **Relative timestamps:** use date-fns `formatDistanceToNow` everywhere
- **Phone formatting:** Brazilian format `(11) 99999-9999`
- All text in **Brazilian Portuguese**
- **404 page:** centered illustration + "Página não encontrada" + "Voltar ao início" button

---

### TypeScript Types (`/src/types/index.ts`)

Define interfaces for: `User`, `Workspace`, `Conversation`, `Message`, `Contact`, `Agent`, `Instance`, `Campaign`, `Flow`, `FlowNode`, `Label`, `CannedResponse`, `Notification`, `AnalyticsOverview`, `Pipeline`, `PipelineStage`

---

### ENV VARIABLES

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/ws
```

---

### README.md

Include:
- Project overview
- Setup instructions (`npm install`, `npm run dev`)
- All env variables documented
- Folder structure explanation
- How to connect to Go (Fiber) backend — mention that all API calls are in `/src/lib/api.ts` and all endpoints follow REST conventions

---

### FINAL NOTES FOR THE AI

- Generate all pages as separate files in `/src/pages/`
- Generate reusable components in `/src/components/`
- Keep each file under 300 lines — split large components into sub-components
- Use realistic mock/placeholder data using `useQuery` with `placeholderData` so the UI renders beautifully even without a real backend
- The code must be clean, modular, and immediately usable in a real Go (Fiber) + PostgreSQL + Redis backend project
- Do NOT use any PHP, Laravel, or server-side rendering — this is a pure SPA
- Do NOT install `react-flow` or `reactflow` — build the flow canvas manually
