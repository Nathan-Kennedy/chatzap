import { api, unwrapEnvelope } from '@/lib/api'
import type { LoginRequest, LoginResponseData } from '@/types/auth'
import { getRefreshToken, setTokens } from '@/lib/auth-storage'

export async function loginRequest(
  body: LoginRequest
): Promise<LoginResponseData> {
  const res = await api.post('/auth/login', body)
  const { data } = unwrapEnvelope<LoginResponseData>(res)
  setTokens(data.access_token, data.refresh_token)
  return data
}

export type RegisterRequest = {
  email: string
  password: string
  name: string
  workspace_name: string
}

export async function registerRequest(
  body: RegisterRequest
): Promise<LoginResponseData> {
  const res = await api.post('/auth/register', body)
  const { data } = unwrapEnvelope<LoginResponseData>(res)
  setTokens(data.access_token, data.refresh_token)
  return data
}

export async function logoutRequest(): Promise<void> {
  try {
    const rt = getRefreshToken()
    await api.post('/auth/logout', { refresh_token: rt ?? '' })
  } catch {
    /* ignorar falha de rede no logout */
  }
}
