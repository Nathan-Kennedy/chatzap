import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { getAccessToken, isAuthMockEnabled } from '@/lib/auth-storage'
import type { WsPayload } from '@/types/ws'

const BASE_DELAY_MS = 1000
const MAX_DELAY_MS = 30000

type Options = {
  enabled?: boolean
  onEvent?: (msg: WsPayload) => void
}

function buildWsUrl(): string | null {
  const base = import.meta.env.VITE_WS_URL
  if (!base) return null
  const token = getAccessToken()
  if (!token && !isAuthMockEnabled()) return null
  if (isAuthMockEnabled()) return null
  const sep = base.includes('?') ? '&' : '?'
  return token ? `${base}${sep}token=${encodeURIComponent(token)}` : base
}

export function useRealtime(options: Options = {}) {
  const { enabled = true, onEvent } = options
  const qc = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)
  const attemptRef = useRef(0)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const onEventRef = useRef(onEvent)

  useEffect(() => {
    onEventRef.current = onEvent
  }, [onEvent])

  useEffect(() => {
    if (!enabled) return

    const url = buildWsUrl()
    if (!url) return

    let closed = false

    const connect = () => {
      if (closed) return
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        attemptRef.current = 0
      }

      ws.onmessage = (ev) => {
        try {
          const raw = JSON.parse(ev.data as string) as Record<string, unknown>
          const msg = raw as unknown as WsPayload
          const t = raw.type
          if (t === 'ping') return
          if (t === 'message.created' || t === 'conversation.updated') {
            void qc.invalidateQueries({ queryKey: ['conversations'] })
            const pl = raw.payload as { conversation_id?: string } | undefined
            const cid =
              (typeof pl?.conversation_id === 'string' && pl.conversation_id) ||
              (typeof raw.conversation_id === 'string' ? raw.conversation_id : undefined)
            if (cid) {
              void qc.invalidateQueries({
                queryKey: ['conversation', cid, 'messages'],
              })
            }
          }
          onEventRef.current?.(msg)
        } catch {
          /* ignore parse errors */
        }
      }

      ws.onclose = () => {
        wsRef.current = null
        if (closed) return
        const delay = Math.min(
          MAX_DELAY_MS,
          BASE_DELAY_MS * Math.pow(2, attemptRef.current)
        )
        attemptRef.current += 1
        timerRef.current = setTimeout(connect, delay)
      }

      ws.onerror = () => {
        ws.close()
      }
    }

    const ping = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'ping' }))
      }
    }, 25000)

    connect()

    return () => {
      closed = true
      clearInterval(ping)
      if (timerRef.current) clearTimeout(timerRef.current)
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [enabled, qc])
}
