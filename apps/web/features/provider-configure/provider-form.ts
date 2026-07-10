import type { ProviderConfig, ProviderType } from '~/lib/types'

export interface ProviderFormState {
  id: string
  name: string
  provider_type: ProviderType
  base_url: string
  api_key: string
  enabled: boolean
  streaming: boolean
  trace_enabled: boolean
  trace_retention_days: string
  default_request_timeout_sec: string
  default_model_id: string
  metadataText: string
  created_at?: string
}

export function createProviderForm(provider?: ProviderConfig): ProviderFormState {
  return {
    id: provider?.id || '',
    name: provider?.name || '',
    provider_type: provider?.provider_type || 'openai-responses',
    base_url: provider?.base_url || '',
    api_key: '',
    enabled: provider?.enabled ?? true,
    streaming: provider?.streaming ?? true,
    trace_enabled: provider?.trace_enabled ?? false,
    trace_retention_days: String(provider?.trace_retention_days ?? 7),
    default_request_timeout_sec: String(provider?.default_request_timeout_sec ?? 120),
    default_model_id: provider?.default_model_id || '',
    metadataText: stringifyStringRecord(provider?.metadata),
    created_at: provider?.created_at
  }
}

export function providerFormToConfig(form: ProviderFormState, original?: ProviderConfig): ProviderConfig {
  const name = form.name.trim()
  if (!name) throw validationError('name', 'Provider name is required.')
  const baseUrl = form.base_url.trim()
  if (!baseUrl) throw validationError('base_url', 'Provider base URL is required.')
  const timeout = parseNonNegativeNumber(form.default_request_timeout_sec, 'default_request_timeout_sec')
  const retention = parseNonNegativeNumber(form.trace_retention_days, 'trace_retention_days')
  const metadata = parseStringRecord(form.metadataText, 'metadata')

  return {
    ...(original || {}),
    id: form.id.trim(),
    name,
    provider_type: form.provider_type,
    type: form.provider_type,
    base_url: baseUrl,
    api_key: form.api_key.trim() || undefined,
    enabled: form.enabled,
    streaming: form.streaming,
    trace_enabled: form.trace_enabled,
    trace_retention_days: retention,
    default_request_timeout_sec: timeout,
    default_model_id: form.default_model_id.trim() || undefined,
    metadata,
    status: original?.status || 'unknown',
    created_at: original?.created_at
  }
}

export function parseStringRecord(value: string, field: string): Record<string, string> | undefined {
  const normalized = value.trim()
  if (!normalized) return undefined
  let parsed: unknown
  try {
    parsed = JSON.parse(normalized)
  } catch (error) {
    console.error(`[provider-configure] Invalid ${field} JSON.`, error)
    throw validationError(field, `${field} must be valid JSON.`)
  }
  if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') throw validationError(field, `${field} must be a JSON object.`)
  const entries = Object.entries(parsed)
  if (entries.some(([, item]) => typeof item !== 'string')) throw validationError(field, `${field} values must be strings.`)
  return Object.fromEntries(entries) as Record<string, string>
}

function stringifyStringRecord(value?: Record<string, string>) {
  return value && Object.keys(value).length ? JSON.stringify(value, null, 2) : ''
}

function parseNonNegativeNumber(value: string, field: string) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed < 0) throw validationError(field, `${field} must be a non-negative number.`)
  return parsed
}

function validationError(field: string, message: string) {
  const error = new Error(message)
  Object.assign(error, { field })
  console.error('[provider-configure] Validation failed.', { field, message })
  return error
}
