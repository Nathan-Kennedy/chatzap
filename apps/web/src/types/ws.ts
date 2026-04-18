/** Eventos esperados do canal WebSocket (playbook). */

export type WsEventType =
  | 'message.created'
  | 'conversation.updated'
  | 'notification.created'

export type WsPayload = {
  type: WsEventType
  payload: Record<string, unknown>
}
