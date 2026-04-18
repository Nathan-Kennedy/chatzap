import { describe, it, expect } from 'vitest'
import { sanitizeMessageBody } from './sanitize'

describe('sanitizeMessageBody', () => {
  it('remove tags script', () => {
    expect(sanitizeMessageBody('<script>alert(1)</script>olá')).toBe('olá')
  })

  it('mantém texto plano', () => {
    expect(sanitizeMessageBody('Oi\nlinha')).toBe('Oi\nlinha')
  })
})
