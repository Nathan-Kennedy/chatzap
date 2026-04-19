"""
POST /v1/tts com JSON (o servidor Fish Speech aceita application/json ou msgpack).
Uso: ver voice-samples/fish-speech/README_FISH_TAGS.md
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

try:
    import requests
except ImportError:
    print("Instala: pip install requests")
    sys.exit(1)


def health_url_from_tts(tts_url: str) -> str:
    u = tts_url.rstrip("/")
    if u.endswith("/v1/tts"):
        return u[: -len("/v1/tts")] + "/v1/health"
    # http://host:port/... -> strip path for default health on same host:port
    if "/v1/" in u:
        return u.rsplit("/", 1)[0] + "/health"
    return u.rstrip("/") + "/v1/health"


def check_server_reachable(tts_url: str) -> str | None:
    """Devolve mensagem de erro em portugues se o servidor nao responder; None se OK."""
    h = health_url_from_tts(tts_url)
    try:
        r = requests.get(h, timeout=3)
        if r.status_code == 200:
            return None
        return f"Servidor respondeu HTTP {r.status_code} em {h}"
    except requests.exceptions.RequestException as e:
        base = tts_url.split("/v1")[0] if "/v1" in tts_url else tts_url
        return (
            f"Nao foi possivel ligar ao Fish Speech em {base} (erro: {e}).\n"
            "Arranca o servidor noutro terminal antes de gerar o WAV:\n"
            "  npm run fish:server\n"
            "Requisitos: clone + venv em services/tts-extras/vendor/fish-speech, checkpoints s2-pro, "
            "ver services/tts-extras/README.md e scripts/run-fish-speech-server.ps1"
        )


def build_payload(text: str) -> dict:
    return {
        "text": text,
        "chunk_length": 300,
        "format": "wav",
        "latency": "normal",
        "references": [],
        "reference_id": None,
        "seed": None,
        "use_memory_cache": "off",
        "normalize": True,
        "streaming": False,
        "max_new_tokens": 1024,
        "top_p": 0.8,
        "repetition_penalty": 1.1,
        "temperature": 0.8,
    }


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--url", default="http://127.0.0.1:8080/v1/tts")
    ap.add_argument("--text-file", type=Path, required=True)
    ap.add_argument("--out", type=Path, required=True)
    ap.add_argument(
        "--api-key",
        default="",
        help="Bearer token se o servidor Fish foi arrancado com --api-key; vazio = sem cabecalho Authorization.",
    )
    args = ap.parse_args()

    text = args.text_file.read_text(encoding="utf-8").strip()
    if not text:
        print("Texto vazio:", args.text_file)
        return 1

    url = args.url.rstrip("/")
    if not url.endswith("/v1/tts"):
        url = url.rstrip("/") + "/v1/tts"

    err = check_server_reachable(url)
    if err:
        print(err)
        return 1

    body = build_payload(text)
    headers = {"Content-Type": "application/json"}
    if (args.api_key or "").strip():
        headers["Authorization"] = f"Bearer {args.api_key.strip()}"
    print("POST", url, "chars", len(text))
    try:
        r = requests.post(url, json=body, headers=headers, timeout=600)
    except requests.exceptions.ConnectionError as e:
        print(
            "Ligacao perdida durante o POST. Confirma que o Fish Speech continua a correr.\n",
            e,
            sep="",
        )
        return 1
    if r.status_code != 200:
        print("HTTP", r.status_code, r.text[:800])
        return 1

    ct = (r.headers.get("Content-Type") or "").lower()
    if "wav" not in ct and not r.content[:4] == b"RIFF":
        print("Resposta inesperada Content-Type:", ct, "primeiros bytes:", r.content[:32])
        return 1

    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_bytes(r.content)
    print("OK ->", args.out, "size", len(r.content))
    return 0


if __name__ == "__main__":
    sys.exit(main())
