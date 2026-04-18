import { useEffect, useState } from 'react'
import { api } from '@/lib/api'

/** Detecta formato real do ficheiro (o servidor por vezes declara OGG e os bytes são MP4/WebM). */
function sniffAudioMime(buf: ArrayBuffer): string | undefined {
  const u = new Uint8Array(buf)
  if (u.length < 4) return undefined
  const ascii = (i: number, n: number) => {
    let s = ''
    for (let k = 0; k < n; k++) s += String.fromCharCode(u[i + k]!)
    return s
  }
  // PTT WhatsApp: Opus em OGG — alguns browsers (ex. Chromium) tratam melhor com codecs na hint.
  if (ascii(0, 4) === 'OggS') return 'audio/ogg; codecs=opus'
  if (u[0] === 0x1a && u[1] === 0x45 && u[2] === 0xdf && u[3] === 0xa3) return 'audio/webm'
  if (u.length >= 12 && ascii(4, 4) === 'ftyp') return 'audio/mp4'
  if (u.length >= 12 && ascii(0, 4) === 'RIFF' && ascii(8, 4) === 'WAVE') return 'audio/wav'
  if (u.length >= 3 && ascii(0, 3) === 'ID3') return 'audio/mpeg'
  if (u.length >= 2 && u[0] === 0xff && (u[1]! & 0xe0) === 0xe0) return 'audio/mpeg'
  return undefined
}

/** Escolhe tipo do Blob: sem isto, OGG/Opus com Content-Type octet-stream ou omitido pode falhar no elemento audio. */
function blobTypeForAttachment(rawContentType: string, axiosBlob: Blob, hint?: string): string {
  const r = rawContentType.trim()
  const rl = r.toLowerCase()
  if (r.length > 0 && rl !== 'application/octet-stream') {
    return r
  }
  const ax = (axiosBlob.type || '').trim()
  const axl = ax.toLowerCase()
  if (ax.length > 0 && axl !== 'application/octet-stream') {
    return ax
  }
  const h = hint?.trim()
  if (h) return h
  return 'application/octet-stream'
}

/** URL blob para GET /conversations/:cid/messages/:mid/attachment (com JWT). */
export function useMessageAttachmentBlobUrl(
  conversationId: string | undefined,
  messageId: string | undefined,
  /** true: tenta GET (ex.: tipo image/audio ou has_attachment na API). Mensagens recuperadas podem ter URL sem flag. */
  shouldFetch: boolean | undefined,
  /** ex.: áudio WhatsApp → `audio/ogg; codecs=opus` se o servidor enviar octet-stream */
  blobTypeHint?: string,
) {
  const [url, setUrl] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (!conversationId || !messageId || !shouldFetch) {
      setUrl(null)
      setFailed(false)
      setLoading(false)
      return
    }
    let cancelled = false
    let blobUrl: string | null = null
    setLoading(true)
    setFailed(false)
    ;(async () => {
      try {
        const res = await api.get<Blob>(
          `/conversations/${conversationId}/messages/${messageId}/attachment`,
          { responseType: 'blob' },
        )
        const blob = res.data
        const hdr = res.headers as Record<string, string | undefined>
        const rawCt = String(hdr['content-type'] ?? hdr['Content-Type'] ?? '').trim()
        const ct = rawCt.toLowerCase()
        // Erros da API vêm como JSON mesmo com responseType blob.
        if (ct.includes('application/json')) {
          throw new Error('attachment_json_response')
        }
        const buf = await blob.arrayBuffer()
        if (buf.byteLength === 0) {
          throw new Error('attachment_empty')
        }
        // Erro JSON por vezes vem com Content-Type errado (ex.: text/html ou octet-stream).
        const u8 = new Uint8Array(buf)
        let i = 0
        while (i < u8.length && (u8[i] === 0x20 || u8[i] === 0x09 || u8[i] === 0x0a || u8[i] === 0x0d)) i++
        if (i < u8.length && u8[i] === 0x7b) {
          throw new Error('attachment_json_body')
        }
        const rl = rawCt.toLowerCase()
        const hintAudio = blobTypeHint?.toLowerCase().startsWith('audio') ?? false
        const sniffed = hintAudio || rl.includes('audio') ? sniffAudioMime(buf) : undefined
        const typ = sniffed ?? blobTypeForAttachment(rawCt, blob, blobTypeHint)
        const typed =
          typ !== 'application/octet-stream' ? new Blob([buf], { type: typ }) : new Blob([buf])
        blobUrl = URL.createObjectURL(typed)
        if (!cancelled) setUrl(blobUrl)
      } catch {
        if (!cancelled) {
          setUrl(null)
          setFailed(true)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
      if (blobUrl) URL.revokeObjectURL(blobUrl)
    }
  }, [conversationId, messageId, shouldFetch, blobTypeHint])

  return { url, loading, failed }
}
