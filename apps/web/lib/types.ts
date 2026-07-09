export type ProviderType = 'openai-responses' | 'openai' | 'anthropic' | 'gemini'
export type ModelKind = 'text' | 'embedding'
export type AgentRole =
  | 'genesis-optimizer'
  | 'plot-architect'
  | 'world-builder'
  | 'character-keeper'
  | 'continuity-auditor'
  | 'writer'
  | 'editor'
  | 'fact-extractor'
  | 'graph-curator'
export type ModelUsageKey = AgentRole | 'embedding'
export type ModelUsageSettings = Record<ModelUsageKey, string>

export interface AppSetting {
  scope: string
  key: string
  value: Record<string, unknown>
  updated_at?: string
}

export interface ApiErrorState {
  message: string
  endpoint: string
  status?: number
  cause?: unknown
}

export interface HealthStatus {
  ok: boolean
  service: string
  version: string
  indexedProjects: number
  queueDepth: number
  lastHeartbeat: string
  warnings: string[]
  qdrant_configured?: boolean
  postgres_configured?: boolean
  status?: string
  time?: string
}

export interface SystemStatus {
  status: string
  postgres_configured: boolean
  qdrant_configured: boolean
  provider_count: number
  model_count: number
  pending_jobs_count: number
  checked_at: string
}

export interface ProviderConfig {
  id: string
  name: string
  provider_type: ProviderType
  type?: ProviderType
  base_url: string
  api_key?: string
  api_key_hint?: string
  streaming: boolean
  enabled: boolean
  trace_enabled?: boolean
  trace_retention_days?: number
  default_request_timeout_sec?: number
  default_model_id?: string
  metadata?: Record<string, string>
  created_at?: string
  updated_at?: string
  last_checked_at?: string
  last_model_refresh_at?: string
  status: 'online' | 'degraded' | 'offline' | 'unknown'
}

export interface ModelConfig {
  id: string
  provider_id: string
  provider_type?: ProviderType
  name: string
  display_name: string
  kind?: ModelKind
  context_window?: number
  max_output_tokens?: number
  dimension?: number
  supports_tools?: boolean
  supports_streaming?: boolean
  default_for_kind?: boolean
  routing_weight?: number
  allowed_agent_roles?: AgentRole[]
  metadata?: Record<string, string>
  created_at?: string
  updated_at?: string
  last_seen_at?: string
  enabled: boolean
}

export interface ProjectSeed {
  title?: string
  premise?: string
  genre?: string
  tone?: string
  audience?: string
  language?: string
  setting?: string
  themes?: string[]
  main_characters?: string[]
  constraints?: string[]
  target_chapters?: number
  metadata?: Record<string, string>
  one_sentence_core: string
  tags: string[]
  world_background: string
  protagonist: string
  central_conflict: string
  style: string
  taboos: string
  optimized_prompt?: string
}

export interface Project {
  id: string
  title: string
  slug: string
  status: string
  seed: ProjectSeed
  active_story_bible_id?: string
  default_worldline_id?: string
  metadata?: Record<string, string>
  created_at: string
  updated_at: string
}

export type ChapterStatus = 'planned' | 'drafting' | 'reviewing' | 'locked' | string

export interface StoryBibleChapter {
  id: string
  title: string
  status: ChapterStatus
  summary: string
}

export interface Chapter extends StoryBibleChapter {
  project_id: string
  number: number
  metadata?: Record<string, string>
  created_at?: string
  updated_at?: string
}

export interface EnsureChapterRequest {
  chapter_id?: string
  number?: number
  title?: string
  status?: ChapterStatus
  summary?: string
  metadata?: Record<string, string>
}

export interface EnsureChapterResponse {
  chapter: Chapter
  created: boolean
  requested_chapter_id?: string
}

