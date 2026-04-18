import { useQuery } from '@tanstack/react-query'
import { api, unwrapEnvelope } from '@/lib/api'

export type InstanceListRow = {
  id: string
  name: string
  evolution_instance_name?: string
  number: string
  status: 'connected' | 'qr_pending' | 'disconnected'
  messages_today: number
}

export function useInstances() {
  return useQuery({
    queryKey: ['instances'],
    queryFn: async (): Promise<InstanceListRow[]> => {
      const res = await api.get<unknown>('/instances')
      const { data } = unwrapEnvelope<InstanceListRow[]>(res)
      return data
    },
  })
}
