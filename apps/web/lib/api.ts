import type {
  AgentConfig,
  AgentRun,
  AgentRunRequest,
  AgentRunResult,
  CharacterSyncResponse,
  Chapter,
  EnsureChapterRequest,
  EnsureChapterResponse,
  AIWorkflow,
  ApiErrorState,
  AppSetting,
  ChapterVersion,
  Entity,
  Fact,
  GraphEdge,
  GraphExpandRequest,
  GraphExpandResponse,
  GraphExpansion,
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

export class ApiClientError extends Error {
  readonly state: ApiErrorState

  constructor(state: ApiErrorState) {
    super(state.message)
    this.name = 'ApiClientError'
    this.state = state
  }
}

export interface ApiResult<T> {
  data: T
  error?: ApiErrorState
}

type RequestOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  query?: Record<string, string | number | boolean | undefined | null>
}

type LocaleCode = 'zh-CN' | 'en-US'

type ApiCopy = {
  untitledProject: string
  defaultGenre: string
  defaultTone: string
  defaultAudience: string
  chapterTitle(index: number): string
  defaultChapterSummary: string
  plannedChapterSummary: string
  storyBiblePendingSummary: string
  protagonistRole: string
  supportingRole: string
  characterDesire: string
  characterWound: string
  indexStatus(status?: string): string
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

type HealthResponse = {
  status: string
  time: string
  qdrant_configured: boolean
  postgres_configured: boolean
}

type RefreshModelsResponse = {
  models: ModelConfig[]
  count: number
  provider?: ProviderConfig
}

type ModelRoutingResponse = {
  routes: Partial<ModelUsageSettings>
}

type V1Envelope<T> = {
  data?: T
  meta?: {
    request_id?: string
  }
  page?: {
    count: number
    limit?: number
  }
  error?: V1ErrorPayload
}

type V1ErrorPayload = {
  code?: string
  message?: string
  status?: number
  request_id?: string
  details?: unknown
}

type BackendContextSelection = {
  chapter_ids?: string[]
  character_ids?: string[]
  character_names?: string[]
  include_world_rules?: boolean
}

export const DEFAULT_API_BASE = 'http://localhost:8080/api/v1'

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

type ProviderSaveRequest = {
  id?: string
  name: string
  type: ProviderConfig['provider_type']
  base_url: string
  api_key?: string
  enabled: boolean
  trace_enabled?: boolean
  trace_retention_days?: number
  default_request_timeout_sec?: number
  metadata?: Record<string, string>
}

export type ProviderDeleteResponse = {
  status: string
}

type ModelSaveRequest = {
  id?: string
  provider_id: string
  name: string
  display_name: string
  kind: ModelConfig['kind']
  context_window: number
  max_output_tokens: number
  dimension?: number
  supports_tools: boolean
  supports_streaming: boolean
  default_for_kind: boolean
  enabled: boolean
  routing_weight: number
  allowed_agent_roles?: ModelConfig['allowed_agent_roles']
  metadata?: Record<string, string>
}

type BackendProjectSeedRequest = {
  title: string
  premise: string
  genre: string
  tone: string
  audience: string
  language: string
  setting: string
  themes?: string[]
  main_characters?: string[]
  constraints?: string[]
  target_chapters: number
  metadata?: Record<string, string>
}

type BackendStoryBibleRequest = {
  id: string
  project_id: string
  version: number
  title: string
  logline: string
  synopsis: string
  genre: string
  tone: string
  audience: string
  language: string
  themes: string[]
  rules?: Record<string, string>
  worldline_ids?: string[]
  entity_ids?: string[]
  plot_thread_ids?: string[]
  source_seed: BackendProjectSeedRequest
  genesis_workflow_id?: string
  approved: boolean
  premise: string
  world_rules: string[]
  characters: StoryBible['characters']
  foreshadows: StoryBible['foreshadows']
  chapter_plan: StoryBible['chapters']
}

type BackendCharacterSyncRequest = {
  story_bible_id?: string
  characters: Array<{
    name: string
    role: string
    desire: string
    wound: string
    secret?: string
    summary?: string
  }>
}

type SkillSaveRequest = {
  id?: string
  project_id?: string
  source_id?: string
  name: string
  description?: string
  content?: string
  path?: string
  enabled: boolean
  metadata?: Record<string, string>
}

type MCPServerSaveRequest = {
  id?: string
  project_id?: string
  name: string
  transport: MCPServerConfig['transport']
  status?: MCPServerConfig['status']
  enabled: boolean
  command?: string
  args?: string[]
  url?: string
  headers?: Record<string, string>
  secret_headers?: Record<string, string>
  env?: Record<string, string>
  secret_env?: Record<string, string>
  timeout_sec?: number
  metadata?: Record<string, string>
}

type BackendChapterVersionRequest = {
  id?: string
  title: string
  content: string
  summary?: string
  author_role: NonNullable<ChapterVersion['author_role']>
  source_workflow_id?: string
  index_status: string
  metadata?: Record<string, string>
}

export type AgentListOptions = {
  projectId?: string
  enabled?: boolean
  limit?: number
}

export type AgentRunListOptions = {
  agentId?: string
  projectId?: string
  status?: AgentRun['status']
  limit?: number
}

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

export interface ApiClient {
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
  ensureChapter(projectId: string, request: EnsureChapterRequest): Promise<ApiResult<EnsureChapterResponse>>
  listChapterVersions(projectId: string, chapterId: string): Promise<ApiResult<ChapterVersion[]>>
  saveChapterVersion(projectId: string, version: Partial<ChapterVersion>): Promise<ApiResult<SaveChapterVersionResponse>>
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
      chapterTitle: (index) => (index === 0 ? 'Chapter 1' : `Chapter ${index + 1}`),
      defaultChapterSummary: 'Open the story through the central conflict.',
      plannedChapterSummary: 'Chapter outline pending.',
      storyBiblePendingSummary: 'Story bible created. Synopsis is pending review.',
      protagonistRole: 'Protagonist',
      supportingRole: 'Key character',
      characterDesire: 'Advance the central conflict',
      characterWound: 'To be developed in later chapters',
      indexStatus: (status) => `Index status: ${status || 'unknown'}`,
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
    chapterTitle: (index) => (index === 0 ? '第一章' : `第 ${index + 1} 章`),
    defaultChapterSummary: '从核心冲突切入故事。',
    plannedChapterSummary: '章节大纲待补充。',
    storyBiblePendingSummary: 'Story Bible 已创建，摘要待确认。',
    protagonistRole: '主角',
    supportingRole: '关键角色',
    characterDesire: '推动故事核心冲突',
    characterWound: '将在后续章节中细化',
    indexStatus: (status) => `索引状态：${status || 'unknown'}`,
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

function pathSegment(value: string): string {
  return encodeURIComponent(value)
}

function normalizeApiBase(baseUrl?: string): string {
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
    if (trimmed === '' || trimmed === '/') return '/api/v1'
    console.warn('Using custom API base that does not end with /api/v1', { baseUrl: trimmed, error })
  }

  return trimmed
}

function requirePathId(value: string | undefined, label: string): string {
  const trimmed = value?.trim()
  if (!trimmed) {
    throw new ApiClientError({ endpoint: label, message: `${label} is required` })
  }
  return trimmed
}

function buildQuery(query?: RequestOptions['query']) {
  if (!query) return ''
  const search = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    if (value !== undefined && value !== null) search.set(key, String(value))
  })
  const value = search.toString()
  return value ? `?${value}` : ''
}