export interface StoryBible {
  id: string
  project_id: string
  version?: number
  title?: string
  logline?: string
  synopsis?: string
  genre?: string
  tone?: string
  audience?: string
  language?: string
  rules?: Record<string, string>
  worldline_ids?: string[]
  entity_ids?: string[]
  plot_thread_ids?: string[]
  source_seed?: ProjectSeed
  genesis_workflow_id?: string
  approved?: boolean
  created_at?: string
  premise: string
  themes: string[]
  world_rules: string[]
  characters: Array<{
    id: string
    name: string
    role: string
    desire: string
    wound: string
    secret?: string
    summary?: string
    entity_id?: string
    sync_status?: 'pending' | 'synced' | 'failed' | string
    synced_at?: string
    metadata?: Record<string, string>
  }>
  foreshadows: Array<{
    id: string
    title: string
    planted_in: string
    payoff_hint: string
    status: 'planted' | 'active' | 'paid_off'
  }>
  chapters: StoryBibleChapter[]
  chapter_plan?: StoryBibleChapter[]
}

export interface ProjectSummary {
  id: string
  title: string
  slug?: string
  status?: string
  logline: string
  tags: string[]
  seed?: ProjectSeed
  active_story_bible_id?: string
  created_at?: string
  updated_at: string
  bible_status: 'missing' | 'draft' | 'ready'
  chapter_count: number
  target_chapters?: number
}

export interface Entity {
  id: string
  project_id: string
  worldline_id?: string
  name: string
  type: string
  aliases?: string[]
  summary: string
  traits?: Record<string, string>
  importance: number
  status: string
  metadata?: Record<string, string>
  created_at: string
  updated_at: string
}

export interface Fact {
  id: string
  project_id: string
  worldline_id?: string
  entity_id?: string
  chapter_id?: string
  chapter_version_id?: string
  claim: string
  source: string
  confidence: number
  status: string
  embedding_ref?: string
  metadata?: Record<string, string>
  created_at: string
  updated_at: string
}

export interface GraphNode {
  id: string
  label: string
  type: 'story_start' | 'character' | 'location' | 'event' | 'clue' | 'chapter' | 'rule'
  depth: number
  timeline: number
  status: 'stable' | 'draft' | 'conflict' | 'resolved'
  metadata: Record<string, string | number | boolean | string[]>
}

export interface GraphEdge {
  id: string
  project_id?: string
  worldline_id?: string
  source: string
  target: string
  source_entity_id?: string
  target_entity_id?: string
  label: string
  type: 'causes' | 'reveals' | 'depends_on' | 'appears_in' | 'contradicts' | 'foreshadows' | string
  weight: number
  timeline: number
  evidence_fact_ids?: string[]
  metadata?: Record<string, string | number | boolean>
  created_at?: string
  updated_at?: string
}

export interface PlotThread {
  id: string
  project_id: string
  worldline_id?: string
  title: string
  summary: string
  status: string
  priority: number
  related_entity_ids?: string[]
  opened_chapter_id?: string
  closed_chapter_id?: string
  metadata?: Record<string, string>
  created_at: string
  updated_at: string
}

export interface GraphExpandRequest {
  project_id: string
  root?: string
  depth: number
  timeline?: number
  filters?: string[]
  entity_ids?: string[]
}

export interface GraphExpansion {
  project_id: string
  depth: number
  entities: Entity[]
  edges: GraphEdge[]
  facts: Fact[]
}

export interface GraphExpandResponse {
  nodes: GraphNode[]
  edges: GraphEdge[]
  facts?: Fact[]
  generated_at: string
}

export interface SemanticSearchRequest {
  query: string
  limit?: number
  filters?: Record<string, string>
}

export interface SemanticSearchItem {
  source_id: string
  score: number
  payload?: Record<string, unknown>
}

export interface SemanticSearchResponse {
  query: string
  project_id: string
  items: SemanticSearchItem[]
}

