import type { Conversation, Message } from '@/types/conversation'

const now = new Date().toISOString()

export const mockConversations: Conversation[] = [
  {
    id: 'c1',
    contact_id: 'p1',
    contact_name: 'Maria Costa',
    contact_phone: '+55 11 98888-7777',
    channel: 'whatsapp',
    last_message_preview: 'Olá, gostaria de saber sobre...',
    unread_count: 1,
    updated_at: now,
    assigned_agent_initials: 'JD',
    status: 'open',
  },
  {
    id: 'c2',
    contact_id: 'p2',
    contact_name: 'João Batista',
    contact_phone: '+55 21 97777-6666',
    channel: 'whatsapp',
    last_message_preview: 'Obrigado pelo retorno.',
    unread_count: 0,
    updated_at: new Date(Date.now() - 3600000).toISOString(),
    status: 'open',
  },
]

export const mockMessagesByConversation: Record<string, Message[]> = {
  c1: [
    {
      id: 'm1',
      conversation_id: 'c1',
      direction: 'inbound',
      body: 'Olá, gostaria de saber sobre os planos do WhatsSaaS.',
      created_at: new Date(Date.now() - 7200000).toISOString(),
    },
    {
      id: 'm2',
      conversation_id: 'c1',
      direction: 'outbound',
      body: 'Olá Maria! Claro, nossa plataforma possui 3 planos principais. Qual o tamanho da sua equipe atual?',
      created_at: new Date(Date.now() - 7100000).toISOString(),
      is_ai: true,
    },
  ],
  c2: [
    {
      id: 'm3',
      conversation_id: 'c2',
      direction: 'inbound',
      body: 'Obrigado pelo retorno.',
      created_at: new Date(Date.now() - 86400000).toISOString(),
    },
  ],
}

export function delay(ms: number) {
  return new Promise((r) => setTimeout(r, ms))
}
