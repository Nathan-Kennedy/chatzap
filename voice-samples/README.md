# Amostras de voz

Frase canónica em `PHRASE_PT.txt` (PT-BR, atendimento curto). Para **Kokoro** com texto mais longo e natural (construção / orçamento), usa `PHRASE_PT_CONSTRUCAO_ORCAMENTO.txt` — copia o conteúdo para a pré-visualização do agente ou para um pedido manual ao servidor TTS.

## OpenAI TTS + Gemini TTS (cloud, pagas)

Na raiz, com `OPENAI_API_KEY` e `GEMINI_API_KEY` no ambiente (ou em `backend/.env`, carregado pelo script):

```bash
npm run voice-samples:paid
```

Saída: `openai-tts/openai_<voz>_phrase_pt.mp3` e `gemini-tts/gemini_<voz>_phrase_pt.wav`. Opções: `--skip-openai`, `--skip-gemini`, `--openai-model`, `--gemini-model`.

Cenário longo (corretora, tags [PAUSA]/[HESITA]/[GAGUEJA]/[CORRIGE], só vozes femininas Kore/Aoede/Leda/Zephyr): texto em `gemini-tts/PHRASE_VENDA_CASA_FEM_TAGS.txt` — saída `gemini_*_venda_casa_tags.wav`.

```bash
npm run voice-samples:paid:gemini-female-venda
```

---

# Amostras open source (Kokoro / OmniVoice)

**Fish Speech** (PT-BR + tags emocionais): ficheiro `fish-speech/INPUT_CONSTRUCAO_PT_BR_TAGS.txt` e comando `npm run fish:tts:construcao-tags` (com servidor Fish em `:8080`); detalhes em `fish-speech/README_FISH_TAGS.md`.

## Gerar ficheiros

1. **Kokoro** (ex. Kokoro-FastAPI): `http://127.0.0.1:8880` (ou o teu `KOKORO_TTS_BASE_URL`).
2. **OmniVoice**: `npm run omnivoice:server` na raiz (ou `http://127.0.0.1:8000`).

Na raiz do repositório:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\generate-voice-samples.ps1
```

Parâmetros úteis: `-KokoroBaseUrl`, `-OmniBaseUrl`, `-SkipKokoro`, `-SkipOmni`.

## TTS extras (Fish Speech, CosyVoice, F5-TTS)

Instalação e limitações: `services/tts-extras/README.md`.

```powershell
npm run tts:extras:install
npm run tts:extras:samples
```

Ou só **F5** (o mais simples no Windows): `powershell -File scripts\install-tts-extras.ps1 -Components F5` e depois `npm run tts:extras:samples` (com `--SkipFish --SkipCosy` se não tiveres servidores).

**Fish** precisa de API em `:8080` (`npm run fish:server`) e checkpoints **s2-pro**. **CosyVoice** precisa de FastAPI em `:50000` (`npm run cosyvoice:server`) e modelo em `pretrained_models/` (`npm run cosyvoice:download-model`).

Saída extra: `f5-tts/f5_phrase_pt.wav`, `fish-speech/fish_phrase_pt.wav`, `cosyvoice/cosy_cross_lingual_phrase_pt.wav`.

## Saída

| Pasta | Conteúdo |
|--------|-----------|
| `openai-tts/` | MP3 por voz (`npm run voice-samples:paid`); instruções pedem PT-BR. |
| `gemini-tts/` | WAV por voz (`voice-samples:paid`); cenário venda-casa: `gemini_*_venda_casa_tags.wav` (`voice-samples:paid:gemini-female-venda`). |
| `kokoro/` | `pf_dora`, `pm_alex`, `pm_santa` (todas as vozes PT-BR do Kokoro-82M; presets na UI Agentes) em MP3. |
| `omnivoice/` | `auto` (modo automático) e `design_female_defaults` (mesmo instruct + `speed`/`num_step` que a auto-resposta Go usa com voz nova/shimmer). |

**Clone** (`clone:atendimento_br`): só é gerado se existir perfil em `services/omnivoice-server/voice-profiles/<id>/` com `ref_audio.wav` + `meta.json` (ver README dessa pasta). Sem perfil, o script ignora clones.

## Carga da máquina

- **Kokoro (CPU)** costuma ser mais leve que **OmniVoice** (modelo grande; primeira corrida descarrega pesos).
- Compara ouvindo os ficheiros gerados e o tempo de cada pedido no terminal.
