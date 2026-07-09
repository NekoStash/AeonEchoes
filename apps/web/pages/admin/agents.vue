<script setup lang="ts">
import {
  Bot,
  Hammer,
  Pencil,
  PlugZap,
  Plus,
  RefreshCw,
  Save,
  Sparkles,
  TestTube2,
  Trash2
} from '@lucide/vue'
import DataCardGrid from '~/components/data/DataCardGrid.vue'
import DataCollection from '~/components/data/DataCollection.vue'
import DataEmptyState from '~/components/data/EmptyState.vue'
import DataFilterBar from '~/components/data/FilterBar.vue'
import DataNoResultsState from '~/components/data/NoResultsState.vue'
import DataTable from '~/components/data/DataTable.vue'
import DensityToggle from '~/components/data/DensityToggle.vue'
import SearchInput from '~/components/data/SearchInput.vue'
import SortSelect from '~/components/data/SortSelect.vue'
import ViewModeToggle from '~/components/data/ViewModeToggle.vue'
import Panel from '~/components/ds/Panel.vue'
import StatCard from '~/components/ds/StatCard.vue'
import StatGrid from '~/components/ds/StatGrid.vue'
import StatusStack from '~/components/ds/StatusStack.vue'
import PageHeader from '~/components/layout/PageHeader.vue'
import PageShell from '~/components/layout/PageShell.vue'
import Toolbar from '~/components/layout/Toolbar.vue'
import type { AgentConfig, AgentRole, MCPServerConfig, Skill, ToolDefinition } from '~/lib/types'
import { formatDateTime } from '~/lib/utils'

const { t } = useI18n()
const api = useApi()
const workspace = useWorkspaceStore()

type ResourceTab = 'agents' | 'skills' | 'mcp' | 'tools'
type ResourceViewMode = 'table' | 'grid'
type ResourceDensity = 'compact' | 'comfortable' | 'relaxed'
type EnabledFilter = '' | 'enabled' | 'disabled'
type ResourceSortKey =
  | 'name:asc'
  | 'name:desc'
  | 'status:asc'
  | 'status:desc'
  | 'updated_at:desc'
  | 'updated_at:asc'
  | 'created_at:desc'
  | 'created_at:asc'
type DeleteTarget = {
  type: 'agent' | 'skill' | 'mcp'
  id: string
  name: string
}

const agentRoleValues: AgentRole[] = [
  'writer',
  'editor',
  'genesis-optimizer',
  'plot-architect',
  'world-builder',
  'character-keeper',
  'continuity-auditor',
  'fact-extractor',
  'graph-curator'
]
const knownMCPTransports = ['stdio', 'streamable_http', 'sse']
const knownMCPStatuses = ['online', 'offline', 'disabled', 'failed', 'unknown']
const knownToolStatuses = ['active', 'disabled', 'unavailable']
const knownToolKinds = ['builtin', 'mcp', 'skill']

const agents = ref<AgentConfig[]>([])
const skills = ref<Skill[]>([])
const mcpServers = ref<MCPServerConfig[]>([])
const tools = ref<ToolDefinition[]>([])
const loading = ref(false)
const savingAgent = ref(false)
const savingSkill = ref(false)
const savingMCP = ref(false)
const errorMessage = ref('')
const loadErrorMessage = ref('')
const successMessage = ref('')
const pendingKeys = ref<string[]>([])

const activeTab = ref<ResourceTab>('agents')
const viewMode = ref<ResourceViewMode>('table')
const density = ref<ResourceDensity>('comfortable')

const agentSearchQuery = ref('')
const agentFilterEnabled = ref<EnabledFilter>('')
const agentFilterRole = ref<AgentRole | ''>('')
const agentSortKey = ref<ResourceSortKey>('name:asc')

const skillSearchQuery = ref('')
const skillFilterEnabled = ref<EnabledFilter>('')
const skillFilterSource = ref('')
const skillSortKey = ref<ResourceSortKey>('name:asc')

const mcpSearchQuery = ref('')
const mcpFilterEnabled = ref<EnabledFilter>('')
const mcpFilterStatus = ref('')
const mcpFilterTransport = ref('')
const mcpSortKey = ref<ResourceSortKey>('name:asc')

const toolSearchQuery = ref('')
const toolFilterKind = ref('')
const toolFilterStatus = ref('')
const toolSortKey = ref<ResourceSortKey>('name:asc')

const agentDialogOpen = ref(false)
const skillDialogOpen = ref(false)
const mcpDialogOpen = ref(false)
const confirmDialogOpen = ref(false)
const deleteTarget = ref<DeleteTarget | null>(null)

const agentForm = reactive<AgentConfig>({
  id: '',
  name: '',
  description: '',
  role: 'writer',
  enabled: true,
  system_prompt: '',
  skill_ids: [],
  tool_ids: [],
  mcp_server_ids: []
})

const skillForm = reactive<Skill>({
  id: '',
  source_id: '',
  name: '',
  description: '',
  content: '',
  enabled: true,
  metadata: {}
})

const mcpForm = reactive<MCPServerConfig>({
  id: '',
  name: '',
  transport: 'stdio',
  status: 'unknown',
  enabled: true,
  command: '',
  args: [],
  url: '',
  timeout_sec: 30,
  metadata: {}
})

const activeAgents = computed(() => agents.value.filter((item) => item.enabled).length)
const activeSkills = computed(() => skills.value.filter((item) => item.enabled).length)
const activeMCPServers = computed(() => mcpServers.value.filter((item) => item.enabled).length)
const activeTools = computed(() => tools.value.filter((item) => item.status === 'active').length)
const collectionDensity = computed<'compact' | 'comfortable'>(() => density.value === 'compact' ? 'compact' : 'comfortable')
const panelPadding = computed<'sm' | 'md'>(() => density.value === 'compact' ? 'sm' : 'md')

const pageTabs = computed(() => [
  { label: t('agents.tabs.agents'), value: 'agents', badge: String(agents.value.length) },
  { label: t('agents.tabs.skills'), value: 'skills', badge: String(skills.value.length) },
  { label: t('agents.tabs.mcp'), value: 'mcp', badge: String(mcpServers.value.length) },
  { label: t('agents.tabs.tools'), value: 'tools', badge: String(tools.value.length) }
])
const roleOptions = computed(() => agentRoleValues.map((value) => ({ label: roleLabel(value), value })))
const agentRoleFilterOptions = computed(() => [
  { label: t('agents.filters.allRoles'), value: '' },
  ...roleOptions.value
])
const enabledFilterOptions = computed(() => [
  { label: t('agents.filters.allStatuses'), value: '' },
  { label: t('agents.enabled'), value: 'enabled' },
  { label: t('agents.disabled'), value: 'disabled' }
])
const skillSourceFilterOptions = computed(() => [
  { label: t('agents.filters.allSources'), value: '' },
  ...uniqueTokens(skills.value.map((skill) => skill.source_id)).map((value) => ({ label: sourceLabel(value), value }))
])
const mcpTransportOptions = computed(() => uniqueTokens([...knownMCPTransports, ...mcpServers.value.map((server) => server.transport)])
  .map((value) => ({ label: transportLabel(value), value })))
const mcpTransportFilterOptions = computed(() => [
  { label: t('agents.filters.allTransports'), value: '' },
  ...mcpTransportOptions.value
])
const mcpStatusFilterOptions = computed(() => [
  { label: t('agents.filters.allStatuses'), value: '' },
  ...uniqueTokens([...knownMCPStatuses, ...mcpServers.value.map((server) => server.status)])
    .map((value) => ({ label: statusLabel(value), value }))
])
const toolKindFilterOptions = computed(() => [
  { label: t('agents.filters.allKinds'), value: '' },
  ...uniqueTokens([...knownToolKinds, ...tools.value.map((tool) => tool.kind)])
    .map((value) => ({ label: kindLabel(value), value }))
])
const toolStatusFilterOptions = computed(() => [
  { label: t('agents.filters.allStatuses'), value: '' },
  ...uniqueTokens([...knownToolStatuses, ...tools.value.map((tool) => tool.status)])
    .map((value) => ({ label: statusLabel(value), value }))
])
const sortOptions = computed(() => [
  { label: t('agents.sort.nameAsc'), value: 'name:asc' },
  { label: t('agents.sort.nameDesc'), value: 'name:desc' },
  { label: t('agents.sort.statusAsc'), value: 'status:asc' },
  { label: t('agents.sort.statusDesc'), value: 'status:desc' },
  { label: t('agents.sort.updatedDesc'), value: 'updated_at:desc' },
  { label: t('agents.sort.updatedAsc'), value: 'updated_at:asc' },
  { label: t('agents.sort.createdDesc'), value: 'created_at:desc' },
  { label: t('agents.sort.createdAsc'), value: 'created_at:asc' }
])

const agentTableColumns = computed(() => [
  { key: 'resource', label: t('agents.table.agent'), class: 'min-w-[260px]', headerClass: 'min-w-[260px]' },
  { key: 'role', label: t('agents.table.role'), class: 'min-w-[150px]' },
  { key: 'status', label: t('agents.table.status'), class: 'min-w-[170px]' },
  { key: 'connections', label: t('agents.table.connections'), class: 'min-w-[220px]' },
  { key: 'updated', label: t('agents.table.updatedAt'), class: 'min-w-[150px]' },
  { key: 'actions', label: t('agents.table.actions'), align: 'right' as const, class: 'min-w-[180px]' }
])
const skillTableColumns = computed(() => [
  { key: 'resource', label: t('agents.table.skill'), class: 'min-w-[260px]', headerClass: 'min-w-[260px]' },
  { key: 'source', label: t('agents.table.source'), class: 'min-w-[180px]' },
  { key: 'status', label: t('agents.table.status'), class: 'min-w-[170px]' },
  { key: 'updated', label: t('agents.table.updatedAt'), class: 'min-w-[150px]' },
  { key: 'actions', label: t('agents.table.actions'), align: 'right' as const, class: 'min-w-[180px]' }
])
const mcpTableColumns = computed(() => [
  { key: 'resource', label: t('agents.table.server'), class: 'min-w-[260px]', headerClass: 'min-w-[260px]' },
  { key: 'transport', label: t('agents.table.transport'), class: 'min-w-[140px]' },
  { key: 'status', label: t('agents.table.status'), class: 'min-w-[170px]' },
  { key: 'endpoint', label: t('agents.table.endpoint'), class: 'min-w-[240px]' },
  { key: 'updated', label: t('agents.table.updatedAt'), class: 'min-w-[150px]' },
  { key: 'actions', label: t('agents.table.actions'), align: 'right' as const, class: 'min-w-[260px]' }
])
const toolTableColumns = computed(() => [
  { key: 'resource', label: t('agents.table.tool'), class: 'min-w-[260px]', headerClass: 'min-w-[260px]' },
  { key: 'kind', label: t('agents.table.kind'), class: 'min-w-[130px]' },
  { key: 'status', label: t('agents.table.status'), class: 'min-w-[170px]' },
  { key: 'origin', label: t('agents.table.origin'), class: 'min-w-[220px]' },
  { key: 'updated', label: t('agents.table.updatedAt'), class: 'min-w-[150px]' }
])

