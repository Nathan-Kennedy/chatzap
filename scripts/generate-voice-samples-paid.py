#!/usr/bin/env python3
"""
Gera MP3 (OpenAI TTS) e WAV (Gemini TTS) em voice-samples/openai-tts e voice-samples/gemini-tts.
Texto: voice-samples/PHRASE_PT.txt

Chaves (não commitar):
  OPENAI_API_KEY — mesma variável do backend (.env)
  GEMINI_API_KEY — mesma que GEMINI_API_KEY no backend

Opcional: carrega backend/.env se existir (linhas KEY=val).

Cenário feminino venda-casa (tags): --gemini-female-venda-casa (lê gemini-tts/PHRASE_VENDA_CASA_FEM_TAGS.txt).
"""
from __future__ import annotations

import argparse
import base64
import json
import os
import re
import ssl
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import wave
from pathlib import Path

# Vozes alinhadas à lista em apps/web/src/pages/Agents.tsx (OPENAI_TTS_VOICES)
OPENAI_VOICES = [
    "coral",
    "nova",
    "shimmer",
    "sage",
    "marin",
    "cedar",
    "alloy",
    "ash",
    "ballad",
    "echo",
    "fable",
    "onyx",
    "verse",
]

# Vozes pré-definidas Gemini TTS (API Speech); nomes oficiais Google AI
GEMINI_VOICES = [
    "Kore",
    "Puck",
    "Charon",
    "Zephyr",
    "Aoede",
    "Fenrir",
    "Leda",
    "Orus",
]

# Subconjunto tipicamente feminino (Kore, Aoede, Leda, Zephyr) — ver documentação Google Speech / exemplos oficiais.
GEMINI_FEMALE_VOICES = ["Kore", "Aoede", "Leda", "Zephyr"]

GEMINI_VENDA_CASA_INSTRUCTION = """\
Segue estas etiquetas de interpretação (não digas a palavra "etiqueta" nem o nome das tags em voz alta; são só para ritmo e expressão):
- [PAUSA]: silêncio curto, como quem respira no microfone.
- [HESITA]: hesitação natural antes do próximo trecho.
- [GAGUEJA]: um único tropeço breve no sítio marcado (não inventes mais gaguejos noutros sítios).

Personagem: corretora brasileira num áudio de WhatsApp para um cliente, tom caloroso, leve nervosismo credível.
Lê o texto seguinte em português do Brasil:
"""

OPENAI_TTS_INSTRUCTIONS_PT = (
    "Fala em português brasileiro claro e natural, tom de atendimento ao cliente, sem sotaque europeu."
)

# Free tier: o Pro TTS costuma ter quota 0; fallback opcional por env.
GEMINI_TTS_MODELS_DEFAULT = ["gemini-2.5-flash-preview-tts"]


def repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


def load_dotenv_file(path: Path) -> None:
    if not path.is_file():
        return
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, _, v = line.partition("=")
        k, v = k.strip(), v.strip().strip('"').strip("'")
        if k and k not in os.environ:
            os.environ[k] = v


def read_phrase(repo: Path) -> str:
    p = repo / "voice-samples" / "PHRASE_PT.txt"
    t = p.read_text(encoding="utf-8").strip()
    if not t:
        raise SystemExit(f"Vazio: {p}")
    return t


def http_post_json(url: str, body: dict, headers: dict[str, str], timeout: int = 180) -> tuple[int, bytes]:
    data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method="POST")
    ctx = ssl.create_default_context()
    try:
        with urllib.request.urlopen(req, timeout=timeout, context=ctx) as resp:
            return resp.status, resp.read()
    except urllib.error.HTTPError as e:
        raw = e.read() if e.fp else b""
        return e.code, raw


def synth_openai_mp3(api_key: str, model: str, voice: str, text: str) -> bytes:
    body = {
        "model": model or "gpt-4o-mini-tts",
        "input": text,
        "voice": voice,
        "response_format": "mp3",
        "instructions": OPENAI_TTS_INSTRUCTIONS_PT,
    }
    status, raw = http_post_json(
        "https://api.openai.com/v1/audio/speech",
        body,
        {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {api_key}",
        },
        timeout=120,
    )
    if status != 200:
        raise RuntimeError(f"OpenAI TTS HTTP {status}: {raw[:800]!r}")
    return raw


def parse_rate_from_mime(mime: str) -> int:
    m = re.search(r"rate=(\d+)", mime, re.I)
    if m:
        return int(m.group(1))
    return 24000


def extract_gemini_audio(data: dict) -> tuple[bytes, str, int]:
    """Devolve (pcm_bytes, mime, sample_rate) ou levanta."""
    cands = data.get("candidates") or []
    for cand in cands:
        parts = (cand.get("content") or {}).get("parts") or []
        for part in parts:
            inline = part.get("inlineData") or part.get("inline_data")
            if not inline:
                continue
            b64 = inline.get("data")
            mime = inline.get("mimeType") or inline.get("mime_type") or "application/octet-stream"
            if not b64:
                continue
            raw = base64.b64decode(b64)
            sr = parse_rate_from_mime(mime)
            return raw, mime, sr
    err = data.get("error") or {}
    msg = err.get("message") if isinstance(err, dict) else str(data)[:500]
    raise RuntimeError(f"Resposta Gemini sem áudio: {msg}")


