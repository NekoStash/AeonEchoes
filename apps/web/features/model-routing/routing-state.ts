import type { AgentRole, ModelConfig, ModelUsageKey, ModelUsageSettings } from '~/lib/types'

export const ROUTING_KEYS: readonly ModelUsageKey[] = [
  'writer',
  'editor',
  'genesis-optimizer',
  'plot-architect',
  'world-builder',
  'character-keeper',
  'continuity-auditor',
  'fact-extractor',
  'graph-curator',
  'embedding'
]

export type RoutingEligibilityReason = 'unknown' | 'disabled' | 'kind' | 'role'

export interface RoutingEligibility {
  eligible: boolean
  reason?: RoutingEligibilityReason
}

export interface RoutingOptionState {
  value: string
  model?: ModelConfig
  disabled: boolean
  reason?: RoutingEligibilityReason
}

export interface RoutingValidationError {
  key: ModelUsageKey
  value: string
  reason: RoutingEligibilityReason
  model?: ModelConfig
}

export interface RoutingReferences {
  baseline: ModelUsageKey[]
  draft: ModelUsageKey[]
  all: ModelUsageKey[]
}

export function cloneRouting(settings: Partial<ModelUsageSettings> = {}): ModelUsageSettings {
  return ROUTING_KEYS.reduce<ModelUsageSettings>((result, key) => {
    result[key] = typeof settings[key] === 'string' ? settings[key] : ''
    return result
  }, {} as ModelUsageSettings)
}

export function isRoutingDirty(baseline: ModelUsageSettings, draft: ModelUsageSettings): boolean {
  return ROUTING_KEYS.some((key) => baseline[key] !== draft[key])
}

export function qualifiedModelId(model: ModelConfig): string {
  return model.id.includes(':') ? model.id : `${model.provider_id}:${model.name}`
}

export function getRoutingEligibility(model: ModelConfig | undefined, key: ModelUsageKey): RoutingEligibility {
  if (!model) return { eligible: false, reason: 'unknown' }
  if (!model.enabled) return { eligible: false, reason: 'disabled' }

  if (key === 'embedding') {
    return model.kind === 'embedding'
      ? { eligible: true }
      : { eligible: false, reason: 'kind' }
  }

  if (model.kind !== 'text') return { eligible: false, reason: 'kind' }
  const allowedRoles = model.allowed_agent_roles || []
  return allowedRoles.length === 0 || allowedRoles.includes(key as AgentRole)
    ? { eligible: true }
    : { eligible: false, reason: 'role' }
}

export function buildRoutingOptions(models: ModelConfig[], key: ModelUsageKey, currentValue = ''): RoutingOptionState[] {
  const options: RoutingOptionState[] = models.map((model) => {
    const eligibility = getRoutingEligibility(model, key)
    return {
      value: qualifiedModelId(model),
      model,
      disabled: !eligibility.eligible,
      reason: eligibility.reason
    }
  })

  if (currentValue && !options.some((option) => option.value === currentValue)) {
    options.unshift({ value: currentValue, disabled: true, reason: 'unknown' })
  }

  return options.sort((left, right) => {
    if (left.value === currentValue) return -1
    if (right.value === currentValue) return 1
    if (left.disabled !== right.disabled) return left.disabled ? 1 : -1
    const leftName = left.model?.display_name || left.model?.name || left.value
    const rightName = right.model?.display_name || right.model?.name || right.value
    return leftName.localeCompare(rightName)
  })
}

export function validateRoutingValue(models: ModelConfig[], key: ModelUsageKey, value: string): RoutingValidationError | null {
  if (!value) return null
  const model = models.find((item) => qualifiedModelId(item) === value)
  const eligibility = getRoutingEligibility(model, key)
  return eligibility.eligible
    ? null
    : { key, value, reason: eligibility.reason || 'unknown', model }
}

export function validateRouting(settings: ModelUsageSettings, models: ModelConfig[]): RoutingValidationError[] {
  return ROUTING_KEYS.flatMap((key) => {
    const error = validateRoutingValue(models, key, settings[key])
    return error ? [error] : []
  })
}

export function findRoutingReferences(model: ModelConfig, baseline: ModelUsageSettings, draft: ModelUsageSettings): RoutingReferences {
  const modelId = qualifiedModelId(model)
  const baselineReferences = ROUTING_KEYS.filter((key) => baseline[key] === modelId)
  const draftReferences = ROUTING_KEYS.filter((key) => draft[key] === modelId)
  return {
    baseline: baselineReferences,
    draft: draftReferences,
    all: ROUTING_KEYS.filter((key) => baselineReferences.includes(key) || draftReferences.includes(key))
  }
}
