import type { UserRole } from '@/types/user'

/** Ações de UI alinhadas à matriz RBAC do playbook. */
export type NavKey =
  | 'inbox'
  | 'contacts'
  | 'kanban'
  | 'campaigns'
  | 'agents'
  | 'instances'
  | 'flows'
  | 'analytics'
  | 'settings'

const navRules: Record<NavKey, UserRole[]> = {
  inbox: ['admin', 'supervisor', 'agent'],
  contacts: ['admin', 'supervisor', 'agent'],
  kanban: ['admin', 'supervisor', 'agent'],
  campaigns: ['admin', 'supervisor'],
  agents: ['admin', 'supervisor'],
  instances: ['admin'],
  flows: ['admin', 'supervisor'],
  analytics: ['admin', 'supervisor'],
  settings: ['admin', 'supervisor', 'agent'],
}

export function canAccessNav(role: UserRole | undefined, key: NavKey): boolean {
  if (!role) return false
  return navRules[key]?.includes(role) ?? false
}

export function filterNavPaths<T extends { path: string }>(
  role: UserRole | undefined,
  items: (T & { navKey: NavKey })[]
): (T & { navKey: NavKey })[] {
  return items.filter((item) => canAccessNav(role, item.navKey))
}