def write_wav_pcm16(path: Path, pcm: bytes, sample_rate: int) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with wave.open(str(path), "wb") as wf:
        wf.setnchannels(1)
        wf.setsampwidth(2)
        wf.setframerate(sample_rate)
        wf.writeframes(pcm)


def gemini_wav_path(repo: Path, voice_name: str) -> Path:
    return repo / "voice-samples" / "gemini-tts" / f"gemini_{voice_name.lower()}_phrase_pt.wav"


def parse_retry_seconds_429(raw: bytes) -> float:
    try:
        m = re.search(rb"retry in ([\d.]+)\s*s", raw, re.I)
        if m:
            return min(float(m.group(1)) + 1.0, 90.0)
    except (ValueError, IndexError):
        pass
    return 18.0


def synth_gemini_wav(
    api_key: str,
    phrase: str,
    voice_name: str,
    model: str,
    *,
    instruction_prefix: str | None = None,
    output_path: Path | None = None,
) -> Path:
    """Chama generateContent com AUDIO; grava WAV PCM no path devolvido."""
    if instruction_prefix is None:
        instruction_prefix = (
            "Lê em voz alta, em português brasileiro natural (tom de atendimento), sem traduzir o conteúdo: "
        )
    text = instruction_prefix.rstrip() + "\n\n" + phrase.strip()
    url = (
        "https://generativelanguage.googleapis.com/v1beta/models/"
        + urllib.parse.quote(model, safe="")
        + ":generateContent"
    )
    url = url + "?" + urllib.parse.urlencode({"key": api_key})
    body = {
        "contents": [{"role": "user", "parts": [{"text": text}]}],
        "generationConfig": {
            "responseModalities": ["AUDIO"],
            "speechConfig": {
                "voiceConfig": {
                    "prebuiltVoiceConfig": {"voiceName": voice_name},
                }
            },
        },
    }
    last_raw = b""
    for attempt in range(6):
        status, raw = http_post_json(url, body, {"Content-Type": "application/json"}, timeout=180)
        last_raw = raw
        if status == 200:
            break
        if status == 429 and attempt < 5:
            wait = parse_retry_seconds_429(raw)
            print(f"[Gemini] 429, a aguardar {wait:.1f}s antes de repetir ({voice_name})...")
            time.sleep(wait)
            continue
        raise RuntimeError(f"Gemini HTTP {status}: {raw[:1200]!r}")
    data = json.loads(last_raw.decode("utf-8"))
    pcm, mime, sr = extract_gemini_audio(data)
    out = output_path if output_path is not None else gemini_wav_path(repo_root(), voice_name)
    if "wav" in mime.lower() and pcm[:4] == b"RIFF":
        out.write_bytes(pcm)
        return out
    if "L16" in mime or "pcm" in mime.lower():
        write_wav_pcm16(out, pcm, sr)
        return out
    # último recurso: assumir PCM16 mono @ sr
    write_wav_pcm16(out, pcm, sr)
    return out


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--openai-model", default=os.environ.get("OPENAI_TTS_MODEL", "gpt-4o-mini-tts"))
    ap.add_argument("--skip-openai", action="store_true")
    ap.add_argument("--skip-gemini", action="store_true")
    ap.add_argument("--gemini-model", default="", help="Força um modelo TTS Gemini (senão só flash TTS)")
    ap.add_argument(
        "--gemini-voices",
        default="",
        help="Lista separada por vírgulas (ex. Leda,Orus). Vazio = todas.",
    )
    ap.add_argument(
        "--gemini-only-missing",
        action="store_true",
        help="Só gera WAV Gemini se o ficheiro ainda não existir.",
    )
    ap.add_argument("--gemini-sleep", type=float, default=2.5, help="Segundos entre pedidos Gemini (rate limit).")
    ap.add_argument(
        "--gemini-female-venda-casa",
        action="store_true",
        help="Só vozes femininas (Kore,Aoede,Leda,Zephyr) + texto longo com tags em gemini-tts/PHRASE_VENDA_CASA_FEM_TAGS.txt",
    )
    args = ap.parse_args()
    if args.gemini_female_venda_casa:
        args.skip_openai = True
        args.skip_gemini = False

    repo = repo_root()
    load_dotenv_file(repo / "backend" / ".env")

    phrase = read_phrase(repo)
    ok_any = False
    gemini_attempted = False

    okey = (os.environ.get("OPENAI_API_KEY") or "").strip()
    if not args.skip_openai:
        if not okey:
            print("[OpenAI] SKIP: define OPENAI_API_KEY (ex. no backend/.env)")
        else:
            out_dir = repo / "voice-samples" / "openai-tts"
            out_dir.mkdir(parents=True, exist_ok=True)
            for voice in OPENAI_VOICES:
                path = out_dir / f"openai_{voice}_phrase_pt.mp3"
                try:
                    mp3 = synth_openai_mp3(okey, args.openai_model, voice, phrase)
                    path.write_bytes(mp3)
                    print("[OpenAI] OK ->", path)
                    ok_any = True
                except Exception as e:
                    print("[OpenAI]", voice, "ERRO:", e)

    gkey = (os.environ.get("GEMINI_API_KEY") or "").strip()
    if not args.skip_gemini:
        if not gkey:
            print("[Gemini] SKIP: define GEMINI_API_KEY (ex. no backend/.env)")
        else:
            (repo / "voice-samples" / "gemini-tts").mkdir(parents=True, exist_ok=True)
            extra = (os.environ.get("GEMINI_TTS_FALLBACK_MODEL") or "").strip()
            if args.gemini_model.strip():
                models = [args.gemini_model.strip()]
            else:
                models = list(GEMINI_TTS_MODELS_DEFAULT)
                if extra:
                    models.append(extra)

            if args.gemini_female_venda_casa:
                scenario_path = repo / "voice-samples" / "gemini-tts" / "PHRASE_VENDA_CASA_FEM_TAGS.txt"
                if not scenario_path.is_file():
                    print("[Gemini] ERRO: falta ficheiro", scenario_path)
                    return 1
                scenario_text = scenario_path.read_text(encoding="utf-8").strip()
                if not scenario_text:
                    print("[Gemini] ERRO: ficheiro vazio:", scenario_path)
                    return 1
                female_voices = list(GEMINI_FEMALE_VOICES)
                if (args.gemini_voices or "").strip():
                    want = {x.strip().lower() for x in args.gemini_voices.split(",") if x.strip()}
                    female_voices = [v for v in GEMINI_FEMALE_VOICES if v.lower() in want]
                    unknown = want - {v.lower() for v in female_voices}
                    if unknown:
                        print(
                            "[Gemini] AVISO: nomes que nao sao voz feminina listada (ignorados):",
                            ", ".join(sorted(unknown)),
                        )
                    if not female_voices:
                        print("[Gemini] ERRO: --gemini-voices nao coincide com Kore,Aoede,Leda,Zephyr.")
                        return 1
                first = True
                for voice in female_voices:
                    out_path = (
                        repo / "voice-samples" / "gemini-tts" / f"gemini_{voice.lower()}_venda_casa_tags.wav"
                    )
                    if args.gemini_only_missing and out_path.is_file():
                        print("[Gemini] SKIP (já existe):", out_path)
                        continue
                    if not first and args.gemini_sleep > 0:
                        time.sleep(args.gemini_sleep)
                    first = False
                    last_err: Exception | None = None
                    for model in models:
                        try:
                            gemini_attempted = True
                            p = synth_gemini_wav(
                                gkey,
                                scenario_text,
                                voice,
                                model,
                                instruction_prefix=GEMINI_VENDA_CASA_INSTRUCTION,
                                output_path=out_path,
                            )
                            print("[Gemini]", model, voice, "venda-casa OK ->", p)
                            ok_any = True
                            last_err = None
                            break
                        except Exception as e:
                            last_err = e
                    if last_err is not None:
                        print("[Gemini]", voice, "venda-casa ERRO:", last_err)
            else:
                voices = list(GEMINI_VOICES)
                if (args.gemini_voices or "").strip():
                    want = {x.strip().lower() for x in args.gemini_voices.split(",") if x.strip()}
                    voices = [v for v in GEMINI_VOICES if v.lower() in want]
                    unknown = want - {v.lower() for v in voices}
                    if unknown:
                        print("[Gemini] AVISO: nomes desconhecidos (ignorados):", ", ".join(sorted(unknown)))
                first = True
                for voice in voices:
                    out_path = gemini_wav_path(repo, voice)
                    if args.gemini_only_missing and out_path.is_file():
                        print("[Gemini] SKIP (já existe):", out_path)
                        continue
                    if not first and args.gemini_sleep > 0:
                        time.sleep(args.gemini_sleep)
                    first = False
                    last_err: Exception | None = None
                    for model in models:
                        try:
                            gemini_attempted = True
                            p = synth_gemini_wav(gkey, phrase, voice, model)
                            print("[Gemini]", model, voice, "OK ->", p)
                            ok_any = True
                            last_err = None
                            break
                        except Exception as e:
                            last_err = e
                    if last_err is not None:
                        print("[Gemini]", voice, "ERRO (todos os modelos):", last_err)

    if not ok_any:
        if not args.skip_gemini and gkey and args.gemini_only_missing and not gemini_attempted:
            print("Gemini: nada em falta (todos os WAV já existem).")
            return 0
        print("Nenhum ficheiro gerado. Configura chaves e volta a correr.")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