const activeAgentFilterCount = computed(() => [agentSearchQuery.value.trim(), agentFilterEnabled.value, agentFilterRole.value].filter(Boolean).length)
const activeSkillFilterCount = computed(() => [skillSearchQuery.value.trim(), skillFilterEnabled.value, skillFilterSource.value].filter(Boolean).length)
const activeMCPFilterCount = computed(() => [mcpSearchQuery.value.trim(), mcpFilterEnabled.value, mcpFilterStatus.value, mcpFilterTransport.value].filter(Boolean).length)
const activeToolFilterCount = computed(() => [toolSearchQuery.value.trim(), toolFilterKind.value, toolFilterStatus.value].filter(Boolean).length)

const visibleAgents = computed(() => [...agents.value]
  .filter(agentMatchesFilters)
  .sort((left, right) => compareResourceItems(left, right, agentSortKey.value, agentTitle, agentStatusValue)))
const visibleSkills = computed(() => [...skills.value]
  .filter(skillMatchesFilters)
  .sort((left, right) => compareResourceItems(left, right, skillSortKey.value, skillTitle, skillStatusValue)))
const visibleMCPServers = computed(() => [...mcpServers.value]
  .filter(mcpMatchesFilters)
  .sort((left, right) => compareResourceItems(left, right, mcpSortKey.value, mcpTitle, (server) => server.status)))
const visibleTools = computed(() => [...tools.value]
  .filter(toolMatchesFilters)
  .sort((left, right) => compareResourceItems(left, right, toolSortKey.value, toolTitle, (tool) => tool.status)))

const agentRows = computed<Array<Record<string, unknown>>>(() => visibleAgents.value.map((agent) => agent as unknown as Record<string, unknown>))
const skillRows = computed<Array<Record<string, unknown>>>(() => visibleSkills.value.map((skill) => skill as unknown as Record<string, unknown>))
const mcpRows = computed<Array<Record<string, unknown>>>(() => visibleMCPServers.value.map((server) => server as unknown as Record<string, unknown>))
const toolRows = computed<Array<Record<string, unknown>>>(() => visibleTools.value.map((tool) => tool as unknown as Record<string, unknown>))

const statusItems = computed(() => {
  const items = []
  if (loadErrorMessage.value) {
    items.push({ id: 'load-error', tone: 'danger' as const, title: t('apiError.title'), description: loadErrorMessage.value })
  }
  if (errorMessage.value) {
    items.push({ id: 'operation-error', tone: 'danger' as const, title: t('common.error'), description: errorMessage.value })
  }
  if (successMessage.value) {
    items.push({ id: 'success', tone: 'success' as const, title: t('actions.saved'), description: successMessage.value })
  }
  return items
})

onMounted(() => {
  refreshAll()
})

async function refreshAll() {
  errorMessage.value = ''
  loadErrorMessage.value = ''
  loading.value = true
  try {
    const [agentResult, skillResult, mcpResult, toolResult] = await Promise.all([
      api.listAgents({ limit: 100 }),
      api.listSkills({ limit: 100 }),
      api.listMCPServers({ limit: 100 }),
      api.listToolCatalog({ limit: 200 })
    ])
    agents.value = agentResult.data
    skills.value = skillResult.data
    mcpServers.value = mcpResult.data
    tools.value = toolResult.data
  } catch (error) {
    const apiError = workspace.recordError(t('agents.resultScopes.load'), error)
    loadErrorMessage.value = apiError.message || t('agents.errors.loadFailed')
  } finally {
    loading.value = false
  }
}

function resetAgentForm() {
  Object.assign(agentForm, {
    id: '',
    project_id: undefined,
    name: '',
    description: '',
    role: 'writer',
    model_id: undefined,
    enabled: true,
    system_prompt: '',
    skill_ids: [],
    tool_ids: [],
    mcp_server_ids: [],
    memory_policy: undefined,
    runtime_options: undefined,
    metadata: undefined,
    created_at: undefined,
    updated_at: undefined
  })
}

function resetSkillForm() {
  Object.assign(skillForm, {
    id: '',
    project_id: undefined,
    source_id: '',
    name: '',
    description: '',
    content: '',
    path: undefined,
    enabled: true,
    metadata: {},
    created_at: undefined,
    updated_at: undefined
  })
}

function resetMCPForm() {
  Object.assign(mcpForm, {
    id: '',
    project_id: undefined,
    name: '',
    transport: 'stdio',
    status: 'unknown',
    enabled: true,
    command: '',
    args: [],
    url: '',
    headers: undefined,
    secret_headers: undefined,
    env: undefined,
    secret_env: undefined,
    timeout_sec: 30,
    metadata: {},
    last_seen_at: undefined,
    created_at: undefined,
    updated_at: undefined
  })
}

function openAgentDialog(agent?: AgentConfig) {
  if (agent) {
    Object.assign(agentForm, {
      ...agent,
      description: agent.description || '',
      system_prompt: agent.system_prompt || '',
      skill_ids: [...(agent.skill_ids || [])],
      tool_ids: [...(agent.tool_ids || [])],
      mcp_server_ids: [...(agent.mcp_server_ids || [])]
    })
  } else {
    resetAgentForm()
  }
  agentDialogOpen.value = true
}

function openSkillDialog(skill?: Skill) {
  if (skill) {
    Object.assign(skillForm, {
      ...skill,
      source_id: skill.source_id || '',
      description: skill.description || '',
      content: skill.content || '',
      metadata: { ...(skill.metadata || {}) }
    })
  } else {
    resetSkillForm()
  }
  skillDialogOpen.value = true
}

function openMCPDialog(server?: MCPServerConfig) {
  if (server) {
    Object.assign(mcpForm, {
      ...server,
      command: server.command || '',
      url: server.url || '',
      args: [...(server.args || [])],
      metadata: { ...(server.metadata || {}) }
    })
  } else {
    resetMCPForm()
  }
  mcpDialogOpen.value = true
}

async function saveAgent() {
  errorMessage.value = ''
  successMessage.value = ''
  savingAgent.value = true
  try {
    const payload: AgentConfig = {
      ...agentForm,
      name: agentForm.name.trim(),
      description: agentForm.description?.trim() || undefined,
      system_prompt: agentForm.system_prompt?.trim() || undefined,
      skill_ids: agentForm.skill_ids || [],
      tool_ids: agentForm.tool_ids || [],
      mcp_server_ids: agentForm.mcp_server_ids || []
    }
    if (!payload.name) throw new Error(t('agents.errors.agentNameRequired'))
    const result = await api.saveAgent(payload, payload.id ? 'edit' : 'create')
    workspace.recordResult(t('agents.resultScopes.agentSave'), result)
    upsertById(agents.value, result.data)
    successMessage.value = t('agents.messages.agentSaved')
    agentDialogOpen.value = false
    resetAgentForm()
  } catch (error) {
    const apiError = workspace.recordError(t('agents.resultScopes.agentSave'), error)
    errorMessage.value = apiError.message || t('agents.errors.saveAgentFailed')
  } finally {
    savingAgent.value = false
  }
}

async function saveSkill() {
  errorMessage.value = ''
  successMessage.value = ''
  savingSkill.value = true
  try {
    const payload: Skill = {
      ...skillForm,
      source_id: skillForm.source_id?.trim() || '',
      name: skillForm.name.trim(),
      description: skillForm.description?.trim() || undefined,
      content: skillForm.content?.trim() || undefined
    }
    if (!payload.name) throw new Error(t('agents.errors.skillNameRequired'))
    const result = await api.saveSkill(payload, payload.id ? 'edit' : 'create')
    workspace.recordResult(t('agents.resultScopes.skillSave'), result)
    upsertById(skills.value, result.data)
    successMessage.value = t('agents.messages.skillSaved')
    skillDialogOpen.value = false
    resetSkillForm()
  } catch (error) {
    const apiError = workspace.recordError(t('agents.resultScopes.skillSave'), error)
    errorMessage.value = apiError.message || t('agents.errors.saveSkillFailed')
  } finally {
    savingSkill.value = false
  }
}

async function saveMCPServer() {
  errorMessage.value = ''
  successMessage.value = ''
  savingMCP.value = true
  try {
    const payload: MCPServerConfig = {
      ...mcpForm,
      name: mcpForm.name.trim(),
      command: mcpForm.transport === 'stdio' ? mcpForm.command?.trim() : undefined,
      url: mcpForm.transport === 'stdio' ? undefined : mcpForm.url?.trim(),
      args: parseArgs(mcpForm.args),
      status: mcpForm.status || 'unknown'
    }
    if (!payload.name) throw new Error(t('agents.errors.mcpNameRequired'))
    if (payload.transport === 'stdio' && !payload.command) throw new Error(t('agents.errors.mcpCommandRequired'))
    if (payload.transport !== 'stdio' && !payload.url) throw new Error(t('agents.errors.mcpUrlRequired'))
    const result = await api.saveMCPServer(payload, payload.id ? 'edit' : 'create')
    workspace.recordResult(t('agents.resultScopes.mcpSave'), result)
    upsertById(mcpServers.value, result.data)
    successMessage.value = t('agents.messages.mcpSaved')
    mcpDialogOpen.value = false
    resetMCPForm()
  } catch (error) {
    const apiError = workspace.recordError(t('agents.resultScopes.mcpSave'), error)
    errorMessage.value = apiError.message || t('agents.errors.saveMCPFailed')
  } finally {
    savingMCP.value = false
  }
}

