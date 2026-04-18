import { describe, it, expect } from 'vitest'
import { canAccessNav } from './permissions'

describe('canAccessNav', () => {
  it('agent não acessa instâncias', () => {
    expect(canAccessNav('agent', 'instances')).toBe(false)
  })

  it('admin acessa instâncias', () => {
    expect(canAccessNav('admin', 'instances')).toBe(true)
  })

  it('sem role nega', () => {
    expect(canAccessNav(undefined, 'inbox')).toBe(false)
  })
})
