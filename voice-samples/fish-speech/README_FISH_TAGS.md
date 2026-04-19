# Fish Speech — texto com tags (PT-BR / construção)

O ficheiro `INPUT_CONSTRUCAO_PT_BR_TAGS.txt` mistura **português brasileiro** com **tags em inglês** entre colchetes, no estilo suportado pelo Fish Audio S2 (controlo livre de prosódia / tom).

## “Voz brasileira”

O modelo é **multilíngue**; o texto em PT-BR já puxa para sotaque brasileiro. Para **timbre** mais BR, o ideal é **áudio de referência** de um falante brasileiro (`--reference-audio` + `--reference-text` no `api_client`, ou `reference_id` no servidor). Sem referência, o resultado depende do checkpoint e da sorte do sampling.

## Gerar o WAV

1. Clone + venv Fish: `services/tts-extras/README.md`. **Checkpoints** (primeira vez, varios GB): na raiz do repo `npm run fish:download-checkpoints`.
2. **Arranca** o Fish noutro terminal: `npm run fish:server`. Espera `GET http://127.0.0.1:8080/v1/health` = `{"status":"ok"}`. Sem servidor aparece *connection refused*.
3. Na raiz do repo:

```powershell
& "services\tts-extras\venvs\f5-tts\Scripts\python.exe" scripts\fish_speech_post_tts.py `
  --url "http://127.0.0.1:8080/v1/tts" `
  --text-file "voice-samples\fish-speech\INPUT_CONSTRUCAO_PT_BR_TAGS.txt" `
  --out "voice-samples\fish-speech\construcao_pt_br_tags.wav"
```

**Licença:** uso comercial do Fish Speech exige acordo com a Fish Audio — ver `services/tts-extras/README.md`.
