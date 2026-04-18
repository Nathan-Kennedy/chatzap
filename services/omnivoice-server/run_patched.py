"""
Workaround: omnivoice_server.ModelService._has_nan usa torch.isnan em todos os elementos,
mas OmniVoice.generate devolve list[np.ndarray] (ver omnivoice.models.omnivoice.OmniVoice.generate).
Sem isto, o arranque falha com: isnan(): argument 'input' must be Tensor, not numpy.ndarray.

Workaround 2: omnivoice_server.utils.audio.* assume torch.Tensor; na inferência real os tensores
podem ser numpy — tensor_to_wav_bytes chama .cpu() e falha com AttributeError.
Isto tem de ser aplicado antes de importar o CLI (routers importam audio uma vez).
"""
from __future__ import annotations

import numpy as np
import torch


def _has_nan_fixed(tensors: list) -> bool:
    for t in tensors:
        if isinstance(t, torch.Tensor):
            if torch.isnan(t).any():
                return True
        else:
            arr = np.asarray(t)
            if arr.size and np.isnan(arr).any():
                return True
    return False


def _install_patch() -> None:
    from omnivoice_server.services import model as model_mod

    model_mod.ModelService._has_nan = staticmethod(_has_nan_fixed)  # type: ignore[method-assign]


def _ensure_torch_float_tensor(x):
    if isinstance(x, torch.Tensor):
        return x
    arr = np.ascontiguousarray(np.asarray(x, dtype=np.float32))
    return torch.from_numpy(arr)


def _install_audio_numpy_patch() -> None:
    import omnivoice_server.utils.audio as audio_mod

    _orig_t2w = audio_mod.tensor_to_wav_bytes
    _orig_pcm = audio_mod.tensor_to_pcm16_bytes

    def tensor_to_wav_bytes(tensor) -> bytes:
        return _orig_t2w(_ensure_torch_float_tensor(tensor))

    def tensors_to_wav_bytes(tensors: list) -> bytes:
        # Não delegar a audio.tensors_to_wav_bytes original: esse corpo chama
        # tensor_to_wav_bytes via LOAD_GLOBAL e, em alguns arranques, ainda pode
        # resolver para a implementação antiga. Reimplementamos com numpy→tensor.
        ts = [_ensure_torch_float_tensor(t) for t in tensors]
        if len(ts) == 1:
            return _orig_t2w(ts[0])
        combined = torch.cat([t.cpu() for t in ts], dim=-1)
        return _orig_t2w(combined)

    def tensor_to_pcm16_bytes(tensor) -> bytes:
        return _orig_pcm(_ensure_torch_float_tensor(tensor))

    audio_mod.tensor_to_wav_bytes = tensor_to_wav_bytes
    audio_mod.tensors_to_wav_bytes = tensors_to_wav_bytes
    audio_mod.tensor_to_pcm16_bytes = tensor_to_pcm16_bytes


_install_patch()
_install_audio_numpy_patch()

from omnivoice_server.cli import main

if __name__ == "__main__":
    main()
