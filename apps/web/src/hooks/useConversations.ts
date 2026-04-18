import { useQuery } from '@tanstack/react-query'
import { api, unwrapEnvelope } from '@/lib/api'
import type { Conversation } from '@/types/conversation'

export function useConversations(search?: string) {
  return useQuery({
    queryKey: ['conversations', { search: search ?? '' }],
    queryFn: async (): Promise<Conversation[]> => {
      const params = search?.trim() ? { search: search.trim() } : undefined
      const res = await api.get<unknown>('/conversations', { params })
      const { data } = unwrapEnvelope<Conversation[]>(res)
      return data
    },
  })
}