export interface ChapterVersion {
  id: string
  project_id: string
  chapter_id: string
  version: number
  title: string
  content: string
  summary?: string
  author_role?: AgentRole
  source_workflow_id?: string
  index_status?: string
  metadata?: Record<string, string>
  created_at: string
  author: 'human' | 'ai'
  change_note: string
  metrics?: {
    word_count?: number
  }
}

export type IndexJobStatus = 'pending' | 'running' | 'completed' | 'failed' | string

export interface IndexJobListOptions {
  projectId?: string
  status?: IndexJobStatus
  limit?: number
}

export interface IndexJob {
  id: string
  project_id: string
  chapter_id?: string
  chapter_version_id?: string
  kind: string
  status: IndexJobStatus
  attempts: number
  error?: string
  payload?: Record<string, string>
  created_at: string
  updated_at: string
  scheduled_at?: string
  started_at?: string
  completed_at?: string
}

export type ToolTrace = string | {
  tool?: string
  name?: string
  status?: string
  chapter_id?: string
  chapter_ids?: string[]
  character_id?: string
  character_ids?: string[]
  entity_id?: string
  entity_ids?: string[]
  event_id?: string
  event_ids?: string[]
  timeline?: string | number
  depth?: number
  count?: number
  message?: string
  input?: Record<string, unknown>
  output?: Record<string, unknown>
  metadata?: Record<string, string | number | boolean | string[] | undefined>
}

export interface AIWorkflowStep {
  id: string
  name: string
  status: 'idle' | 'running' | 'succeeded' | 'failed' | 'completed' | string
  message: string
  updated_at?: string
  started_at?: string
  ended_at?: string
  error?: string
  metadata?: Record<string, string>
}

export interface AIWorkflow {
  id: string
  project_id: string
  chapter_id?: string
  intent: 'optimize_seed' | 'draft_chapter' | 'reflect' | 'expand_graph' | 'refresh_models' | string
  kind?: string
  role?: AgentRole
  status: 'idle' | 'running' | 'succeeded' | 'failed' | 'completed' | string
  model_id?: string
  context_pack_id?: string
  model_resolution?: ModelResolution
  steps: AIWorkflowStep[]
  input?: Record<string, string>
  output?: Record<string, string>
  error?: ApiErrorState | string
  created_at?: string
  updated_at?: string
}

export interface AgentConfig {
  id: string
  project_id?: string
  name: string
  description?: string
  role?: AgentRole
  model_id?: string
  enabled: boolean
  system_prompt?: string
  skill_ids?: string[]
  tool_ids?: string[]
  mcp_server_ids?: string[]
  memory_policy?: Record<string, unknown>
  runtime_options?: Record<string, unknown>
  metadata?: Record<string, string>
  created_at?: string
  updated_at?: string
}

export type AgentRunStatus = 'running' | 'completed' | 'failed' | string

export interface AgentRun {
  id: string
  agent_id: string
  project_id?: string
  status: AgentRunStatus
  input?: Record<string, unknown>
  output?: Record<string, unknown>
  error?: string
  tool_invocation_ids?: string[]
  started_at?: string
  completed_at?: string
  created_at?: string
  updated_at?: string
}

export interface AgentRunRequest {
  project_id?: string
  task_type?: string
  input?: Record<string, unknown>
  context_selection?: ContextSelection
  max_output_tokens?: number
}

export interface AgentRunResult {
  run: AgentRun
  content: string
  tool_trace?: ToolTrace[]
  model_resolution: ModelResolution
}

export type SkillSourceType = 'inline_text' | 'directory' | string

export interface SkillSource {
  id: string
  project_id?: string
  name: string
  type: SkillSourceType
  path?: string
  inline_text?: string
  enabled: boolean
  metadata?: Record<string, string>
  created_at?: string
  updated_at?: string
}

export interface Skill {
  id: string
  project_id?: string
  source_id: string
  name: string
  description?: string
  content?: string
  path?: string
  enabled: boolean
  metadata?: Record<string, string>
  created_at?: string
  updated_at?: string
}

