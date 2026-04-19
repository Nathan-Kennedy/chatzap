"""
Gera WAV em voice-samples/{f5-tts,fish-speech,cosyvoice} para comparar TTS extras.
Requer: venv F5 (f5-tts_infer-cli) para F5; servidor Fish :8080; servidor Cosy :50000.
Executar normalmente com o Python do venv F5 para ter `requests` disponivel.
"""
from __future__ import annotations

import argparse
import array
import os
import subprocess
import sys
import wave
from pathlib import Path

try:
    import requests
except ImportError:
    print("Instala requests no Python usado (ex.: services/tts-extras/venvs/f5-tts/Scripts/pip install requests)")
    sys.exit(1)


def repo_root_from_script() -> Path:
    return Path(__file__).resolve().parents[1]


def read_phrase(repo: Path) -> str:
    p = repo / "voice-samples" / "PHRASE_PT.txt"
    text = p.read_text(encoding="utf-8").strip()
    if not text:
        raise SystemExit(f"Vazio: {p}")
    return text


def venv_cli(repo: Path, name: str) -> Path | None:
    win = repo / "services" / "tts-extras" / "venvs" / "f5-tts" / "Scripts" / f"{name}.exe"
    if win.is_file():
        return win
    nix = repo / "services" / "tts-extras" / "venvs" / "f5-tts" / "bin" / name
    if nix.is_file():
        return nix
    return None


def run_f5(
    repo: Path,
    phrase: str,
    *,
    model: str = "F5TTS_v1_Base",
    device: str | None = None,
) -> bool:
    cli = venv_cli(repo, "f5-tts_infer-cli")
    if not cli:
        print("[F5] Skip: f5-tts_infer-cli nao encontrado (corre install-tts-extras.ps1 -Components F5)")
        return False
    out_dir = repo / "voice-samples" / "f5-tts"
    out_dir.mkdir(parents=True, exist_ok=True)
    out_wav = "f5_phrase_pt.wav"
    cmd = [
        str(cli),
        "--model",
        model,
        "-t",
        phrase,
        "-o",
        str(out_dir),
        "-w",
        out_wav,
    ]
    if device:
        cmd += ["--device", device]
    env = os.environ.copy()
    # Reduz fragmentacao de alocacao (ajuda marginalmente em picos de RAM).
    env.setdefault("PYTORCH_ALLOC_CONF", "expandable_segments:True")
    print("[F5]", " ".join(cmd))
    r = subprocess.run(cmd, cwd=str(repo), env=env)
    if r.returncode != 0:
        print("[F5] falhou com codigo", r.returncode)
        print(
            "[F5] Se foi 1455/paginacao: aumenta memoria virtual; fecha apps; ou --f5-model F5TTS_Base. "
            "Se foi 'Torch not compiled with CUDA enabled': o venv tem PyTorch CPU — nao uses --f5-device cuda "
            "sem reinstalar torch+torchaudio com CUDA (ver README tts-extras)."
        )
        return False
    target = out_dir / out_wav
    if target.is_file():
        print("[F5] OK ->", target)
        return True
    print("[F5] ficheiro em falta:", target)
    return False


def fish_health(url: str) -> bool:
    try:
        r = requests.get(f"{url.rstrip('/')}/v1/health", timeout=3)
        return r.status_code == 200
    except OSError:
        return False


def run_fish(repo: Path, phrase: str, base_url: str) -> bool:
    if not fish_health(base_url):
        print("[Fish] Skip: servidor nao responde em", base_url, "(npm run fish:server)")
        return False
    fish_repo = repo / "services" / "tts-extras" / "vendor" / "fish-speech"
    py = repo / "services" / "tts-extras" / "venvs" / "fish-speech" / "Scripts" / "python.exe"
    if not py.is_file():
        py = repo / "services" / "tts-extras" / "venvs" / "fish-speech" / "bin" / "python"
    client = fish_repo / "tools" / "api_client.py"
    if not client.is_file():
        print("[Fish] Skip: clone em falta:", fish_repo)
        return False
    out_dir = repo / "voice-samples" / "fish-speech"
    out_dir.mkdir(parents=True, exist_ok=True)
    out_base = out_dir / "fish_phrase_pt"
    cmd = [
        str(py),
        str(client),
        "-u",
        f"{base_url.rstrip('/')}/v1/tts",
        "-t",
        phrase,
        "--no-play",
        "-o",
        str(out_base),
        "--format",
        "wav",
        "--api_key",
        "local",
    ]
    print("[Fish]", " ".join(cmd))
    r = subprocess.run(cmd, cwd=str(fish_repo))
    if r.returncode != 0:
        print("[Fish] api_client falhou:", r.returncode)
        return False
    wav = Path(str(out_base) + ".wav")
    if wav.is_file():
        print("[Fish] OK ->", wav)
        return True
    return False


