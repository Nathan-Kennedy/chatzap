# Perfis de voz (clone)

Coloca aqui **um perfil por pasta**: o servidor OmniVoice lê este diretório (`--profile-dir`) e expõe cada perfil como `clone:<id_da_pasta>` no `POST /v1/audio/speech`.

## Estrutura (obrigatória)

```
voice-profiles/
  <id_do_perfil>/          # só letras, números, hífen e underscore (ex.: atendimento_br)
    ref_audio.wav          # áudio de referência (WAV)
    meta.json              # metadados (ver exemplo abaixo)
```

- **`ref_audio.wav`**: **obrigatório este nome e extensão** — o servidor não lê `.mp3` nem outros nomes. Converte o teu MP3 para WAV e grava como `ref_audio.wav`. **Usa só 3–10 s de áudio limpo** (o motor avisa se passar de ~20 s: clone pior, mais RAM e mais lento). Para cortar: `..\venv\Scripts\python.exe ..\scripts\trim_ref_audio.py entrada.wav saida.wav 8` (na pasta `omnivoice-server`).
- **`meta.json`**: JSON com pelo menos o formato esperado pelo pacote:

```json
{
  "name": "atendimento_br",
  "ref_text": null,
  "created_at": "2026-01-01T12:00:00+00:00"
}
```

O campo **`ref_text` não é** a frase que o bot vai dizer no WhatsApp (isso vem do `input` do TTS / texto do LLM). Aqui vai **só a transcrição do que está no `ref_audio.wav`**. Notas ou instruções (“palavra por palavra…”) **não** — desalinham o clone. Enquanto não transcreveres, usa `null`.

Podes copiar [`meta.json.example`](meta.json.example) para `meu_perfil/meta.json` e editar.

## Uso na API

Com o perfil `atendimento_br`, o corpo do TTS fica:

```json
{
  "model": "tts-1",
  "input": "Olá, em que posso ajudar?",
  "voice": "clone:atendimento_br"
}
```

Lista de perfis: `GET http://127.0.0.1:8000/v1/voices/profiles` (quando o servidor está a correr).

## ChatBot (API Go)

Para a auto-resposta WhatsApp usar este clone, é preciso que o agente envie `voice: clone:<id>` (extensão futura na UI Agentes) ou variável de ambiente dedicada — hoje o fluxo mapeia `nova`/`shimmer` para modo `design`. O servidor OmniVoice já suporta `clone:` quando o perfil existe nesta pasta.

## Direitos

Usa apenas áudio para o qual tens **direito de uso** (gravação própria, contrato com locutor, licença explícita).
