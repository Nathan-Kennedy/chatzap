import { format, formatDistanceToNow } from 'date-fns'
import { ptBR } from 'date-fns/locale'

export function formatRelativeShort(iso: string): string {
  const d = new Date(iso)
  const diff = Date.now() - d.getTime()
  if (diff < 60000) return 'Agora'
  if (diff < 86400000) return format(d, 'HH:mm', { locale: ptBR })
  return formatDistanceToNow(d, { addSuffix: true, locale: ptBR })
}

export function formatMessageTime(iso: string): string {
  return format(new Date(iso), 'HH:mm', { locale: ptBR })
}
