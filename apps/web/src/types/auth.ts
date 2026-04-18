import type { UserRole } from './user'

export type LoginRequest = {
  email: string
  password: string
}

export type AuthTokens = {
  access_token: string
  refresh_token?: string
  expires_in?: number
}

export type AuthUser = {
  id: string
  email: string
  name?: string
  role: UserRole
}

export type LoginResponseData = AuthTokens & {
  user: AuthUser
  workspace_id: string
  workspace_name?: string
}
