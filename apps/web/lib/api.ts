import type {
  AgentConfig,
  AgentListOptions,
  AgentRun,
  AgentRunListOptions,
  AgentRunRequest,
  AgentRunResult,
  CharacterSyncResponse,
  Chapter,
  ChapterWriteRequest,
  AIWorkflow,
  AppSetting,
  ChapterVersion,
  ChapterVersionWriteRequest,
  Entity,
  Fact,
  GraphEdge,
  GraphExpandRequest,
  GraphExpandResponse,
  GraphNode,
  HealthStatus,
  IndexJob,
  IndexJobListOptions,
  InitializeProjectResponse,
  MCPServerConfig,
  ModelConfig,
  ModelResolution,
  ModelUsageSettings,
  Project,
  ProjectSeed,
  ProjectSummary,
  ProviderConfig,
  RebuildVectorsResponse,
  RunPendingIndexResponse,
  SaveChapterVersionResponse,
  SemanticSearchRequest,
  SemanticSearchResponse,
  Skill,
  SkillScanResult,
  SkillSource,
  StoryBible,
  SystemStatus,
  ToolDefinition,
  ToolInvocation
} from './types'
import { CHAPTER_STATUS_VALUES } from './types'

import * as apiSdk from './generated/api/sdk.gen'
import type * as GeneratedApi from './generated/api/types.gen'
import type { AgentApi } from '~/entities/agent'
import type { ChapterApi } from '~/entities/chapter'
import type { CreateChapterRequest, UpdateChapterRequest } from '~/entities/chapter'
import type { GraphApi } from '~/entities/graph'
import type { IndexJobApi } from '~/entities/index-job'
import type { ModelApi } from '~/entities/model'
import type { ProjectApi } from '~/entities/project'
import type { StoryBibleApi } from '~/entities/story-bible'
import {
  ApiClientError,
  apiValidationError,
  callGeneratedApi,
  configureGeneratedClient,
  DEFAULT_API_BASE,
  isRecord,
  optionalApiArray,
  optionalStringRecord,
  requireApiArray,
  requireApiBoolean,
  requireApiNumber,
  requireApiRecord,
  requireApiString
} from '~/shared/api'
import type { ApiResult } from '~/shared/api'

export { ApiClientError, DEFAULT_API_BASE }
export type { ApiResult }

type LocaleCode = 'zh-CN' | 'en-US'

type ApiCopy = {
  untitledProject: string
  defaultGenre: string
  defaultTone: string
  defaultAudience: string
  healthWarnings: {
    qdrant: string
    postgres: string
  }
  draftRequestLabels: {
    styleConstraints: string
    selectedChapterPlan: string
    selectedChapterSummary: string
    selectedWorldRules: string
    selectedCharacters: string
  }
}

type HealthResponse = GeneratedApi.Health

type ApiSuccessEnvelope<T> = {
  data?: T
  meta?: GeneratedApi.Meta
  page?: GeneratedApi.Page
}

type GeneratedFieldsResult<TEnvelope> = {
  data?: TEnvelope
  error?: unknown
  request?: unknown
  response?: unknown
}

type ApiOperationResult<TEnvelope> = TEnvelope | GeneratedFieldsResult<TEnvelope>

type ApiEnvelopeData<TEnvelope> = TEnvelope extends ApiSuccessEnvelope<infer TData> ? NonNullable<TData> : never

type BackendContextSelection = GeneratedApi.ContextSelection

