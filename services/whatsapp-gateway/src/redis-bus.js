/**
 * Pub/sub para a app web / workers consumirem QR e estado em tempo real.
 */
import Redis from 'ioredis'

let client

export function getRedis(url) {
  if (!url) return null
  if (!client) {
    client = new Redis(url, { maxRetriesPerRequest: 2, lazyConnect: true })
  }
  return client
}

export async function publishJson(redis, channel, obj) {
  if (!redis) return
  await redis.publish(channel, JSON.stringify({ ...obj, ts: Date.now() }))
}

export async function closeRedis() {
  if (client) {
    await client.quit()
    client = null
  }
}
