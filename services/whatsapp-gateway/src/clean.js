import 'dotenv/config'
import fs from 'node:fs'
import { authPathForInstance, config } from './config.js'

const name = process.argv[2] || process.env.WA_INSTANCE_NAME || config.instanceName
const dir = authPathForInstance(name)
fs.rmSync(dir, { recursive: true, force: true })
console.log('Sessão removida:', dir)
console.log('Corre: npm run pair')