const modelUsageKeys: Array<keyof ModelUsageSettings> = [
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

type ProviderSaveRequest = GeneratedApi.ProviderRequestWritable

export type ProviderDeleteResponse = {
  status: string
}

type ModelSaveRequest = GeneratedApi.ModelRequest

type BackendProjectSeedRequest = GeneratedApi.ProjectSeed

type BackendStoryBibleRequest = GeneratedApi.StoryBible

type BackendCharacterSyncRequest = GeneratedApi.CharacterSyncRequest

type SkillSaveRequest = GeneratedApi.SkillRequest & { id?: string }

type MCPServerSaveRequest = GeneratedApi.McpServerRequest

type BackendChapterVersionRequest = GeneratedApi.ChapterVersionRequest & Pick<ChapterVersionWriteRequest, 'change_note' | 'parent_version_id'>

export type SkillSourceListOptions = {
  projectId?: string
  enabled?: boolean
  limit?: number
}

export type SkillListOptions = {
  projectId?: string
  sourceId?: string
  enabled?: boolean
  limit?: number
}

export type MCPServerListOptions = {
  projectId?: string
  enabled?: boolean
  status?: MCPServerConfig['status']
  limit?: number
}

export type ToolCatalogListOptions = {
  projectId?: string
  kind?: ToolDefinition['kind']
  status?: ToolDefinition['status']
  mcpServerId?: string
  sourceId?: string
  skillId?: string
  limit?: number
}

export type ToolInvocationListOptions = {
  agentRunId?: string
  agentId?: string
  projectId?: string
  toolId?: string
  status?: ToolInvocation['status']
  limit?: number
}

export interface ApiDomains {
  project: ProjectApi
  storyBible: StoryBibleApi
  chapter: ChapterApi
  graph: GraphApi
  model: ModelApi
  agent: AgentApi
  indexJob: IndexJobApi
}

export interface ApiClient extends ApiDomains {
  health(): Promise<ApiResult<HealthStatus>>
  systemStatus(): Promise<ApiResult<SystemStatus>>
  listProviders(): Promise<ApiResult<ProviderConfig[]>>
  saveProvider(provider: ProviderConfig, mode?: 'create' | 'edit'): Promise<ApiResult<ProviderConfig>>
  deleteProvider(id: string): Promise<ApiResult<ProviderDeleteResponse>>
  listModels(kind?: string): Promise<ApiResult<ModelConfig[]>>
  saveModel(model: ModelConfig): Promise<ApiResult<ModelConfig>>
  deleteModel(id: string): Promise<ApiResult<{ status: string }>>
  refreshModels(providerId: string): Promise<ApiResult<ModelConfig[]>>
  listSettings(scope?: string): Promise<ApiResult<AppSetting[]>>
  saveSetting(setting: AppSetting): Promise<ApiResult<AppSetting>>
  getModelUsageSettings(): Promise<ApiResult<ModelUsageSettings>>
  saveModelUsageSettings(settings: ModelUsageSettings): Promise<ApiResult<ModelUsageSettings>>
  listProjects(): Promise<ApiResult<ProjectSummary[]>>
  initializeProject(seed: ProjectSeed): Promise<ApiResult<StoryBible>>
  initializeProjectFull(seed: ProjectSeed): Promise<ApiResult<InitializeProjectResponse>>
  getStoryBible(projectId: string): Promise<ApiResult<StoryBible>>
  updateStoryBible(projectId: string, bible: StoryBible): Promise<ApiResult<StoryBible>>
  syncCharacters(projectId: string, bible: StoryBible): Promise<ApiResult<CharacterSyncResponse>>
  expandGraph(request: GraphExpandRequest): Promise<ApiResult<GraphExpandResponse>>
  semanticSearch(projectId: string, request: SemanticSearchRequest): Promise<ApiResult<SemanticSearchResponse>>
  listChapters(projectId: string): Promise<ApiResult<Chapter[]>>
  createChapter(projectId: string, request: CreateChapterRequest): Promise<ApiResult<Chapter>>
  updateChapter(projectId: string, request: UpdateChapterRequest): Promise<ApiResult<Chapter>>
  listChapterVersions(projectId: string, chapterId: string): Promise<ApiResult<ChapterVersion[]>>
  saveChapterVersion(projectId: string, version: ChapterVersionWriteRequest): Promise<ApiResult<SaveChapterVersionResponse>>
  listAgents(options?: AgentListOptions): Promise<ApiResult<AgentConfig[]>>
  saveAgent(agent: AgentConfig, mode?: 'create' | 'edit'): Promise<ApiResult<AgentConfig>>
  deleteAgent(id: string): Promise<ApiResult<{ status: string }>>
  runAgent(agentId: string, request: AgentRunRequest): Promise<ApiResult<AgentRunResult>>
  listAgentRuns(options?: AgentRunListOptions): Promise<ApiResult<AgentRun[]>>
  listSkillSources(options?: SkillSourceListOptions): Promise<ApiResult<SkillSource[]>>
  scanDefaultSkillSource(): Promise<ApiResult<SkillScanResult>>
  scanSkillSource(id: string): Promise<ApiResult<SkillScanResult>>
  listSkills(options?: SkillListOptions): Promise<ApiResult<Skill[]>>
  saveSkill(skill: Skill, mode?: 'create' | 'edit'): Promise<ApiResult<Skill>>
  deleteSkill(id: string): Promise<ApiResult<{ status: string }>>
  setSkillEnabled(id: string, enabled: boolean): Promise<ApiResult<Skill>>
  listMCPServers(options?: MCPServerListOptions): Promise<ApiResult<MCPServerConfig[]>>
  saveMCPServer(server: MCPServerConfig, mode?: 'create' | 'edit'): Promise<ApiResult<MCPServerConfig>>
  deleteMCPServer(id: string): Promise<ApiResult<{ status: string }>>
  setMCPServerEnabled(id: string, enabled: boolean): Promise<ApiResult<MCPServerConfig>>
  testMCPServer(id: string): Promise<ApiResult<{ ok: boolean; server: MCPServerConfig }>>
  refreshMCPTools(id: string): Promise<ApiResult<{ tools: ToolDefinition[]; count: number; unavailable: number }>>
  listMCPServerTools(id: string): Promise<ApiResult<ToolDefinition[]>>
  listToolCatalog(options?: ToolCatalogListOptions): Promise<ApiResult<ToolDefinition[]>>
  setToolEnabled(id: string, enabled: boolean): Promise<ApiResult<ToolDefinition>>
  listToolInvocations(options?: ToolInvocationListOptions): Promise<ApiResult<ToolInvocation[]>>
  listIndexJobs(options?: string | IndexJobListOptions): Promise<ApiResult<IndexJob[]>>
  runIndexJob(id: string): Promise<ApiResult<IndexJob>>
  runPendingIndexJobs(projectId?: string, limit?: number): Promise<ApiResult<RunPendingIndexResponse>>
  rebuildVectors(): Promise<ApiResult<RebuildVectorsResponse>>
  optimizeProjectSeed(seed: ProjectSeed): Promise<ApiResult<ProjectSeed>>
}

function normalizeLocale(locale?: string): LocaleCode {
  return locale === 'en-US' ? 'en-US' : 'zh-CN'
}

function getApiCopy(locale: LocaleCode): ApiCopy {
  if (locale === 'en-US') {
    return {
      untitledProject: 'Untitled project',
      defaultGenre: 'Long-form fiction',
      defaultTone: 'Clear and grounded',
      defaultAudience: 'General readers',
      healthWarnings: {
        qdrant: 'Qdrant is not configured. Index operations will fail until vector storage is available.',
        postgres: 'Postgres is not configured. The service may be using volatile storage.'
      },
      draftRequestLabels: {
        styleConstraints: 'Style constraints',
        selectedChapterPlan: 'Focus on current editable chapter plan',
        selectedChapterSummary: 'Focus on current chapter summary',
        selectedWorldRules: 'Respect world rules',
        selectedCharacters: 'Focus characters'
      }
    }
  }

  return {
    untitledProject: '未命名项目',
    defaultGenre: '长篇小说',
    defaultTone: '稳健、清晰',
    defaultAudience: '通用读者',
    healthWarnings: {
      qdrant: 'Qdrant 未配置：索引任务会在向量存储可用前失败。',
      postgres: 'Postgres 未配置：服务可能正在使用临时存储。'
    },
    draftRequestLabels: {
      styleConstraints: '风格约束',
      selectedChapterPlan: '重点参考当前可编辑章节方案',
      selectedChapterSummary: '重点参考当前章节摘要',
      selectedWorldRules: '遵守世界规则',
      selectedCharacters: '重点参考角色'
    }
  }
}

function normalizeProviderType(value?: string): ProviderConfig['provider_type'] {
  if (value === 'openai' || value === 'anthropic' || value === 'gemini' || value === 'openai-responses') return value
  return 'openai-responses'
}

function normalizeProviderStatus(value: string | undefined, enabled?: boolean): ProviderConfig['status'] {
  if (value === 'online' || value === 'degraded' || value === 'offline' || value === 'unknown') return value
  return enabled ? 'online' : 'unknown'
}

function normalizeModelKind(value?: string): NonNullable<ModelConfig['kind']> {
  return value === 'embedding' ? 'embedding' : 'text'
}

function normalizeAgentRole(value?: string): NonNullable<ModelConfig['allowed_agent_roles']>[number] | undefined {
  if (
    value === 'genesis-optimizer'
    || value === 'plot-architect'
    || value === 'world-builder'
    || value === 'character-keeper'
    || value === 'continuity-auditor'
    || value === 'writer'
    || value === 'editor'
    || value === 'fact-extractor'
    || value === 'graph-curator'
  ) {
    return value
  }
  return undefined
}

function normalizeAgentRoles(values?: string[]): ModelConfig['allowed_agent_roles'] {
  return (values || []).map(normalizeAgentRole).filter((value): value is NonNullable<ModelConfig['allowed_agent_roles']>[number] => Boolean(value))
}

function optionalAgentString(value: unknown, endpoint: string, field: string): string | undefined {
  if (value === undefined || value === null) return undefined
  return requireApiString(value, endpoint, field, { allowEmpty: true })
}

export function decodeAgentResponse(value: unknown, index = 0, endpoint = 'listAgents'): AgentConfig {
  const item = requireApiRecord(value, endpoint, `agents[${index}]`)
  const roleValue = optionalAgentString(item.role, endpoint, `agents[${index}].role`)
  const role = roleValue ? normalizeAgentRole(roleValue) : undefined
  if (roleValue && !role) {
    throw apiValidationError(endpoint, `agents[${index}].role`, `role must be one of the documented Agent roles`, roleValue)
  }
  return {
    id: requireApiString(item.id, endpoint, `agents[${index}].id`),
    project_id: optionalAgentString(item.project_id, endpoint, `agents[${index}].project_id`),
    name: requireApiString(item.name, endpoint, `agents[${index}].name`),
    description: optionalAgentString(item.description, endpoint, `agents[${index}].description`),
    role,
    model_id: optionalAgentString(item.model_id, endpoint, `agents[${index}].model_id`),
    enabled: requireApiBoolean(item.enabled, endpoint, `agents[${index}].enabled`),
    system_prompt: optionalAgentString(item.system_prompt, endpoint, `agents[${index}].system_prompt`),
    skill_ids: optionalApiArray(item.skill_ids, endpoint, `agents[${index}].skill_ids`, (entry, entryIndex) => requireApiString(entry, endpoint, `agents[${index}].skill_ids[${entryIndex}]`)),
    tool_ids: optionalApiArray(item.tool_ids, endpoint, `agents[${index}].tool_ids`, (entry, entryIndex) => requireApiString(entry, endpoint, `agents[${index}].tool_ids[${entryIndex}]`)),
    mcp_server_ids: optionalApiArray(item.mcp_server_ids, endpoint, `agents[${index}].mcp_server_ids`, (entry, entryIndex) => requireApiString(entry, endpoint, `agents[${index}].mcp_server_ids[${entryIndex}]`)),
    memory_policy: item.memory_policy === undefined || item.memory_policy === null ? undefined : requireApiRecord(item.memory_policy, endpoint, `agents[${index}].memory_policy`),
    runtime_options: item.runtime_options === undefined || item.runtime_options === null ? undefined : requireApiRecord(item.runtime_options, endpoint, `agents[${index}].runtime_options`),
    metadata: optionalStringRecord(item.metadata, endpoint, `agents[${index}].metadata`),
    created_at: optionalAgentString(item.created_at, endpoint, `agents[${index}].created_at`),
    updated_at: optionalAgentString(item.updated_at, endpoint, `agents[${index}].updated_at`)
  }
}

function requirePathId(value: string | undefined, label: string): string {
  return requireApiString(value, label, label).trim()
}

async function mapApi<TEnvelope extends ApiSuccessEnvelope<unknown>, TData>(
  operationName: string,
  operation: () => Promise<ApiOperationResult<TEnvelope>>,
  mapData: (raw: ApiEnvelopeData<TEnvelope>) => TData
): Promise<ApiResult<TData>> {
  return callGeneratedApi(operationName, operation, (raw) => mapData(raw as ApiEnvelopeData<TEnvelope>))
}

function projectSeedToBackend(seed: ProjectSeed, copy: ApiCopy): BackendProjectSeedRequest {
  const constraints = [seed.central_conflict, seed.taboos, ...(seed.constraints || [])]
    .map((item) => item.trim())
    .filter(Boolean)
  const mainCharacters = Array.from(new Set([seed.protagonist, ...(seed.main_characters || [])]
    .map((item) => item.trim())
    .filter(Boolean)))

  return {
    title: seed.title || seed.one_sentence_core.slice(0, 32) || copy.untitledProject,
    premise: seed.premise || seed.one_sentence_core,
    genre: seed.genre || seed.tags[0] || copy.defaultGenre,
    tone: seed.tone || seed.style || copy.defaultTone,
    audience: seed.audience || copy.defaultAudience,
    language: seed.language || 'zh-CN',
    setting: seed.setting || seed.world_background,
    themes: seed.themes?.length ? seed.themes : seed.tags,
    main_characters: mainCharacters,
    constraints,
    target_chapters: seed.target_chapters,
    metadata: {
      ...(seed.metadata || {}),
      one_sentence_core: seed.one_sentence_core,
      tags: seed.tags.join(','),
      world_background: seed.world_background,
      protagonist: seed.protagonist,
      central_conflict: seed.central_conflict,
      style: seed.style,
      taboos: seed.taboos
    }
  }
}

function normalizeProvider(provider: GeneratedApi.Provider): ProviderConfig {
  const providerType = normalizeProviderType(provider.type)
  const enabled = provider.enabled ?? true
  return {
    ...provider,
    id: provider.id || '',
    name: provider.name || '',
    provider_type: providerType,
    type: providerType,
    base_url: provider.base_url || '',
    streaming: provider.streaming ?? provider.metadata?.streaming === 'true',
    enabled,
    api_key_hint: provider.api_key_hint,
    default_model_id: provider.default_model_id || provider.metadata?.default_model_id,
    last_checked_at: provider.last_checked_at || provider.last_model_refresh_at || provider.updated_at,
    status: normalizeProviderStatus(provider.status, enabled)
  }
}

function providerToBackend(provider: ProviderConfig): ProviderSaveRequest {
  const metadata: Record<string, string> = { ...(provider.metadata || {}) }
  metadata.streaming = String(provider.streaming ?? false)
  if (provider.default_model_id?.trim()) {
    metadata.default_model_id = provider.default_model_id.trim()
  } else {
    delete metadata.default_model_id
  }

  const request: ProviderSaveRequest = {
    name: provider.name.trim(),
    type: provider.provider_type || provider.type || 'openai-responses',
    base_url: provider.base_url.trim(),
    enabled: provider.enabled,
    streaming: provider.streaming,
    trace_enabled: provider.trace_enabled,
    trace_retention_days: provider.trace_retention_days,
    default_request_timeout_sec: provider.default_request_timeout_sec,
    metadata: Object.keys(metadata).length > 0 ? metadata : undefined
  }

  if (provider.id.trim()) request.id = provider.id.trim()
  if (provider.api_key?.trim()) request.api_key = provider.api_key.trim()

  return request
}

function skillToBackend(skill: Skill, includeId: boolean): SkillSaveRequest {
  const body: SkillSaveRequest = {
    project_id: skill.project_id || undefined,
    source_id: skill.source_id || undefined,
    name: skill.name.trim(),
    description: skill.description?.trim() || undefined,
    content: skill.content || undefined,
    path: skill.path || undefined,
    enabled: skill.enabled,
    metadata: skill.metadata
  }
  if (includeId && skill.id?.trim()) body.id = skill.id.trim()
  return body
}

function mcpServerToBackend(server: MCPServerConfig, includeId: boolean): MCPServerSaveRequest {
  const body: MCPServerSaveRequest = {
    project_id: server.project_id || undefined,
    name: server.name.trim(),
    transport: server.transport,
    status: server.status || undefined,
    enabled: server.enabled,
    command: server.command?.trim() || undefined,
    args: server.args?.filter((item) => item.trim()),
    url: server.url?.trim() || undefined,
    headers: server.headers,
    secret_headers: server.secret_headers,
    env: server.env,
    secret_env: server.secret_env,
    timeout_sec: server.timeout_sec,
    metadata: server.metadata
  }
  if (includeId && server.id?.trim()) body.id = server.id.trim()
  return body
}

function normalizeModel(model: GeneratedApi.Model): ModelConfig {
  const name = model.name || ''
  const providerId = model.provider_id || ''
  return {
    ...model,
    id: model.id || (providerId && name ? `${providerId}:${name}` : name),
    provider_id: providerId,
    provider_type: normalizeProviderType(model.provider_type),
    name,
    display_name: model.display_name || name,
    kind: normalizeModelKind(model.kind),
    enabled: model.enabled ?? true,
    context_window: model.context_window || 0,
    max_output_tokens: model.max_output_tokens || 0,
    dimension: model.dimension || 0,
    supports_streaming: model.supports_streaming ?? false,
    supports_tools: model.supports_tools ?? false,
    default_for_kind: model.default_for_kind ?? false,
    cost_input_per_mtok: model.cost_input_per_mtok || 0,
    cost_output_per_mtok: model.cost_output_per_mtok || 0,
    routing_weight: model.routing_weight || 100,
    allowed_agent_roles: normalizeAgentRoles(model.allowed_agent_roles)
  }
}

function modelToBackend(model: ModelConfig): ModelSaveRequest {
  const request: ModelSaveRequest = {
    provider_id: model.provider_id.trim(),
    name: model.name.trim(),
    display_name: (model.display_name || model.name).trim(),
    kind: model.kind || 'text',
    context_window: Number(model.context_window || 0),
    max_output_tokens: Number(model.max_output_tokens || 0),
    dimension: Number(model.dimension || 0),
    supports_tools: Boolean(model.supports_tools),
    supports_streaming: Boolean(model.supports_streaming),
    default_for_kind: Boolean(model.default_for_kind),
    enabled: Boolean(model.enabled),
    cost_input_per_mtok: Number(model.cost_input_per_mtok || 0),
    cost_output_per_mtok: Number(model.cost_output_per_mtok || 0),
    routing_weight: Number(model.routing_weight || 0),
    allowed_agent_roles: model.allowed_agent_roles || [],
    metadata: model.metadata
  }
  const id = model.id?.trim()
  if (id) request.id = id
  return request
}

function emptyModelUsageSettings(): ModelUsageSettings {
  return {
    writer: '',
    editor: '',
    'genesis-optimizer': '',
    'plot-architect': '',
    'world-builder': '',
    'character-keeper': '',
    'continuity-auditor': '',
    'fact-extractor': '',
    'graph-curator': '',
    embedding: ''
  }
}

function normalizeModelUsageSettings(routes: Partial<ModelUsageSettings> = {}): ModelUsageSettings {
  const settings = emptyModelUsageSettings()
  modelUsageKeys.forEach((key) => {
    const raw = routes[key]
    settings[key] = typeof raw === 'string' ? raw.trim() : ''
  })
  return settings
}

function normalizeModelResolution(value: unknown, endpoint: string, field = 'model_resolution'): ModelResolution {
  const resolution = requireApiRecord(value, endpoint, field)
  const providerType = requireApiString(resolution.provider_type, endpoint, `${field}.provider_type`)
  const modelKind = requireApiString(resolution.model_kind, endpoint, `${field}.model_kind`)
  if (providerType !== 'openai-responses' && providerType !== 'openai' && providerType !== 'anthropic' && providerType !== 'gemini') {
    throw new ApiClientError({ endpoint, field: `${field}.provider_type`, kind: 'validation', code: 'invalid_api_response', message: `invalid_api_response: unsupported provider type ${providerType}`, cause: value })
  }
  if (modelKind !== 'text' && modelKind !== 'embedding') {
    throw new ApiClientError({ endpoint, field: `${field}.model_kind`, kind: 'validation', code: 'invalid_api_response', message: `invalid_api_response: unsupported model kind ${modelKind}`, cause: value })
  }
  return {
    route_key: requireApiString(resolution.route_key, endpoint, `${field}.route_key`, { allowEmpty: true }),
    resolution_source: requireApiString(resolution.resolution_source, endpoint, `${field}.resolution_source`),
    provider_id: requireApiString(resolution.provider_id, endpoint, `${field}.provider_id`),
    provider_name: requireApiString(resolution.provider_name, endpoint, `${field}.provider_name`, { allowEmpty: true }),
    provider_type: providerType,
    model_id: requireApiString(resolution.model_id, endpoint, `${field}.model_id`),
    model_name: requireApiString(resolution.model_name, endpoint, `${field}.model_name`, { allowEmpty: true }),
    model_kind: modelKind
  }
}

function normalizeProjectEntity(value: unknown, endpoint: string): Project {
  const project = requireApiRecord(value, endpoint, 'project')
  return {
    ...(project as unknown as Partial<Project>),
    id: requireApiString(project.id, endpoint, 'project.id'),
    title: requireApiString(project.title, endpoint, 'project.title', { allowEmpty: true }),
    slug: requireApiString(project.slug, endpoint, 'project.slug', { allowEmpty: true }),
    status: requireApiString(project.status, endpoint, 'project.status'),
    seed: requireApiRecord(project.seed, endpoint, 'project.seed') as unknown as ProjectSeed,
    created_at: requireApiString(project.created_at, endpoint, 'project.created_at'),
    updated_at: requireApiString(project.updated_at, endpoint, 'project.updated_at'),
    metadata: optionalStringRecord(project.metadata, endpoint, 'project.metadata')
  }
}

function normalizeWorkflowStep(value: unknown, index: number, endpoint: string): AIWorkflow['steps'][number] {
  const step = requireApiRecord(value, endpoint, `workflow.steps[${index}]`)
  return {
    id: typeof step.id === 'string' ? step.id : undefined,
    name: requireApiString(step.name, endpoint, `workflow.steps[${index}].name`),
    status: requireApiString(step.status, endpoint, `workflow.steps[${index}].status`),
    message: typeof step.message === 'string' ? step.message : undefined,
    updated_at: typeof step.updated_at === 'string' ? step.updated_at : undefined,
    started_at: typeof step.started_at === 'string' ? step.started_at : undefined,
    ended_at: typeof step.ended_at === 'string' ? step.ended_at : undefined,
    error: typeof step.error === 'string' ? step.error : undefined,
    metadata: optionalStringRecord(step.metadata, endpoint, `workflow.steps[${index}].metadata`)
  }
}

function normalizeWorkflow(value: unknown, endpoint = 'workflow'): AIWorkflow {
  const workflow = requireApiRecord(value, endpoint, 'workflow')
  return {
    ...(workflow as unknown as Partial<AIWorkflow>),
    id: requireApiString(workflow.id, endpoint, 'workflow.id'),
    project_id: requireApiString(workflow.project_id, endpoint, 'workflow.project_id'),
    intent: requireApiString(workflow.intent ?? workflow.kind, endpoint, 'workflow.intent'),
    status: requireApiString(workflow.status, endpoint, 'workflow.status'),
    steps: requireApiArray(workflow.steps, endpoint, 'workflow.steps', (step, index) => normalizeWorkflowStep(step, index, endpoint)),
    model_resolution: workflow.model_resolution === undefined ? undefined : normalizeModelResolution(workflow.model_resolution, endpoint)
  }
}

const STORY_BIBLE_METADATA_KEYS = new Set([
  'story_bible_premise',
  'story_bible_world_rules',
  'story_bible_characters',
  'story_bible_foreshadows',
  'story_bible_chapter_plan'
])

function stripStoryBibleMetadata(metadata?: Record<string, string>): Record<string, string> | undefined {
  if (!metadata) return undefined
  const next = Object.entries(metadata).reduce<Record<string, string>>((items, [key, value]) => {
    if (!STORY_BIBLE_METADATA_KEYS.has(key)) items[key] = value
    return items
  }, {})
  return Object.keys(next).length > 0 ? next : undefined
}

function sanitizeStoryBibleChapterPlan(bible: StoryBible): StoryBible['chapter_plan'] {
  return bible.chapter_plan.map((chapter, index) => ({
    id: requireApiString(chapter.id, 'updateStoryBible', `chapter_plan[${index}].id`).trim(),
    title: chapter.title.trim(),
    status: requireChapterStatus(chapter.status, 'updateStoryBible', `chapter_plan[${index}].status`),
    summary: chapter.summary.trim()
  }))
}

function storyBibleToBackend(bible: StoryBible): BackendStoryBibleRequest {
  const worldRules = (bible.world_rules || []).map((rule) => rule.trim()).filter(Boolean)
  const characters = (bible.characters || []).map((character, index) => ({
    ...character,
    id: requireApiString(character.id, 'updateStoryBible', `characters[${index}].id`).trim(),
    name: character.name.trim(),
    role: character.role.trim(),
    desire: character.desire.trim(),
    wound: character.wound.trim(),
    secret: character.secret?.trim(),
    summary: character.summary?.trim(),
    metadata: stripStoryBibleMetadata(character.metadata)
  }))
  const foreshadows = (bible.foreshadows || []).map((item, index) => ({
    ...item,
    id: requireApiString(item.id, 'updateStoryBible', `foreshadows[${index}].id`).trim(),
    title: item.title.trim(),
    planted_in: item.planted_in.trim(),
    payoff_hint: item.payoff_hint.trim(),
    status: requireForeshadowStatus(item.status, 'updateStoryBible', `foreshadows[${index}].status`, item)
  }))
  const chapterPlan = sanitizeStoryBibleChapterPlan(bible)
  const sourceSeed = bible.source_seed
  const themes = (bible.themes || []).map((theme) => theme.trim()).filter(Boolean)
  const premise = bible.premise || sourceSeed?.premise || bible.logline || bible.synopsis || ''
  const sourceMetadata = stripStoryBibleMetadata(sourceSeed?.metadata)
  const worldBackground = worldRules.join('\n') || sourceSeed?.world_background || sourceSeed?.setting || ''
  const rules = worldRules.reduce<Record<string, string>>((items, rule, index) => {
    items[`rule_${index + 1}`] = rule
    return items
  }, {})
  const sanitizedSourceSeed: BackendProjectSeedRequest = {
    title: sourceSeed?.title || bible.title || '',
    premise,
    genre: bible.genre || sourceSeed?.genre || '',
    tone: bible.tone || sourceSeed?.tone || '',
    audience: bible.audience || sourceSeed?.audience || '',
    language: bible.language || sourceSeed?.language || 'zh-CN',
    setting: sourceSeed?.setting || worldBackground,
    themes,
    main_characters: characters.map((character) => character.name).filter(Boolean),
    constraints: sourceSeed?.constraints?.length ? sourceSeed.constraints.map((item) => item.trim()).filter(Boolean) : worldRules,
    target_chapters: sourceSeed?.target_chapters || chapterPlan.length || undefined,
    metadata: sourceMetadata
  }

  return {
    id: bible.id,
    version: bible.version || 0,
    project_id: bible.project_id,
    title: bible.title || sanitizedSourceSeed.title,
    logline: bible.logline || premise,
    synopsis: bible.synopsis || premise,
    genre: bible.genre || sanitizedSourceSeed.genre,
    tone: bible.tone || sanitizedSourceSeed.tone,
    audience: bible.audience || sanitizedSourceSeed.audience,
    language: bible.language || sanitizedSourceSeed.language,
    themes,
    rules,
    worldline_ids: bible.worldline_ids || [],
    entity_ids: bible.entity_ids || [],
    plot_thread_ids: bible.plot_thread_ids || [],
    source_seed: sanitizedSourceSeed,
    genesis_workflow_id: bible.genesis_workflow_id || '',
    approved: Boolean(bible.approved),
    premise,
    world_rules: worldRules,
    characters,
    foreshadows,
    chapter_plan: chapterPlan
  }
}

function normalizeStoryBibleCharacter(value: unknown, index: number, endpoint: string): StoryBible['characters'][number] {
  const character = requireApiRecord(value, endpoint, `characters[${index}]`)
  return {
    id: requireApiString(character.id, endpoint, `characters[${index}].id`),
    name: requireApiString(character.name, endpoint, `characters[${index}].name`),
    role: requireApiString(character.role, endpoint, `characters[${index}].role`),
    desire: requireApiString(character.desire, endpoint, `characters[${index}].desire`),
    wound: requireApiString(character.wound, endpoint, `characters[${index}].wound`),
    secret: typeof character.secret === 'string' ? character.secret : undefined,
    summary: typeof character.summary === 'string' ? character.summary : undefined,
    entity_id: typeof character.entity_id === 'string' ? character.entity_id : undefined,
    sync_status: typeof character.sync_status === 'string' ? character.sync_status : undefined,
    synced_at: typeof character.synced_at === 'string' ? character.synced_at : undefined,
    metadata: optionalStringRecord(character.metadata, endpoint, `characters[${index}].metadata`)
  }
}

function requireForeshadowStatus(status: string, endpoint: string, field: string, cause?: unknown): StoryBible['foreshadows'][number]['status'] {
  if (status === 'planted' || status === 'active' || status === 'paid_off') return status
  throw new ApiClientError({
    endpoint,
    field,
    kind: 'validation',
    code: 'invalid_api_response',
    message: `invalid_api_response: unsupported foreshadow status ${status}`,
    cause
  })
}

function normalizeStoryBibleForeshadow(value: unknown, index: number, endpoint: string): StoryBible['foreshadows'][number] {
  const item = requireApiRecord(value, endpoint, `foreshadows[${index}]`)
  const status = requireForeshadowStatus(
    requireApiString(item.status, endpoint, `foreshadows[${index}].status`),
    endpoint,
    `foreshadows[${index}].status`,
    item
  )
  return {
    id: requireApiString(item.id, endpoint, `foreshadows[${index}].id`),
    title: requireApiString(item.title, endpoint, `foreshadows[${index}].title`, { allowEmpty: true }),
    planted_in: requireApiString(item.planted_in, endpoint, `foreshadows[${index}].planted_in`, { allowEmpty: true }),
    payoff_hint: requireApiString(item.payoff_hint, endpoint, `foreshadows[${index}].payoff_hint`, { allowEmpty: true }),
    status
  }
}

function normalizeStoryBibleChapter(value: unknown, index: number, endpoint: string): StoryBible['chapter_plan'][number] {
  const chapter = requireApiRecord(value, endpoint, `chapter_plan[${index}]`)
  return {
    id: requireApiString(chapter.id, endpoint, `chapter_plan[${index}].id`),
    title: requireApiString(chapter.title, endpoint, `chapter_plan[${index}].title`, { allowEmpty: true }),
    status: requireChapterStatus(chapter.status, endpoint, `chapter_plan[${index}].status`),
    summary: requireApiString(chapter.summary, endpoint, `chapter_plan[${index}].summary`, { allowEmpty: true })
  }
}

function normalizeStoryBible(value: unknown, endpoint = 'storyBible'): StoryBible {
  const bible = requireApiRecord(value, endpoint)
  const sourceSeed = isRecord(bible.source_seed) ? bible.source_seed as unknown as ProjectSeed : undefined
  const chapterPlan = optionalApiArray(bible.chapter_plan, endpoint, 'chapter_plan', (item, index) => normalizeStoryBibleChapter(item, index, endpoint))
  const rules = optionalStringRecord(bible.rules, endpoint, 'rules')
  return {
    ...(bible as unknown as Partial<StoryBible>),
    id: requireApiString(bible.id, endpoint, 'id'),
    project_id: requireApiString(bible.project_id, endpoint, 'project_id'),
    premise: requireApiString(bible.premise, endpoint, 'premise', { allowEmpty: true }),
    themes: optionalApiArray(bible.themes, endpoint, 'themes', (item, index) => requireApiString(item, endpoint, `themes[${index}]`, { allowEmpty: true })),
    world_rules: optionalApiArray(bible.world_rules, endpoint, 'world_rules', (item, index) => requireApiString(item, endpoint, `world_rules[${index}]`, { allowEmpty: true })),
    characters: optionalApiArray(bible.characters, endpoint, 'characters', (item, index) => normalizeStoryBibleCharacter(item, index, endpoint)),
    foreshadows: optionalApiArray(bible.foreshadows, endpoint, 'foreshadows', (item, index) => normalizeStoryBibleForeshadow(item, index, endpoint)),
    chapter_plan: chapterPlan,
    source_seed: sourceSeed,
    rules
  }
}

function normalizeProject(project: Project, endpoint = 'listProjects'): ProjectSummary {
  const targetChapters = project.seed?.target_chapters
  const seedTags = project.seed?.themes?.length
    ? project.seed.themes
    : project.seed?.metadata?.tags
      ? project.seed.metadata.tags.split(',').map((tag) => tag.trim()).filter(Boolean)
      : project.seed?.genre
        ? [project.seed.genre]
        : []

  return {
    id: requireApiString(project.id, endpoint, 'project.id'),
    title: requireApiString(project.title, endpoint, 'project.title', { allowEmpty: true }),
    slug: project.slug,
    status: project.status,
    logline: project.seed?.premise || project.seed?.metadata?.one_sentence_core || project.metadata?.logline || project.status,
    tags: seedTags,
    seed: project.seed,
    active_story_bible_id: project.active_story_bible_id,
    created_at: project.created_at,
    updated_at: requireApiString(project.updated_at, endpoint, 'project.updated_at'),
    bible_status: project.active_story_bible_id ? 'draft' : 'missing',
    chapter_count: undefined,
    target_chapters: targetChapters
  }
}

function requireChapterStatus(value: unknown, endpoint: string, field: string): Chapter['status'] {
  const status = requireApiString(value, endpoint, field)
  if (CHAPTER_STATUS_VALUES.some((candidate) => candidate === status)) return status as Chapter['status']
  throw new ApiClientError({ endpoint, field, kind: 'validation', code: 'invalid_api_response', message: `invalid_api_response: unsupported chapter status ${status}`, cause: value })
}

function normalizeChapter(value: unknown, index: number, endpoint = 'chapter'): Chapter {
  const chapter = requireApiRecord(value, endpoint, `chapters[${index}]`)
  return {
    ...(chapter as unknown as Partial<Chapter>),
    id: requireApiString(chapter.id, endpoint, `chapters[${index}].id`),
    project_id: requireApiString(chapter.project_id, endpoint, `chapters[${index}].project_id`),
    number: requireApiNumber(chapter.number, endpoint, `chapters[${index}].number`),
    title: requireApiString(chapter.title, endpoint, `chapters[${index}].title`, { allowEmpty: true }),
    status: requireChapterStatus(chapter.status, endpoint, `chapters[${index}].status`),
    summary: requireApiString(chapter.summary, endpoint, `chapters[${index}].summary`, { allowEmpty: true }),
    metadata: optionalStringRecord(chapter.metadata, endpoint, `chapters[${index}].metadata`) || {}
  }
}

function normalizeChapterVersion(value: unknown, index = 0, endpoint = 'chapterVersion'): ChapterVersion {
  const version = requireApiRecord(value, endpoint, `versions[${index}]`)
  const authorRoleValue = typeof version.author_role === 'string' ? version.author_role : undefined
  const authorRole = authorRoleValue === undefined ? undefined : normalizeAgentRole(authorRoleValue)
  if (authorRoleValue !== undefined && authorRole === undefined) {
    throw new ApiClientError({
      endpoint,
      field: `versions[${index}].author_role`,
      kind: 'validation',
      code: 'invalid_api_response',
      message: `invalid_api_response: unsupported author role ${authorRoleValue}`,
      cause: value
    })
  }
  const content = requireApiString(version.content, endpoint, `versions[${index}].content`, { allowEmpty: true })
  const metadata = optionalStringRecord(version.metadata, endpoint, `versions[${index}].metadata`)
  return {
    ...(version as unknown as Partial<ChapterVersion>),
    id: requireApiString(version.id, endpoint, `versions[${index}].id`),
    project_id: requireApiString(version.project_id, endpoint, `versions[${index}].project_id`),
    chapter_id: requireApiString(version.chapter_id, endpoint, `versions[${index}].chapter_id`),
    parent_version_id: typeof version.parent_version_id === 'string' ? version.parent_version_id : undefined,
    version: requireApiNumber(version.version, endpoint, `versions[${index}].version`),
    title: requireApiString(version.title, endpoint, `versions[${index}].title`, { allowEmpty: true }),
    content,
    created_at: requireApiString(version.created_at, endpoint, `versions[${index}].created_at`),
    author_role: authorRole,
    author: undefined,
    change_note: metadata?.change_note,
    metadata,
    metrics: { word_count: content.replace(/\s/g, '').length }
  }
}

function normalizeGraphType(type: string, endpoint: string, field: string): GraphNode['type'] {
  if (type === 'story_start') return 'story_start'
  if (type === 'character') return 'character'
  if (type === 'place' || type === 'location') return 'location'
  if (type === 'object' || type === 'item' || type === 'clue') return 'clue'
  if (type === 'concept' || type === 'rule') return 'rule'
  if (type === 'event' || type === 'time_node') return 'event'
  if (type === 'chapter') return 'chapter'
  throw new ApiClientError({ endpoint, field, kind: 'validation', code: 'invalid_api_response', message: `invalid_api_response: unsupported graph node type ${type}`, cause: type })
}

function normalizeGraphStatus(status: string, endpoint: string, field: string): GraphNode['status'] {
  if (status === 'conflict' || status === 'stable' || status === 'draft' || status === 'resolved') return status
  if (status === 'active' || status === 'canonical') return 'stable'
  if (status === 'deprecated') return 'conflict'
  throw new ApiClientError({ endpoint, field, kind: 'validation', code: 'invalid_api_response', message: `invalid_api_response: unsupported graph node status ${status}`, cause: status })
}

function optionalGraphMetadataNumber(
  metadata: Record<string, string> | undefined,
  key: string,
  endpoint: string,
  field: string
): number | undefined {
  const value = metadata?.[key]
  if (value === undefined) return undefined
  if (!value.trim()) {
    throw new ApiClientError({
      endpoint,
      field,
      kind: 'validation',
      code: 'invalid_api_response',
      message: `invalid_api_response: ${field} must be a finite number when present`,
      cause: value
    })
  }
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) {
    throw new ApiClientError({
      endpoint,
      field,
      kind: 'validation',
      code: 'invalid_api_response',
      message: `invalid_api_response: ${field} must be a finite number when present`,
      cause: value
    })
  }
  return parsed
}

