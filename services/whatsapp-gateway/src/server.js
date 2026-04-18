/**
 * HTTP: health / readiness para orquestração (Docker, K8s).
 * O pareamento continua a ser `npm run pair` ou `node src/cli-pair.js`.
 */
import 'dotenv/config'
import Fastify from 'fastify'
import Redis from 'ioredis'
import { config } from './config.js'
import { getPool } from './db.js'

const pool = getPool(config.databaseUrl)
let redisClient = null
if (config.redisUrl) {
  redisClient = new Redis(config.redisUrl, { maxRetriesPerRequest: 2, lazyConnect: true })
}

const app = Fastify({ logger: { level: config.logLevel } })

app.get('/health', async () => ({
  status: 'ok',
  service: 'whatsapp-gateway',
}))

app.get('/ready', async () => {
  const checks = {}
  if (pool) {
    try {
      await pool.query('SELECT 1')
      checks.postgres = 'ok'
    } catch (e) {
      checks.postgres = { error: e.message }
    }
  } else {
    checks.postgres = 'skipped'
  }
  if (redisClient) {
    try {
      await redisClient.connect()
      await redisClient.ping()
      checks.redis = 'ok'
    } catch (e) {
      checks.redis = { error: e.message }
    }
  } else {
    checks.redis = 'skipped'
  }
  const pgOk = !pool || checks.postgres === 'ok'
  const redisOk = !redisClient || checks.redis === 'ok'
  return { ready: pgOk && redisOk, checks }
})

const port = config.httpPort
await app.listen({ port, host: '0.0.0.0' })
console.log(`whatsapp-gateway HTTP em http://0.0.0.0:${port} (GET /health, /ready)`)
