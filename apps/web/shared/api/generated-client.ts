import { client as generatedClient } from '~/lib/generated/api/client.gen'
import type * as GeneratedApi from '~/lib/generated/api/types.gen'
import { ApiClientError, toApiErrorState } from './error'
import type { ApiErrorState, ApiResult } from './types'
import { isRecord } from './validation'

export const DEFAULT_API_BASE = 'http://localhost:8080/api/v1'

type V1ErrorPayload = Partial<GeneratedApi.ApiError>

type ApiEnvelope<T = unknown> = {
  data?: T
  meta?: GeneratedApi.Meta
  page?: GeneratedApi.Page
}

type GeneratedFieldsResult = {
  data?: unknown
  error?: unknown
  request?: unknown
  response?: unknown
}

export function normalizeApiBase(baseUrl?: string): string {
  const trimmed = (baseUrl || DEFAULT_API_BASE).trim().replace(/\/+$/, '')
  if (!trimmed) return DEFAULT_API_BASE
  if (/\/api$/i.test(trimmed)) return `${trimmed}/v1`
  if (/\/api\/v1$/i.test(trimmed) || /\/v1$/i.test(trimmed)) return trimmed

  try {
    const parsed = new URL(trimmed)
    if (parsed.pathname === '' || parsed.pathname === '/') {
      parsed.pathname = '/api/v1'
      return parsed.toString().replace(/\/+$/, '')
    }
  } catch (error) {
    console.warn('[AeonEchoes API] Using a custom API base without /api/v1', { baseUrl: trimmed, error })
  }

  return trimmed
}

function isV1ErrorPayload(value: unknown): value is V1ErrorPayload {
  if (!isRecord(value)) return false
  return typeof value.message === 'string' || typeof value.code === 'string' || typeof value.status === 'number'
}

function errorMessageFromV1(endpoint: string, status: number, error: V1ErrorPayload): ApiErrorState {
  const code = error.code || 'request_error'
  const responseStatus = error.status || status
  const requestId = error.request_id
  return {
    endpoint,
    status: responseStatus,
    code,
    kind: 'response',
    requestId,
    message: `${code} (${responseStatus}): ${error.message || 'request failed'}${requestId ? ` request_id=${requestId}` : ''}`,
    cause: error.details
  }
}

function createGeneratedErrorState(endpoint: string, cause: unknown): ApiErrorState {
  if (cause instanceof ApiClientError) return cause.state
  if (isRecord(cause) && isV1ErrorPayload(cause.error)) {
    return errorMessageFromV1(endpoint, cause.error.status || 0, cause.error)
  }
  if (isV1ErrorPayload(cause)) return errorMessageFromV1(endpoint, cause.status || 0, cause)
  return toApiErrorState(endpoint, cause)
}

function isEnvelope(value: unknown): value is ApiEnvelope {
  return isRecord(value)
    && Object.prototype.hasOwnProperty.call(value, 'data')
    && (Object.prototype.hasOwnProperty.call(value, 'meta') || Object.prototype.hasOwnProperty.call(value, 'page'))
}

function unwrapGeneratedResult(endpoint: string, result: unknown): ApiEnvelope {
  if (isRecord(result) && Object.prototype.hasOwnProperty.call(result, 'error') && result.error !== undefined) {
    throw new ApiClientError(createGeneratedErrorState(endpoint, result.error))
  }
  if (isEnvelope(result)) return result

  const fields = result as GeneratedFieldsResult
  if (isRecord(fields) && isEnvelope(fields.data)) return fields.data

  throw new ApiClientError({
    endpoint,
    kind: 'validation',
    code: 'invalid_v1_envelope',
    message: `invalid_v1_envelope: expected an API envelope from ${endpoint}`,
    cause: result
  })
}

function unwrapEnvelopeData(endpoint: string, envelope: ApiEnvelope): unknown {
  if (!Object.prototype.hasOwnProperty.call(envelope, 'data') || envelope.data === undefined || envelope.data === null) {
    throw new ApiClientError({
      endpoint,
      kind: 'validation',
      code: 'invalid_v1_envelope',
      message: `invalid_v1_envelope: missing data from ${endpoint}`,
      cause: envelope
    })
  }
  return envelope.data
}

export function configureGeneratedClient(rawBaseUrl: string): string {
  const baseUrl = normalizeApiBase(rawBaseUrl)
  generatedClient.setConfig({ baseUrl, throwOnError: true })
  return baseUrl
}

export async function callGeneratedApi<T>(
  endpoint: string,
  operation: () => Promise<unknown>,
  decode: (data: unknown) => T
): Promise<ApiResult<T>> {
  try {
    const data = unwrapEnvelopeData(endpoint, unwrapGeneratedResult(endpoint, await operation()))
    return { data: decode(data) }
  } catch (cause) {
    const error = createGeneratedErrorState(endpoint, cause)
    console.error(`[AeonEchoes API] ${error.message}`, error)
    throw new ApiClientError(error)
  }
}