function entityToNode(entity: Entity, index: number, endpoint: string): GraphNode {
  return {
    id: entity.id,
    label: entity.name,
    type: normalizeGraphType(entity.type, endpoint, `entities[${index}].type`),
    importance: entity.importance,
    depth: optionalGraphMetadataNumber(entity.metadata, 'depth', endpoint, `entities[${index}].metadata.depth`),
    timeline: optionalGraphMetadataNumber(entity.metadata, 'timeline', endpoint, `entities[${index}].metadata.timeline`),
    status: normalizeGraphStatus(entity.status, endpoint, `entities[${index}].status`),
    metadata: {
      summary: entity.summary,
      importance: entity.importance,
      ...(entity.traits || {}),
      ...(entity.metadata || {})
    }
  }
}

function normalizeGraphEdge(value: unknown, index: number, endpoint: string): GraphEdge {
  const edge = requireApiRecord(value, endpoint, `edges[${index}]`)
  const source = requireApiString(edge.source_entity_id, endpoint, `edges[${index}].source_entity_id`)
  const target = requireApiString(edge.target_entity_id, endpoint, `edges[${index}].target_entity_id`)
  const metadata = optionalStringRecord(edge.metadata, endpoint, `edges[${index}].metadata`)
  const evidenceFactIds = edge.evidence_fact_ids === undefined || edge.evidence_fact_ids === null
    ? undefined
    : requireApiArray(edge.evidence_fact_ids, endpoint, `edges[${index}].evidence_fact_ids`, (item, evidenceIndex) => requireApiString(item, endpoint, `edges[${index}].evidence_fact_ids[${evidenceIndex}]`))
  return {
    id: requireApiString(edge.id, endpoint, `edges[${index}].id`),
    project_id: requireApiString(edge.project_id, endpoint, `edges[${index}].project_id`),
    worldline_id: edge.worldline_id === undefined || edge.worldline_id === null
      ? undefined
      : requireApiString(edge.worldline_id, endpoint, `edges[${index}].worldline_id`),
    source,
    target,
    source_entity_id: source,
    target_entity_id: target,
    label: requireApiString(edge.label, endpoint, `edges[${index}].label`, { allowEmpty: true }),
    type: requireApiString(edge.type, endpoint, `edges[${index}].type`),
    weight: requireApiNumber(edge.weight, endpoint, `edges[${index}].weight`),
    timeline: optionalGraphMetadataNumber(metadata, 'timeline', endpoint, `edges[${index}].metadata.timeline`),
    evidence_fact_ids: evidenceFactIds,
    metadata,
    created_at: requireApiString(edge.created_at, endpoint, `edges[${index}].created_at`),
    updated_at: requireApiString(edge.updated_at, endpoint, `edges[${index}].updated_at`)
  }
}

