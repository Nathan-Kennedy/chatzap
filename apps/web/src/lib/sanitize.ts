import DOMPurify from 'dompurify'

/** Texto de chat: remove HTML/scripts; exibe como texto seguro. */
export function sanitizeMessageBody(raw: string): string {
  return DOMPurify.sanitize(raw, { ALLOWED_TAGS: [] })
}