export interface SkillScanResult {
  source_id: string
  path: string
  created: number
  updated: number
  deleted: number
  unchanged: number
  errors?: string[]
  scanned_at: string
}

export type MCPTransport = 'stdio' | 'streamable_http' | 'sse' | string
export type MCPServerStatus = 'online' | 'offline' | 'disabled' | 'failed' | 'unknown' | string

export interface MCPServerConfig {
  id: string
  project_id?: string
  name: string
  transport: MCPTransport
  status: MCPServerStatus
  enabled: boolean
  command?: string
  args?: string[]
  url?: string
  headers?: Record<string, string>
  secret_headers?: Record<string, string>
  secret_headers_hint?: string[]
  env?: Record<string, string>
  secret_env?: Record<string, string>
  secret_env_hint?: string[]
  timeout_sec?: number
  metadata?: Record<string, string>
  last_seen_at?: string
  created_at?: string
  updated_at?: string
}

export type ToolDefinitionKind = 'builtin' | 'mcp' | 'skill' | string
export type ToolStatus = 'active' | 'disabled' | 'unavailable' | string

export interface ToolDefinition {
  id: string
  project_id?: string
  name: string
  display_name?: string
  description?: string
  kind: ToolDefinitionKind
  status: ToolStatus
  mcp_server_id?: string
  source_id?: string
  skill_id?: string
  input_schema?: Record<string, unknown>
  metadata?: Record<string, string>
  created_at?: string
  updated_at?: string
}

export type ToolInvocationStatus = 'running' | 'succeeded' | 'failed' | string

export interface ToolInvocation {
  id: string
  agent_run_id?: string
  agent_id?: string
  project_id?: string
  tool_id?: string
  tool_name: string
  status: ToolInvocationStatus
  arguments?: Record<string, unknown>
  result?: Record<string, unknown>
  error?: string
  started_at?: string
  completed_at?: string
  created_at?: string
  updated_at?: string
}

export interface InitializeProjectResponse {
  project: Project
  story_bible: StoryBible
  workflow: AIWorkflow
}

export interface ModelResolution {
  route_key: string
  resolution_source: string
  provider_id: string
  provider_name: string
  provider_type: ProviderType
  model_id: string
  model_name: string
  model_kind: ModelKind
}

export interface IndexFreshness {
  project_id: string
  chapter_id?: string
  status: 'missing' | 'pending' | 'stale' | 'fresh' | string
  latest_chapter_version_id?: string
  latest_chapter_version_created_at?: string
  latest_indexed_chapter_version_id?: string
  latest_indexed_at?: string
  pending_job_count: number
}

export interface ContinuityEvidenceRef {
  source_type: string
  source_id?: string
  label: string
  excerpt?: string
}

export interface ContinuityIssue {
  type: string
  severity: string
  message: string
  draft_excerpt: string
  suggestion: string
  evidence: ContinuityEvidenceRef[]
}

export interface ContinuityAudit {
  status: string
  issues: ContinuityIssue[]
}

export interface ContextPreviewSummary {
  chapter_summary_count: number
  entity_count: number
  fact_count: number
  plot_thread_count: number
  world_rule_count: number
  text: string
}

export interface ContextPreviewResponse {
  context_pack: ContextPack
  summary: string
  estimated_tokens: number
  index_freshness: IndexFreshness
  model_resolution: ModelResolution
  tool_trace?: ToolTrace[]
}

export interface SaveChapterVersionResponse {
  chapter_version: ChapterVersion
  index_job: IndexJob
}

export interface RunPendingIndexResponse {
  processed: IndexJob[]
  count: number
  error?: string
}

export interface RebuildVectorsResponse {
  embedding_model_id: string
  embedding_model_name: string
  embedding_dimension: number
  project_count: number
  chapter_version_count: number
  job_count: number
}

