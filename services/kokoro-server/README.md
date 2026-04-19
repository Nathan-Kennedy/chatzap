# Kokoro TTS (servidor local compatível OpenAI)

Este repositório **não** inclui o motor Kokoro. Para a API Go falar com Kokoro, corre um servidor com endpoint estilo OpenAI, por exemplo [remsky/Kokoro-FastAPI](https://github.com/remsky/Kokoro-FastAPI) (`POST /v1/audio/speech`).

## Arranque no PC **sem Docker** (recomendado se só usas Docker na Hetzner)

Na raiz do monorepo (Windows, com **[uv](https://docs.astral.sh/uv/)** e **Git**). Se `uv` não estiver no PATH, o script tenta `%USERPROFILE%\.local\bin\uv.exe` (instalação padrão do instalador oficial).

Instalar **uv** (PowerShell):

```powershell
powershell -ExecutionPolicy Bypass -c "irm https://astral.sh/uv/install.ps1 | iex"
```

Depois **fecha e reabre o terminal**, ou define `UV_EXE` ao caminho de `uv.exe`.

```powershell
npm run kokoro:server:native
```

- Clona (uma vez) o upstream **[remsky/Kokoro-FastAPI](https://github.com/remsky/Kokoro-FastAPI)** na pasta **irmã** do ChatBot (ex.: `F:\Projetos Dev\Kokoro-FastAPI`). Para outro sítio: `KOKORO_FASTAPI_DIR=D:\src\Kokoro-FastAPI`.
- Corre **`uv sync --extra cpu`**, descarrega o modelo e **`uvicorn`** na porta **8880** (compatível com **uv 0.11+**; o `start-cpu.ps1` upstream pode falhar sem `.venv` pré-criado).
- Primeira execução: PyTorch + Kokoro + modelo — demora e ocupa disco.
- Se uma tentativa anterior falhou a meio: apaga **`Kokoro-FastAPI/.venv`** ou corre com **`KOKORO_RESET_VENV=1`** na mesma sessão antes de `npm run kokoro:server:native`.

URL: `http://127.0.0.1:8880` — alinha `KOKORO_DEFAULT_BASE_URL` no `backend/.env`.

## Arranque rápido (Docker, CPU)

Na raiz do monorepo:

```bash
npm run kokoro:server
```

Isto expõe **8880** no host (`http://127.0.0.1:8880`), alinhado a `KOKORO_DEFAULT_BASE_URL` em `backend/.env`. Requer **Docker Desktop**.

### Variante GPU

Define `KOKORO_DOCKER_IMAGE=ghcr.io/remsky/kokoro-fastapi-gpu:latest` (ou outra tag suportada pelo projeto) antes de correr o script, ou ajusta o comando em `scripts/run-kokoro-server.ps1`.

## Integração com a API

1. `backend/.env`: `KOKORO_DEFAULT_BASE_URL=http://127.0.0.1:8880` (ou URL por agente na UI **Agentes**).
2. Na UI, provedor **Kokoro** e voz (ex. `pf_dora` para PT-BR feminina). Lista de vozes: [VOICES.md do Kokoro-82M](https://huggingface.co/hexgrad/Kokoro-82M/blob/main/VOICES.md).

## Saúde

Com o contentor a correr, podes testar `GET http://127.0.0.1:8880/health` (se o upstream expuser esse caminho).
