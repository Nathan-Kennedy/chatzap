# Kokoro TTS (servidor local compatível OpenAI)

Este repositório **não** inclui o motor Kokoro. Para a API Go falar com Kokoro, corre um servidor com endpoint estilo OpenAI, por exemplo [remsky/Kokoro-FastAPI](https://github.com/remsky/Kokoro-FastAPI) (`POST /v1/audio/speech`).

## Arranque rápido (Docker, CPU)

Na raiz do monorepo:

```bash
npm run kokoro:server
```

Isto expõe **8880** no host (`http://127.0.0.1:8880`), alinhado a `KOKORO_DEFAULT_BASE_URL` em `backend/.env`.

### Variante GPU

Define `KOKORO_DOCKER_IMAGE=ghcr.io/remsky/kokoro-fastapi-gpu:latest` (ou outra tag suportada pelo projeto) antes de correr o script, ou ajusta o comando em `scripts/run-kokoro-server.ps1`.

## Integração com a API

1. `backend/.env`: `KOKORO_DEFAULT_BASE_URL=http://127.0.0.1:8880` (ou URL por agente na UI **Agentes**).
2. Na UI, provedor **Kokoro** e voz (ex. `pf_dora` para PT-BR feminina). Lista de vozes: [VOICES.md do Kokoro-82M](https://huggingface.co/hexgrad/Kokoro-82M/blob/main/VOICES.md).

## Saúde

Com o contentor a correr, podes testar `GET http://127.0.0.1:8880/health` (se o upstream expuser esse caminho).
