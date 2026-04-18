import type { Contact } from '@/types/contact'
import { delay } from '@/mocks/inbox-mocks'

export const mockContacts: Contact[] = [
  {
    id: 'ct1',
    name: 'Maria Costa',
    phone: '+55 11 98888-7777',
    email: 'maria@empresa.com',
    tags: ['VIP', 'Lead'],
    lastInteractionAt: new Date().toISOString(),
    agentName: 'João Doe',
  },
  {
    id: 'ct2',
    name: 'Pedro Santos',
    phone: '+55 21 97777-6666',
    tags: ['Suporte'],
    lastInteractionAt: new Date(Date.now() - 86400000).toISOString(),
  },
]

export async function fetchContactsMock(): Promise<Contact[]> {
  await delay(100)
  return [...mockContacts]
}
