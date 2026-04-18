/** Envelope REST alinhado ao playbook (saas-whatsapp-playbook.md, seção 6). */

export type ApiMeta = {
  cursor?: string
  has_more?: boolean
}

export type ApiSuccessEnvelope<T> = {
  data: T
  meta?: ApiMeta
}

export type ApiErrorBody = {
  code: string
  message: string
  details?: Array<{ field?: string; message: string }>
}

export type ApiErrorEnvelope = {
  error: ApiErrorBody
}

export class ApiEnvelopeError extends Error {
  readonly code: string
  readonly details?: ApiErrorBody['details']

  constructor(body: ApiErrorBody) {
    super(body.message)
    this.name = 'ApiEnvelopeError'
    this.code = body.code
    this.details = body.details
  }
}