def run_cosy(
    repo: Path,
    phrase: str,
    base_url: str,
    prompt_wav: Path,
    target_sr: int = 22050,
) -> bool:
    url = f"{base_url.rstrip('/')}/inference_cross_lingual"
    if not prompt_wav.is_file():
        print("[Cosy] Skip: prompt WAV em falta:", prompt_wav)
        return False
    try:
        with prompt_wav.open("rb") as f:
            r = requests.post(
                url,
                data={"tts_text": phrase},
                files={"prompt_wav": ("prompt.wav", f, "application/octet-stream")},
                timeout=600,
            )
    except OSError as e:
        print("[Cosy] erro rede:", e)
        return False
    if r.status_code != 200:
        print("[Cosy] HTTP", r.status_code, r.text[:500])
        return False
    pcm = r.content
    if len(pcm) < 100:
        print("[Cosy] resposta vazia")
        return False
    out_dir = repo / "voice-samples" / "cosyvoice"
    out_dir.mkdir(parents=True, exist_ok=True)
    out_path = out_dir / "cosy_cross_lingual_phrase_pt.wav"
    arr = array.array("h")
    arr.frombytes(pcm)
    with wave.open(str(out_path), "wb") as wf:
        wf.setnchannels(1)
        wf.setsampwidth(2)
        wf.setframerate(target_sr)
        wf.writeframes(arr.tobytes())
    print("[Cosy] OK ->", out_path)
    return True


def ensure_hf_cache(repo: Path) -> None:
    """Alinha com scripts PowerShell: CHATBOT_TTS_CACHE_ROOT\\hf-cache ou services/tts-extras/hf-cache."""
    ext = (os.environ.get("CHATBOT_TTS_CACHE_ROOT") or "").strip()
    if ext:
        cache = Path(ext) / "hf-cache"
    else:
        cache = repo / "services" / "tts-extras" / "hf-cache"
    cache.mkdir(parents=True, exist_ok=True)
    os.environ.setdefault("HF_HUB_CACHE", str(cache))
    os.environ.setdefault("HF_HOME", str(cache))


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--repo-root", type=Path, default=None)
    ap.add_argument("--fish-url", default="http://127.0.0.1:8080")
    ap.add_argument("--cosy-url", default="http://127.0.0.1:50000")
    ap.add_argument("--skip-f5", action="store_true")
    ap.add_argument("--skip-fish", action="store_true")
    ap.add_argument("--skip-cosy", action="store_true")
    ap.add_argument(
        "--f5-model",
        default="F5TTS_v1_Base",
        help="Ex.: F5TTS_Base (checkpoint menor) se F5TTS_v1_Base falhar por memoria.",
    )
    ap.add_argument(
        "--f5-device",
        default="",
        help="Ex.: cpu ou cuda — vazio deixa o default do F5-TTS.",
    )
    args = ap.parse_args()
    repo = args.repo_root or repo_root_from_script()
    repo = repo.resolve()
    ensure_hf_cache(repo)
    phrase = read_phrase(repo)
    prompt = repo / "services" / "tts-extras" / "assets" / "cross_lingual_prompt.wav"

    ok_any = False
    f5_dev = (args.f5_device or "").strip() or None
    if not args.skip_f5 and run_f5(repo, phrase, model=args.f5_model, device=f5_dev):
        ok_any = True
    if not args.skip_fish and run_fish(repo, phrase, args.fish_url):
        ok_any = True
    if not args.skip_cosy and run_cosy(repo, phrase, args.cosy_url, prompt):
        ok_any = True

    if not ok_any:
        print("Nenhuma amostra gerada. Verifica venv F5, servidores Fish/Cosy e assets.")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