function normalizeEntity(value: unknown, index: number, endpoint: string): Entity {
  const entity = requireApiRecord(value, endpoint, `entities[${index}]`)
  return {
    ...(entity as unknown as Partial<Entity>),
    id: requireApiString(entity.id, endpoint, `entities[${index}].id`),
    project_id: requireApiString(entity.project_id, endpoint, `entities[${index}].project_id`),
    name: requireApiString(entity.name, endpoint, `entities[${index}].name`, { allowEmpty: true }),
    type: requireApiString(entity.type, endpoint, `entities[${index}].type`),
    summary: requireApiString(entity.summary, endpoint, `entities[${index}].summary`, { allowEmpty: true }),
    importance: requireApiNumber(entity.importance, endpoint, `entities[${index}].importance`),
    status: requireApiString(entity.status, endpoint, `entities[${index}].status`),
    created_at: requireApiString(entity.created_at, endpoint, `entities[${index}].created_at`),
    updated_at: requireApiString(entity.updated_at, endpoint, `entities[${index}].updated_at`),
    metadata: optionalStringRecord(entity.metadata, endpoint, `entities[${index}].metadata`),
    traits: optionalStringRecord(entity.traits, endpoint, `entities[${index}].traits`)
  }
}

function graphExpansionMismatch(field: string, expected: string | number, actual: string | number, cause: unknown): never {
  const endpoint = 'expandGraph'
  throw new ApiClientError({
    endpoint,
    field,
    kind: 'validation',
    code: 'invalid_api_response',
    message: `invalid_api_response: ${field} ${String(actual)} does not match requested ${String(expected)}`,
    cause
  })
}

