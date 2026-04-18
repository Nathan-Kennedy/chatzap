/**
 * Remove a sessão local (.auth). Usa antes de um novo QR se tiveres 401/erro de sessão.
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const dir = path.join(path.dirname(fileURLToPath(import.meta.url)), '.auth')
fs.rmSync(dir, { recursive: true, force: true })
console.log('Sessão removida:', dir)
console.log('Corre agora: npm run pair')
