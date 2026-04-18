/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
  readonly VITE_WS_URL?: string
  readonly VITE_ENABLE_AUTH_MOCK?: string
  readonly VITE_API_WITH_CREDENTIALS?: string
  readonly VITE_SENTRY_DSN?: string
  readonly VITE_PRIVACY_URL?: string
  readonly VITE_TERMS_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
