import axios, { type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'
import {
  ApiEnvelopeError,
  type ApiErrorEnvelope,
  type ApiSuccessEnvelope,
} from '@/types/api'
import {
  clearAuthStorage,
  getAccessToken,
  getRefreshToken,
  setTokens,
} from '@/lib/auth-storage'

const baseURL =
  import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'

export const api = axios.create({
  baseURL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: import.meta.env.VITE_API_WITH_CREDENTIALS === 'true',
})

const refreshClient = axios.create({
  baseURL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: import.meta.env.VITE_API_WITH_CREDENTIALS === 'true',
})

let isRefreshing = false
let refreshWaitQueue: Array<(token: string | null) => void> = []

function flushQueue(token: string | null) {
  refreshWaitQueue.forEach((cb) => cb(token))
  refreshWaitQueue = []
}

function isAuthPath(url?: string): boolean {
  if (!url) return false
  return (
    url.includes('/auth/login') ||
    url.includes('/auth/refresh') ||
    url.includes('/auth/forgot-password') ||
    url.includes('/auth/reset-password')
  )
}

api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response: AxiosResponse) => {
    if (
      response.config.responseType === 'blob' ||
      (typeof Blob !== 'undefined' && response.data instanceof Blob)
    ) {
      return response
    }
    const body = response.data as
      | ApiSuccessEnvelope<unknown>
      | ApiErrorEnvelope
      | unknown
    if (
      body &&
      typeof body === 'object' &&
      'error' in body &&
      (body as ApiErrorEnvelope).error &&
      typeof (body as ApiErrorEnvelope).error === 'object'
    ) {
      return Promise.reject(
        new ApiEnvelopeError((body as ApiErrorEnvelope).error)
      )
    }
    return response
  },
  async (error) => {
    const original = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean
    }
    const status = error.response?.status
    const resData = error.response?.data

    if (
      status === 401 &&
      original &&
      !original._retry &&
      !isAuthPath(original.url)
    ) {
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          refreshWaitQueue.push((token) => {
            if (!token) {
              reject(error)
              return
            }
            original.headers.Authorization = `Bearer ${token}`
            original._retry = true
            resolve(api(original))
          })
        })
      }

      original._retry = true
      isRefreshing = true

      try {
        const rt = getRefreshToken()
        if (!rt) {
          throw new Error('Sem refresh token')
        }
        const res = await refreshClient.post<
          ApiSuccessEnvelope<{
            access_token: string
            refresh_token?: string
          }>
        >('/auth/refresh', { refresh_token: rt })

        const raw = res.data as
          | ApiSuccessEnvelope<{
              access_token: string
              refresh_token?: string
            }>
          | ApiErrorEnvelope
        if ('error' in raw && raw.error) {
          throw new ApiEnvelopeError(raw.error)
        }
        if (!('data' in raw)) {
          throw new Error('Resposta de refresh inválida')
        }
        const inner = raw.data
        if (!inner?.access_token) {
          throw new Error('Resposta de refresh inválida')
        }
        setTokens(inner.access_token, inner.refresh_token ?? rt)
        flushQueue(inner.access_token)
        original.headers.Authorization = `Bearer ${inner.access_token}`
        return api(original)
      } catch {
        clearAuthStorage()
        flushQueue(null)
        if (typeof window !== 'undefined') {
          window.location.href = '/login'
        }
        return Promise.reject(error)
      } finally {
        isRefreshing = false
      }
    }

    if (
      resData &&
      typeof resData === 'object' &&
      'error' in resData &&
      (resData as ApiErrorEnvelope).error
    ) {
      return Promise.reject(
        new ApiEnvelopeError((resData as ApiErrorEnvelope).error)
      )
    }

    return Promise.reject(error)
  }
)

export function unwrapEnvelope<T>(response: AxiosResponse<unknown>): {
  data: T
  meta?: ApiSuccessEnvelope<T>['meta']
} {
  const body = response.data as ApiSuccessEnvelope<T>
  return { data: body.data, meta: body.meta }
}

/** Multipart sem Content-Type fixo — o axios define boundary automaticamente. */
export async function postMultipart<T>(
  path: string,
  formData: FormData,
  onProgress?: (percent: number) => void,
): Promise<{ data: T }> {
  const res = await api.post<unknown>(path, formData, {
    transformRequest: [
      (data, headers) => {
        delete (headers as Record<string, unknown>)['Content-Type']
        return data as FormData
      },
    ],
    onUploadProgress:
      onProgress &&
      ((e) => {
        if (e.total && e.total > 0) {
          onProgress(Math.round((e.loaded * 100) / e.total))
        }
      }),
  })
  return unwrapEnvelope<T>(res)
}