function normalizeGraphExpansion(value: unknown, expected?: Pick<GraphExpandRequest, 'project_id' | 'depth'>): GraphExpandResponse {
  const endpoint = 'expandGraph'
  const expansion = requireApiRecord(value, endpoint)
  const projectId = requireApiString(expansion.project_id, endpoint, 'project_id')
  const depth = requireApiNumber(expansion.depth, endpoint, 'depth')
  const generatedAt = requireApiString(expansion.generated_at, endpoint, 'generated_at')
  if (!Number.isInteger(depth) || depth < 1 || depth > 4) {
    throw new ApiClientError({ endpoint, field: 'depth', kind: 'validation', code: 'invalid_api_response', message: 'invalid_api_response: graph expansion depth must be an integer between 1 and 4', cause: depth })
  }
  if (expected && projectId !== expected.project_id) graphExpansionMismatch('project_id', expected.project_id, projectId, expansion)
  if (expected && depth !== expected.depth) graphExpansionMismatch('depth', expected.depth, depth, expansion)
  const entities = requireApiArray(expansion.entities, endpoint, 'entities', (item, index) => normalizeEntity(item, index, endpoint))
  entities.forEach((entity, index) => {
    if (entity.project_id !== projectId) graphExpansionMismatch(`entities[${index}].project_id`, projectId, entity.project_id, entity)
  })
  const nodes = entities.map((entity, index) => entityToNode(entity, index, endpoint))
  const nodeIDs = new Set(nodes.map((node) => node.id))
  const edges = requireApiArray(expansion.edges, endpoint, 'edges', (item, index) => {
    const normalized = normalizeGraphEdge(item, index, endpoint)
    if (normalized.project_id !== projectId) graphExpansionMismatch(`edges[${index}].project_id`, projectId, normalized.project_id, item)
    if (!nodeIDs.has(normalized.source) || !nodeIDs.has(normalized.target)) {
      throw new ApiClientError({ endpoint, field: `edges[${index}]`, kind: 'validation', code: 'invalid_api_response', message: 'invalid_api_response: graph edge endpoint is missing from entities', cause: item })
    }
    return normalized
  })
  return {
    project_id: projectId,
    depth,
    nodes,
    edges,
    facts: requireApiArray(expansion.facts, endpoint, 'facts', (item) => item as Fact),
    generated_at: generatedAt
  }
}