function createErrorState(endpoint: string, cause: unknown): ApiErrorState {
  if (cause instanceof ApiClientError) return cause.state
  if (cause instanceof Error) {
    return {
      endpoint,
      message: cause.message,
      cause
    }
  }
  return {
    endpoint,
    message: `API request failed for ${endpoint}`,
    cause
  }
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
    target_chapters: seed.target_chapters || 12,
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

function normalizeProvider(provider: ProviderConfig): ProviderConfig {
  const providerType = provider.provider_type || provider.type || 'openai-responses'
  return {
    ...provider,
    provider_type: providerType,
    type: provider.type || providerType,
    streaming: provider.streaming ?? provider.metadata?.streaming === 'true',
    enabled: provider.enabled ?? true,
    api_key_hint: provider.api_key_hint,
    default_model_id: provider.default_model_id || provider.metadata?.default_model_id,
    last_checked_at: provider.last_checked_at || provider.last_model_refresh_at || provider.updated_at,
    status: provider.status || (provider.enabled ? 'online' : 'unknown')
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

function normalizeModel(model: ModelConfig): ModelConfig {
  return {
    ...model,
    id: model.id || (model.provider_id && model.name ? `${model.provider_id}:${model.name}` : model.name),
    display_name: model.display_name || model.name,
    kind: model.kind || 'text',
    enabled: model.enabled ?? true,
    context_window: model.context_window || 0,
    max_output_tokens: model.max_output_tokens || 0,
    dimension: model.dimension || 0,
    supports_streaming: model.supports_streaming ?? false,
    supports_tools: model.supports_tools ?? false,
    default_for_kind: model.default_for_kind ?? false,
    routing_weight: model.routing_weight || 100,
    allowed_agent_roles: model.allowed_agent_roles || []
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

function normalizeModelResolution(modelResolution: ModelResolution): ModelResolution {
  return {
    ...modelResolution,
    route_key: modelResolution.route_key || '',
    resolution_source: modelResolution.resolution_source || '',
    provider_id: modelResolution.provider_id || '',
    provider_name: modelResolution.provider_name || '',
    provider_type: modelResolution.provider_type || 'openai-responses',
    model_id: modelResolution.model_id || '',
    model_name: modelResolution.model_name || '',
    model_kind: modelResolution.model_kind || 'text'
  }
}

function normalizeWorkflow(workflow: AIWorkflow): AIWorkflow {
  return {
    ...workflow,
    intent: workflow.intent || workflow.kind || 'draft_chapter',
    model_resolution: workflow.model_resolution ? normalizeModelResolution(workflow.model_resolution) : undefined,
    steps: (workflow.steps || []).map((step, index) => ({
      ...step,
      id: step.id || `${step.name || 'step'}-${index}`,
      status: step.status === 'completed' ? 'succeeded' : step.status,
      message: step.message || step.error || step.metadata?.message || step.name
    }))
  }
}

const STORY_BIBLE_METADATA_KEYS = new Set([
  'story_bible_premise',
  'story_bible_world_rules',
  'story_bible_characters',
  'story_bible_foreshadows',
  'story_bible_chapter_plan',
  'story_bible_chapters'
])

function stripStoryBibleMetadata(metadata?: Record<string, string>): Record<string, string> | undefined {
  if (!metadata) return undefined
  const next = Object.entries(metadata).reduce<Record<string, string>>((items, [key, value]) => {
    if (!STORY_BIBLE_METADATA_KEYS.has(key)) items[key] = value
    return items
  }, {})
  return Object.keys(next).length > 0 ? next : undefined
}

function sanitizeStoryBibleChapterPlan(bible: StoryBible): StoryBible['chapters'] {
  const source = bible.chapter_plan?.length ? bible.chapter_plan : bible.chapters
  return (source || []).map((chapter, index) => ({
    id: chapter.id?.trim() || `chapter-${index + 1}`,
    title: chapter.title?.trim() || '',
    status: chapter.status || 'planned',
    summary: chapter.summary?.trim() || ''
  }))
}

function storyBibleToBackend(bible: StoryBible): BackendStoryBibleRequest {
  const worldRules = (bible.world_rules || []).map((rule) => rule.trim()).filter(Boolean)
  const characters = (bible.characters || []).map((character, index) => ({
    ...character,
    id: character.id?.trim() || `character-${index + 1}`,
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
    id: item.id?.trim() || `foreshadow-${index + 1}`,
    title: item.title.trim(),
    planted_in: item.planted_in.trim(),
    payoff_hint: item.payoff_hint.trim(),
    status: item.status || 'planted'
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
    target_chapters: chapterPlan.length || sourceSeed?.target_chapters || 1,
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

function parseStoryBibleMetadata<T>(sourceSeed: ProjectSeed | undefined, key: string, fallback: T): T {
  const raw = sourceSeed?.metadata?.[key]
  if (!raw) return fallback
  try {
    return JSON.parse(raw) as T
  } catch (error) {
    console.error(`Failed to parse story bible metadata ${key}`, error)
    return fallback
  }
}

function normalizeStoryBible(bible: StoryBible, copy: ApiCopy): StoryBible {
  const rules = bible.rules || {}
  const sourceSeed = bible.source_seed
  const generatedChapters = Array.from({ length: Math.max(sourceSeed?.target_chapters || 3, 1) }).map((_, index) => ({
    id: `chapter-${index + 1}`,
    title: copy.chapterTitle(index),
    status: index === 0 ? ('drafting' as const) : ('planned' as const),
    summary: index === 0 ? bible.logline || bible.premise || copy.defaultChapterSummary : copy.plannedChapterSummary
  }))
  const mainCharacters = sourceSeed?.main_characters?.map((name) => name.trim()).filter(Boolean) || []
  const fallbackWorldRules = Object.values(rules)
  const fallbackCharacters = mainCharacters.map((name, index) => ({
    id: `character-${index + 1}`,
    name,
    role: index === 0 ? copy.protagonistRole : copy.supportingRole,
    desire: sourceSeed?.premise || copy.characterDesire,
    wound: copy.characterWound
  }))
  const fallbackForeshadows = Object.entries(rules).slice(0, 3).map(([key, value]) => ({
    id: `rule-${key}`,
    title: key,
    planted_in: 'Story Bible',
    payoff_hint: value,
    status: 'planted' as const
  }))
  const premise = bible.premise || sourceSeed?.metadata?.story_bible_premise || bible.logline || bible.synopsis || sourceSeed?.premise || copy.storyBiblePendingSummary
  const chapterPlan = bible.chapter_plan?.length
    ? bible.chapter_plan
    : bible.chapters?.length
      ? bible.chapters
      : parseStoryBibleMetadata(sourceSeed, 'story_bible_chapter_plan', parseStoryBibleMetadata(sourceSeed, 'story_bible_chapters', generatedChapters))
  return {
    ...bible,
    premise,
    themes: bible.themes || sourceSeed?.themes || [],
    world_rules: bible.world_rules || parseStoryBibleMetadata(sourceSeed, 'story_bible_world_rules', fallbackWorldRules),
    characters: bible.characters || parseStoryBibleMetadata(sourceSeed, 'story_bible_characters', fallbackCharacters),
    foreshadows: bible.foreshadows || parseStoryBibleMetadata(sourceSeed, 'story_bible_foreshadows', fallbackForeshadows),
    chapter_plan: chapterPlan,
    chapters: chapterPlan
  }
}

function normalizeProject(project: Project): ProjectSummary {
  const targetChapters = project.seed?.target_chapters || 0
  const seedTags = project.seed?.themes?.length
    ? project.seed.themes
    : project.seed?.metadata?.tags
      ? project.seed.metadata.tags.split(',').map((tag) => tag.trim()).filter(Boolean)
      : project.seed?.genre
        ? [project.seed.genre]
        : []

  return {
    id: project.id,
    title: project.title,
    slug: project.slug,
    status: project.status,
    logline: project.seed?.premise || project.seed?.metadata?.one_sentence_core || project.metadata?.logline || project.status,
    tags: seedTags,
    seed: project.seed,
    active_story_bible_id: project.active_story_bible_id,
    created_at: project.created_at,
    updated_at: project.updated_at,
    bible_status: project.active_story_bible_id ? 'draft' : 'missing',
    chapter_count: targetChapters,
    target_chapters: targetChapters
  }
}

function normalizeChapter(chapter: Chapter, index: number): Chapter {
  return {
    ...chapter,
    id: chapter.id || `chapter-${chapter.number || index + 1}`,
    project_id: chapter.project_id || '',
    number: Number(chapter.number || index + 1),
    title: chapter.title || '',
    status: chapter.status || 'planned',
    summary: chapter.summary || chapter.metadata?.summary || '',
    metadata: chapter.metadata || {}
  }
}

function normalizeEnsureChapterResponse(response: EnsureChapterResponse | Chapter): EnsureChapterResponse {
  const chapter = 'chapter' in response ? response.chapter : response
  return {
    ...('chapter' in response ? response : {}),
    chapter: normalizeChapter(chapter, Math.max(Number(chapter?.number || 1) - 1, 0)),
    created: 'created' in response ? Boolean(response.created) : false
  }
}

function normalizeChapterVersion(version: ChapterVersion, copy: ApiCopy): ChapterVersion {
  const authorRole = version.author_role || 'writer'
  const wordCount = version.metrics?.word_count || version.content.replace(/\s/g, '').length
  return {
    ...version,
    author_role: authorRole,
    author: version.author || (authorRole === 'writer' ? 'ai' : 'human'),
    change_note: version.change_note || version.metadata?.change_note || version.summary || copy.indexStatus(version.index_status),
    metrics: {
      ...(version.metrics || {}),
      word_count: wordCount
    }
  }
}

function normalizeGraphType(type: string): GraphNode['type'] {
  if (type === 'place' || type === 'location') return 'location'
  if (type === 'object' || type === 'clue') return 'clue'
  if (type === 'concept' || type === 'rule') return 'rule'
  if (type === 'event') return 'event'
  if (type === 'chapter') return 'chapter'
  return 'character'
}

function normalizeGraphStatus(status: string): GraphNode['status'] {
  if (status === 'conflict' || status === 'stable' || status === 'draft' || status === 'resolved') return status
  if (status === 'active' || status === 'canonical') return 'stable'
  if (status === 'deprecated') return 'conflict'
  return 'draft'
}

function entityToNode(entity: Entity, index: number): GraphNode {
  return {
    id: entity.id,
    label: entity.name,
    type: normalizeGraphType(entity.type),
    depth: Number(entity.metadata?.depth || 1),
    timeline: Number(entity.metadata?.timeline || 0),
    status: normalizeGraphStatus(entity.status),
    metadata: {
      summary: entity.summary,
      importance: entity.importance,
      ...(entity.traits || {}),
      ...(entity.metadata || {})
    }
  }
}

function normalizeGraphEdge(edge: GraphEdge): GraphEdge {
  return {
    ...edge,
    source: edge.source || edge.source_entity_id || '',
    target: edge.target || edge.target_entity_id || '',
    timeline: edge.timeline || Number(edge.metadata?.timeline || 0)
  }
}

function normalizeGraphExpansion(expansion: GraphExpansion): GraphExpandResponse {
  return {
    nodes: expansion.entities.map(entityToNode),
    edges: expansion.edges.map(normalizeGraphEdge),
    facts: expansion.facts,
    generated_at: new Date().toISOString()
  }
}

function normalizeSaveChapterVersionResponse(response: SaveChapterVersionResponse, copy: ApiCopy): SaveChapterVersionResponse {
  return {
    ...response,
    chapter_version: normalizeChapterVersion(response.chapter_version, copy),
    index_job: response.index_job
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

function chapterRequestToBackend(request: EnsureChapterRequest): EnsureChapterRequest {
  const metadata = { ...(request.metadata || {}) }
  delete metadata.summary
  const body: EnsureChapterRequest = {
    chapter_id: request.chapter_id,
    number: request.number,
    title: request.title,
    status: request.status,
    summary: request.summary,
    metadata: Object.keys(metadata).length > 0 ? metadata : undefined
  }
  return body
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

function errorMessageFromV1(endpoint: string, status: number, error?: V1ErrorPayload): ApiErrorState {
  const code = error?.code || 'request_error'
  const message = error?.message || 'request failed'
  const responseStatus = error?.status || status
  const requestId = error?.request_id ? ` request_id=${error.request_id}` : ''
  return {
    endpoint,
    status: responseStatus,
    message: `${code} (${responseStatus}): ${message}${requestId}`,
    cause: error?.details
  }
}

async function parseV1Payload<T>(response: Response, endpoint: string): Promise<T> {
  const payload = await response.json() as V1Envelope<T>
  if (payload?.error) {
    throw new ApiClientError(errorMessageFromV1(endpoint, response.status, payload.error))
  }
  if (!Object.prototype.hasOwnProperty.call(payload, 'data')) {
    throw new ApiClientError({
      endpoint,
      status: response.status,
      message: `invalid_v1_envelope (${response.status}): missing data`
    })
  }
  return payload.data as T
}

async function requestJson<T>(baseUrl: string, endpoint: string, options: RequestOptions = {}): Promise<T> {
  const url = `${baseUrl.replace(/\/$/, '')}${endpoint}${buildQuery(options.query)}`
  try {
    const response = await fetch(url, {
      method: options.method || 'GET',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json'
      },
      body: options.body === undefined ? undefined : JSON.stringify(options.body)
    })

    if (!response.ok) {
      try {
        await parseV1Payload<T>(response, endpoint)
      } catch (cause) {
        if (cause instanceof ApiClientError) throw cause
        console.error('Failed to parse v1 API error response', cause)
      }
      throw new ApiClientError(errorMessageFromV1(endpoint, response.status, {
        code: 'request_error',
        message: response.statusText || 'request failed',
        status: response.status
      }))
    }

    return await parseV1Payload<T>(response, endpoint)
  } catch (cause) {
    if (cause instanceof ApiClientError) throw cause
    const error = createErrorState(endpoint, cause)
    console.error(`[AeonEchoes API] ${error.message}`, error)
    throw new ApiClientError(error)
  }
}

async function requestResult<T>(baseUrl: string, endpoint: string, options: RequestOptions = {}): Promise<ApiResult<T>> {
  return {
    data: await requestJson<T>(baseUrl, endpoint, options)
  }
}

async function requestMapped<TRaw, TData>(
  baseUrl: string,
  endpoint: string,
  options: RequestOptions,
  mapData: (raw: TRaw) => TData
): Promise<ApiResult<TData>> {
  return {
    data: mapData(await requestJson<TRaw>(baseUrl, endpoint, options))
  }
}

export function createApiClient(rawBaseUrl: string, locale?: string): ApiClient {
  const baseUrl = normalizeApiBase(rawBaseUrl)
  const normalizedLocale = normalizeLocale(locale)
  const copy = getApiCopy(normalizedLocale)

  return {
    health() {
      return requestMapped<HealthResponse, HealthStatus>(baseUrl, '/health', {}, (response) => mapHealth(response, copy))
    },
    systemStatus() {
      return requestResult<SystemStatus>(baseUrl, '/system/status')
    },
    listProviders() {
      return requestMapped<ProviderConfig[], ProviderConfig[]>(baseUrl, '/providers', {}, (items) => items.map(normalizeProvider))
    },
    saveProvider(provider, mode) {
      const request = providerToBackend(provider)
      const isExisting = mode === 'edit' || (!mode && Boolean(provider.id && provider.created_at))
      const endpoint = isExisting ? `/providers/${pathSegment(request.id || provider.id.trim())}` : '/providers'
      const method = isExisting ? 'PUT' : 'POST'
      return requestMapped<ProviderConfig, ProviderConfig>(baseUrl, endpoint, { method, body: request }, normalizeProvider)
    },
    deleteProvider(id) {
      return requestResult<ProviderDeleteResponse>(baseUrl, `/providers/${pathSegment(id)}`, { method: 'DELETE' })
    },
    listModels(kind) {
      return requestMapped<ModelConfig[], ModelConfig[]>(baseUrl, '/models', { query: { 'filter[kind]': kind } }, (items) => items.map(normalizeModel))
    },
    saveModel(model) {
      const isExisting = Boolean(model.id && model.created_at)
      const endpoint = isExisting ? `/models/${pathSegment(model.id)}` : '/models'
      const method = isExisting ? 'PUT' : 'POST'
      return requestMapped<ModelConfig, ModelConfig>(baseUrl, endpoint, { method, body: modelToBackend(model) }, normalizeModel)
    },
    deleteModel(id) {
      return requestResult<{ status: string }>(baseUrl, `/models/${pathSegment(id)}`, { method: 'DELETE' })
    },
    refreshModels(providerId) {
      return requestMapped<RefreshModelsResponse, ModelConfig[]>(
        baseUrl,
        `/providers/${pathSegment(providerId)}/model-refreshes`,
        { method: 'POST' },
        (response) => response.models.map(normalizeModel)
      )
    },
    listSettings(scope) {
      return requestResult<AppSetting[]>(baseUrl, '/settings', { query: { scope } })
    },
    saveSetting(setting) {
      return requestResult<AppSetting>(baseUrl, `/settings/${pathSegment(setting.scope)}/${pathSegment(setting.key)}`, {
        method: 'PUT',
        body: setting
      })
    },
    async getModelUsageSettings() {
      const result = await requestResult<ModelRoutingResponse>(baseUrl, '/model-routing')
      return {
        data: normalizeModelUsageSettings(result.data.routes),
        error: result.error
      }
    },
    async saveModelUsageSettings(settings) {
      const routes = modelUsageKeys.reduce<Partial<ModelUsageSettings>>((items, key) => {
        items[key] = settings[key].trim()
        return items
      }, {})
      const result = await requestResult<ModelRoutingResponse>(baseUrl, '/model-routing', {
        method: 'PUT',
        body: { routes }
      })
      return {
        data: normalizeModelUsageSettings(result.data.routes),
        error: result.error
      }
    },
    listProjects() {
      return requestMapped<Project[], ProjectSummary[]>(baseUrl, '/projects', {}, (items) => items.map(normalizeProject))
    },
    initializeProject(seed) {
      return requestMapped<InitializeProjectResponse, StoryBible>(
        baseUrl,
        '/projects',
        { method: 'POST', body: projectSeedToBackend(seed, copy) },
        (response) => normalizeStoryBible(response.story_bible, copy)
      )
    },
    initializeProjectFull(seed) {
      return requestMapped<InitializeProjectResponse, InitializeProjectResponse>(
        baseUrl,
        '/projects',
        { method: 'POST', body: projectSeedToBackend(seed, copy) },
        (response) => ({
          ...response,
          story_bible: normalizeStoryBible(response.story_bible, copy),
          workflow: normalizeWorkflow(response.workflow)
        })
      )
    },
    getStoryBible(projectId) {
      return requestMapped<StoryBible, StoryBible>(baseUrl, `/projects/${pathSegment(projectId)}/story-bibles/current`, {}, (storyBible) => normalizeStoryBible(storyBible, copy))
    },
    updateStoryBible(projectId, bible) {
      const storyBibleId = requirePathId(bible.id, 'story_bible_id')
      return requestMapped<StoryBible, StoryBible>(
        baseUrl,
        `/projects/${pathSegment(projectId)}/story-bibles/${pathSegment(storyBibleId)}`,
        { method: 'PUT', body: storyBibleToBackend(bible) },
        (storyBible) => normalizeStoryBible(storyBible, copy)
      )
    },
    syncCharacters(projectId, bible) {
      const characters: BackendCharacterSyncRequest['characters'] = (bible.characters || [])
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
        return Promise.resolve({
          data: {
            project_id: projectId,
            story_bible_id: bible.id || '',
            characters: [],
            mappings: []
          }
        })
      }
      const body: BackendCharacterSyncRequest = {
        story_bible_id: bible.id || undefined,
        characters
      }
      return requestResult<CharacterSyncResponse>(
        baseUrl,
        `/projects/${pathSegment(projectId)}/story-bibles/${pathSegment(requirePathId(bible.id, 'story_bible_id'))}/character-syncs`,
        {
          method: 'POST',
          body
        }
      )
    },
    expandGraph(request) {
      const projectId = requirePathId(request.project_id, 'project_id')
      const entityIds = request.entity_ids?.map((item) => item.trim()).filter(Boolean) || []
      return requestMapped<GraphExpansion, GraphExpandResponse>(
        baseUrl,
        `/projects/${pathSegment(projectId)}/graph/expansions`,
        { method: 'POST', body: { entity_ids: entityIds.length > 0 ? entityIds : undefined, depth: request.depth } },
        normalizeGraphExpansion
      )
    },
    semanticSearch(projectId, request) {
      return requestResult<SemanticSearchResponse>(baseUrl, `/projects/${pathSegment(projectId)}/retrieval/semantic-searches`, {
        method: 'POST',
        body: request
      })
    },
    listChapters(projectId) {
      return requestMapped<Chapter[], Chapter[]>(
        baseUrl,
        `/projects/${pathSegment(projectId)}/chapters`,
        {},
        (items) => items.map(normalizeChapter)
      )
    },
    ensureChapter(projectId, request) {
      const chapterId = request.chapter_id?.trim()
      const endpoint = chapterId
        ? `/projects/${pathSegment(projectId)}/chapters/${pathSegment(chapterId)}`
        : `/projects/${pathSegment(projectId)}/chapters`
      return requestMapped<EnsureChapterResponse | Chapter, EnsureChapterResponse>(
        baseUrl,
        endpoint,
        { method: chapterId ? 'PUT' : 'POST', body: chapterRequestToBackend(request) },
        normalizeEnsureChapterResponse
      )
    },
    listChapterVersions(projectId, chapterId) {
      return requestMapped<ChapterVersion[], ChapterVersion[]>(
        baseUrl,
        `/projects/${pathSegment(projectId)}/chapters/${pathSegment(chapterId)}/versions`,
        {},
        (items) => items.map((item) => normalizeChapterVersion(item, copy))
      )
    },
    saveChapterVersion(projectId, version) {
      const chapterId = requirePathId(version.chapter_id, 'chapter_id')
      const body: BackendChapterVersionRequest = {
        id: version.id,
        title: version.title || '',
        content: version.content || '',
        summary: version.summary,
        author_role: version.author_role || 'editor',
        source_workflow_id: version.source_workflow_id,
        index_status: version.index_status || 'pending',
        metadata: version.metadata
      }
      return requestMapped<SaveChapterVersionResponse, SaveChapterVersionResponse>(
        baseUrl,
        `/projects/${pathSegment(projectId)}/chapters/${pathSegment(chapterId)}/versions`,
        {
          method: 'POST',
          body
        },
        (response) => normalizeSaveChapterVersionResponse(response, copy)
      )
    },
    listAgents(options) {
      return requestResult<AgentConfig[]>(baseUrl, '/agents', {
        query: { project_id: options?.projectId, enabled: options?.enabled, limit: options?.limit }
      })
    },
    saveAgent(agent, mode) {
      const isExisting = mode === 'edit' || (!mode && Boolean(agent.id && agent.created_at))
      const endpoint = isExisting ? `/agents/${pathSegment(agent.id)}` : '/agents'
      const method = isExisting ? 'PUT' : 'POST'
      return requestResult<AgentConfig>(baseUrl, endpoint, { method, body: agent })
    },
    deleteAgent(id) {
      return requestResult<{ status: string }>(baseUrl, `/agents/${pathSegment(id)}`, { method: 'DELETE' })
    },
    runAgent(agentId, request) {
      return requestResult<AgentRunResult>(baseUrl, `/agents/${pathSegment(agentId)}/runs`, { method: 'POST', body: agentRunRequestToBackend(request) })
    },
    listAgentRuns(options) {
      return requestResult<AgentRun[]>(baseUrl, '/agent-runs', {
        query: { agent_id: options?.agentId, project_id: options?.projectId, status: options?.status, limit: options?.limit }
      })
    },
    listSkillSources(options) {
      return requestResult<SkillSource[]>(baseUrl, '/skill-sources', {
        query: { project_id: options?.projectId, enabled: options?.enabled, limit: options?.limit }
      })
    },
    scanDefaultSkillSource() {
      return requestResult<SkillScanResult>(baseUrl, '/skill-sources/default/scans', { method: 'POST' })
    },
    scanSkillSource(id) {
      return requestResult<SkillScanResult>(baseUrl, `/skill-sources/${pathSegment(id)}/scans`, { method: 'POST' })
    },
    listSkills(options) {
      return requestResult<Skill[]>(baseUrl, '/skills', {
        query: { project_id: options?.projectId, source_id: options?.sourceId, enabled: options?.enabled, limit: options?.limit }
      })
    },
    saveSkill(skill, mode) {
      const isExisting = mode === 'edit' || (!mode && Boolean(skill.id && skill.created_at))
      const endpoint = isExisting ? `/skills/${pathSegment(skill.id)}` : '/skills'
      const method = isExisting ? 'PUT' : 'POST'
      return requestResult<Skill>(baseUrl, endpoint, { method, body: skillToBackend(skill, isExisting) })
    },
    deleteSkill(id) {
      return requestResult<{ status: string }>(baseUrl, `/skills/${pathSegment(id)}`, { method: 'DELETE' })
    },
    setSkillEnabled(id, enabled) {
      return requestResult<Skill>(baseUrl, `/skills/${pathSegment(id)}`, { method: 'PATCH', body: { enabled } })
    },
    listMCPServers(options) {
      return requestResult<MCPServerConfig[]>(baseUrl, '/mcp-servers', {
        query: { project_id: options?.projectId, enabled: options?.enabled, status: options?.status, limit: options?.limit }
      })
    },
    saveMCPServer(server, mode) {
      const isExisting = mode === 'edit' || (!mode && Boolean(server.id && server.created_at))
      const endpoint = isExisting ? `/mcp-servers/${pathSegment(server.id)}` : '/mcp-servers'
      const method = isExisting ? 'PUT' : 'POST'
      return requestResult<MCPServerConfig>(baseUrl, endpoint, { method, body: mcpServerToBackend(server, isExisting) })
    },
    deleteMCPServer(id) {
      return requestResult<{ status: string }>(baseUrl, `/mcp-servers/${pathSegment(id)}`, { method: 'DELETE' })
    },
    setMCPServerEnabled(id, enabled) {
      return requestResult<MCPServerConfig>(baseUrl, `/mcp-servers/${pathSegment(id)}`, { method: 'PATCH', body: { enabled } })
    },
    testMCPServer(id) {
      return requestResult<{ ok: boolean; server: MCPServerConfig }>(baseUrl, `/mcp-servers/${pathSegment(id)}/connection-tests`, { method: 'POST' })
    },
    refreshMCPTools(id) {
      return requestResult<{ tools: ToolDefinition[]; count: number; unavailable: number }>(baseUrl, `/mcp-servers/${pathSegment(id)}/tool-refreshes`, { method: 'POST' })
    },
    listMCPServerTools(id) {
      return requestResult<ToolDefinition[]>(baseUrl, `/mcp-servers/${pathSegment(id)}/tools`)
    },
    listToolCatalog(options) {
      return requestResult<ToolDefinition[]>(baseUrl, '/tools', {
        query: {
          project_id: options?.projectId,
          kind: options?.kind,
          status: options?.status,
          mcp_server_id: options?.mcpServerId,
          source_id: options?.sourceId,
          skill_id: options?.skillId,
          limit: options?.limit
        }
      })
    },
    setToolEnabled(id, enabled) {
      return requestResult<ToolDefinition>(baseUrl, `/tools/${pathSegment(id)}`, { method: 'PATCH', body: { enabled } })
    },
    listToolInvocations(options) {
      return requestResult<ToolInvocation[]>(baseUrl, '/tool-invocations', {
        query: {
          agent_run_id: options?.agentRunId,
          agent_id: options?.agentId,
          project_id: options?.projectId,
          tool_id: options?.toolId,
          status: options?.status,
          limit: options?.limit
        }
      })
    },
    listIndexJobs(options) {
      const query = typeof options === 'string'
        ? { project_id: options }
        : { project_id: options?.projectId, status: options?.status, limit: options?.limit }
      return requestResult<IndexJob[]>(baseUrl, '/index-jobs', { query })
    },
    runIndexJob(id) {
      return requestResult<IndexJob>(baseUrl, `/index-jobs/${pathSegment(id)}/runs`, { method: 'POST' })
    },
    runPendingIndexJobs(projectId, limit = 10) {
      return requestResult<RunPendingIndexResponse>(baseUrl, '/index-runs', { method: 'POST', query: { project_id: projectId, limit } })
    },
    rebuildVectors() {
      return requestResult<RebuildVectorsResponse>(baseUrl, '/vector-index-rebuilds', { method: 'POST' })
    },
    optimizeProjectSeed(seed) {
      return requestMapped<ProjectSeed, ProjectSeed>(
        baseUrl,
        '/project-seed-optimizations',
        { method: 'POST', body: projectSeedToBackend(seed, copy) },
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
}
