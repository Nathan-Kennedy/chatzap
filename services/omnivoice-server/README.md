# omnivoice-server (TTS local, compat. OpenAI)

Servidor HTTP à parte do ChatBot: expõe `POST /v1/audio/speech` para a API Go usar em **Agentes → OmniVoice**.

- Pacote: [omnivoice-server no PyPI](https://pypi.org/project/omnivoice-server/) / [GitHub](https://github.com/maemreyo/omnivoice-server)
- Requer **Python 3.10+**. Este diretório usa um **venv** em `venv/` (não commitado).

## Instalação (Windows, NVIDIA / CUDA)

1. Drivers NVIDIA atualizados; `nvidia-smi` deve funcionar.
2. Na raiz do repo:

```powershell
cd services\omnivoice-server
python -m venv venv
.\venv\Scripts\Activate.ps1
python -m pip install --upgrade pip
# PyTorch CUDA: o índice cu128 não inclui torchcodec — instalar em dois passos.
pip install torch==2.8.0+cu128 torchaudio==2.8.0+cu128 --index-url https://download.pytorch.org/whl/cu128
pip install torchcodec==0.11 omnivoice-server
```

Se a instalação CUDA falhar, usa CPU (mais lento):

```powershell
pip install torchcodec==0.11 torch==2.8.0 torchaudio==2.8.0 --index-url https://download.pytorch.org/whl/cpu
pip install omnivoice-server
```

## Arranque (porta 8000 — alinhada ao agente / `OMNIVOICE_DEFAULT_BASE_URL`)

```powershell
omnivoice-server --host 127.0.0.1 --port 8000 --device cpu
```

**CUDA** só se tiveres VRAM e memória virtual suficientes; se aparecer **`os error 1455`** (ficheiro de paginação pequeno) ou OOM, usa **`--device cpu`** (mais lento) ou aumenta o [pagefile](https://support.microsoft.com/windows) do Windows. Opcional: **ffmpeg** no PATH para silenciar avisos do pydub.

### Script npm (`npm run omnivoice:server` / `npm run tts:local`)

Na raiz do repo, [`scripts/run-omnivoice-server.ps1`](../../scripts/run-omnivoice-server.ps1) usa **`--device cpu` por defeito** (evita 1455 no Windows). Para **GPU**:

- `npm run omnivoice:server -- --device cuda`, ou
- no PowerShell antes do npm: `$env:OMNIVOICE_DEVICE = "cuda"`

`--profile-dir` aponta para [`voice-profiles`](voice-profiles/README.md).

### Clonar uma voz (áudio de referência)

Cria uma pasta `voice-profiles/<id>/` com `ref_audio.wav` + `meta.json` — guia passo a passo em [`voice-profiles/README.md`](voice-profiles/README.md). Na API usa `"voice": "clone:<id>"`.

A **primeira** execução descarrega modelos do Hugging Face (demorado).

## Patch local (`run_patched.py`)

O CLI oficial chama `torch.isnan` nos outputs de um teste de geração, mas o modelo **OmniVoice** devolve **`numpy.ndarray`**, não `Tensor` — o arranque falhava com `isnan(): ... must be Tensor, not numpy.ndarray`. O script [`run_patched.py`](run_patched.py) corrige isso antes de importar o servidor; **`npm run omnivoice:server`** e [`scripts/run-omnivoice-server.ps1`](../../scripts/run-omnivoice-server.ps1) usam esse script em vez do `.exe` direto.

O mesmo pacote assume tensores PyTorch em `tensor_to_wav_bytes` / `tensors_to_wav_bytes` (`.cpu()`); na geração real os outputs podem ser **numpy**, o que dava **500** com `AttributeError: 'numpy.ndarray' object has no attribute 'cpu'`. O patch converte para `torch.Tensor` antes de delegar ao código original.

## Resolução de problemas

| Sintoma | O que fazer |
|--------|-------------|
| `os error 1455` / paging ao carregar em CUDA | Aumenta o [ficheiro de paginação](https://support.microsoft.com/windows) do Windows ou fecha outras apps; o modelo é grande. |
| `Couldn't find ffmpeg` (pydub) | Opcional: instala [ffmpeg](https://ffmpeg.org/) e adiciona ao PATH. |

## Teste rápido

O servidor aceita `response_format` só **`wav`** ou **`pcm`** (não `mp3`). No **PowerShell**, JSON inline em `--data-raw` pode falhar com `json_invalid` (encoding/aspas na linha de comandos). Usa corpo em ficheiro ou o script na raiz do repo:

```powershell
# Na raiz do repo (UTF-8 real no corpo — evita 422 json_invalid)
curl.exe -s -o $env:TEMP\omni-test.wav -w "%{http_code}" -X POST "http://127.0.0.1:8000/v1/audio/speech" -H "Content-Type: application/json; charset=utf-8" --data-binary "@scripts\omnivoice-smoke-body.json"
```

Ou: `powershell -ExecutionPolicy Bypass -File scripts\test-omnivoice-speech.ps1`

Esperado: código HTTP `200` e ficheiro WAV não vazio (no script, passa `-OutPath` para gravar). A API Go do ChatBot omite `response_format` e o servidor usa WAV por defeito.

Na **auto-resposta WhatsApp**, respostas longas são divididas em **vários** áudios em sequência (como bolhas de texto). A API Go envia `speed`, `num_step` e voz `design:…` com tokens **só da lista** que o modelo aceita (erro 500 se usares frases livres).

Para soar mais próximo de **ElevenLabs**, o OmniVoice suporta **`clone:nome_do_perfil`** com áudio de referência no servidor (perfis em disco — ver documentação upstream). O modo `design:` só combina atributos fixos (ex. `female`, `portuguese accent`, `low pitch`). Ajuste fino sem recompilar: no `backend/.env` define `OMNIVOICE_DESIGN_INSTRUCT` e opcionalmente `OMNIVOICE_TTS_SPEED` / `OMNIVOICE_TTS_NUM_STEP`.