function normalizeIndexJob(value: unknown, index = 0, endpoint = 'indexJob'): IndexJob {
  const job = requireApiRecord(value, endpoint, `jobs[${index}]`)
  return {
    ...(job as unknown as Partial<IndexJob>),
    id: requireApiString(job.id, endpoint, `jobs[${index}].id`),
    project_id: requireApiString(job.project_id, endpoint, `jobs[${index}].project_id`),
    kind: requireApiString(job.kind, endpoint, `jobs[${index}].kind`),
    status: requireApiString(job.status, endpoint, `jobs[${index}].status`),
    attempts: requireApiNumber(job.attempts, endpoint, `jobs[${index}].attempts`),
    created_at: requireApiString(job.created_at, endpoint, `jobs[${index}].created_at`),
    updated_at: requireApiString(job.updated_at, endpoint, `jobs[${index}].updated_at`),
    payload: optionalStringRecord(job.payload, endpoint, `jobs[${index}].payload`)
  }
}

function normalizeSaveChapterVersionResponse(value: unknown): SaveChapterVersionResponse {
  const endpoint = 'createChapterVersion'
  const response = requireApiRecord(value, endpoint)
  return {
    chapter_version: normalizeChapterVersion(response.chapter_version, 0, endpoint),
    index_job: normalizeIndexJob(response.index_job, 0, endpoint)
  }
}

function normalizeCharacterSyncResponse(value: unknown, endpoint = 'syncStoryBibleCharacters'): CharacterSyncResponse {
  const response = requireApiRecord(value, endpoint)
  const characters = requireApiArray(response.characters, endpoint, 'characters', (item, index) => normalizeEntity(item, index, endpoint))
  const mappings = requireApiArray(response.mappings, endpoint, 'mappings', (item, index) => {
    const mapping = requireApiRecord(item, endpoint, `mappings[${index}]`)
    return {
      name: requireApiString(mapping.name, endpoint, `mappings[${index}].name`),
      entity_id: requireApiString(mapping.entity_id, endpoint, `mappings[${index}].entity_id`),
      action: requireApiString(mapping.action, endpoint, `mappings[${index}].action`)
    }
  })
  return {
    project_id: requireApiString(response.project_id, endpoint, 'project_id'),
    story_bible_id: requireApiString(response.story_bible_id, endpoint, 'story_bible_id'),
    characters,
    mappings
  }
}

function isSyncableCharacter(character: StoryBible['characters'][number]) {
  return Boolean(
    character.name.trim()
    && character.role.trim()
    && character.desire.trim()
    && character.wound.trim()
  )
}

function chapterRequestFields(request: ChapterWriteRequest) {
  const metadata = { ...(request.metadata || {}) }
  delete metadata.summary
  return {
    number: request.number,
    title: request.title,
    status: request.status,
    summary: request.summary,
    metadata: Object.keys(metadata).length > 0 ? metadata : undefined
  }
}

function createChapterRequestToBackend(request: CreateChapterRequest): GeneratedApi.CreateChapterRequest {
  return {
    ...chapterRequestFields(request),
    title: requireApiString(request.title, 'createChapter', 'title')
  }
}

function updateChapterRequestToBackend(request: UpdateChapterRequest): GeneratedApi.UpdateChapterRequest {
  return chapterRequestFields(request)
}

function contextSelectionToBackend(selection: AgentRunRequest['context_selection']): BackendContextSelection | undefined {
  if (!selection) return undefined
  const chapterIds = selection.chapter_ids?.map((item) => item.trim()).filter(Boolean)
  const characterIds = selection.character_ids?.map((item) => item.trim()).filter(Boolean)
  const characterNames = selection.character_names?.map((item) => item.trim()).filter(Boolean)
  const body: BackendContextSelection = {
    chapter_ids: chapterIds?.length ? chapterIds : undefined,
    character_ids: characterIds?.length ? characterIds : undefined,
    character_names: characterNames?.length ? characterNames : undefined,
    include_world_rules: selection.include_world_rules || undefined
  }
  return Object.values(body).some((value) => value !== undefined) ? body : undefined
}

function agentRunRequestToBackend(request: AgentRunRequest): AgentRunRequest {
  return {
    ...request,
    context_selection: contextSelectionToBackend(request.context_selection)
  }
}

function mapHealth(response: HealthResponse, copy: ApiCopy): HealthStatus {
  const warnings = []
  if (!response.qdrant_configured) warnings.push(copy.healthWarnings.qdrant)
  if (!response.postgres_configured) warnings.push(copy.healthWarnings.postgres)
  return {
    ok: response.status === 'ok',
    status: response.status,
    time: response.time,
    service: 'aeon-echoes-api',
    version: 'go-backend',
    indexedProjects: response.qdrant_configured ? 1 : 0,
    queueDepth: 0,
    lastHeartbeat: response.time,
    warnings,
    qdrant_configured: response.qdrant_configured,
    postgres_configured: response.postgres_configured
  }
}

export function decodeStoryBibleResponse(value: unknown, endpoint = 'storyBible'): StoryBible {
  return normalizeStoryBible(value, endpoint)
}

export function decodeProjectSummaryResponse(value: unknown, endpoint = 'project'): ProjectSummary {
  return normalizeProject(value as Project, endpoint)
}

export function decodeWorkflowResponse(value: unknown, endpoint = 'workflow'): AIWorkflow {
  return normalizeWorkflow(value, endpoint)
}

export function decodeChapterVersionResponse(value: unknown, endpoint = 'chapterVersion'): ChapterVersion {
  return normalizeChapterVersion(value, 0, endpoint)
}

export function decodeGraphExpansionResponse(value: unknown): GraphExpandResponse {
  return normalizeGraphExpansion(value)
}