export interface ContextPack {
  id: string
  project_id: string
  chapter_id?: string
  role: AgentRole
  token_budget: number
  query: string
  story_bible_id?: string
  world_rules?: Record<string, string>
  facts?: Fact[]
  entities?: Entity[]
  edges?: GraphEdge[]
  plot_threads?: PlotThread[]
  chapter_summaries?: Array<{
    chapter_id: string
    chapter_version_id: string
    title: string
    summary: string
  }>
  tool_trace?: ToolTrace[]
  metadata?: Record<string, string>
  created_at: string
}

export interface ContextSelection {
  chapter_ids?: string[]
  previous_chapter_count?: number
  include_current_chapter?: boolean
  character_ids?: string[]
  character_names?: string[]
  include_world_rules?: boolean
}

export type CharacterProfileMode = 'protagonist' | 'character'

export interface CharacterProfile {
  id?: string
  name: string
  role: string
  desire: string
  wound: string
  secret?: string
  summary?: string
  metadata?: Record<string, string>
}

export interface CharacterProfileRequest {
  project_id: string
  focus: string
  count: number
  brief: string
  chapter_id?: string
  context_node_ids?: string[]
  context_selection?: ContextSelection
  max_output_tokens?: number
}

export interface CharacterProfileResponse {
  characters: CharacterProfile[]
  workflow?: AIWorkflow
  model_resolution?: ModelResolution
  tool_trace?: ToolTrace[]
  mappings?: Array<{
    local_id?: string
    name: string
    entity_id: string
    action?: string
  }>
}

export interface CharacterSyncRequest {
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

export interface CharacterSyncResponse {
  project_id: string
  story_bible_id: string
  characters: Entity[]
  mappings: Array<{
    name: string
    entity_id: string
    action: string
  }>
}

export interface ContextPreviewRequest {
  project_id: string
  chapter_id: string
  title?: string
  brief?: string
  prompt?: string
  selection?: ContextSelection
  style_constraints?: string[]
  role?: AgentRole
  token_budget?: number
}

export interface AIDraftRequest {
  project_id: string
  chapter_id: string
  prompt?: string
  brief?: string
  title?: string
  chapter_idea?: string
  chapter_idea_workflow_id?: string
  max_output_tokens?: number
  selection?: ContextSelection
  style_constraints?: string[]
}

export interface ChapterIdeaRequest {
  project_id: string
  chapter_id: string
  title?: string
  brief: string
  prompt?: string
  selection?: ContextSelection
  style_constraints?: string[]
  max_output_tokens?: number
}

export interface ChapterIdeaResponse {
  workflow: AIWorkflow
  context_pack: ContextPack
  chapter_idea: string
  model_resolution: ModelResolution
  tool_trace?: ToolTrace[]
}

export interface DraftWithIdeaRequest {
  project_id: string
  chapter_id: string
  title?: string
  brief: string
  prompt?: string
  selection?: ContextSelection
  style_constraints?: string[]
  max_idea_output_tokens?: number
  max_draft_output_tokens?: number
}

export interface DraftResultResponse {
  workflow: AIWorkflow
  context_pack: ContextPack
  chapter_version: ChapterVersion
  index_job: IndexJob
  index_freshness: IndexFreshness
  model_resolution: ModelResolution
  continuity_audit: ContinuityAudit
  tool_trace?: ToolTrace[]
}

export interface DraftWithIdeaResponse {
  chapter_idea: ChapterIdeaResponse
  draft: DraftResultResponse
  model_resolution?: ModelResolution
  tool_trace?: ToolTrace[]
}

export interface AIDraftResponse {
  content: string
  workflow: AIWorkflow
  warnings: string[]
  context_pack: ContextPack
  chapter_version?: ChapterVersion
  index_job?: IndexJob
  index_freshness: IndexFreshness
  model_resolution: ModelResolution
  continuity_audit: ContinuityAudit
  tool_trace?: ToolTrace[]
}
