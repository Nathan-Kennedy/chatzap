import type { UserRole } from '@/types/user'

const ACCESS = 'access_token'
const REFRESH = 'refresh_token'
const LEGACY = 'token'
const PROFILE = 'auth_profile'

export type AuthProfile = {
  workspace_id: string
  workspace_name?: string
  role: UserRole
  user_name?: string
  user_email?: string
}

export function setAuthProfile(profile: AuthProfile): void {
  localStorage.setItem(PROFILE, JSON.stringify(profile))
}

export function getAuthProfile(): AuthProfile | null {
  const raw = localStorage.getItem(PROFILE)
  if (!raw) return null
  try {
    return JSON.parse(raw) as AuthProfile
  } catch {
    return null
  }
}

export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS) ?? localStorage.getItem(LEGACY)
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH)
}

export function setTokens(access: string, refresh?: string | null): void {
  localStorage.setItem(ACCESS, access)
  if (refresh) localStorage.setItem(REFRESH, refresh)
  localStorage.removeItem(LEGACY)
}

export function clearAuthStorage(): void {
  localStorage.removeItem(ACCESS)
  localStorage.removeItem(REFRESH)
  localStorage.removeItem(LEGACY)
  localStorage.removeItem(PROFILE)
}

/** Mock explícito só em dev (VITE_ENABLE_AUTH_MOCK=true). */
export function isAuthMockEnabled(): boolean {
  return (
    import.meta.env.DEV === true &&
    import.meta.env.VITE_ENABLE_AUTH_MOCK === 'true'
  )
}

export function hasAuthSession(): boolean {
  if (isAuthMockEnabled()) return true
  return !!getAccessToken()
}
