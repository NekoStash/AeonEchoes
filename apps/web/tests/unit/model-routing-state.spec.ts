import { describe, expect, it } from 'vitest'
import {
  buildRoutingOptions,
  cloneRouting,
  findRoutingReferences,
  getRoutingEligibility,
  isRoutingDirty,
  qualifiedModelId,
  validateRoutingValue
} from '../../features/model-routing/routing-state'
import type { ModelConfig } from '../../lib/types'

function model(overrides: Partial<ModelConfig> = {}): ModelConfig {
  return {
    id: 'provider:text-model',
    provider_id: 'provider',
    name: 'text-model',
    display_name: 'Text Model',
    kind: 'text',
    enabled: true,
    allowed_agent_roles: [],
    ...overrides
  }
}

describe('model routing state', () => {
  it('Agent 角色只接受已启用文本模型，空角色列表表示全部角色可用', () => {
    expect(getRoutingEligibility(model(), 'writer')).toEqual({ eligible: true })
    expect(getRoutingEligibility(model({ kind: 'embedding' }), 'writer')).toMatchObject({ eligible: false, reason: 'kind' })
    expect(getRoutingEligibility(model({ enabled: false }), 'writer')).toMatchObject({ eligible: false, reason: 'disabled' })
  })

  it('受限文本模型只允许声明的 Agent 角色', () => {
    const restricted = model({ allowed_agent_roles: ['editor'] })
    expect(getRoutingEligibility(restricted, 'editor')).toEqual({ eligible: true })
    expect(getRoutingEligibility(restricted, 'writer')).toMatchObject({ eligible: false, reason: 'role' })
  })

  it('embedding 路由只接受已启用向量模型', () => {
    expect(getRoutingEligibility(model({ id: 'provider:embedding', name: 'embedding', kind: 'embedding' }), 'embedding')).toEqual({ eligible: true })
    expect(getRoutingEligibility(model(), 'embedding')).toMatchObject({ eligible: false, reason: 'kind' })
    expect(getRoutingEligibility(model({ kind: 'embedding', enabled: false }), 'embedding')).toMatchObject({ eligible: false, reason: 'disabled' })
  })

  it('区分未知、停用、类型不匹配和角色不允许的无效路由', () => {
    const models = [
      model(),
      model({ id: 'provider:disabled', name: 'disabled', enabled: false }),
      model({ id: 'provider:embedding', name: 'embedding', kind: 'embedding' }),
      model({ id: 'provider:restricted', name: 'restricted', allowed_agent_roles: ['editor'] })
    ]
    expect(validateRoutingValue(models, 'writer', 'provider:missing')?.reason).toBe('unknown')
    expect(validateRoutingValue(models, 'writer', 'provider:disabled')?.reason).toBe('disabled')
    expect(validateRoutingValue(models, 'writer', 'provider:embedding')?.reason).toBe('kind')
    expect(validateRoutingValue(models, 'writer', 'provider:restricted')?.reason).toBe('role')
    expect(validateRoutingValue(models, 'writer', '')).toBeNull()
  })

  it('路由选项保留当前无效值但禁止选择，并禁用其他无资格模型', () => {
    const models = [
      model(),
      model({ id: 'provider:restricted', name: 'restricted', allowed_agent_roles: ['editor'] })
    ]
    const options = buildRoutingOptions(models, 'writer', 'provider:missing')
    expect(options[0]).toMatchObject({ value: 'provider:missing', disabled: true, reason: 'unknown' })
    expect(options.find((option) => option.value === 'provider:text-model')).toMatchObject({ disabled: false })
    expect(options.find((option) => option.value === 'provider:restricted')).toMatchObject({ disabled: true, reason: 'role' })
  })

  it('克隆固定完整路由键并正确比较草稿脏状态', () => {
    const baseline = cloneRouting({ writer: 'provider:text-model' })
    const draft = cloneRouting(baseline)
    expect(Object.keys(baseline)).toHaveLength(10)
    expect(isRoutingDirty(baseline, draft)).toBe(false)
    draft.embedding = 'provider:embedding'
    expect(isRoutingDirty(baseline, draft)).toBe(true)
  })

  it('同时返回已保存与草稿对模型的路由引用', () => {
    const target = model()
    const baseline = cloneRouting({ writer: qualifiedModelId(target), editor: qualifiedModelId(target) })
    const draft = cloneRouting({ writer: qualifiedModelId(target), embedding: qualifiedModelId(target) })
    expect(findRoutingReferences(target, baseline, draft)).toEqual({
      baseline: ['writer', 'editor'],
      draft: ['writer', 'embedding'],
      all: ['writer', 'editor', 'embedding']
    })
  })
})
