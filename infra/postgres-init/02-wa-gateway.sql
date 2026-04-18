-- WhatsApp Gateway (Baileys): metadados e auditoria (credenciais ficam em WA_AUTH_DIR / volume).
SET search_path TO public;

CREATE TABLE IF NOT EXISTS wa_instances (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'disconnected',
  phone_jid TEXT,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wa_connection_events (
  id BIGSERIAL PRIMARY KEY,
  instance_id UUID NOT NULL REFERENCES wa_instances (id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  payload JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_wa_events_instance_created
  ON wa_connection_events (instance_id, created_at DESC);
