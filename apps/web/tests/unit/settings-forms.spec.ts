import { describe, expect, it } from 'vitest'
import { createProviderForm, providerFormToConfig } from '../../features/provider-configure/provider-form'
import { createModelForm, modelFormToConfig } from '../../features/model-configure/model-form'
import { agentFormToConfig, createAgentForm, createMCPForm, mcpFormToConfig } from '../../features/agent-configure/resource-forms'

describe('settings contract forms', () => {
  it('提供商表单保留 streaming、trace、timeout、default_model_id 和 metadata', () => {
    const form = createProviderForm()
    Object.assign(form, { id: 'p1', name: 'Provider', base_url: 'https://example.test', streaming: false, trace_enabled: true, trace_retention_days: '14', default_request_timeout_sec: '60', default_model_id: 'p1:m1', metadataText: '{"region":"us"}' })
    expect(providerFormToConfig(form)).toMatchObject({ id: 'p1', streaming: false, trace_enabled: true, trace_retention_days: 14, default_request_timeout_sec: 60, default_model_id: 'p1:m1', metadata: { region: 'us' } })
  })

  it('模型表单保留成本、能力、权重和角色限制', () => {
    const form = createModelForm(undefined, 'p1')
    Object.assign(form, { id: 'p1:m1', name: 'm1', display_name: 'Model', cost_input_per_mtok: '1.5', cost_output_per_mtok: '3', supports_tools: true, routing_weight: '80', allowed_agent_roles: ['writer'] })
    expect(modelFormToConfig(form)).toMatchObject({ provider_id: 'p1', cost_input_per_mtok: 1.5, cost_output_per_mtok: 3, supports_tools: true, routing_weight: 80, allowed_agent_roles: ['writer'] })
  })

  it('Agent JSON 策略与绑定会进入真实请求对象', () => {
    const form = createAgentForm()
    Object.assign(form, { name: 'Writer', skillIdsText: 's1\ns2', memoryPolicyText: '{"scope":"project"}', runtimeOptionsText: '{"temperature":0.3}' })
    expect(agentFormToConfig(form)).toMatchObject({ name: 'Writer', skill_ids: ['s1', 's2'], memory_policy: { scope: 'project' }, runtime_options: { temperature: 0.3 } })
  })

  it('MCP transport 缺少必需端点时失败', () => {
    const form = createMCPForm()
    form.name = 'Server'
    form.transport = 'streamable_http'
    form.url = ''
    expect(() => mcpFormToConfig(form)).toThrow(/url/i)
  })
})
