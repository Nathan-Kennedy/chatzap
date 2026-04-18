#!/usr/bin/env node
/**
 * Chama Docker Compose: tenta `docker compose` (plugin v2) e depois `docker-compose` (v1).
 * Uso (na raiz do repo): node scripts/docker-compose-exec.js <compose relativo> [args...]
 * Ex.: node scripts/docker-compose-exec.js infra/docker-compose.dev.yml up -d
 */
const { spawnSync } = require('child_process')
const path = require('path')
const fs = require('fs')

const repoRoot = path.join(__dirname, '..')
const argv = process.argv.slice(2)
if (argv.length < 2) {
  console.error('Uso: node scripts/docker-compose-exec.js <ficheiro-compose.yml> <args...>')
  process.exit(1)
}
const composeRel = argv[0]
const rest = argv.slice(1)
const composeFile = path.join(repoRoot, composeRel)
if (!fs.existsSync(composeFile)) {
  console.error(`Ficheiro não encontrado: ${composeFile}`)
  process.exit(1)
}

function run(cmd, args) {
  const r = spawnSync(cmd, args, { stdio: 'inherit', shell: false, cwd: repoRoot })
  return r.status === null ? 1 : r.status
}

// Teste rápido: plugin v2 disponível?
const tryCompose = spawnSync('docker', ['compose', 'version'], { stdio: 'ignore', shell: false })
if (tryCompose.status === 0) {
  process.exit(run('docker', ['compose', '-f', composeFile, ...rest]))
}

const tryLegacy = spawnSync('docker-compose', ['version'], { stdio: 'ignore', shell: false })
if (tryLegacy.status === 0) {
  process.exit(run('docker-compose', ['-f', composeFile, ...rest]))
}

console.error(`
Docker Compose não encontrado.
- Ativa o plugin no Docker Desktop (Compose V2), ou
- Instala: https://github.com/docker/compose/releases
`)
process.exit(127)
