import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

function getEnv(name, fallback = '') {
  const v = process.env[name]
  return v === undefined || v === '' ? fallback : v
}

export const config = {
  instanceName: getEnv('WA_INSTANCE_NAME', 'default'),
  authDir: path.resolve(getEnv('WA_AUTH_DIR', path.join(__dirname, '..', 'data', 'auth'))),
  databaseUrl: getEnv('DATABASE_URL', ''),
  redisUrl: getEnv('REDIS_URL', ''),
  httpPort: Number(getEnv('WA_GATEWAY_HTTP_PORT', '3090')),
  logLevel: getEnv('LOG_LEVEL', 'warn'),
  /** Canal Redis para a plataforma subscrever (QR, estado). */
  redisChannelPrefix: getEnv('WA_REDIS_CHANNEL_PREFIX', 'wa:gw'),
}

export function authPathForInstance(name) {
  return path.join(config.authDir, name.replace(/[^a-zA-Z0-9_-]/g, '_'))
}
