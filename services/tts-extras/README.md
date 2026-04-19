# TTS extras (Fish Speech, CosyVoice, F5-TTS)

Instalação **local** para comparar qualidade com Kokoro / OmniVoice. Nada disto é usado pela API Go em produção — é só para testes e amostras em `voice-samples/`.

## Requisitos gerais

| Motor | Notas |
|--------|--------|
| **F5-TTS** | O mais simples no **Windows**: venv + `f5-tts`; o instalador fixa **torch/torchaudio 2.8.x** e remove **torchcodec** (evita erro de DLL sem FFmpeg “full-shared”). Primeira inferência descarrega ~1.3 GB para o **cache HF** (ver secção abaixo). **ffmpeg** no PATH ajuda o pydub; sem ele ainda pode funcionar para WAV. |
| **Fish Speech (S2)** | Documentação oficial: **Linux / WSL**; GPU ~24 GB para S2-Pro. No Windows nativo usa **Docker** (recomendado) ou tenta `pip install -e ".[cpu]"` no clone (pode falhar por dependências). **Checkpoints:** `npm run fish:download-checkpoints` (vários GB) + `npm run fish:server`. Caminhos em disco: secção **Pesos fora do C:**. Ver [instalação Fish](https://speech.fish.audio/install/). |
| **CosyVoice** | `requirements.txt` + **submódulos**; muitos utilizadores usam **Linux / Docker**. Em Windows, `ttsfrd` tem wheels só para Linux — o projeto cai em **WeTextProcessing**. Modelos: ver `scripts/download-cosyvoice-model.ps1` (download grande). |

## Erro Windows **1455** (“ficheiro de paginação é muito pequeno”)

Aparece ao **carregar pesos** (F5/OmniVoice/etc.) quando a **memória virtual** (RAM + ficheiro de paginação) não chega.

1. **Aumenta o ficheiro de paginação** do Windows (tamanho gerido pelo sistema ou valor fixo maior). Guia Microsoft: [alterar o tamanho da memória virtual](https://support.microsoft.com/windows).
2. **Fecha** browser com muitos separadores, jogos, outros modelos em memória.
3. **F5 mais leve**: gera amostra com checkpoint menor:
   ```powershell
   powershell -File scripts\generate-voice-samples-extras.ps1 -SkipFish -SkipCosy -F5Model F5TTS_Base
   ```
4. **`-F5Device cuda`**: só funciona se o **PyTorch do venv tiver CUDA** (wheels `+cu128` etc.). O instalador fixa **torch CPU** no Windows para evitar TorchCodec; sem reinstalar torch com CUDA, vais ver `Torch not compiled with CUDA enabled`.

## Português brasileiro (PT-BR)

Nenhum destes três substitui o **Kokoro `pf_*`** como voz “nativa” PT-BR leve. Para PT-BR nos extras: **testa** texto em português com **CosyVoice** (ex. `cross_lingual` + prompt), **F5** (multilíngue; referência em inglês tende a sotaque misto) e **Fish** (lista `pt` como língua suportada). A qualidade depende do modo, prompt e clone.

## Uso comercial no teu site (resumo; **não** é aconselhamento jurídico)

| Motor | Uso comercial com os pesos “oficiais” típicos |
|--------|-----------------------------------------------|
| **Fish Speech (S2)** | A [Fish Audio Research License](https://github.com/fishaudio/fish-speech/blob/main/LICENSE) diz que **uso comercial exige licença escrita separada** da Fish Audio (`business@fish.audio`). **Não** contes com hospedagem paga / SaaS só com esta licença. |
| **F5-TTS** | Código **MIT**, mas os **checkpoints pré-treinados** são descritos pelo projeto como **CC-BY-NC** (componente **Non-Commercial** pelos dados Emilia). **Evita** usar esses pesos por defeito em produto comercial; alternativa comercial típica seria **treinar / licenciar** pesos próprios ou outro checkpoint com licença explícita. |
| **CosyVoice** | O cartão Hugging Face do **CosyVoice2-0.5B** indica **`license: apache-2.0`**, o que em geral é **compatível com uso comercial** nas condições Apache (atribuição, etc.). **Confirma sempre** o ficheiro `LICENSE` / cartão do **modelo exato** que vais usar (ex. Fun-CosyVoice3 pode ter texto próprio). Clones com voz de terceiros têm **direitos de imagem** à parte. |

Para um produto comercial com **menos risco de licença** entre estes três, o caminho mais alinhado a OSS “permissivo” costuma ser **CosyVoice (pesos Apache-2.0 conforme o cartão do modelo)** — ainda assim: **advogado + leitura do license do checkpoint + política de clone**.

## Pesos e cache fora do disco C: (`CHATBOT_TTS_CACHE_ROOT`)

Se o **C:** está cheio, define **`CHATBOT_TTS_CACHE_ROOT`** para uma pasta noutro disco (ex.: `F:\ChatBotTts`). Os scripts PowerShell (`install-tts-extras`, `download-*`, `run-*-server`) e `scripts/tts_extras_generate_samples.py` alinham o cache Hugging Face e os destinos grandes com o helper `scripts/_tts_cache_paths.ps1`.

**Sessão atual (PowerShell):**

```powershell
$env:CHATBOT_TTS_CACHE_ROOT = "F:\ChatBotTts"
```

**Persistente (utilizador Windows):** `setx CHATBOT_TTS_CACHE_ROOT "F:\ChatBotTts"` — fecha e reabre o terminal (ou reinicia a IDE) para aplicar.

Com `CHATBOT_TTS_CACHE_ROOT` definido, usam-se subpastas:

| Subpasta | Conteúdo |
|----------|----------|
| `{root}\hf-cache` | `HF_HOME` / `HF_HUB_CACHE` (F5, downloads HF, etc.) |
| `{root}\fish-speech-s2-pro` | Snapshot `fishaudio/s2-pro` (corre `npm run fish:download-checkpoints` **depois** de definires a variável) |
| `{root}\cosyvoice-pretrained_models\{Model}` | Modelos CosyVoice (`npm run cosyvoice:download-model`) |

O **código** (`vendor/fish-speech`, `vendor/CosyVoice`) continua no repositório; mudam sobretudo **cache HF** e **checkpoints / pretrained**.

**Sem** a variável, mantém-se o layout anterior: `services/tts-extras/hf-cache`, Fish em `vendor/fish-speech/checkpoints/s2-pro`, Cosy em `services/tts-extras/pretrained_models/`.

Se já tiveres pesos no C:, podes **mover** as pastas para `F:\...` e apontar `CHATBOT_TTS_CACHE_ROOT` para o pai correcto (ou copiar só `hf-cache`, `fish-speech-s2-pro` e `cosyvoice-pretrained_models` para dentro do mesmo `{root}` que os scripts esperam).

## Instalação (raiz do repo)

```powershell
powershell -ExecutionPolicy Bypass -File scripts\install-tts-extras.ps1
```

Só um componente:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\install-tts-extras.ps1 -Components F5
powershell -ExecutionPolicy Bypass -File scripts\install-tts-extras.ps1 -Components Fish
powershell -ExecutionPolicy Bypass -File scripts\install-tts-extras.ps1 -Components Cosy
```

Isto cria `venvs/` e `vendor/` (ignorados pelo Git) sob `services/tts-extras/`.

## Arranque de servidores (quando aplicável)

- **Fish** (API, porta **8080**): `npm run fish:server` — só depois de checkpoints + `pip install -e ".[cpu]"` ou extras CUDA no clone.
- **CosyVoice** (FastAPI, porta **50000**): `npm run cosyvoice:server` — depois do modelo existir na pasta indicada em **Pesos fora do C:** (ou `pretrained_models/` no repo se não usares a variável).

## Gerar WAV em `voice-samples/`

Com a frase em `voice-samples/PHRASE_PT.txt`:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\generate-voice-samples-extras.ps1
```

Ou: `npm run tts:extras:samples`

- **F5**: usa o venv `services/tts-extras/venvs/f5-tts` (sem servidor HTTP).
- **Fish**: espera `GET http://127.0.0.1:8080/v1/health` OK e chama `tools/api_client.py` do clone.
- **CosyVoice**: espera FastAPI em `50000` e modo **cross_lingual** (prompt WAV em `assets/`).

## Licenças

- **Fish Speech**: [FISH AUDIO RESEARCH LICENSE](https://github.com/fishaudio/fish-speech/blob/main/LICENSE) (não é MIT).
- **F5-TTS**: MIT (pesos CC-BY-NC por causa dos dados de treino).
- **CosyVoice**: Apache 2.0 (ver repositório).
