/**
 * Pareamento WhatsApp: QR no terminal (Baileys).
 * Sessão em ./.auth — tratar como credencial (não commitar).
 *
 * 515 (restartRequired) após o QR é normal — reconectamos automaticamente.
 * 401 (loggedOut) com .auth antigo = sessão inválida — corre `npm run clean` e de novo `npm run pair`.
 */
import makeWASocket, {
  Browsers,
  DisconnectReason,
  fetchLatestBaileysVersion,
  useMultiFileAuthState,
} from '@whiskeysockets/baileys'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import pino from 'pino'
import qrcode from 'qrcode-terminal'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const AUTH_DIR = path.join(__dirname, '.auth')

const logger = pino({
  level: process.env.LOG_LEVEL ?? 'fatal',
})

/** Códigos em que não adianta reconectar sem limpar sessão ou mudar condições. */
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
    return 'A pasta .auth tem credenciais recusadas pelo WhatsApp. Corre: npm run clean   e depois: npm run pair'
  }
  if (code === DisconnectReason.connectionReplaced) {
    return 'Esta sessão foi substituída por outro aparelho/browser. Fecha outras ligações WhatsApp Web ou corre npm run clean e volta a parear.'
  }
  if (code === DisconnectReason.forbidden) {
    return 'Acesso recusado (403). Verifica conta/rede ou tenta mais tarde.'
  }
  return ''
}

function printQr(qr) {
  console.log('\n--- Escaneia no telemóvel: WhatsApp → Definições → Aparelhos ligados → Ligar um aparelho ---\n')
  qrcode.generate(qr, { small: true })
  console.log('')
}

async function connect() {
  const { state, saveCreds } = await useMultiFileAuthState(AUTH_DIR)
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
    if (qr) printQr(qr)

    if (connection === 'open') {
      console.log('✓ Ligação estabelecida. Sessão em:', AUTH_DIR)
      console.log('  Ctrl+C para sair (a sessão mantém-se para a próxima execução).')
    }

    if (connection === 'close') {
      const statusCode = lastDisconnect?.error?.output?.statusCode
      const reason = lastDisconnect?.error?.message ?? lastDisconnect?.error
      console.warn('Conexão fechada:', reason, 'código:', statusCode)

      const hint = hintForCode(statusCode)
      if (hint) console.error('\n' + hint + '\n')

      if (isFatalWithoutUserAction(statusCode)) {
        process.exit(1)
      }

      console.log('Reconexão automática em 2s (pedido do servidor / ligação instável)…')
      await new Promise((r) => setTimeout(r, 2000))
      await connect()
    }
  })
}

const hasAuth = fs.existsSync(AUTH_DIR)
console.log('A iniciar. Pasta de sessão:', AUTH_DIR)
if (hasAuth) {
  console.log('Existe sessão anterior: se vires 401 sem QR, corre primeiro: npm run clean\n')
} else {
  console.log('Primeira vez: aparece o QR em baixo.\n')
}

await connect()
