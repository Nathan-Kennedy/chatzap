import pg from 'pg'

let pool

export function getPool(databaseUrl) {
  if (!databaseUrl || String(databaseUrl).trim() === '') return null
  if (!pool) {
    pool = new pg.Pool({ connectionString: databaseUrl, max: 5 })
  }
  return pool
}

export async function closePool() {
  if (pool) {
    await pool.end()
    pool = null
  }
}