async function toggleAgent(agent: AgentConfig, enabled: boolean) {
  const key = `agent-toggle:${agent.id}`
  await withPending(key, async () => {
    errorMessage.value = ''
    successMessage.value = ''
    try {
      const result = await api.saveAgent({ ...agent, enabled }, 'edit')
      workspace.recordResult(t('agents.resultScopes.agentToggle'), result)
      upsertById(agents.value, result.data)
      successMessage.value = t(enabled ? 'agents.messages.agentEnabled' : 'agents.messages.agentDisabled')
    } catch (error) {
      const apiError = workspace.recordError(t('agents.resultScopes.agentToggle'), error)
      errorMessage.value = apiError.message || t('agents.errors.toggleAgentFailed')
    }
  })
}

async function toggleSkill(skill: Skill, enabled: boolean) {
  const key = `skill-toggle:${skill.id}`
  await withPending(key, async () => {
    errorMessage.value = ''
    successMessage.value = ''
    try {
      const result = await api.setSkillEnabled(skill.id, enabled)
      workspace.recordResult(t('agents.resultScopes.skillToggle'), result)
      upsertById(skills.value, result.data)
      successMessage.value = t(enabled ? 'agents.messages.skillEnabled' : 'agents.messages.skillDisabled')
    } catch (error) {
      const apiError = workspace.recordError(t('agents.resultScopes.skillToggle'), error)
      errorMessage.value = apiError.message || t('agents.errors.toggleSkillFailed')
    }
  })
}

async function toggleMCP(server: MCPServerConfig, enabled: boolean) {
  const key = `mcp-toggle:${server.id}`
  await withPending(key, async () => {
    errorMessage.value = ''
    successMessage.value = ''
    try {
      const result = await api.setMCPServerEnabled(server.id, enabled)
      workspace.recordResult(t('agents.resultScopes.mcpToggle'), result)
      upsertById(mcpServers.value, result.data)
      successMessage.value = t(enabled ? 'agents.messages.mcpEnabled' : 'agents.messages.mcpDisabled')
    } catch (error) {
      const apiError = workspace.recordError(t('agents.resultScopes.mcpToggle'), error)
      errorMessage.value = apiError.message || t('agents.errors.toggleMCPFailed')
    }
  })
}

async function toggleTool(tool: ToolDefinition, enabled: boolean) {
  const key = `tool-toggle:${tool.id}`
  await withPending(key, async () => {
    errorMessage.value = ''
    successMessage.value = ''
    try {
      const result = await api.setToolEnabled(tool.id, enabled)
      workspace.recordResult(t('agents.resultScopes.toolToggle'), result)
      upsertById(tools.value, result.data)
      successMessage.value = t(enabled ? 'agents.messages.toolEnabled' : 'agents.messages.toolDisabled')
    } catch (error) {
      const apiError = workspace.recordError(t('agents.resultScopes.toolToggle'), error)
      errorMessage.value = apiError.message || t('agents.errors.toggleToolFailed')
    }
  })
}

async function scanDefaultSkills() {
  const key = 'skill-scan:default'
  await withPending(key, async () => {
    errorMessage.value = ''
    successMessage.value = ''
    try {
      const result = await api.scanDefaultSkillSource()
      workspace.recordResult(t('agents.resultScopes.skillScan'), result)
      successMessage.value = t('agents.messages.skillScanComplete', { count: result.data.created + result.data.updated })
      await refreshAll()
    } catch (error) {
      const apiError = workspace.recordError(t('agents.resultScopes.skillScan'), error)
      errorMessage.value = apiError.message || t('agents.errors.scanSkillsFailed')
    }
  })
}

async function testMCPServer(server: MCPServerConfig) {
  const key = `mcp-test:${server.id}`
  await withPending(key, async () => {
    errorMessage.value = ''
    successMessage.value = ''
    try {
      const result = await api.testMCPServer(server.id)
      workspace.recordResult(t('agents.resultScopes.mcpTest'), result)
      upsertById(mcpServers.value, result.data.server)
      successMessage.value = result.data.ok ? t('agents.messages.mcpTestPassed') : t('agents.messages.mcpTestFailed')
    } catch (error) {
      const apiError = workspace.recordError(t('agents.resultScopes.mcpTest'), error)
      errorMessage.value = apiError.message || t('agents.errors.testMCPFailed')
    }
  })
}

async function refreshMCPTools(server: MCPServerConfig) {
  const key = `mcp-refresh:${server.id}`
  await withPending(key, async () => {
    errorMessage.value = ''
    successMessage.value = ''
    try {
      const result = await api.refreshMCPTools(server.id)
      workspace.recordResult(t('agents.resultScopes.toolRefresh'), result)
      successMessage.value = t('agents.messages.toolsRefreshed', { count: result.data.count })
      await refreshAll()
    } catch (error) {
      const apiError = workspace.recordError(t('agents.resultScopes.toolRefresh'), error)
      errorMessage.value = apiError.message || t('agents.errors.refreshToolsFailed')
    }
  })
}

function requestDeleteAgent(agent: AgentConfig) {
  deleteTarget.value = { type: 'agent', id: agent.id, name: agentTitle(agent) }
  confirmDialogOpen.value = true
}

function requestDeleteSkill(skill: Skill) {
  deleteTarget.value = { type: 'skill', id: skill.id, name: skillTitle(skill) }
  confirmDialogOpen.value = true
}

function requestDeleteMCP(server: MCPServerConfig) {
  deleteTarget.value = { type: 'mcp', id: server.id, name: mcpTitle(server) }
  confirmDialogOpen.value = true
}

async function confirmDelete() {
  const target = deleteTarget.value
  if (!target) return
  const key = `${target.type}-delete:${target.id}`
  await withPending(key, async () => {
    errorMessage.value = ''
    successMessage.value = ''
    try {
      if (target.type === 'agent') {
        const result = await api.deleteAgent(target.id)
        workspace.recordResult(t('agents.resultScopes.agentDelete'), result)
        removeById(agents.value, target.id)
        if (agentForm.id === target.id) resetAgentForm()
        successMessage.value = t('agents.messages.agentDeleted')
      } else if (target.type === 'skill') {
        const result = await api.deleteSkill(target.id)
        workspace.recordResult(t('agents.resultScopes.skillDelete'), result)
        removeById(skills.value, target.id)
        if (skillForm.id === target.id) resetSkillForm()
        successMessage.value = t('agents.messages.skillDeleted')
      } else {
        const result = await api.deleteMCPServer(target.id)
        workspace.recordResult(t('agents.resultScopes.mcpDelete'), result)
        removeById(mcpServers.value, target.id)
        if (mcpForm.id === target.id) resetMCPForm()
        successMessage.value = t('agents.messages.mcpDeleted')
      }
      confirmDialogOpen.value = false
      deleteTarget.value = null
    } catch (error) {
      const apiError = workspace.recordError(t('agents.resultScopes.delete'), error)
      errorMessage.value = apiError.message || t('agents.errors.deleteFailed')
    }
  })
}

async function withPending(key: string, operation: () => Promise<void>) {
  pendingKeys.value = [...new Set([...pendingKeys.value, key])]
  try {
    await operation()
  } finally {
    pendingKeys.value = pendingKeys.value.filter((item) => item !== key)
  }
}

function isPending(key: string) {
  return pendingKeys.value.includes(key)
}

function parseArgs(args?: string[] | string) {
  if (Array.isArray(args)) return args.map((item) => item.trim()).filter(Boolean)
  return String(args || '').split(/\s+/).map((item) => item.trim()).filter(Boolean)
}

function argsText(args?: string[]) {
  return (args || []).join(' ')
}

function updateArgs(value: string) {
  mcpForm.args = parseArgs(value)
}

function upsertById<T extends { id: string }>(items: T[], item: T) {
  const index = items.findIndex((candidate) => candidate.id === item.id)
  if (index >= 0) {
    items[index] = item
    return
  }
  items.unshift(item)
}

function removeById<T extends { id: string }>(items: T[], id: string) {
  const index = items.findIndex((candidate) => candidate.id === id)
  if (index >= 0) items.splice(index, 1)
}

function clearAgentFilters() {
  agentSearchQuery.value = ''
  agentFilterEnabled.value = ''
  agentFilterRole.value = ''
}

function clearSkillFilters() {
  skillSearchQuery.value = ''
  skillFilterEnabled.value = ''
  skillFilterSource.value = ''
}

function clearMCPFilters() {
  mcpSearchQuery.value = ''
  mcpFilterEnabled.value = ''
  mcpFilterStatus.value = ''
  mcpFilterTransport.value = ''
}

function clearToolFilters() {
  toolSearchQuery.value = ''
  toolFilterKind.value = ''
  toolFilterStatus.value = ''
}

function resourceLoading(total: number) {
  return loading.value && total === 0
}

function resourceError(total: number) {
  if (resourceLoading(total) || total > 0) return ''
  return loadErrorMessage.value
}

function resourceEmpty(total: number) {
  return !resourceLoading(total) && !resourceError(total) && total === 0
}

function resourceNoResults(total: number, visible: number) {
  return !resourceLoading(total) && !resourceError(total) && total > 0 && visible === 0
}

function normalizeSearch(value: unknown) {
  return String(value || '').trim().toLowerCase()
}

function uniqueTokens(values: Array<string | undefined>) {
  return Array.from(new Set(values.map((value) => String(value || '').trim()).filter(Boolean)))
    .sort((left, right) => compareText(left, right))
}

