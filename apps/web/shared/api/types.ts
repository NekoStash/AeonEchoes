export type ApiErrorKind = 'transport' | 'response' | 'validation'

export interface ApiErrorState {
  message: string
  endpoint: string
  status?: number
  code?: string
  kind?: ApiErrorKind
  field?: string
  requestId?: string
  cause?: unknown
}

export interface ApiResult<T> {
  data: T
  error?: ApiErrorState
}

export interface ApiRequestState {
  loading: boolean
  error: ApiErrorState | null
}
