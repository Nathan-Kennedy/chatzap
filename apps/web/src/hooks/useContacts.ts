import { useQuery } from '@tanstack/react-query'
import { api, unwrapEnvelope } from '@/lib/api'

export type ContactRow = {
  id: string
  name: string
  phone: string
  channel: string
  last_seen_at: string
}

export function useContacts(search?: string) {
  return useQuery({
    queryKey: ['contacts', { search: search ?? '' }],
    queryFn: async (): Promise<ContactRow[]> => {
      const params = search?.trim() ? { search: search.trim() } : undefined
      const res = await api.get<unknown>('/contacts', { params })
      const { data } = unwrapEnvelope<ContactRow[]>(res)
      return data
    },
  })
}
