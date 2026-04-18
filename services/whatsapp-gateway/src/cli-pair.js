import 'dotenv/config'
import { runPairingLoop, shutdown } from './pairing.js'

async function main() {
  const name = process.argv[2] || process.env.WA_INSTANCE_NAME || 'default'
  await runPairingLoop({ instanceName: name })
}

process.on('SIGINT', async () => {
  await shutdown().catch(() => {})
  process.exit(0)
})
process.on('SIGTERM', async () => {
  await shutdown().catch(() => {})
  process.exit(0)
})

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