function matchesQuery(query: string, fields: unknown[]) {
  const normalizedQuery = normalizeSearch(query)
  if (!normalizedQuery) return true
  return fields.some((field) => normalizeSearch(field).includes(normalizedQuery))
}

function matchesEnabledFilter(enabled: boolean, filter: EnabledFilter) {
  if (filter === 'enabled') return enabled
  if (filter === 'disabled') return !enabled
  return true
}

function agentMatchesFilters(agent: AgentConfig) {
  if (!matchesQuery(agentSearchQuery.value, agentSearchFields(agent))) return false
  if (!matchesEnabledFilter(agent.enabled, agentFilterEnabled.value)) return false
  if (agentFilterRole.value && agent.role !== agentFilterRole.value) return false
  return true
}

function skillMatchesFilters(skill: Skill) {
  if (!matchesQuery(skillSearchQuery.value, skillSearchFields(skill))) return false
  if (!matchesEnabledFilter(skill.enabled, skillFilterEnabled.value)) return false
  if (skillFilterSource.value && skill.source_id !== skillFilterSource.value) return false
  return true
}

function mcpMatchesFilters(server: MCPServerConfig) {
  if (!matchesQuery(mcpSearchQuery.value, mcpSearchFields(server))) return false
  if (!matchesEnabledFilter(server.enabled, mcpFilterEnabled.value)) return false
  if (mcpFilterStatus.value && server.status !== mcpFilterStatus.value) return false
  if (mcpFilterTransport.value && server.transport !== mcpFilterTransport.value) return false
  return true
}

function toolMatchesFilters(tool: ToolDefinition) {
  if (!matchesQuery(toolSearchQuery.value, toolSearchFields(tool))) return false
  if (toolFilterKind.value && tool.kind !== toolFilterKind.value) return false
  if (toolFilterStatus.value && tool.status !== toolFilterStatus.value) return false
  return true
}

function agentSearchFields(agent: AgentConfig) {
  return [
    agent.name,
    agent.id,
    agent.description,
    agent.role,
    roleLabel(agent.role),
    agent.project_id,
    agent.model_id,
    agent.system_prompt,
    agent.enabled ? t('agents.enabled') : t('agents.disabled'),
    ...(agent.skill_ids || []),
    ...(agent.tool_ids || []),
    ...(agent.mcp_server_ids || [])
  ]
}

function skillSearchFields(skill: Skill) {
  return [
    skill.name,
    skill.id,
    skill.description,
    skill.project_id,
    skill.source_id,
    sourceLabel(skill.source_id),
    skill.path,
    skill.enabled ? t('agents.enabled') : t('agents.disabled')
  ]
}

function mcpSearchFields(server: MCPServerConfig) {
  return [
    server.name,
    server.id,
    server.project_id,
    server.transport,
    transportLabel(server.transport),
    server.status,
    statusLabel(server.status),
    server.command,
    server.url,
    server.enabled ? t('agents.enabled') : t('agents.disabled'),
    ...(server.args || [])
  ]
}

function toolSearchFields(tool: ToolDefinition) {
  return [
    tool.display_name,
    tool.name,
    tool.id,
    tool.description,
    tool.kind,
    kindLabel(tool.kind),
    tool.status,
    statusLabel(tool.status),
    tool.project_id,
    tool.mcp_server_id,
    tool.source_id,
    tool.skill_id
  ]
}

function compareResourceItems<T extends { created_at?: string; updated_at?: string }>(
  left: T,
  right: T,
  sortKey: ResourceSortKey,
  getName: (item: T) => string,
  getStatus: (item: T) => string
) {
  const [field, direction] = sortKey.split(':') as ['name' | 'status' | 'updated_at' | 'created_at', 'asc' | 'desc']
  const multiplier = direction === 'asc' ? 1 : -1
  if (field === 'status') {
    return compareText(getStatus(left), getStatus(right)) * multiplier || compareText(getName(left), getName(right))
  }
  if (field === 'updated_at') {
    return (timestampValue(left.updated_at) - timestampValue(right.updated_at)) * multiplier || compareText(getName(left), getName(right))
  }
  if (field === 'created_at') {
    return (timestampValue(left.created_at) - timestampValue(right.created_at)) * multiplier || compareText(getName(left), getName(right))
  }
  return compareText(getName(left), getName(right)) * multiplier || compareText(getStatus(left), getStatus(right))
}

function compareText(left: string, right: string) {
  return left.localeCompare(right, undefined, { numeric: true, sensitivity: 'base' })
}

function timestampValue(value?: string) {
  const timestamp = Date.parse(value || '')
  return Number.isFinite(timestamp) ? timestamp : 0
}

function statusVariant(status?: string) {
  if (status === 'active' || status === 'online' || status === 'enabled') return 'success' as const
  if (status === 'disabled' || status === 'offline') return 'muted' as const
  if (status === 'failed' || status === 'unavailable') return 'rose' as const
  return 'gold' as const
}

function enabledVariant(enabled: boolean) {
  return enabled ? 'success' as const : 'muted' as const
}

function roleLabel(role?: AgentRole | string) {
  if (!role) return t('agents.roles.default')
  return translatedToken(`agents.roles.${role.replace(/-/g, '_')}`, role)
}

function statusLabel(status?: string) {
  if (!status) return t('common.emptyValue')
  return translatedToken(`agents.statuses.${status.replace(/-/g, '_')}`, status)
}

function transportLabel(transport?: string) {
  if (!transport) return t('common.emptyValue')
  return translatedToken(`agents.transports.${transport.replace(/-/g, '_')}`, transport)
}

function kindLabel(kind?: string) {
  if (!kind) return t('common.emptyValue')
  return translatedToken(`agents.kinds.${kind.replace(/-/g, '_')}`, kind)
}

function sourceLabel(sourceId?: string) {
  return sourceId || t('agents.filters.noSource')
}

function translatedToken(key: string, fallback: string) {
  const value = t(key)
  return value === key ? prettifyToken(fallback) : value
}

function prettifyToken(value: string) {
  if (!value) return t('common.emptyValue')
  return value
    .split(/[-_]/g)
    .filter(Boolean)
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join(' ')
}

function dateLabel(value?: string) {
  return value ? formatDateTime(value) : t('common.emptyValue')
}

function agentTitle(agent: AgentConfig) {
  return agent.name || agent.id
}

function skillTitle(skill: Skill) {
  return skill.name || skill.id
}

function mcpTitle(server: MCPServerConfig) {
  return server.name || server.id
}

function toolTitle(tool: ToolDefinition) {
  return tool.display_name || tool.name || tool.id
}

function agentStatusValue(agent: AgentConfig) {
  return agent.enabled ? 'enabled' : 'disabled'
}

function skillStatusValue(skill: Skill) {
  return skill.enabled ? 'enabled' : 'disabled'
}

function mcpEndpoint(server: MCPServerConfig) {
  return server.transport === 'stdio' ? [server.command, ...(server.args || [])].filter(Boolean).join(' ') : server.url || t('common.emptyValue')
}

function toolOrigin(tool: ToolDefinition) {
  if (tool.mcp_server_id) return `${t('agents.fields.mcpServerId')}: ${tool.mcp_server_id}`
  if (tool.source_id) return `${t('agents.fields.sourceId')}: ${tool.source_id}`
  if (tool.skill_id) return `${t('agents.fields.skillId')}: ${tool.skill_id}`
  if (tool.project_id) return `${t('agents.fields.projectId')}: ${tool.project_id}`
  return t('common.emptyValue')
}

function resourceSummary(visible: number, total: number) {
  return t('agents.filters.resultSummary', { visible, total })
}

function agentFromRow(row: unknown) {
  return row as AgentConfig
}

function skillFromRow(row: unknown) {
  return row as Skill
}

function mcpFromRow(row: unknown) {
  return row as MCPServerConfig
}

function toolFromRow(row: unknown) {
  return row as ToolDefinition
}
</script>

