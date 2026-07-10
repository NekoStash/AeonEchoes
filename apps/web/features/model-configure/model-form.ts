import type { AgentRole, ModelConfig, ModelKind } from '~/lib/types'

export interface ModelFormState {
  id: string
  provider_id: string
  name: string
  display_name: string
  kind: ModelKind
  context_window: string
  max_output_tokens: string
  dimension: string
  supports_tools: boolean
  supports_streaming: boolean
  default_for_kind: boolean
  enabled: boolean
  cost_input_per_mtok: string
  cost_output_per_mtok: string
  routing_weight: string
  allowed_agent_roles: AgentRole[]
  metadataText: string
  created_at?: string
}

export const configurableAgentRoles: AgentRole[] = [
  'writer', 'editor', 'genesis-optimizer', 'plot-architect', 'world-builder', 'character-keeper', 'continuity-auditor', 'fact-extractor', 'graph-curator'
]

export function createModelForm(model?: ModelConfig, providerId = ''): ModelFormState {
  return {
    id: model?.id || '',
    provider_id: model?.provider_id || providerId,
    name: model?.name || '',
    display_name: model?.display_name || '',
    kind: model?.kind || 'text',
    context_window: String(model?.context_window ?? 0),
    max_output_tokens: String(model?.max_output_tokens ?? 0),
    dimension: String(model?.dimension ?? 0),
    supports_tools: model?.supports_tools ?? false,
    supports_streaming: model?.supports_streaming ?? false,
    default_for_kind: model?.default_for_kind ?? false,
    enabled: model?.enabled ?? true,
    cost_input_per_mtok: String(model?.cost_input_per_mtok ?? 0),
    cost_output_per_mtok: String(model?.cost_output_per_mtok ?? 0),
    routing_weight: String(model?.routing_weight ?? 100),
    allowed_agent_roles: [...(model?.allowed_agent_roles || [])],
    metadataText: model?.metadata && Object.keys(model.metadata).length ? JSON.stringify(model.metadata, null, 2) : '',
    created_at: model?.created_at
  }
}

export function modelFormToConfig(form: ModelFormState, original?: ModelConfig): ModelConfig {
  const providerId = required(form.provider_id, 'provider_id')
  const name = required(form.name, 'name')
  const displayName = required(form.display_name || form.name, 'display_name')
  const metadata = parseMetadata(form.metadataText)
  const config: ModelConfig = {
    ...(original || {}),
    id: form.id.trim(),
    provider_id: providerId,
    name,
    display_name: displayName,
    kind: form.kind,
    context_window: numberField(form.context_window, 'context_window'),
    max_output_tokens: numberField(form.max_output_tokens, 'max_output_tokens'),
    dimension: numberField(form.dimension, 'dimension'),
    supports_tools: form.supports_tools,
    supports_streaming: form.supports_streaming,
    default_for_kind: form.default_for_kind,
    enabled: form.enabled,
    cost_input_per_mtok: numberField(form.cost_input_per_mtok, 'cost_input_per_mtok'),
    cost_output_per_mtok: numberField(form.cost_output_per_mtok, 'cost_output_per_mtok'),
    routing_weight: numberField(form.routing_weight, 'routing_weight'),
    allowed_agent_roles: [...form.allowed_agent_roles],
    metadata,
    created_at: original?.created_at
  }
  return config
}

export function toggleRole(form: ModelFormState, role: AgentRole) {
  form.allowed_agent_roles = form.allowed_agent_roles.includes(role)
    ? form.allowed_agent_roles.filter((item) => item !== role)
    : [...form.allowed_agent_roles, role]
}

function required(value: string, field: string) {
  const normalized = value.trim()
  if (!normalized) throw validationError(field, `${field} is required.`)
  return normalized
}

function numberField(value: string, field: string) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed < 0) throw validationError(field, `${field} must be a non-negative number.`)
  return parsed
}

function parseMetadata(value: string) {
  if (!value.trim()) return undefined
  let parsed: unknown
  try {
    parsed = JSON.parse(value)
  } catch (error) {
    console.error('[model-configure] Invalid metadata JSON.', error)
    throw validationError('metadata', 'metadata must be valid JSON.')
  }
  if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object' || Object.values(parsed).some((item) => typeof item !== 'string')) {
    throw validationError('metadata', 'metadata must be a string-to-string JSON object.')
  }
  return parsed as Record<string, string>
}

function validationError(field: string, message: string) {
  const error = new Error(message)
  Object.assign(error, { field })
  console.error('[model-configure] Validation failed.', { field, message })
  return error
}
