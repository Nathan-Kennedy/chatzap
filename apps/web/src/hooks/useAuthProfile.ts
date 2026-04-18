import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  getAuthProfile,
  isAuthMockEnabled,
  clearAuthStorage,
  getAccessToken,
  setAuthProfile,
  type AuthProfile,
} from '@/lib/auth-storage'
import { api, unwrapEnvelope } from '@/lib/api'
import type { UserRole } from '@/types/user'

export const authProfileQueryKey = ['auth', 'profile'] as const

const mockProfile: AuthProfile = {
  workspace_id: 'mock-workspace',
  workspace_name: 'Workspace (mock)',
  role: 'admin',
  user_name: 'Dev Mock',
  user_email: 'mock@local.dev',
}

type MeResponse = {
  user: {
    id: string
    email: string
    name?: string
    role: UserRole
  }
  workspace_id: string
  workspace_name?: string
}

async function loadProfile(): Promise<AuthProfile | null> {
  if (isAuthMockEnabled()) {
    return mockProfile
  }
  const cached = getAuthProfile()
  const token = getAccessToken()
  if (!token) {
    return cached
  }
  try {
    const res = await api.get('/auth/me')
    const { data } = unwrapEnvelope<MeResponse>(res)
    const p: AuthProfile = {
      workspace_id: data.workspace_id,
      workspace_name: data.workspace_name,
      role: data.user.role,
      user_name: data.user.name,
      user_email: data.user.email,
    }
    setAuthProfile(p)
    return p
  } catch {
    return cached
  }
}

export function useAuthProfile(): AuthProfile | null {
  const { data } = useQuery({
    queryKey: authProfileQueryKey,
    queryFn: loadProfile,
    staleTime: 60_000,
  })
  if (data) return data
  if (isAuthMockEnabled()) return mockProfile
  return getAuthProfile()
}

export function useUserRole(): UserRole | undefined {
  return useAuthProfile()?.role
}

export function useLogoutCleanup() {
  const qc = useQueryClient()
  return () => {
    clearAuthStorage()
    qc.removeQueries({ queryKey: authProfileQueryKey })
    qc.clear()
  }
}
