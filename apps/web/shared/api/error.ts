import type { ApiErrorState } from './types'

export class ApiClientError extends Error {
  readonly state: ApiErrorState

  constructor(state: ApiErrorState) {
    super(state.message)
    this.name = 'ApiClientError'
    this.state = state
  }
}

export function apiValidationError(endpoint: string, field: string, message?: string, cause?: unknown): ApiClientError {
  return new ApiClientError({
    endpoint,
    field,
    kind: 'validation',
    code: 'invalid_api_response',
    message: message || `invalid_api_response: ${endpoint} is missing required field ${field}`,
    cause
  })
}

export function toApiErrorState(endpoint: string, cause: unknown): ApiErrorState {
  if (cause instanceof ApiClientError) return cause.state
  if (cause instanceof Error) {
    return {
      endpoint,
      kind: 'transport',
      code: 'request_error',
      message: cause.message,
      cause
    }
  }
  return {
    endpoint,
    kind: 'transport',
    code: 'request_error',
    message: `API request failed for ${endpoint}`,
    cause
  }
}
