/**
 * Núcleo Baileys: QR no terminal + hooks Postgres/Redis.
 */
import makeWASocket, {
  Browsers,
  DisconnectReason,
  fetchLatestBaileysVersion,
  useMultiFileAuthState,
} from '@whiskeysockets/baileys'
import fs from 'node:fs'
import pino from 'pino'
import qrcode from 'qrcode-terminal'
import { authPathForInstance, config } from './config.js'
import { closePool, getPool } from './db.js'
import {
  insertEvent,
  markConnected,
  markDisconnected,
  upsertInstance,
} from './registry.js'
import { getRedis, publishJson, closeRedis } from './redis-bus.js'

const logger = pino({ level: config.logLevel })

function isFatalWithoutUserAction(code) {
  if (code === undefined || code === null) return false
  return (
    code === DisconnectReason.loggedOut ||
    code === DisconnectReason.badSession ||
    code === DisconnectReason.forbidden ||
    code === DisconnectReason.connectionReplaced
  )
}

function hintForCode(code) {
  if (code === DisconnectReason.loggedOut || code === DisconnectReason.badSession) {
    return 'Credenciais recusadas. Corre: npm run clean'
  }
  if (code === DisconnectReason.connectionReplaced) {
    return 'Sessão substituída por outro aparelho. Fecha outras ligações ou npm run clean.'
  }
  if (code === DisconnectReason.forbidden) {
    return 'Acesso recusado (403). Verifica rede/conta.'
  }
  return ''
}

function printQr(qr) {
  console.log('\n--- WhatsApp → Definições → Aparelhos ligados → Ligar um aparelho ---\n')
  qrcode.generate(qr, { small: true })
  console.log('')
}

export async function runPairingLoop({ instanceName = config.instanceName } = {}) {
  const authDir = authPathForInstance(instanceName)
  const pool = getPool(config.databaseUrl)
  const redis = getRedis(config.redisUrl)

  if (redis) {
    await redis.connect().catch(() => {})
  }

  let instanceRow = null
  if (pool) {
    instanceRow = await upsertInstance(pool, instanceName, 'pairing')
    await insertEvent(pool, instanceRow.id, 'pairing_started', { name: instanceName })
  }

  const prefix = config.redisChannelPrefix

  async function emit(channelSuffix, payload) {
    try {
      const ch = `${prefix}:${instanceName}:${channelSuffix}`
      await publishJson(redis, ch, { instance: instanceName, ...payload })
      await publishJson(redis, `${prefix}:broadcast`, { instance: instanceName, ...payload })
    } catch (e) {
      logger.warn({ err: e }, 'redis publish falhou (ignorado)')
    }
  }

  async function connect() {
    const { state, saveCreds } = await useMultiFileAuthState(authDir)
    const versionInfo = await fetchLatestBaileysVersion().catch(() => null)
    const version = versionInfo?.version ?? [2, 3000, 1015901307]

    const sock = makeWASocket({
      version,
      logger,
      printQRInTerminal: false,
      browser: Browsers.macOS('Chrome'),
      auth: state,
      markOnlineOnConnect: false,
    })

    sock.ev.on('creds.update', saveCreds)

    sock.ev.on('connection.update', async (update) => {
      const { connection, lastDisconnect, qr } = update

      if (qr) {
        printQr(qr)
        if (pool && instanceRow) {
          await insertEvent(pool, instanceRow.id, 'qr', { length: qr.length })
        }
        await emit('qr', { qr })
      }

      if (connection === 'open') {
        const phone = sock.user?.id ? String(sock.user.id) : null
        console.log('✓ Ligação estabelecida. Sessão em:', authDir)
        console.log('  Ctrl+C para sair.\n')
        if (pool && instanceRow) {
          await markConnected(pool, instanceRow.id, phone)
          await insertEvent(pool, instanceRow.id, 'open', { phone_jid: phone })
        }
        await emit('connection', { state: 'open', phone_jid: phone })
      }

      if (connection === 'close') {
        const statusCode = lastDisconnect?.error?.output?.statusCode
        const reason = lastDisconnect?.error?.message ?? String(lastDisconnect?.error ?? '')
        console.warn('Conexão fechada:', reason, 'código:', statusCode)

        const hint = hintForCode(statusCode)
        if (hint) console.error('\n' + hint + '\n')

        if (pool && instanceRow) {
          await markDisconnected(pool, instanceRow.id, 'disconnected', reason.slice(0, 2000))
          await insertEvent(pool, instanceRow.id, 'close', { code: statusCode, reason })
        }
        await emit('connection', { state: 'close', code: statusCode, reason })

        if (isFatalWithoutUserAction(statusCode)) {
          process.exit(1)
        }

        console.log('Reconexão automática em 2s…')
        await new Promise((r) => setTimeout(r, 2000))
        await connect()
      }
    })
  }

  const hasAuth = fs.existsSync(authDir)
  console.log('Instância:', instanceName)
  console.log('Auth:', authDir)
  console.log('Postgres:', pool ? 'sim' : 'não (defina DATABASE_URL)')
  console.log('Redis:', redis ? 'sim' : 'não (defina REDIS_URL)')
  if (hasAuth) {
    console.log('Existe sessão anterior; 401 sem QR → npm run clean\n')
  } else {
    console.log('Primeira vez: QR em baixo.\n')
  }

  await connect()
}

export async function shutdown() {
  await closeRedis()
  await closePool()
}
