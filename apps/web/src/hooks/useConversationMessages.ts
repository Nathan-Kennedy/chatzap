import { useQuery } from '@tanstack/react-query'
import { api, unwrapEnvelope } from '@/lib/api'
import type { Message } from '@/types/conversation'

const WS_POLL_MS = 12_000

export function useConversationMessages(conversationId: string | null) {
  const wsConfigured = Boolean(import.meta.env.VITE_WS_URL?.trim())

  return useQuery({
    queryKey: ['conversation', conversationId, 'messages'],
    queryFn: async (): Promise<Message[]> => {
      const res = await api.get<unknown>(`/conversations/${conversationId}/messages`)
      const { data } = unwrapEnvelope<Message[]>(res)
      return data
    },
    enabled: !!conversationId,
    // Sem WebSocket, a Inbox não atualizava mensagens recebidas até refrescar manualmente.
    refetchInterval: conversationId && !wsConfigured ? WS_POLL_MS : false,
    refetchIntervalInBackground: false,
  })
}
