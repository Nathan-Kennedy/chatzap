/** Dígitos internacionais para wa.me / tel (sem +). */
export function digitsForDial(phoneOrJid: string): string {
  const s = phoneOrJid.trim()
  const head = s.split('@')[0] ?? s
  return head.replace(/\D/g, '')
}

export function waMeUrl(phoneOrJid: string): string | null {
  const d = digitsForDial(phoneOrJid)
  if (d.length < 8) return null
  return `https://wa.me/${d}`
}

export function telUrl(phoneOrJid: string): string | null {
  const d = digitsForDial(phoneOrJid)
  if (d.length < 8) return null
  return `tel:+${d}`
}