export function createApiClient(rawBaseUrl: string, locale?: string): ApiClient {
  configureGeneratedClient(rawBaseUrl)
  const normalizedLocale = normalizeLocale(locale)
  const copy = getApiCopy(normalizedLocale)

  const client: Omit<ApiClient, keyof ApiDomains> = {
    health() {
      return mapApi('getHealth', () => apiSdk.getHealth(), (response) => mapHealth(response, copy))
    },
    systemStatus() {
      return mapApi('getSystemStatus', () => apiSdk.getSystemStatus(), (status) => ({
        status: status.status || 'unknown',
        postgres_configured: Boolean(status.postgres_configured),
        qdrant_configured: Boolean(status.qdrant_configured),
        provider_count: Number(status.provider_count || 0),
        model_count: Number(status.model_count || 0),
        pending_jobs_count: Number(status.pending_jobs_count || 0),
        checked_at: status.checked_at || ''
      }))
    },
    listProviders() {
      return mapApi('listProviders', () => apiSdk.listProviders(), (items) => items.map(normalizeProvider))
    },
    saveProvider(provider, mode) {
      const request = providerToBackend(provider)
      const isExisting = mode === 'edit' || (!mode && Boolean(provider.id && provider.created_at))
      if (isExisting) {
        return mapApi(
          'updateProvider',
          () => apiSdk.updateProvider({ path: { id: requirePathId(request.id || provider.id, 'provider_id') }, body: request }),
          normalizeProvider
        )
      }
      return mapApi('createProvider', () => apiSdk.createProvider({ body: request }), normalizeProvider)
    },
    deleteProvider(id) {
      return mapApi('deleteProvider', () => apiSdk.deleteProvider({ path: { id } }), (status) => ({ status: status.status }))
    },
    listModels(kind) {
      return mapApi('listModels', () => apiSdk.listModels({ query: { kind } }), (items) => items.map(normalizeModel))
    },
    saveModel(model) {
      const isExisting = Boolean(model.id && model.created_at)
      if (isExisting) {
        return mapApi('updateModel', () => apiSdk.updateModel({ path: { id: requirePathId(model.id, 'model_id') }, body: modelToBackend(model) }), normalizeModel)
      }
      return mapApi('createModel', () => apiSdk.createModel({ body: modelToBackend(model) }), normalizeModel)
    },
    deleteModel(id) {
      return mapApi('deleteModel', () => apiSdk.deleteModel({ path: { id } }), (status) => ({ status: status.status }))
    },
    refreshModels(providerId) {
      return mapApi('refreshProviderModels', () => apiSdk.refreshProviderModels({ path: { id: providerId } }), (response) => (response.models || []).map(normalizeModel))
    },
    listSettings(scope) {
      return mapApi('listSettings', () => apiSdk.listSettings({ query: { scope } }), (items) => items.map((item) => item as AppSetting))
    },
    saveSetting(setting) {
      return mapApi(
        'upsertSetting',
        () => apiSdk.upsertSetting({ path: { scope: setting.scope, key: setting.key }, body: setting }),
        (item) => item as AppSetting
      )
    },
    getModelUsageSettings() {
      return mapApi('getModelRouting', () => apiSdk.getModelRouting(), (response) => normalizeModelUsageSettings(response.routes))
    },
    saveModelUsageSettings(settings) {
      const routes = modelUsageKeys.reduce<Partial<ModelUsageSettings>>((items, key) => {
        items[key] = settings[key].trim()
        return items
      }, {})
      return mapApi('putModelRouting', () => apiSdk.putModelRouting({ body: { routes } }), (response) => normalizeModelUsageSettings(response.routes))
    },
    listProjects() {
      return mapApi('listProjects', () => apiSdk.listProjects(), (items) => requireApiArray(items, 'listProjects', 'projects', (project) => normalizeProject(project as Project)))
    },
    initializeProject(seed) {
      return mapApi(
        'createProject',
        () => apiSdk.createProject({ body: projectSeedToBackend(seed, copy) }),
        (value) => {
          const response = requireApiRecord(value, 'createProject')
          return normalizeStoryBible(response.story_bible, 'createProject')
        }
      )
    },
    initializeProjectFull(seed) {
      return mapApi(
        'createProject',
        () => apiSdk.createProject({ body: projectSeedToBackend(seed, copy) }),
        (value) => {
          const response = requireApiRecord(value, 'createProject')
          return {
            project: normalizeProjectEntity(response.project, 'createProject'),
            story_bible: normalizeStoryBible(response.story_bible, 'createProject'),
            workflow: normalizeWorkflow(response.workflow, 'createProject')
          }
        }
      )
    },
    getStoryBible(projectId) {
      return mapApi(
        'getCurrentStoryBible',
        () => apiSdk.getCurrentStoryBible({ path: { projectID: projectId } }),
        (storyBible) => normalizeStoryBible(storyBible, 'getCurrentStoryBible')
      )
    },
    updateStoryBible(projectId, bible) {
      const storyBibleId = requirePathId(bible.id, 'story_bible_id')
      return mapApi(
        'updateStoryBible',
        () => apiSdk.updateStoryBible({ path: { projectID: projectId, storyBibleID: storyBibleId }, body: storyBibleToBackend(bible) }),
        (storyBible) => normalizeStoryBible(storyBible, 'updateStoryBible')
      )
    },
    syncCharacters(projectId, bible) {
      const characters: NonNullable<BackendCharacterSyncRequest['characters']> = (bible.characters || [])
        .filter(isSyncableCharacter)
        .map((character) => {
          const secret = character.secret?.trim()
          const summary = character.summary?.trim()
          return {
            name: character.name.trim(),
            role: character.role.trim(),
            desire: character.desire.trim(),
            wound: character.wound.trim(),
            ...(secret ? { secret } : {}),
            ...(summary ? { summary } : {})
          }
        })
      if (characters.length === 0) {
        const error = new ApiClientError({
          endpoint: 'syncStoryBibleCharacters',
          field: 'characters',
          kind: 'validation',
          code: 'characters_required',
          message: 'characters_required: at least one complete character is required'
        })
        console.error('[AeonEchoes API] Cannot sync an empty character collection', error.state)
        return Promise.reject(error)
      }
      const storyBibleId = requirePathId(bible.id, 'story_bible_id')
      const body: BackendCharacterSyncRequest = {
        story_bible_id: bible.id || undefined,
        characters
      }
      return mapApi(
        'syncStoryBibleCharacters',
        () => apiSdk.syncStoryBibleCharacters({ path: { projectID: projectId, storyBibleID: storyBibleId }, body }),
        (response) => normalizeCharacterSyncResponse(response)
      )
    },
    expandGraph(request) {
      const projectId = requirePathId(request.project_id, 'project_id')
      if (!Number.isInteger(request.depth) || request.depth < 1 || request.depth > 4) {
        return Promise.reject(new ApiClientError({ endpoint: 'expandGraph', field: 'depth', kind: 'validation', code: 'invalid_request', message: 'Graph expansion depth must be an integer between 1 and 4.', cause: request.depth }))
      }
      const entityIds = request.entity_ids?.map((item) => requireApiString(item, 'expandGraph', 'entity_ids').trim()) || []
      return mapApi(
        'expandGraph',
        () => apiSdk.expandGraph({ path: { projectID: projectId }, body: { entity_ids: entityIds.length > 0 ? entityIds : undefined, depth: request.depth } }),
        (expansion) => normalizeGraphExpansion(expansion, { project_id: projectId, depth: request.depth })
      )
    },
    semanticSearch(projectId, request) {
      return mapApi('semanticSearch', () => apiSdk.semanticSearch({ path: { projectID: projectId }, body: request }), (response) => response as SemanticSearchResponse)
    },
    listChapters(projectId) {
      return mapApi(
        'listChapters',
        () => apiSdk.listChapters({ path: { projectID: projectId } }),
        (items) => requireApiArray(items, 'listChapters', 'chapters', (item, index) => normalizeChapter(item, index, 'listChapters'))
      )
    },
    createChapter(projectId, request) {
      const body = createChapterRequestToBackend(request)
      return mapApi(
        'createChapter',
        () => apiSdk.createChapter({ path: { projectID: projectId }, body }),
        (chapter) => normalizeChapter(chapter, 0, 'createChapter')
      )
    },
    updateChapter(projectId, request) {
      const chapterId = requirePathId(request.chapter_id, 'chapter_id')
      const body = updateChapterRequestToBackend(request)
      return mapApi(
        'updateChapter',
        () => apiSdk.updateChapter({ path: { projectID: projectId, chapterID: chapterId }, body }),
        (chapter) => normalizeChapter(chapter, 0, 'updateChapter')
      )
    },
    listChapterVersions(projectId, chapterId) {
      return mapApi(
        'listChapterVersions',
        () => apiSdk.listChapterVersions({ path: { projectID: projectId, chapterID: chapterId } }),
        (items) => requireApiArray(items, 'listChapterVersions', 'versions', (item, index) => normalizeChapterVersion(item, index, 'listChapterVersions'))
      )
    },
    saveChapterVersion(projectId, version) {
      const chapterId = requirePathId(version.chapter_id, 'chapter_id')
      const body: BackendChapterVersionRequest = {
        title: requireApiString(version.title, 'createChapterVersion', 'title'),
        content: requireApiString(version.content, 'createChapterVersion', 'content'),
        author_role: version.author_role,
        summary: version.summary,
        change_note: version.change_note,
        metadata: version.metadata,
        parent_version_id: version.parent_version_id
      }
      return mapApi(
        'createChapterVersion',
        () => apiSdk.createChapterVersion({ path: { projectID: projectId, chapterID: chapterId }, body }),
        (response) => normalizeSaveChapterVersionResponse(response)
      )
    },
    listAgents(options) {
      return mapApi(
        'listAgents',
        () => apiSdk.listAgents({ query: { project_id: options?.projectId, enabled: options?.enabled, limit: options?.limit } }),
        (items) => requireApiArray(items, 'listAgents', 'agents', (item, index) => decodeAgentResponse(item, index, 'listAgents'))
      )
    },
    saveAgent(agent, mode) {
      const isExisting = mode === 'edit' || (!mode && Boolean(agent.id && agent.created_at))
      if (isExisting) {
        return mapApi('updateAgent', () => apiSdk.updateAgent({ path: { id: requirePathId(agent.id, 'agent_id') }, body: agent }), (item) => decodeAgentResponse(item, 0, 'updateAgent'))
      }
      return mapApi('createAgent', () => apiSdk.createAgent({ body: agent }), (item) => decodeAgentResponse(item, 0, 'createAgent'))
    },
    deleteAgent(id) {
      return mapApi('deleteAgent', () => apiSdk.deleteAgent({ path: { id } }), (status) => ({ status: status.status }))
    },
    runAgent(agentId, request) {
      return mapApi('runAgent', () => apiSdk.runAgent({ path: { id: agentId }, body: agentRunRequestToBackend(request) }), (response) => response as AgentRunResult)
    },
    listAgentRuns(options) {
      return mapApi('listAgentRuns', () => apiSdk.listAgentRuns({
        query: { agent_id: options?.agentId, project_id: options?.projectId, status: options?.status, limit: options?.limit }
      }), (items) => items.map((item) => item as AgentRun))
    },
    listSkillSources(options) {
      return mapApi('listSkillSources', () => apiSdk.listSkillSources({
        query: { project_id: options?.projectId, enabled: options?.enabled, limit: options?.limit }
      }), (items) => items.map((item) => item as SkillSource))
    },
    scanDefaultSkillSource() {
      return mapApi('scanDefaultSkillSource', () => apiSdk.scanDefaultSkillSource(), (response) => response as SkillScanResult)
    },
    scanSkillSource(id) {
      return mapApi('scanSkillSource', () => apiSdk.scanSkillSource({ path: { id } }), (response) => response as SkillScanResult)
    },
    listSkills(options) {
      return mapApi('listSkills', () => apiSdk.listSkills({
        query: { project_id: options?.projectId, source_id: options?.sourceId, enabled: options?.enabled, limit: options?.limit }
      }), (items) => items.map((item) => item as Skill))
    },
    saveSkill(skill, mode) {
      const isExisting = mode === 'edit' || (!mode && Boolean(skill.id && skill.created_at))
      const body = skillToBackend(skill, isExisting)
      if (isExisting) {
        return mapApi('updateSkill', () => apiSdk.updateSkill({ path: { id: requirePathId(skill.id, 'skill_id') }, body }), (item) => item as Skill)
      }
      return mapApi('createSkill', () => apiSdk.createSkill({ body }), (item) => item as Skill)
    },
    deleteSkill(id) {
      return mapApi('deleteSkill', () => apiSdk.deleteSkill({ path: { id } }), (status) => ({ status: status.status }))
    },
    setSkillEnabled(id, enabled) {
      return mapApi('patchSkill', () => apiSdk.patchSkill({ path: { id }, body: { enabled } }), (item) => item as Skill)
    },
    listMCPServers(options) {
      return mapApi('listMcpServers', () => apiSdk.listMcpServers({
        query: { project_id: options?.projectId, enabled: options?.enabled, status: options?.status, limit: options?.limit }
      }), (items) => items.map((item) => item as MCPServerConfig))
    },
    saveMCPServer(server, mode) {
      const isExisting = mode === 'edit' || (!mode && Boolean(server.id && server.created_at))
      const body = mcpServerToBackend(server, isExisting)
      if (isExisting) {
        return mapApi('updateMcpServer', () => apiSdk.updateMcpServer({ path: { id: requirePathId(server.id, 'mcp_server_id') }, body }), (item) => item as MCPServerConfig)
      }
      return mapApi('createMcpServer', () => apiSdk.createMcpServer({ body }), (item) => item as MCPServerConfig)
    },
    deleteMCPServer(id) {
      return mapApi('deleteMcpServer', () => apiSdk.deleteMcpServer({ path: { id } }), (status) => ({ status: status.status }))
    },
    setMCPServerEnabled(id, enabled) {
      return mapApi('patchMcpServer', () => apiSdk.patchMcpServer({ path: { id }, body: { enabled } }), (item) => item as MCPServerConfig)
    },
    testMCPServer(id) {
      return mapApi('testMcpServerConnection', () => apiSdk.testMcpServerConnection({ path: { id } }), (response) => ({
        ok: Boolean(response.ok),
        server: response.server as MCPServerConfig
      }))
    },
    refreshMCPTools(id) {
      return mapApi('refreshMcpTools', () => apiSdk.refreshMcpTools({ path: { id } }), (response) => ({
        tools: (response.tools || []).map((item) => item as ToolDefinition),
        count: Number(response.count || 0),
        unavailable: Number(response.unavailable || 0)
      }))
    },
    listMCPServerTools(id) {
      return mapApi('listMcpServerTools', () => apiSdk.listMcpServerTools({ path: { id } }), (items) => items.map((item) => item as ToolDefinition))
    },
    listToolCatalog(options) {
      return mapApi('listTools', () => apiSdk.listTools({
        query: {
          project_id: options?.projectId,
          kind: options?.kind,
          status: options?.status,
          mcp_server_id: options?.mcpServerId,
          source_id: options?.sourceId,
          skill_id: options?.skillId,
          limit: options?.limit
        }
      }), (items) => items.map((item) => item as ToolDefinition))
    },
    setToolEnabled(id, enabled) {
      return mapApi('patchTool', () => apiSdk.patchTool({ path: { id }, body: { enabled } }), (item) => item as ToolDefinition)
    },
    listToolInvocations(options) {
      return mapApi('listToolInvocations', () => apiSdk.listToolInvocations({
        query: {
          agent_run_id: options?.agentRunId,
          agent_id: options?.agentId,
          project_id: options?.projectId,
          tool_id: options?.toolId,
          status: options?.status,
          limit: options?.limit
        }
      }), (items) => items.map((item) => item as ToolInvocation))
    },
    listIndexJobs(options) {
      const query = typeof options === 'string'
        ? { project_id: options }
        : { project_id: options?.projectId, status: options?.status, limit: options?.limit }
      return mapApi('listIndexJobs', () => apiSdk.listIndexJobs({ query }), (items) => requireApiArray(items, 'listIndexJobs', 'jobs', (item, index) => normalizeIndexJob(item, index, 'listIndexJobs')))
    },
    runIndexJob(id) {
      return mapApi('runIndexJob', () => apiSdk.runIndexJob({ path: { id } }), (item) => normalizeIndexJob(item, 0, 'runIndexJob'))
    },
    runPendingIndexJobs(projectId, limit = 10) {
      return mapApi('runPendingIndexJobs', () => apiSdk.runPendingIndexJobs({ query: { project_id: projectId, limit } }), (value) => {
        const response = requireApiRecord(value, 'runPendingIndexJobs')
        return {
          processed: requireApiArray(response.processed, 'runPendingIndexJobs', 'processed', (item, index) => normalizeIndexJob(item, index, 'runPendingIndexJobs')),
          count: requireApiNumber(response.count, 'runPendingIndexJobs', 'count'),
          error: typeof response.error === 'string' ? response.error : undefined
        }
      })
    },
    rebuildVectors() {
      return mapApi('rebuildVectorIndex', () => apiSdk.rebuildVectorIndex(), (response) => response as RebuildVectorsResponse)
    },
    optimizeProjectSeed(seed) {
      return mapApi(
        'optimizeProjectSeed',
        () => apiSdk.optimizeProjectSeed({ body: projectSeedToBackend(seed, copy) }),
        (response) => ({
          ...seed,
          title: response.title || seed.title,
          premise: response.premise || seed.premise,
          genre: response.genre || seed.genre,
          tone: response.tone || seed.tone,
          audience: response.audience || seed.audience,
          language: response.language || seed.language,
          setting: response.setting || seed.setting,
          themes: response.themes || seed.themes,
          main_characters: response.main_characters || seed.main_characters,
          constraints: response.constraints || seed.constraints,
          target_chapters: response.target_chapters || seed.target_chapters,
          metadata: response.metadata || seed.metadata,
          optimized_prompt: response.metadata?.optimized_prompt || seed.optimized_prompt
        })
      )
    }
  }

  const api: ApiClient = {
    ...client,
    project: {
      listProjects: client.listProjects,
      initializeProject: client.initializeProject,
      initializeProjectFull: client.initializeProjectFull,
      optimizeProjectSeed: client.optimizeProjectSeed
    },
    storyBible: {
      getStoryBible: client.getStoryBible,
      updateStoryBible: client.updateStoryBible,
      syncCharacters: client.syncCharacters
    },
    chapter: {
      listChapters: client.listChapters,
      createChapter: client.createChapter,
      updateChapter: client.updateChapter,
      listChapterVersions: client.listChapterVersions,
      saveChapterVersion: client.saveChapterVersion
    },
    graph: {
      expandGraph: client.expandGraph,
      semanticSearch: client.semanticSearch
    },
    model: {
      listModels: client.listModels,
      saveModel: client.saveModel,
      deleteModel: client.deleteModel,
      refreshModels: client.refreshModels,
      getModelUsageSettings: client.getModelUsageSettings,
      saveModelUsageSettings: client.saveModelUsageSettings
    },
    agent: {
      listAgents: client.listAgents,
      saveAgent: client.saveAgent,
      deleteAgent: client.deleteAgent,
      runAgent: client.runAgent,
      listAgentRuns: client.listAgentRuns
    },
    indexJob: {
      listIndexJobs: client.listIndexJobs,
      runIndexJob: client.runIndexJob,
      runPendingIndexJobs: client.runPendingIndexJobs,
      rebuildVectors: client.rebuildVectors
    }
  }

  return api
}
