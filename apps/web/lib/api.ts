import type {
  AIDraftRequest,
  AIDraftResponse,
  CharacterProfileRequest,
  CharacterProfileResponse,
  CharacterSyncResponse,
  ChapterIdeaRequest,
  ChapterIdeaResponse,
  ContextPreviewRequest,
  ContextPreviewResponse,
  ContextSelection,
  DraftResultResponse,
  DraftWithIdeaRequest,
  DraftWithIdeaResponse,
  AIWorkflow,
  ApiErrorState,
  AppSetting,
  ChapterVersion,
  ContextPack,
  Entity,
  Fact,
  GraphEdge,
  GraphExpandRequest,
  GraphExpandResponse,
  GraphExpansion,
  GraphNode,
  HealthStatus,
  IndexFreshness,
  IndexJob,
  InitializeProjectResponse,
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
  StoryBible
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
  query?: Record<string, string | number | boolean | undefined>
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

const MODEL_ROUTING_SCOPE = 'model_routing'
const MODEL_ROUTING_VALUE_KEY = 'model'

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
  api_key_env?: string
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
  provider_type?: ModelConfig['provider_type']
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

type RawDraftResponse = Omit<AIDraftResponse, 'content' | 'warnings' | 'context_pack' | 'index_freshness' | 'model_resolution' | 'continuity_audit'> & {
  content?: string
  warnings?: string[]
  chapter_version?: ChapterVersion
  workflow: AIWorkflow
  context_pack: ContextPack
  index_freshness: IndexFreshness
  model_resolution: ModelResolution
  continuity_audit: AIDraftResponse['continuity_audit']
}

export interface ApiClient {
  health(): Promise<ApiResult<HealthStatus>>
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
  generateCharacterProfiles(request: CharacterProfileRequest): Promise<ApiResult<CharacterProfileResponse>>
  expandGraph(request: GraphExpandRequest): Promise<ApiResult<GraphExpandResponse>>
  listChapterVersions(projectId: string, chapterId: string): Promise<ApiResult<ChapterVersion[]>>
  saveChapterVersion(projectId: string, version: Partial<ChapterVersion>): Promise<ApiResult<SaveChapterVersionResponse>>
  requestAIDraft(request: AIDraftRequest): Promise<ApiResult<AIDraftResponse>>
  requestChapterIdea(request: ChapterIdeaRequest): Promise<ApiResult<ChapterIdeaResponse>>
  previewContextSelection(request: ContextPreviewRequest): Promise<ApiResult<ContextPreviewResponse>>
  requestDraftWithIdea(request: DraftWithIdeaRequest): Promise<ApiResult<DraftWithIdeaResponse>>
  listIndexJobs(projectId?: string): Promise<ApiResult<IndexJob[]>>
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

function buildQuery(query?: RequestOptions['query']) {
  if (!query) return ''
  const search = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    if (value !== undefined) search.set(key, String(value))
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

function projectSeedToBackend(seed: ProjectSeed, copy: ApiCopy) {
  const constraints = [seed.central_conflict, seed.taboos, ...(seed.constraints || [])]
    .map((item) => item.trim())
    .filter(Boolean)
  const mainCharacters = [seed.protagonist, ...(seed.main_characters || [])]
    .map((item) => item.trim())
    .filter(Boolean)

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
    api_key_hint: provider.api_key_hint || provider.api_key_env,
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
  request.api_key_env = provider.api_key_env?.trim() || ''

  return request
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
    provider_type: model.provider_type,
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

function normalizeModelUsageSettings(items: AppSetting[]): ModelUsageSettings {
  const settings = emptyModelUsageSettings()
  items
    .filter((item) => item.scope === MODEL_ROUTING_SCOPE)
    .forEach((item) => {
      if (!modelUsageKeys.includes(item.key as keyof ModelUsageSettings)) return
      const raw = item.value?.[MODEL_ROUTING_VALUE_KEY]
      settings[item.key as keyof ModelUsageSettings] = typeof raw === 'string' ? raw.trim() : ''
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

function normalizeIndexFreshness(indexFreshness: IndexFreshness): IndexFreshness {
  return {
    ...indexFreshness,
    project_id: indexFreshness.project_id || '',
    chapter_id: indexFreshness.chapter_id || '',
    status: indexFreshness.status || 'missing',
    pending_job_count: Number(indexFreshness.pending_job_count || 0)
  }
}

function normalizeContextPack(contextPack: ContextPack): ContextPack {
  return {
    ...contextPack,
    world_rules: contextPack.world_rules || {},
    facts: contextPack.facts || [],
    entities: contextPack.entities || [],
    edges: contextPack.edges || [],
    plot_threads: contextPack.plot_threads || [],
    chapter_summaries: contextPack.chapter_summaries || [],
    tool_trace: contextPack.tool_trace || [],
    metadata: contextPack.metadata || {}
  }
}

function normalizeContinuityEvidenceRef(evidence: AIDraftResponse['continuity_audit']['issues'][number]['evidence'][number]) {
  return {
    ...evidence
  }
}

function normalizeContinuityIssue(issue: AIDraftResponse['continuity_audit']['issues'][number]) {
  return {
    ...issue,
    evidence: issue.evidence.map(normalizeContinuityEvidenceRef)
  }
}

function normalizeContinuityAudit(audit: AIDraftResponse['continuity_audit']): AIDraftResponse['continuity_audit'] {
  return {
    ...audit,
    issues: audit.issues.map(normalizeContinuityIssue)
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

function storyBibleToBackend(bible: StoryBible) {
  const worldRules = (bible.world_rules || []).map((rule) => rule.trim()).filter(Boolean)
  const characters = (bible.characters || []).map((character) => ({
    ...character,
    id: character.id.trim(),
    name: character.name.trim(),
    role: character.role.trim(),
    desire: character.desire.trim(),
    wound: character.wound.trim(),
    secret: character.secret?.trim()
  }))
  const foreshadows = (bible.foreshadows || []).map((item) => ({
    ...item,
    id: item.id.trim(),
    title: item.title.trim(),
    planted_in: item.planted_in.trim(),
    payoff_hint: item.payoff_hint.trim()
  }))
  const chapters = (bible.chapters || []).map((chapter) => ({
    ...chapter,
    id: chapter.id.trim(),
    title: chapter.title.trim(),
    summary: chapter.summary.trim()
  }))
  const sourceSeed = bible.source_seed
  const themes = (bible.themes || []).map((theme) => theme.trim()).filter(Boolean)
  const metadata = {
    ...(sourceSeed?.metadata || {}),
    one_sentence_core: sourceSeed?.one_sentence_core || bible.logline || bible.premise,
    tags: (sourceSeed?.tags || themes).join(','),
    world_background: worldRules.join('\n') || sourceSeed?.world_background || sourceSeed?.setting || '',
    protagonist: characters[0]?.name || sourceSeed?.protagonist || sourceSeed?.main_characters?.[0] || '',
    central_conflict: sourceSeed?.central_conflict || bible.premise,
    style: sourceSeed?.style || bible.tone || '',
    taboos: sourceSeed?.taboos || '',
    story_bible_premise: bible.premise,
    story_bible_world_rules: JSON.stringify(worldRules),
    story_bible_characters: JSON.stringify(characters),
    story_bible_foreshadows: JSON.stringify(foreshadows),
    story_bible_chapters: JSON.stringify(chapters)
  }
  const rules = worldRules.reduce<Record<string, string>>((items, rule, index) => {
    items[`rule_${index + 1}`] = rule
    return items
  }, {})
  const sanitizedSourceSeed = {
    title: sourceSeed?.title || bible.title || '',
    premise: bible.premise,
    genre: bible.genre || sourceSeed?.genre || '',
    tone: bible.tone || sourceSeed?.tone || '',
    audience: bible.audience || sourceSeed?.audience || '',
    language: bible.language || sourceSeed?.language || 'zh-CN',
    setting: sourceSeed?.setting || metadata.world_background,
    themes,
    main_characters: characters.map((character) => character.name).filter(Boolean),
    constraints: sourceSeed?.constraints?.length ? sourceSeed.constraints.map((item) => item.trim()).filter(Boolean) : worldRules,
    target_chapters: chapters.length || sourceSeed?.target_chapters || 1,
    metadata
  }

  return {
    id: '',
    version: 0,
    project_id: bible.project_id,
    title: bible.title || sanitizedSourceSeed.title,
    logline: bible.logline || bible.premise,
    synopsis: bible.synopsis || bible.premise,
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
    approved: Boolean(bible.approved)
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
  const mainCharacters = sourceSeed?.main_characters?.length ? sourceSeed.main_characters : [sourceSeed?.metadata?.protagonist || sourceSeed?.main_characters?.[0]].filter(Boolean) as string[]
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
  return {
    ...bible,
    premise,
    themes: bible.themes || sourceSeed?.themes || [],
    world_rules: bible.world_rules || parseStoryBibleMetadata(sourceSeed, 'story_bible_world_rules', fallbackWorldRules),
    characters: bible.characters || parseStoryBibleMetadata(sourceSeed, 'story_bible_characters', fallbackCharacters),
    foreshadows: bible.foreshadows || parseStoryBibleMetadata(sourceSeed, 'story_bible_foreshadows', fallbackForeshadows),
    chapters: bible.chapters || parseStoryBibleMetadata(sourceSeed, 'story_bible_chapters', generatedChapters)
  }
}

function normalizeProject(project: Project): ProjectSummary {
  return {
    id: project.id,
    title: project.title,
    logline: project.seed?.premise || project.metadata?.logline || project.status,
    tags: project.seed?.themes || (project.seed?.genre ? [project.seed.genre] : []),
    updated_at: project.updated_at,
    bible_status: project.active_story_bible_id ? 'draft' : 'missing',
    chapter_count: project.seed?.target_chapters || 0
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

function summarizeContextPreview(contextPack: ContextPack, summary: string): ContextPreviewResponse['summary'] {
  return summary || [
    `章节摘要 ${contextPack.chapter_summaries?.length || 0} 条`,
    `实体 ${contextPack.entities?.length || 0} 个`,
    `事实 ${contextPack.facts?.length || 0} 条`,
    `情节线 ${contextPack.plot_threads?.length || 0} 条`,
    `世界规则 ${Object.keys(contextPack.world_rules || {}).length} 条`
  ].join('，')
}

function normalizeContextPreviewResponse(response: ContextPreviewResponse): ContextPreviewResponse {
  const contextPack = normalizeContextPack(response.context_pack)
  return {
    ...response,
    context_pack: contextPack,
    summary: summarizeContextPreview(contextPack, response.summary),
    estimated_tokens: Number(response.estimated_tokens || 0),
    index_freshness: normalizeIndexFreshness(response.index_freshness),
    model_resolution: normalizeModelResolution(response.model_resolution)
  }
}

function normalizeChapterIdeaResponse(response: ChapterIdeaResponse): ChapterIdeaResponse {
  return {
    ...response,
    workflow: normalizeWorkflow(response.workflow),
    context_pack: normalizeContextPack(response.context_pack),
    model_resolution: normalizeModelResolution(response.model_resolution)
  }
}

function normalizeDraftResultResponse(response: DraftResultResponse, copy: ApiCopy): DraftResultResponse {
  return {
    ...response,
    workflow: normalizeWorkflow(response.workflow),
    context_pack: normalizeContextPack(response.context_pack),
    chapter_version: normalizeChapterVersion(response.chapter_version, copy),
    index_freshness: normalizeIndexFreshness(response.index_freshness),
    model_resolution: normalizeModelResolution(response.model_resolution),
    continuity_audit: normalizeContinuityAudit(response.continuity_audit)
  }
}

function normalizeDraftWithIdeaResponse(response: DraftWithIdeaResponse, copy: ApiCopy): DraftWithIdeaResponse {
  return {
    chapter_idea: normalizeChapterIdeaResponse(response.chapter_idea),
    draft: normalizeDraftResultResponse(response.draft, copy),
    model_resolution: response.model_resolution ? normalizeModelResolution(response.model_resolution) : undefined
  }
}

function normalizeDraftResponse(response: RawDraftResponse, copy: ApiCopy): AIDraftResponse {
  const chapterVersion = response.chapter_version ? normalizeChapterVersion(response.chapter_version, copy) : undefined
  return {
    ...response,
    workflow: normalizeWorkflow(response.workflow),
    context_pack: normalizeContextPack(response.context_pack),
    chapter_version: chapterVersion,
    content: response.content || chapterVersion?.content || '',
    warnings: response.warnings || [],
    index_job: response.index_job,
    index_freshness: normalizeIndexFreshness(response.index_freshness),
    model_resolution: normalizeModelResolution(response.model_resolution),
    continuity_audit: normalizeContinuityAudit(response.continuity_audit)
  }
}

function normalizeSaveChapterVersionResponse(response: SaveChapterVersionResponse, copy: ApiCopy): SaveChapterVersionResponse {
  return {
    ...response,
    chapter_version: normalizeChapterVersion(response.chapter_version, copy),
    index_job: response.index_job
  }
}

function selectionCharacterLabels(selection?: ContextSelection) {
  return selection?.character_names?.length ? selection.character_names : selection?.character_ids || []
}

function requestWithContextSelection<T extends { selection?: ContextSelection }>(request: T) {
  return {
    ...request,
    context_selection: request.selection,
    selection: undefined
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
      let detail = response.statusText
      try {
        const payload = await response.json()
        detail = typeof payload?.message === 'string' ? payload.message : typeof payload?.error === 'string' ? payload.error : JSON.stringify(payload)
      } catch (error) {
        console.error('Failed to parse API error response', error)
      }
      throw new ApiClientError({
        endpoint,
        status: response.status,
        message: `${response.status} ${detail}`
      })
    }

    return (await response.json()) as T
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

export function createApiClient(baseUrl: string, locale?: string): ApiClient {
  const normalizedLocale = normalizeLocale(locale)
  const copy = getApiCopy(normalizedLocale)

  return {
    health() {
      return requestMapped<HealthResponse, HealthStatus>(baseUrl, '/health', {}, (response) => mapHealth(response, copy))
    },
    listProviders() {
      return requestMapped<ProviderConfig[], ProviderConfig[]>(baseUrl, '/providers', {}, (items) => items.map(normalizeProvider))
    },
    saveProvider(provider, mode) {
      const request = providerToBackend(provider)
      const isExisting = mode === 'edit' || (!mode && Boolean(provider.id && provider.created_at))
      const endpoint = isExisting ? `/providers/${encodeURIComponent(request.id || provider.id.trim())}` : '/providers'
      const method = isExisting ? 'PUT' : 'POST'
      return requestMapped<ProviderConfig, ProviderConfig>(baseUrl, endpoint, { method, body: request }, normalizeProvider)
    },
    deleteProvider(id) {
      return requestResult<ProviderDeleteResponse>(baseUrl, `/providers/${encodeURIComponent(id)}`, { method: 'DELETE' })
    },
    listModels(kind) {
      return requestMapped<ModelConfig[], ModelConfig[]>(baseUrl, '/models', { query: { kind } }, (items) => items.map(normalizeModel))
    },
    saveModel(model) {
      const isExisting = Boolean(model.id && model.created_at)
      const endpoint = isExisting ? `/models/${encodeURIComponent(model.id)}` : '/models'
      const method = isExisting ? 'PUT' : 'POST'
      return requestMapped<ModelConfig, ModelConfig>(baseUrl, endpoint, { method, body: modelToBackend(model) }, normalizeModel)
    },
    deleteModel(id) {
      return requestResult<{ status: string }>(baseUrl, `/models/${encodeURIComponent(id)}`, { method: 'DELETE' })
    },
    refreshModels(providerId) {
      return requestMapped<RefreshModelsResponse, ModelConfig[]>(
        baseUrl,
        `/providers/${encodeURIComponent(providerId)}/refresh-models`,
        { method: 'POST' },
        (response) => response.models.map(normalizeModel)
      )
    },
    listSettings(scope) {
      return requestResult<AppSetting[]>(baseUrl, '/settings', { query: { scope } })
    },
    saveSetting(setting) {
      return requestResult<AppSetting>(baseUrl, `/settings/${encodeURIComponent(setting.scope)}/${encodeURIComponent(setting.key)}`, {
        method: 'PUT',
        body: setting
      })
    },
    async getModelUsageSettings() {
      const result = await requestResult<AppSetting[]>(baseUrl, '/settings', { query: { scope: MODEL_ROUTING_SCOPE } })
      return {
        data: normalizeModelUsageSettings(result.data),
        error: result.error
      }
    },
    async saveModelUsageSettings(settings) {
      const savedSettings = await Promise.all(
        modelUsageKeys.map((key) => requestJson<AppSetting>(baseUrl, `/settings/${encodeURIComponent(MODEL_ROUTING_SCOPE)}/${encodeURIComponent(key)}`, {
          method: 'PUT',
          body: { scope: MODEL_ROUTING_SCOPE, key, value: { [MODEL_ROUTING_VALUE_KEY]: settings[key].trim() } }
        }))
      )
      return {
        data: normalizeModelUsageSettings(savedSettings),
        error: undefined
      }
    },
    listProjects() {
      return requestMapped<Project[], ProjectSummary[]>(baseUrl, '/projects', {}, (items) => items.map(normalizeProject))
    },
    initializeProject(seed) {
      return requestMapped<InitializeProjectResponse, StoryBible>(
        baseUrl,
        '/projects/initialize',
        { method: 'POST', body: projectSeedToBackend(seed, copy) },
        (response) => normalizeStoryBible(response.story_bible, copy)
      )
    },
    initializeProjectFull(seed) {
      return requestMapped<InitializeProjectResponse, InitializeProjectResponse>(
        baseUrl,
        '/projects/initialize',
        { method: 'POST', body: projectSeedToBackend(seed, copy) },
        (response) => ({
          ...response,
          story_bible: normalizeStoryBible(response.story_bible, copy),
          workflow: normalizeWorkflow(response.workflow)
        })
      )
    },
    getStoryBible(projectId) {
      return requestMapped<StoryBible, StoryBible>(baseUrl, `/projects/${projectId}/story-bible`, {}, (storyBible) => normalizeStoryBible(storyBible, copy))
    },
    updateStoryBible(projectId, bible) {
      return requestMapped<StoryBible, StoryBible>(
        baseUrl,
        `/projects/${projectId}/story-bible`,
        { method: 'PUT', body: storyBibleToBackend(bible) },
        (storyBible) => normalizeStoryBible(storyBible, copy)
      )
    },
    syncCharacters(projectId, bible) {
      const characters = (bible.characters || [])
        .map((character) => ({
          id: character.id?.trim(),
          name: character.name.trim(),
          role: character.role.trim(),
          desire: character.desire.trim(),
          wound: character.wound.trim(),
          secret: character.secret?.trim(),
          metadata: character.metadata
        }))
        .filter((character) => character.name)
      return requestMapped<CharacterSyncResponse, CharacterSyncResponse>(
        baseUrl,
        `/projects/${encodeURIComponent(projectId)}/characters/sync`,
        {
          method: 'POST',
          body: {
            story_bible_id: bible.id || undefined,
            characters
          }
        },
        (response) => ({
          ...response,
          story_bible: response.story_bible ? normalizeStoryBible(response.story_bible, copy) : undefined
        })
      )
    },
    generateCharacterProfiles(request) {
      return requestResult<CharacterProfileResponse>(baseUrl, '/ai/character-profiles', { method: 'POST', body: request })
    },
    expandGraph(request) {
      const entityIds = request.entity_ids?.map((item) => item.trim()).filter(Boolean) || []
      return requestMapped<GraphExpansion, GraphExpandResponse>(
        baseUrl,
        '/graph/expand',
        { query: { project_id: request.project_id, entity_ids: entityIds.length > 0 ? entityIds.join(',') : undefined, depth: request.depth } },
        normalizeGraphExpansion
      )
    },
    listChapterVersions(projectId, chapterId) {
      return requestMapped<ChapterVersion[], ChapterVersion[]>(
        baseUrl,
        `/projects/${projectId}/chapter-versions`,
        { query: { chapter_id: chapterId } },
        (items) => items.map((item) => normalizeChapterVersion(item, copy))
      )
    },
    saveChapterVersion(projectId, version) {
      return requestMapped<SaveChapterVersionResponse, SaveChapterVersionResponse>(
        baseUrl,
        `/projects/${projectId}/chapter-versions`,
        {
          method: 'POST',
          body: {
            chapter_id: version.chapter_id,
            title: version.title,
            content: version.content,
            summary: version.summary,
            author_role: version.author_role || 'editor',
            index_status: version.index_status || 'pending',
            metadata: version.metadata
          }
        },
        (response) => normalizeSaveChapterVersionResponse(response, copy)
      )
    },
    requestAIDraft(request) {
      const styleConstraints = request.style_constraints || []
      const selection = request.selection
      const selectedCharacters = selectionCharacterLabels(selection)
      const briefParts = [
        request.brief || request.prompt || '',
        styleConstraints.length ? `${copy.draftRequestLabels.styleConstraints}: ${styleConstraints.join(', ')}` : '',
        request.chapter_idea ? `${copy.draftRequestLabels.selectedChapterPlan}: ${request.chapter_idea}` : '',
        selection?.chapter_ids?.length ? `${copy.draftRequestLabels.selectedChapterSummary}: ${selection.chapter_ids.join(', ')}` : '',
        selection?.include_world_rules ? copy.draftRequestLabels.selectedWorldRules : '',
        selectedCharacters.length ? `${copy.draftRequestLabels.selectedCharacters}: ${selectedCharacters.join(', ')}` : ''
      ]
        .map((item) => item.trim())
        .filter(Boolean)
      return requestMapped<RawDraftResponse, AIDraftResponse>(baseUrl, '/ai/draft', {
        method: 'POST',
        body: {
          project_id: request.project_id,
          chapter_id: request.chapter_id,
          brief: briefParts.join('\n'),
          title: request.title,
          chapter_idea: request.chapter_idea,
          chapter_idea_workflow_id: request.chapter_idea_workflow_id,
          context_selection: request.selection,
          max_output_tokens: request.max_output_tokens
        }

      }, (response) => normalizeDraftResponse(response, copy))
    },
    requestChapterIdea(request) {
      return requestMapped<ChapterIdeaResponse, ChapterIdeaResponse>(baseUrl, '/ai/chapter-idea', {
        method: 'POST',
        body: requestWithContextSelection(request)
      }, normalizeChapterIdeaResponse)
    },
    previewContextSelection(request) {
      return requestMapped<ContextPreviewResponse, ContextPreviewResponse>(baseUrl, '/ai/context-selection/preview', {
        method: 'POST',
        body: {
          project_id: request.project_id,
          chapter_id: request.chapter_id,
          title: request.title,
          brief: request.brief,
          prompt: request.prompt,
          context_selection: request.selection,
          style_constraints: request.style_constraints,
          role: request.role,
          token_budget: request.token_budget
        }
      }, normalizeContextPreviewResponse)
    },
    requestDraftWithIdea(request) {
      return requestMapped<DraftWithIdeaResponse, DraftWithIdeaResponse>(baseUrl, '/ai/draft-with-idea', {
        method: 'POST',
        body: requestWithContextSelection(request)
      }, (response) => normalizeDraftWithIdeaResponse(response, copy))
    },
    listIndexJobs(projectId) {
      return requestResult<IndexJob[]>(baseUrl, '/index/jobs', { query: { project_id: projectId } })
    },
    runIndexJob(id) {
      return requestResult<IndexJob>(baseUrl, `/index/jobs/${id}/run`, { method: 'POST' })
    },
    runPendingIndexJobs(projectId, limit = 10) {
      return requestResult<RunPendingIndexResponse>(baseUrl, '/index/run-pending', { method: 'POST', query: { project_id: projectId, limit } })
    },
    rebuildVectors() {
      return requestResult<RebuildVectorsResponse>(baseUrl, '/index/rebuild-vectors', { method: 'POST' })
    },
    optimizeProjectSeed(seed) {
      return requestMapped<ProjectSeed, ProjectSeed>(
        baseUrl,
        '/projects/seed/optimize',
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
