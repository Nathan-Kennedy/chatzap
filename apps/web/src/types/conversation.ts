export type ConversationStatus = 'open' | 'pending' | 'resolved'

export type Conversation = {
  id: string
  /** Instância WhatsApp (UUID); ausente em respostas antigas — usar picker da Inbox. */
  whatsapp_instance_id?: string
  contact_id: string
  contact_name: string
  contact_phone: string
  channel: 'whatsapp' | 'instagram' | 'email'
  last_message_preview: string
  unread_count: number
  updated_at: string
  assigned_agent_id?: string
  assigned_agent_initials?: string
  status: ConversationStatus
}

export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'document'

export type Message = {
  id: string
  conversation_id: string
  direction: 'inbound' | 'outbound' | 'system'
  body: string
  created_at: string
  message_type?: MessageType | string
  file_name?: string
  mime_type?: string
  /** Há ficheiro local ou URL remota servida por GET .../attachment */
  has_attachment?: boolean
  is_private_note?: boolean
  is_ai?: boolean
}
