#!/usr/bin/env python3
"""Corta WAV de referência para clone OmniVoice (recomendado 3–10 s; aviso upstream >20 s)."""
from __future__ import annotations

import sys
from pathlib import Path

import torchaudio


def main() -> None:
    if len(sys.argv) < 3:
        print("Uso: trim_ref_audio.py <entrada.wav> <saida.wav> [segundos_max]", file=sys.stderr)
        sys.exit(1)
    inp, outp = Path(sys.argv[1]), Path(sys.argv[2])
    max_sec = float(sys.argv[3]) if len(sys.argv) > 3 else 8.0
    wav, sr = torchaudio.load(str(inp))
    if wav.dim() == 1:
        wav = wav.unsqueeze(0)
    n_samples = int(max_sec * sr)
    if wav.shape[1] > n_samples:
        wav = wav[:, :n_samples]
    torchaudio.save(str(outp), wav, sr)
    dur = wav.shape[1] / sr
    print(f"OK: {outp} — {dur:.2f}s, {sr} Hz, canais={wav.shape[0]}")


if __name__ == "__main__":
    main()
