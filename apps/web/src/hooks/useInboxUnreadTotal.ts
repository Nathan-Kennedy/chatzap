import { useMemo } from 'react'
import { useConversations } from '@/hooks/useConversations'

/** Total de mensagens não lidas na inbox (soma dos unread_count). Cache partilhado com a página Inbox (search vazio). */
export function useInboxUnreadTotal(): number {
  const { data: conversations = [] } = useConversations('')
  return useMemo(
    () =>
      conversations.reduce((sum, c) => sum + (Number(c.unread_count) || 0), 0),
    [conversations]
  )
}
