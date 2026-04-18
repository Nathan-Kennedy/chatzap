const palette = ['bg-cyan-600', 'bg-emerald-600', 'bg-violet-600', 'bg-amber-600', 'bg-rose-600']

export function initialsFromName(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

export function avatarColorClass(name: string): string {
  let h = 0
  for (let i = 0; i < name.length; i++) h = (h + name.charCodeAt(i) * (i + 1)) % palette.length
  return palette[h] ?? palette[0]
}
