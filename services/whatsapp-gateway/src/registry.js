/**
 * Registo de instâncias e eventos (Postgres).
 */
export async function upsertInstance(pool, name, status = 'pairing') {
  const r = await pool.query(
    `INSERT INTO wa_instances (name, status, updated_at)
     VALUES ($1, $2, now())
     ON CONFLICT (name) DO UPDATE SET
       status = EXCLUDED.status,
       updated_at = now(),
       last_error = NULL
     RETURNING id, name, status`,
    [name, status],
  )
  return r.rows[0]
}

export async function markConnected(pool, id, phoneJid) {
  await pool.query(
    `UPDATE wa_instances SET status = 'connected', phone_jid = $2, last_error = NULL, updated_at = now()
     WHERE id = $1`,
    [id, phoneJid ?? null],
  )
}

export async function markDisconnected(pool, id, status, lastError) {
  await pool.query(
    `UPDATE wa_instances SET status = $2, last_error = $3, updated_at = now() WHERE id = $1`,
    [id, status, lastError ?? null],
  )
}

export async function insertEvent(pool, instanceId, eventType, payload) {
  await pool.query(
    `INSERT INTO wa_connection_events (instance_id, event_type, payload)
     VALUES ($1, $2, $3::jsonb)`,
    [instanceId, eventType, JSON.stringify(payload ?? {})],
  )
}
