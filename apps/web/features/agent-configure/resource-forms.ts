import type { AgentConfig, AgentRole, MCPServerConfig, Skill } from '~/lib/types'

export interface AgentFormState {
  id: string
  project_id: string
  name: string
  description: string
  role: AgentRole
  model_id: string
  enabled: boolean
  system_prompt: string
  skillIdsText: string
  toolIdsText: string
  mcpServerIdsText: string
  memoryPolicyText: string
  runtimeOptionsText: string
  metadataText: string
  created_at?: string
}

export interface SkillFormState {
  id: string
  project_id: string
  source_id: string
  name: string
  description: string
  content: string
  path: string
  enabled: boolean
  metadataText: string
  created_at?: string
}

export interface MCPFormState {
  id: string
  project_id: string
  name: string
  transport: string
  enabled: boolean
  command: string
  argsText: string
  url: string
  headersText: string
  secretHeadersText: string
  envText: string
  secretEnvText: string
  timeoutSec: string
  metadataText: string
  created_at?: string
}

export function createAgentForm(agent?: AgentConfig): AgentFormState {
  return {
    id: agent?.id || '', project_id: agent?.project_id || '', name: agent?.name || '', description: agent?.description || '', role: agent?.role || 'writer', model_id: agent?.model_id || '', enabled: agent?.enabled ?? true, system_prompt: agent?.system_prompt || '',
    skillIdsText: joinIds(agent?.skill_ids), toolIdsText: joinIds(agent?.tool_ids), mcpServerIdsText: joinIds(agent?.mcp_server_ids), memoryPolicyText: stringify(agent?.memory_policy), runtimeOptionsText: stringify(agent?.runtime_options), metadataText: stringify(agent?.metadata), created_at: agent?.created_at
  }
}

export function agentFormToConfig(form: AgentFormState, original?: AgentConfig): AgentConfig {
  return {
    ...(original || {}), id: form.id.trim(), project_id: optional(form.project_id), name: required(form.name, 'name'), description: optional(form.description), role: form.role, model_id: optional(form.model_id), enabled: form.enabled, system_prompt: optional(form.system_prompt),
    skill_ids: parseIds(form.skillIdsText), tool_ids: parseIds(form.toolIdsText), mcp_server_ids: parseIds(form.mcpServerIdsText), memory_policy: parseObject(form.memoryPolicyText, 'memory_policy'), runtime_options: parseObject(form.runtimeOptionsText, 'runtime_options'), metadata: parseStringObject(form.metadataText, 'metadata'), created_at: original?.created_at
  }
}

export function createSkillForm(skill?: Skill): SkillFormState {
  return { id: skill?.id || '', project_id: skill?.project_id || '', source_id: skill?.source_id || '', name: skill?.name || '', description: skill?.description || '', content: skill?.content || '', path: skill?.path || '', enabled: skill?.enabled ?? true, metadataText: stringify(skill?.metadata), created_at: skill?.created_at }
}

export function skillFormToConfig(form: SkillFormState, original?: Skill): Skill {
  return { ...(original || {}), id: form.id.trim(), project_id: optional(form.project_id), source_id: optional(form.source_id) || '', name: required(form.name, 'name'), description: optional(form.description), content: optional(form.content), path: optional(form.path), enabled: form.enabled, metadata: parseStringObject(form.metadataText, 'metadata'), created_at: original?.created_at }
}

export function createMCPForm(server?: MCPServerConfig): MCPFormState {
  return { id: server?.id || '', project_id: server?.project_id || '', name: server?.name || '', transport: server?.transport || 'stdio', enabled: server?.enabled ?? true, command: server?.command || '', argsText: joinIds(server?.args), url: server?.url || '', headersText: stringify(server?.headers), secretHeadersText: '', envText: stringify(server?.env), secretEnvText: '', timeoutSec: String(server?.timeout_sec ?? 30), metadataText: stringify(server?.metadata), created_at: server?.created_at }
}

export function mcpFormToConfig(form: MCPFormState, original?: MCPServerConfig): MCPServerConfig {
  const command = optional(form.command)
  const url = optional(form.url)
  if (form.transport === 'stdio' && !command) throw validationError('command', 'command is required for stdio.')
  if (form.transport !== 'stdio' && !url) throw validationError('url', 'url is required for HTTP/SSE.')
  const timeout = Number(form.timeoutSec)
  if (!Number.isFinite(timeout) || timeout < 0) throw validationError('timeout_sec', 'timeout_sec must be a non-negative number.')
  return {
    ...(original || {}), id: form.id.trim(), project_id: optional(form.project_id), name: required(form.name, 'name'), transport: form.transport, status: original?.status || 'unknown', enabled: form.enabled, command, args: parseIds(form.argsText), url,
    headers: parseStringObject(form.headersText, 'headers'), secret_headers: parseStringObject(form.secretHeadersText, 'secret_headers'), env: parseStringObject(form.envText, 'env'), secret_env: parseStringObject(form.secretEnvText, 'secret_env'), timeout_sec: timeout, metadata: parseStringObject(form.metadataText, 'metadata'), created_at: original?.created_at
  }
}

export function parseIds(value: string): string[] {
  return Array.from(new Set(value.split(/[\n,，]+/u).map((item) => item.trim()).filter(Boolean)))
}

function joinIds(values?: string[]) { return (values || []).join('\n') }
function optional(value: string) { return value.trim() || undefined }
function required(value: string, field: string) { const normalized = value.trim(); if (!normalized) throw validationError(field, `${field} is required.`); return normalized }
function stringify(value?: Record<string, unknown>) { return value && Object.keys(value).length ? JSON.stringify(value, null, 2) : '' }

function parseObject(value: string, field: string): Record<string, unknown> | undefined {
  const parsed = parseJsonObject(value, field)
  return parsed as Record<string, unknown> | undefined
}

function parseStringObject(value: string, field: string): Record<string, string> | undefined {
  const parsed = parseJsonObject(value, field)
  if (parsed && Object.values(parsed).some((item) => typeof item !== 'string')) throw validationError(field, `${field} values must be strings.`)
  return parsed as Record<string, string> | undefined
}

function parseJsonObject(value: string, field: string): Record<string, unknown> | undefined {
  if (!value.trim()) return undefined
  try {
    const parsed = JSON.parse(value)
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') throw validationError(field, `${field} must be a JSON object.`)
    return parsed as Record<string, unknown>
  } catch (error) {
    if (error instanceof Error && 'field' in error) throw error
    console.error(`[agent-configure] Invalid ${field} JSON.`, error)
    throw validationError(field, `${field} must be valid JSON.`)
  }
}

function validationError(field: string, message: string) {
  const error = Object.assign(new Error(message), { field })
  console.error('[agent-configure] Validation failed.', { field, message })
  return error
}
