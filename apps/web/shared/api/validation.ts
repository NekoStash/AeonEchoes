import { apiValidationError } from './error'

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

export function requireApiRecord(value: unknown, endpoint: string, field = 'data'): Record<string, unknown> {
  if (!isRecord(value)) throw apiValidationError(endpoint, field, undefined, value)
  return value
}

export function requireApiString(value: unknown, endpoint: string, field: string, options: { allowEmpty?: boolean } = {}): string {
  if (typeof value !== 'string' || (!options.allowEmpty && !value.trim())) {
    throw apiValidationError(endpoint, field, undefined, value)
  }
  return value
}

export function requireApiNumber(value: unknown, endpoint: string, field: string): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    throw apiValidationError(endpoint, field, undefined, value)
  }
  return value
}

export function requireApiBoolean(value: unknown, endpoint: string, field: string): boolean {
  if (typeof value !== 'boolean') throw apiValidationError(endpoint, field, undefined, value)
  return value
}

export function requireApiArray<T>(
  value: unknown,
  endpoint: string,
  field: string,
  decode: (item: unknown, index: number) => T
): T[] {
  if (!Array.isArray(value)) throw apiValidationError(endpoint, field, undefined, value)
  return value.map(decode)
}

export function optionalApiArray<T>(
  value: unknown,
  endpoint: string,
  field: string,
  decode: (item: unknown, index: number) => T
): T[] {
  if (value === undefined || value === null) return []
  return requireApiArray(value, endpoint, field, decode)
}

export function optionalStringRecord(value: unknown, endpoint: string, field: string): Record<string, string> | undefined {
  if (value === undefined || value === null) return undefined
  const record = requireApiRecord(value, endpoint, field)
  const entries = Object.entries(record)
  if (entries.some(([, item]) => typeof item !== 'string')) {
    throw apiValidationError(endpoint, field, `${field} must contain only string values`, value)
  }
  return Object.fromEntries(entries) as Record<string, string>
}