<template>
  <PageShell density="normal">
    <PageHeader :eyebrow="t('agents.eyebrow')" :title="t('agents.title')" :description="t('agents.description')">
      <template #actions>
        <UiButton variant="outline" :disabled="loading" class="w-full sm:w-auto" @click="refreshAll">
          <RefreshCw :class="['h-4 w-4', loading && 'animate-spin']" />
          {{ t('actions.refresh') }}
        </UiButton>
      </template>
    </PageHeader>

    <StatusStack v-if="statusItems.length" :items="statusItems" />

    <StatGrid columns="four">
      <StatCard :label="t('agents.stats.agents')" :value="agents.length" :hint="t('agents.stats.enabledCount', { count: activeAgents })" tone="info">
        <template #icon><Bot class="h-5 w-5" /></template>
      </StatCard>
      <StatCard :label="t('agents.stats.skills')" :value="skills.length" :hint="t('agents.stats.enabledCount', { count: activeSkills })" tone="success">
        <template #icon><Sparkles class="h-5 w-5" /></template>
      </StatCard>
      <StatCard :label="t('agents.stats.mcp')" :value="mcpServers.length" :hint="t('agents.stats.enabledCount', { count: activeMCPServers })" tone="warning">
        <template #icon><PlugZap class="h-5 w-5" /></template>
      </StatCard>
      <StatCard :label="t('agents.stats.tools')" :value="tools.length" :hint="t('agents.stats.activeTools', { count: activeTools })" tone="neutral">
        <template #icon><Hammer class="h-5 w-5" /></template>
      </StatCard>
    </StatGrid>

    <UiTabs v-model="activeTab" :tabs="pageTabs" class="w-full" />

    <section v-if="activeTab === 'agents'" class="space-y-4">
      <DataCollection
        :title="t('agents.sections.agents')"
        :description="t('agents.sections.agentsDescription')"
        :loading="resourceLoading(agents.length)"
        :error="resourceError(agents.length)"
        :empty="resourceEmpty(agents.length)"
        :no-results="resourceNoResults(agents.length, visibleAgents.length)"
        :loading-title="t('agents.states.agents.loadingTitle')"
        :loading-description="t('agents.states.agents.loadingDescription')"
        :empty-title="t('agents.states.agents.emptyTitle')"
        :empty-description="t('agents.states.agents.emptyDescription')"
        :no-results-title="t('agents.states.agents.noResultsTitle')"
        :no-results-description="t('agents.states.agents.noResultsDescription')"
      >
        <template #toolbar>
          <Toolbar density="compact" class="w-full lg:w-auto">
            <template #start>
              <span class="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{{ resourceSummary(visibleAgents.length, agents.length) }}</span>
              <UiBadge v-if="activeAgentFilterCount" variant="muted">{{ t('agents.filters.activeCount', { count: activeAgentFilterCount }) }}</UiBadge>
            </template>
            <template #end>
              <ViewModeToggle v-model="viewMode" :modes="['table', 'grid']" :label="t('agents.viewModeLabel')" />
              <DensityToggle v-model="density" :densities="['compact', 'comfortable']" :label="t('agents.densityLabel')" />
              <UiButton class="w-full sm:w-auto" @click="openAgentDialog()">
                <Plus class="h-4 w-4" />
                {{ t('agents.actions.newAgent') }}
              </UiButton>
            </template>
          </Toolbar>
        </template>

        <template #filters>
          <DataFilterBar density="compact">
            <template #search>
              <SearchInput v-model="agentSearchQuery" :label="t('agents.search.agentsLabel')" :placeholder="t('agents.search.agents')" />
            </template>
            <UiSelect v-model="agentFilterEnabled" :options="enabledFilterOptions" class="min-w-[150px] flex-1 sm:max-w-[190px]" />
            <UiSelect v-model="agentFilterRole" :options="agentRoleFilterOptions" searchable :search-placeholder="t('agents.filters.roleSearch')" :empty-text="t('agents.search.empty')" class="min-w-[170px] flex-1 sm:max-w-[240px]" />
            <template #actions>
              <SortSelect v-model="agentSortKey" :options="sortOptions" class="min-w-[190px]" />
              <UiButton v-if="activeAgentFilterCount" variant="outline" @click="clearAgentFilters">{{ t('agents.filters.clear') }}</UiButton>
            </template>
          </DataFilterBar>
        </template>

        <template #empty>
          <DataEmptyState :title="t('agents.states.agents.emptyTitle')" :description="t('agents.states.agents.emptyDescription')">
            <template #actions>
              <UiButton @click="openAgentDialog()"><Plus class="h-4 w-4" />{{ t('agents.actions.newAgent') }}</UiButton>
              <UiButton variant="outline" :disabled="loading" @click="refreshAll"><RefreshCw :class="['h-4 w-4', loading && 'animate-spin']" />{{ t('actions.refresh') }}</UiButton>
            </template>
          </DataEmptyState>
        </template>

        <template #no-results>
          <DataNoResultsState :title="t('agents.states.agents.noResultsTitle')" :description="t('agents.states.agents.noResultsDescription')">
            <template #actions><UiButton variant="outline" @click="clearAgentFilters">{{ t('agents.filters.clear') }}</UiButton></template>
          </DataNoResultsState>
        </template>

        <DataTable v-if="viewMode === 'table'" :columns="agentTableColumns" :rows="agentRows" row-key="id" :density="collectionDensity" :caption="t('agents.table.agentCaption')" class="hidden xl:block">
          <template #cell="{ row, column }">
            <div v-if="column.key === 'resource'" class="min-w-0 space-y-1">
              <p class="break-words font-medium text-foreground" :title="agentTitle(agentFromRow(row))">{{ agentTitle(agentFromRow(row)) }}</p>
              <p class="break-all font-mono text-[11px] text-muted-foreground" :title="agentFromRow(row).id">{{ agentFromRow(row).id }}</p>
              <p v-if="agentFromRow(row).description" class="line-clamp-2 text-xs leading-5 text-muted-foreground">{{ agentFromRow(row).description }}</p>
            </div>
            <div v-else-if="column.key === 'role'" class="space-y-1">
              <UiBadge variant="muted">{{ roleLabel(agentFromRow(row).role) }}</UiBadge>
              <p class="text-xs text-muted-foreground">{{ agentFromRow(row).model_id || t('common.emptyValue') }}</p>
            </div>
            <div v-else-if="column.key === 'status'" class="space-y-2">
              <UiBadge :variant="enabledVariant(agentFromRow(row).enabled)">{{ agentFromRow(row).enabled ? t('agents.enabled') : t('agents.disabled') }}</UiBadge>
              <UiSwitch :model-value="agentFromRow(row).enabled" class="min-h-10 w-36 rounded-xl px-3 py-2" :disabled="isPending('agent-toggle:' + agentFromRow(row).id)" :label="agentFromRow(row).enabled ? t('agents.enabled') : t('agents.disabled')" @update:model-value="toggleAgent(agentFromRow(row), $event)" />
            </div>
            <div v-else-if="column.key === 'connections'" class="flex flex-wrap gap-1.5">
              <UiBadge variant="muted">{{ t('agents.connectionCounts.skills', { count: agentFromRow(row).skill_ids?.length || 0 }) }}</UiBadge>
              <UiBadge variant="muted">{{ t('agents.connectionCounts.tools', { count: agentFromRow(row).tool_ids?.length || 0 }) }}</UiBadge>
              <UiBadge variant="muted">{{ t('agents.connectionCounts.mcp', { count: agentFromRow(row).mcp_server_ids?.length || 0 }) }}</UiBadge>
              <UiBadge v-if="agentFromRow(row).project_id" variant="muted">{{ agentFromRow(row).project_id }}</UiBadge>
            </div>
            <span v-else-if="column.key === 'updated'" class="text-xs text-muted-foreground">{{ dateLabel(agentFromRow(row).updated_at || agentFromRow(row).created_at) }}</span>
            <div v-else-if="column.key === 'actions'" class="flex justify-end gap-2">
              <UiButton size="sm" variant="outline" @click.stop="openAgentDialog(agentFromRow(row))"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton>
              <UiButton size="sm" variant="destructive" :loading="isPending('agent-delete:' + agentFromRow(row).id)" @click.stop="requestDeleteAgent(agentFromRow(row))"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton>
            </div>
          </template>
        </DataTable>

        <DataCardGrid :items="agentRows" :density="collectionDensity" columns="two" :class="viewMode === 'table' ? 'xl:hidden' : ''">
          <template #default="{ item }">
            <Panel as="article" :padding="panelPadding" interactive>
              <div class="flex min-w-0 flex-wrap items-start justify-between gap-4">
                <div class="min-w-0 flex-1">
                  <h3 class="break-words font-semibold" :title="agentTitle(agentFromRow(item))">{{ agentTitle(agentFromRow(item)) }}</h3>
                  <p class="mt-1 break-all font-mono text-[11px] text-muted-foreground">{{ agentFromRow(item).id }}</p>
                  <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ agentFromRow(item).description || t('agents.emptyDescriptions.agent') }}</p>
                </div>
                <UiBadge :variant="enabledVariant(agentFromRow(item).enabled)">{{ agentFromRow(item).enabled ? t('agents.enabled') : t('agents.disabled') }}</UiBadge>
              </div>
              <div class="mt-4 flex flex-wrap gap-2">
                <UiBadge variant="muted">{{ roleLabel(agentFromRow(item).role) }}</UiBadge>
                <UiBadge v-if="agentFromRow(item).model_id" variant="muted">{{ agentFromRow(item).model_id }}</UiBadge>
              </div>
              <div class="mt-4 grid gap-2 sm:grid-cols-3">
                <div class="rounded-xl bg-muted/35 p-3 text-sm"><p class="field-label text-xs">{{ t('agents.table.skills') }}</p><p class="mt-1 font-medium">{{ agentFromRow(item).skill_ids?.length || 0 }}</p></div>
                <div class="rounded-xl bg-muted/35 p-3 text-sm"><p class="field-label text-xs">{{ t('agents.table.tools') }}</p><p class="mt-1 font-medium">{{ agentFromRow(item).tool_ids?.length || 0 }}</p></div>
                <div class="rounded-xl bg-muted/35 p-3 text-sm"><p class="field-label text-xs">{{ t('agents.table.mcp') }}</p><p class="mt-1 font-medium">{{ agentFromRow(item).mcp_server_ids?.length || 0 }}</p></div>
              </div>
              <p class="mt-4 text-xs text-muted-foreground">{{ t('agents.fields.updatedAt') }}: {{ dateLabel(agentFromRow(item).updated_at || agentFromRow(item).created_at) }}</p>
              <div class="mt-5 flex flex-wrap gap-2">
                <UiButton size="sm" variant="outline" @click="openAgentDialog(agentFromRow(item))"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton>
                <UiButton size="sm" variant="destructive" :loading="isPending('agent-delete:' + agentFromRow(item).id)" @click="requestDeleteAgent(agentFromRow(item))"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton>
              </div>
              <UiSwitch :model-value="agentFromRow(item).enabled" class="mt-4" :disabled="isPending('agent-toggle:' + agentFromRow(item).id)" :label="agentFromRow(item).enabled ? t('agents.enabled') : t('agents.disabled')" @update:model-value="toggleAgent(agentFromRow(item), $event)" />
            </Panel>
          </template>
        </DataCardGrid>
      </DataCollection>
    </section>

    <section v-else-if="activeTab === 'skills'" class="space-y-4">
      <DataCollection
        :title="t('agents.sections.skills')"
        :description="t('agents.sections.skillsDescription')"
        :loading="resourceLoading(skills.length)"
        :error="resourceError(skills.length)"
        :empty="resourceEmpty(skills.length)"
        :no-results="resourceNoResults(skills.length, visibleSkills.length)"
        :loading-title="t('agents.states.skills.loadingTitle')"
        :loading-description="t('agents.states.skills.loadingDescription')"
        :empty-title="t('agents.states.skills.emptyTitle')"
        :empty-description="t('agents.states.skills.emptyDescription')"
        :no-results-title="t('agents.states.skills.noResultsTitle')"
        :no-results-description="t('agents.states.skills.noResultsDescription')"
      >
        <template #toolbar>
          <Toolbar density="compact" class="w-full lg:w-auto">
            <template #start>
              <span class="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{{ resourceSummary(visibleSkills.length, skills.length) }}</span>
              <UiBadge v-if="activeSkillFilterCount" variant="muted">{{ t('agents.filters.activeCount', { count: activeSkillFilterCount }) }}</UiBadge>
            </template>
            <template #end>
              <ViewModeToggle v-model="viewMode" :modes="['table', 'grid']" :label="t('agents.viewModeLabel')" />
              <DensityToggle v-model="density" :densities="['compact', 'comfortable']" :label="t('agents.densityLabel')" />
              <UiButton variant="outline" :loading="isPending('skill-scan:default')" @click="scanDefaultSkills"><RefreshCw class="h-4 w-4" />{{ t('agents.actions.scanSkills') }}</UiButton>
              <UiButton @click="openSkillDialog()"><Plus class="h-4 w-4" />{{ t('agents.actions.newSkill') }}</UiButton>
            </template>
          </Toolbar>
        </template>
        <template #filters>
          <DataFilterBar density="compact">
            <template #search><SearchInput v-model="skillSearchQuery" :label="t('agents.search.skillsLabel')" :placeholder="t('agents.search.skills')" /></template>
            <UiSelect v-model="skillFilterEnabled" :options="enabledFilterOptions" class="min-w-[150px] flex-1 sm:max-w-[190px]" />
            <UiSelect v-model="skillFilterSource" :options="skillSourceFilterOptions" searchable :search-placeholder="t('agents.filters.sourceSearch')" :empty-text="t('agents.search.empty')" class="min-w-[170px] flex-1 sm:max-w-[260px]" />
            <template #actions>
              <SortSelect v-model="skillSortKey" :options="sortOptions" class="min-w-[190px]" />
              <UiButton v-if="activeSkillFilterCount" variant="outline" @click="clearSkillFilters">{{ t('agents.filters.clear') }}</UiButton>
            </template>
          </DataFilterBar>
        </template>
        <template #empty>
          <DataEmptyState :title="t('agents.states.skills.emptyTitle')" :description="t('agents.states.skills.emptyDescription')">
            <template #actions>
              <UiButton @click="openSkillDialog()"><Plus class="h-4 w-4" />{{ t('agents.actions.newSkill') }}</UiButton>
              <UiButton variant="outline" :loading="isPending('skill-scan:default')" @click="scanDefaultSkills"><RefreshCw class="h-4 w-4" />{{ t('agents.actions.scanSkills') }}</UiButton>
            </template>
          </DataEmptyState>
        </template>
        <template #no-results>
          <DataNoResultsState :title="t('agents.states.skills.noResultsTitle')" :description="t('agents.states.skills.noResultsDescription')">
            <template #actions><UiButton variant="outline" @click="clearSkillFilters">{{ t('agents.filters.clear') }}</UiButton></template>
          </DataNoResultsState>
        </template>
        <DataTable v-if="viewMode === 'table'" :columns="skillTableColumns" :rows="skillRows" row-key="id" :density="collectionDensity" :caption="t('agents.table.skillCaption')" class="hidden xl:block">
          <template #cell="{ row, column }">
            <div v-if="column.key === 'resource'" class="min-w-0 space-y-1">
              <p class="break-words font-medium text-foreground">{{ skillTitle(skillFromRow(row)) }}</p>
              <p class="break-all font-mono text-[11px] text-muted-foreground">{{ skillFromRow(row).id }}</p>
              <p v-if="skillFromRow(row).description" class="line-clamp-2 text-xs leading-5 text-muted-foreground">{{ skillFromRow(row).description }}</p>
            </div>
            <p v-else-if="column.key === 'source'" class="break-all text-xs text-muted-foreground">{{ sourceLabel(skillFromRow(row).source_id) }}</p>
            <div v-else-if="column.key === 'status'" class="space-y-2">
              <UiBadge :variant="enabledVariant(skillFromRow(row).enabled)">{{ skillFromRow(row).enabled ? t('agents.enabled') : t('agents.disabled') }}</UiBadge>
              <UiSwitch :model-value="skillFromRow(row).enabled" class="min-h-10 w-36 rounded-xl px-3 py-2" :disabled="isPending('skill-toggle:' + skillFromRow(row).id)" :label="skillFromRow(row).enabled ? t('agents.enabled') : t('agents.disabled')" @update:model-value="toggleSkill(skillFromRow(row), $event)" />
            </div>
            <span v-else-if="column.key === 'updated'" class="text-xs text-muted-foreground">{{ dateLabel(skillFromRow(row).updated_at || skillFromRow(row).created_at) }}</span>
            <div v-else-if="column.key === 'actions'" class="flex justify-end gap-2">
              <UiButton size="sm" variant="outline" @click.stop="openSkillDialog(skillFromRow(row))"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton>
              <UiButton size="sm" variant="destructive" :loading="isPending('skill-delete:' + skillFromRow(row).id)" @click.stop="requestDeleteSkill(skillFromRow(row))"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton>
            </div>
          </template>
        </DataTable>
        <DataCardGrid :items="skillRows" :density="collectionDensity" columns="two" :class="viewMode === 'table' ? 'xl:hidden' : ''">
          <template #default="{ item }">
            <Panel as="article" :padding="panelPadding" interactive>
              <div class="flex min-w-0 flex-wrap items-start justify-between gap-4">
                <div class="min-w-0 flex-1">
                  <h3 class="break-words font-semibold">{{ skillTitle(skillFromRow(item)) }}</h3>
                  <p class="mt-1 break-all font-mono text-[11px] text-muted-foreground">{{ skillFromRow(item).id }}</p>
                  <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ skillFromRow(item).description || t('agents.emptyDescriptions.skill') }}</p>
                </div>
                <UiBadge :variant="enabledVariant(skillFromRow(item).enabled)">{{ skillFromRow(item).enabled ? t('agents.enabled') : t('agents.disabled') }}</UiBadge>
              </div>
              <div class="mt-4 flex flex-wrap gap-2">
                <UiBadge variant="muted">{{ sourceLabel(skillFromRow(item).source_id) }}</UiBadge>
                <UiBadge v-if="skillFromRow(item).project_id" variant="muted">{{ skillFromRow(item).project_id }}</UiBadge>
              </div>
              <p class="mt-4 text-xs text-muted-foreground">{{ t('agents.fields.updatedAt') }}: {{ dateLabel(skillFromRow(item).updated_at || skillFromRow(item).created_at) }}</p>
              <div class="mt-5 flex flex-wrap gap-2">
                <UiButton size="sm" variant="outline" @click="openSkillDialog(skillFromRow(item))"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton>
                <UiButton size="sm" variant="destructive" :loading="isPending('skill-delete:' + skillFromRow(item).id)" @click="requestDeleteSkill(skillFromRow(item))"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton>
              </div>
              <UiSwitch :model-value="skillFromRow(item).enabled" class="mt-4" :disabled="isPending('skill-toggle:' + skillFromRow(item).id)" :label="skillFromRow(item).enabled ? t('agents.enabled') : t('agents.disabled')" @update:model-value="toggleSkill(skillFromRow(item), $event)" />
            </Panel>
          </template>
        </DataCardGrid>
      </DataCollection>
    </section>

    <section v-else-if="activeTab === 'mcp'" class="space-y-4">
      <DataCollection
        :title="t('agents.sections.mcp')"
        :description="t('agents.sections.mcpDescription')"
        :loading="resourceLoading(mcpServers.length)"
        :error="resourceError(mcpServers.length)"
        :empty="resourceEmpty(mcpServers.length)"
        :no-results="resourceNoResults(mcpServers.length, visibleMCPServers.length)"
        :loading-title="t('agents.states.mcp.loadingTitle')"
        :loading-description="t('agents.states.mcp.loadingDescription')"
        :empty-title="t('agents.states.mcp.emptyTitle')"
        :empty-description="t('agents.states.mcp.emptyDescription')"
        :no-results-title="t('agents.states.mcp.noResultsTitle')"
        :no-results-description="t('agents.states.mcp.noResultsDescription')"
      >
        <template #toolbar>
          <Toolbar density="compact" class="w-full lg:w-auto">
            <template #start>
              <span class="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{{ resourceSummary(visibleMCPServers.length, mcpServers.length) }}</span>
              <UiBadge v-if="activeMCPFilterCount" variant="muted">{{ t('agents.filters.activeCount', { count: activeMCPFilterCount }) }}</UiBadge>
            </template>
            <template #end>
              <ViewModeToggle v-model="viewMode" :modes="['table', 'grid']" :label="t('agents.viewModeLabel')" />
              <DensityToggle v-model="density" :densities="['compact', 'comfortable']" :label="t('agents.densityLabel')" />
              <UiButton @click="openMCPDialog()"><Plus class="h-4 w-4" />{{ t('agents.actions.newMCP') }}</UiButton>
            </template>
          </Toolbar>
        </template>
        <template #filters>
          <DataFilterBar density="compact">
            <template #search><SearchInput v-model="mcpSearchQuery" :label="t('agents.search.mcpLabel')" :placeholder="t('agents.search.mcp')" /></template>
            <UiSelect v-model="mcpFilterEnabled" :options="enabledFilterOptions" class="min-w-[150px] flex-1 sm:max-w-[190px]" />
            <UiSelect v-model="mcpFilterStatus" :options="mcpStatusFilterOptions" class="min-w-[150px] flex-1 sm:max-w-[190px]" />
            <UiSelect v-model="mcpFilterTransport" :options="mcpTransportFilterOptions" class="min-w-[170px] flex-1 sm:max-w-[220px]" />
            <template #actions>
              <SortSelect v-model="mcpSortKey" :options="sortOptions" class="min-w-[190px]" />
              <UiButton v-if="activeMCPFilterCount" variant="outline" @click="clearMCPFilters">{{ t('agents.filters.clear') }}</UiButton>
            </template>
          </DataFilterBar>
        </template>
        <template #empty>
          <DataEmptyState :title="t('agents.states.mcp.emptyTitle')" :description="t('agents.states.mcp.emptyDescription')">
            <template #actions><UiButton @click="openMCPDialog()"><Plus class="h-4 w-4" />{{ t('agents.actions.newMCP') }}</UiButton></template>
          </DataEmptyState>
        </template>
        <template #no-results>
          <DataNoResultsState :title="t('agents.states.mcp.noResultsTitle')" :description="t('agents.states.mcp.noResultsDescription')">
            <template #actions><UiButton variant="outline" @click="clearMCPFilters">{{ t('agents.filters.clear') }}</UiButton></template>
          </DataNoResultsState>
        </template>
        <DataTable v-if="viewMode === 'table'" :columns="mcpTableColumns" :rows="mcpRows" row-key="id" :density="collectionDensity" :caption="t('agents.table.mcpCaption')" class="hidden xl:block">
          <template #cell="{ row, column }">
            <div v-if="column.key === 'resource'" class="min-w-0 space-y-1">
              <p class="break-words font-medium text-foreground">{{ mcpTitle(mcpFromRow(row)) }}</p>
              <p class="break-all font-mono text-[11px] text-muted-foreground">{{ mcpFromRow(row).id }}</p>
            </div>
            <UiBadge v-else-if="column.key === 'transport'" variant="muted">{{ transportLabel(mcpFromRow(row).transport) }}</UiBadge>
            <div v-else-if="column.key === 'status'" class="space-y-2">
              <div class="flex flex-wrap gap-1.5">
                <UiBadge :variant="statusVariant(mcpFromRow(row).status)">{{ statusLabel(mcpFromRow(row).status) }}</UiBadge>
                <UiBadge :variant="enabledVariant(mcpFromRow(row).enabled)">{{ mcpFromRow(row).enabled ? t('agents.enabled') : t('agents.disabled') }}</UiBadge>
              </div>
              <UiSwitch :model-value="mcpFromRow(row).enabled" class="min-h-10 w-36 rounded-xl px-3 py-2" :disabled="isPending('mcp-toggle:' + mcpFromRow(row).id)" :label="mcpFromRow(row).enabled ? t('agents.enabled') : t('agents.disabled')" @update:model-value="toggleMCP(mcpFromRow(row), $event)" />
            </div>
            <p v-else-if="column.key === 'endpoint'" class="break-all text-xs text-muted-foreground">{{ mcpEndpoint(mcpFromRow(row)) }}</p>
            <span v-else-if="column.key === 'updated'" class="text-xs text-muted-foreground">{{ dateLabel(mcpFromRow(row).updated_at || mcpFromRow(row).last_seen_at || mcpFromRow(row).created_at) }}</span>
            <div v-else-if="column.key === 'actions'" class="flex flex-wrap justify-end gap-2">
              <UiButton size="sm" variant="outline" @click.stop="openMCPDialog(mcpFromRow(row))"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton>
              <UiButton size="sm" variant="outline" :loading="isPending('mcp-test:' + mcpFromRow(row).id)" @click.stop="testMCPServer(mcpFromRow(row))"><TestTube2 class="h-4 w-4" />{{ t('agents.actions.testMCP') }}</UiButton>
              <UiButton size="sm" variant="outline" :loading="isPending('mcp-refresh:' + mcpFromRow(row).id)" @click.stop="refreshMCPTools(mcpFromRow(row))"><RefreshCw class="h-4 w-4" />{{ t('agents.actions.refreshTools') }}</UiButton>
              <UiButton size="sm" variant="destructive" :loading="isPending('mcp-delete:' + mcpFromRow(row).id)" @click.stop="requestDeleteMCP(mcpFromRow(row))"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton>
            </div>
          </template>
        </DataTable>
        <DataCardGrid :items="mcpRows" :density="collectionDensity" columns="two" :class="viewMode === 'table' ? 'xl:hidden' : ''">
          <template #default="{ item }">
            <Panel as="article" :padding="panelPadding" interactive>
              <div class="flex min-w-0 flex-wrap items-start justify-between gap-4">
                <div class="min-w-0 flex-1">
                  <h3 class="break-words font-semibold">{{ mcpTitle(mcpFromRow(item)) }}</h3>
                  <p class="mt-1 break-all font-mono text-[11px] text-muted-foreground">{{ mcpFromRow(item).id }}</p>
                  <p class="mt-2 break-all text-sm leading-6 text-muted-foreground">{{ mcpEndpoint(mcpFromRow(item)) }}</p>
                </div>
                <UiBadge :variant="statusVariant(mcpFromRow(item).status)">{{ statusLabel(mcpFromRow(item).status) }}</UiBadge>
              </div>
              <div class="mt-4 flex flex-wrap gap-2">
                <UiBadge variant="muted">{{ transportLabel(mcpFromRow(item).transport) }}</UiBadge>
                <UiBadge :variant="enabledVariant(mcpFromRow(item).enabled)">{{ mcpFromRow(item).enabled ? t('agents.enabled') : t('agents.disabled') }}</UiBadge>
              </div>
              <p class="mt-4 text-xs text-muted-foreground">{{ t('agents.fields.updatedAt') }}: {{ dateLabel(mcpFromRow(item).updated_at || mcpFromRow(item).last_seen_at || mcpFromRow(item).created_at) }}</p>
              <div class="mt-5 flex flex-wrap gap-2">
                <UiButton size="sm" variant="outline" @click="openMCPDialog(mcpFromRow(item))"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton>
                <UiButton size="sm" variant="outline" :loading="isPending('mcp-test:' + mcpFromRow(item).id)" @click="testMCPServer(mcpFromRow(item))"><TestTube2 class="h-4 w-4" />{{ t('agents.actions.testMCP') }}</UiButton>
                <UiButton size="sm" variant="outline" :loading="isPending('mcp-refresh:' + mcpFromRow(item).id)" @click="refreshMCPTools(mcpFromRow(item))"><RefreshCw class="h-4 w-4" />{{ t('agents.actions.refreshTools') }}</UiButton>
                <UiButton size="sm" variant="destructive" :loading="isPending('mcp-delete:' + mcpFromRow(item).id)" @click="requestDeleteMCP(mcpFromRow(item))"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton>
              </div>
              <UiSwitch :model-value="mcpFromRow(item).enabled" class="mt-4" :disabled="isPending('mcp-toggle:' + mcpFromRow(item).id)" :label="mcpFromRow(item).enabled ? t('agents.enabled') : t('agents.disabled')" @update:model-value="toggleMCP(mcpFromRow(item), $event)" />
            </Panel>
          </template>
        </DataCardGrid>
      </DataCollection>
    </section>

    <section v-else class="space-y-4">
      <DataCollection
        :title="t('agents.sections.tools')"
        :description="t('agents.sections.toolsDescription')"
        :loading="resourceLoading(tools.length)"
        :error="resourceError(tools.length)"
        :empty="resourceEmpty(tools.length)"
        :no-results="resourceNoResults(tools.length, visibleTools.length)"
        :loading-title="t('agents.states.tools.loadingTitle')"
        :loading-description="t('agents.states.tools.loadingDescription')"
        :empty-title="t('agents.states.tools.emptyTitle')"
        :empty-description="t('agents.states.tools.emptyDescription')"
        :no-results-title="t('agents.states.tools.noResultsTitle')"
        :no-results-description="t('agents.states.tools.noResultsDescription')"
      >
        <template #toolbar>
          <Toolbar density="compact" class="w-full lg:w-auto">
            <template #start>
              <span class="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{{ resourceSummary(visibleTools.length, tools.length) }}</span>
              <UiBadge v-if="activeToolFilterCount" variant="muted">{{ t('agents.filters.activeCount', { count: activeToolFilterCount }) }}</UiBadge>
            </template>
            <template #end>
              <ViewModeToggle v-model="viewMode" :modes="['table', 'grid']" :label="t('agents.viewModeLabel')" />
              <DensityToggle v-model="density" :densities="['compact', 'comfortable']" :label="t('agents.densityLabel')" />
            </template>
          </Toolbar>
        </template>
        <template #filters>
          <DataFilterBar density="compact">
            <template #search><SearchInput v-model="toolSearchQuery" :label="t('agents.search.toolsLabel')" :placeholder="t('agents.search.tools')" /></template>
            <UiSelect v-model="toolFilterKind" :options="toolKindFilterOptions" class="min-w-[150px] flex-1 sm:max-w-[190px]" />
            <UiSelect v-model="toolFilterStatus" :options="toolStatusFilterOptions" class="min-w-[150px] flex-1 sm:max-w-[190px]" />
            <template #actions>
              <SortSelect v-model="toolSortKey" :options="sortOptions" class="min-w-[190px]" />
              <UiButton v-if="activeToolFilterCount" variant="outline" @click="clearToolFilters">{{ t('agents.filters.clear') }}</UiButton>
            </template>
          </DataFilterBar>
        </template>
        <template #empty>
          <DataEmptyState :title="t('agents.states.tools.emptyTitle')" :description="t('agents.states.tools.emptyDescription')">
            <template #actions><UiButton variant="outline" :disabled="loading" @click="refreshAll"><RefreshCw :class="['h-4 w-4', loading && 'animate-spin']" />{{ t('actions.refresh') }}</UiButton></template>
          </DataEmptyState>
        </template>
        <template #no-results>
          <DataNoResultsState :title="t('agents.states.tools.noResultsTitle')" :description="t('agents.states.tools.noResultsDescription')">
            <template #actions><UiButton variant="outline" @click="clearToolFilters">{{ t('agents.filters.clear') }}</UiButton></template>
          </DataNoResultsState>
        </template>
        <DataTable v-if="viewMode === 'table'" :columns="toolTableColumns" :rows="toolRows" row-key="id" :density="collectionDensity" :caption="t('agents.table.toolCaption')" class="hidden xl:block">
          <template #cell="{ row, column }">
            <div v-if="column.key === 'resource'" class="min-w-0 space-y-1">
              <p class="break-words font-medium text-foreground">{{ toolTitle(toolFromRow(row)) }}</p>
              <p class="break-all font-mono text-[11px] text-muted-foreground">{{ toolFromRow(row).id }}</p>
              <p v-if="toolFromRow(row).description" class="line-clamp-2 text-xs leading-5 text-muted-foreground">{{ toolFromRow(row).description }}</p>
            </div>
            <UiBadge v-else-if="column.key === 'kind'" variant="muted">{{ kindLabel(toolFromRow(row).kind) }}</UiBadge>
            <div v-else-if="column.key === 'status'" class="space-y-2">
              <UiBadge :variant="statusVariant(toolFromRow(row).status)">{{ statusLabel(toolFromRow(row).status) }}</UiBadge>
              <UiSwitch :model-value="toolFromRow(row).status === 'active'" class="min-h-10 w-36 rounded-xl px-3 py-2" :disabled="isPending('tool-toggle:' + toolFromRow(row).id)" :label="toolFromRow(row).status === 'active' ? t('agents.enabled') : t('agents.disabled')" @update:model-value="toggleTool(toolFromRow(row), $event)" />
            </div>
            <p v-else-if="column.key === 'origin'" class="break-all text-xs text-muted-foreground">{{ toolOrigin(toolFromRow(row)) }}</p>
            <span v-else-if="column.key === 'updated'" class="text-xs text-muted-foreground">{{ dateLabel(toolFromRow(row).updated_at || toolFromRow(row).created_at) }}</span>
          </template>
        </DataTable>
        <DataCardGrid :items="toolRows" :density="collectionDensity" columns="two" :class="viewMode === 'table' ? 'xl:hidden' : ''">
          <template #default="{ item }">
            <Panel as="article" :padding="panelPadding" interactive>
              <div class="flex min-w-0 flex-wrap items-start justify-between gap-4">
                <div class="min-w-0 flex-1">
                  <h3 class="break-words font-semibold">{{ toolTitle(toolFromRow(item)) }}</h3>
                  <p class="mt-1 break-all font-mono text-[11px] text-muted-foreground">{{ toolFromRow(item).id }}</p>
                  <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ toolFromRow(item).description || t('agents.emptyDescriptions.tool') }}</p>
                </div>
                <UiBadge :variant="statusVariant(toolFromRow(item).status)">{{ statusLabel(toolFromRow(item).status) }}</UiBadge>
              </div>
              <div class="mt-4 flex flex-wrap gap-2">
                <UiBadge variant="muted">{{ kindLabel(toolFromRow(item).kind) }}</UiBadge>
                <UiBadge v-if="toolFromRow(item).mcp_server_id" variant="muted">{{ toolFromRow(item).mcp_server_id }}</UiBadge>
                <UiBadge v-if="toolFromRow(item).source_id" variant="muted">{{ toolFromRow(item).source_id }}</UiBadge>
              </div>
              <p class="mt-3 break-all text-xs text-muted-foreground">{{ toolOrigin(toolFromRow(item)) }}</p>
              <p class="mt-3 text-xs text-muted-foreground">{{ t('agents.fields.updatedAt') }}: {{ dateLabel(toolFromRow(item).updated_at || toolFromRow(item).created_at) }}</p>
              <UiSwitch :model-value="toolFromRow(item).status === 'active'" class="mt-4" :disabled="isPending('tool-toggle:' + toolFromRow(item).id)" :label="toolFromRow(item).status === 'active' ? t('agents.enabled') : t('agents.disabled')" @update:model-value="toggleTool(toolFromRow(item), $event)" />
            </Panel>
          </template>
        </DataCardGrid>
      </DataCollection>
    </section>

    <UiDialog v-model:open="agentDialogOpen" size="lg" :title="agentForm.id ? t('agents.dialogs.editAgent') : t('agents.dialogs.newAgent')" :description="t('agents.dialogs.agentDescription')">
      <div class="grid gap-4 md:grid-cols-2">
        <label class="space-y-2">
          <span class="field-label">{{ t('agents.fields.name') }}</span>
          <UiInput v-model="agentForm.name" required :placeholder="t('agents.placeholders.agentName')" />
        </label>
        <label class="space-y-2">
          <span class="field-label">{{ t('agents.fields.role') }}</span>
          <UiSelect v-model="agentForm.role" :options="roleOptions" />
        </label>
        <label class="space-y-2 md:col-span-2">
          <span class="field-label">{{ t('agents.fields.description') }}</span>
          <UiInput v-model="agentForm.description" :placeholder="t('agents.placeholders.description')" />
        </label>
        <label class="space-y-2 md:col-span-2">
          <span class="field-label">{{ t('agents.fields.systemPrompt') }}</span>
          <UiTextarea v-model="agentForm.system_prompt" :placeholder="t('agents.placeholders.systemPrompt')" />
        </label>
      </div>
      <UiSwitch v-model="agentForm.enabled" class="mt-4" :label="agentForm.enabled ? t('agents.enabled') : t('agents.disabled')" />
      <template #footer>
        <div class="flex flex-col gap-3 sm:flex-row sm:justify-end">
          <UiButton variant="outline" @click="agentDialogOpen = false">{{ t('actions.cancel') }}</UiButton>
          <UiButton :disabled="savingAgent" @click="saveAgent"><Save class="h-4 w-4" />{{ savingAgent ? t('actions.saving') : t('actions.save') }}</UiButton>
        </div>
      </template>
    </UiDialog>

    <UiDialog v-model:open="skillDialogOpen" size="lg" :title="skillForm.id ? t('agents.dialogs.editSkill') : t('agents.dialogs.newSkill')" :description="t('agents.dialogs.skillDescription')">
      <div class="grid gap-4 md:grid-cols-2">
        <label class="space-y-2">
          <span class="field-label">{{ t('agents.fields.name') }}</span>
          <UiInput v-model="skillForm.name" required :placeholder="t('agents.placeholders.skillName')" />
        </label>
        <label class="space-y-2">
          <span class="field-label">{{ t('agents.fields.sourceId') }}</span>
          <UiInput v-model="skillForm.source_id" :placeholder="t('agents.placeholders.sourceId')" />
        </label>
        <label class="space-y-2 md:col-span-2">
          <span class="field-label">{{ t('agents.fields.description') }}</span>
          <UiInput v-model="skillForm.description" :placeholder="t('agents.placeholders.description')" />
        </label>
        <label class="space-y-2 md:col-span-2">
          <span class="field-label">{{ t('agents.fields.skillContent') }}</span>
          <UiTextarea v-model="skillForm.content" :placeholder="t('agents.placeholders.skillContent')" />
        </label>
      </div>
      <UiSwitch v-model="skillForm.enabled" class="mt-4" :label="skillForm.enabled ? t('agents.enabled') : t('agents.disabled')" />
      <template #footer>
        <div class="flex flex-col gap-3 sm:flex-row sm:justify-end">
          <UiButton variant="outline" @click="skillDialogOpen = false">{{ t('actions.cancel') }}</UiButton>
          <UiButton :disabled="savingSkill" @click="saveSkill"><Save class="h-4 w-4" />{{ savingSkill ? t('actions.saving') : t('actions.save') }}</UiButton>
        </div>
      </template>
    </UiDialog>

    <UiDialog v-model:open="mcpDialogOpen" size="lg" :title="mcpForm.id ? t('agents.dialogs.editMCP') : t('agents.dialogs.newMCP')" :description="t('agents.dialogs.mcpDescription')">
      <div class="grid gap-4 md:grid-cols-2">
        <label class="space-y-2">
          <span class="field-label">{{ t('agents.fields.name') }}</span>
          <UiInput v-model="mcpForm.name" required :placeholder="t('agents.placeholders.mcpName')" />
        </label>
        <label class="space-y-2">
          <span class="field-label">{{ t('agents.fields.transport') }}</span>
          <UiSelect v-model="mcpForm.transport" :options="mcpTransportOptions" />
        </label>
        <label v-if="mcpForm.transport === 'stdio'" class="space-y-2 md:col-span-2">
          <span class="field-label">{{ t('agents.fields.command') }}</span>
          <UiInput v-model="mcpForm.command" :placeholder="t('agents.placeholders.command')" />
        </label>
        <label v-else class="space-y-2 md:col-span-2">
          <span class="field-label">{{ t('agents.fields.url') }}</span>
          <UiInput v-model="mcpForm.url" :placeholder="t('agents.placeholders.url')" />
        </label>
        <label class="space-y-2 md:col-span-2">
          <span class="field-label">{{ t('agents.fields.args') }}</span>
          <UiInput :model-value="argsText(mcpForm.args)" :placeholder="t('agents.placeholders.args')" @update:model-value="updateArgs" />
        </label>
      </div>
      <UiSwitch v-model="mcpForm.enabled" class="mt-4" :label="mcpForm.enabled ? t('agents.enabled') : t('agents.disabled')" />
      <template #footer>
        <div class="flex flex-col gap-3 sm:flex-row sm:justify-end">
          <UiButton variant="outline" @click="mcpDialogOpen = false">{{ t('actions.cancel') }}</UiButton>
          <UiButton :disabled="savingMCP" @click="saveMCPServer"><Save class="h-4 w-4" />{{ savingMCP ? t('actions.saving') : t('actions.save') }}</UiButton>
        </div>
      </template>
    </UiDialog>

    <UiDialog v-model:open="confirmDialogOpen" size="sm" :title="t('agents.confirmDeleteTitle')" :description="deleteTarget ? t('agents.confirmDeleteDescription', { name: deleteTarget.name }) : ''">
      <div class="rounded-2xl border border-destructive/30 bg-destructive/10 p-4 text-sm leading-6 text-destructive">
        {{ t('agents.confirmDeleteWarning') }}
      </div>
      <template #footer>
        <div class="flex flex-col gap-3 sm:flex-row sm:justify-end">
          <UiButton variant="outline" @click="confirmDialogOpen = false">{{ t('actions.cancel') }}</UiButton>
          <UiButton variant="destructive" :loading="deleteTarget ? isPending(deleteTarget.type + '-delete:' + deleteTarget.id) : false" @click="confirmDelete"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton>
        </div>
      </template>
    </UiDialog>
  </PageShell>
</template>
