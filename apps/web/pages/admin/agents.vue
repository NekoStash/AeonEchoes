<script setup lang="ts">
import { Bot, Hammer, PlugZap, RefreshCw, Save, Sparkles } from '@lucide/vue'
import type { AgentConfig, AgentRole, MCPServerConfig, Skill, ToolDefinition } from '~/lib/types'
import { formatDateTime } from '~/lib/utils'

const { t } = useI18n()
const api = useApi()
const workspace = useWorkspaceStore()

const roleOptions = computed(() => [
  { label: t('agents.roles.writer'), value: 'writer' },
  { label: t('agents.roles.editor'), value: 'editor' },
  { label: t('agents.roles.plot_architect'), value: 'plot-architect' },
  { label: t('agents.roles.world_builder'), value: 'world-builder' },
  { label: t('agents.roles.character_keeper'), value: 'character-keeper' },
  { label: t('agents.roles.continuity_auditor'), value: 'continuity-auditor' }
])

const mcpTransportOptions = computed(() => [
  { label: 'stdio', value: 'stdio' },
  { label: 'streamable_http', value: 'streamable_http' },
  { label: 'sse', value: 'sse' }
])

const agents = ref<AgentConfig[]>([])
const skills = ref<Skill[]>([])
const mcpServers = ref<MCPServerConfig[]>([])
const tools = ref<ToolDefinition[]>([])
const loading = ref(false)
const savingAgent = ref(false)
const savingSkill = ref(false)
const savingMCP = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

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

onMounted(() => {
  refreshAll()
})

async function refreshAll() {
  errorMessage.value = ''
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
    errorMessage.value = apiError.message || t('agents.errors.loadFailed')
  } finally {
    loading.value = false
  }
}

function resetAgentForm() {
  Object.assign(agentForm, {
    id: '',
    name: '',
    description: '',
    role: 'writer',
    enabled: true,
    system_prompt: '',
    skill_ids: [],
    tool_ids: [],
    mcp_server_ids: [],
    created_at: undefined
  })
}

function resetSkillForm() {
  Object.assign(skillForm, {
    id: '',
    source_id: '',
    name: '',
    description: '',
    content: '',
    enabled: true,
    metadata: {},
    created_at: undefined
  })
}

function resetMCPForm() {
  Object.assign(mcpForm, {
    id: '',
    name: '',
    transport: 'stdio',
    status: 'unknown',
    enabled: true,
    command: '',
    args: [],
    url: '',
    timeout_sec: 30,
    created_at: undefined
  })
}

function editAgent(agent: AgentConfig) {
  Object.assign(agentForm, {
    ...agent,
    skill_ids: [...(agent.skill_ids || [])],
    tool_ids: [...(agent.tool_ids || [])],
    mcp_server_ids: [...(agent.mcp_server_ids || [])]
  })
}

function editMCP(server: MCPServerConfig) {
  Object.assign(mcpForm, {
    ...server,
    args: [...(server.args || [])]
  })
}

async function saveAgent() {
  errorMessage.value = ''
  successMessage.value = ''
  savingAgent.value = true
  try {
    const payload: AgentConfig = {
      ...agentForm,
      name: agentForm.name.trim(),
      description: agentForm.description?.trim(),
      system_prompt: agentForm.system_prompt?.trim(),
      skill_ids: agentForm.skill_ids || [],
      tool_ids: agentForm.tool_ids || [],
      mcp_server_ids: agentForm.mcp_server_ids || []
    }
    const result = await api.saveAgent(payload, payload.id ? 'edit' : 'create')
    workspace.recordResult(t('agents.resultScopes.agentSave'), result)
    upsertById(agents.value, result.data)
    successMessage.value = t('agents.messages.agentSaved')
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
    const result = await api.saveSkill({
      ...skillForm,
      name: skillForm.name.trim(),
      description: skillForm.description?.trim(),
      content: skillForm.content?.trim()
    }, skillForm.id ? 'edit' : 'create')
    workspace.recordResult(t('agents.resultScopes.skillSave'), result)
    upsertById(skills.value, result.data)
    successMessage.value = t('agents.messages.skillSaved')
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
    const result = await api.saveMCPServer(payload, payload.id ? 'edit' : 'create')
    workspace.recordResult(t('agents.resultScopes.mcpSave'), result)
    upsertById(mcpServers.value, result.data)
    successMessage.value = t('agents.messages.mcpSaved')
    resetMCPForm()
  } catch (error) {
    const apiError = workspace.recordError(t('agents.resultScopes.mcpSave'), error)
    errorMessage.value = apiError.message || t('agents.errors.saveMCPFailed')
  } finally {
    savingMCP.value = false
  }
}

async function toggleSkill(skill: Skill, enabled: boolean) {
  const result = await api.setSkillEnabled(skill.id, enabled)
  upsertById(skills.value, result.data)
}

async function toggleMCP(server: MCPServerConfig, enabled: boolean) {
  const result = await api.setMCPServerEnabled(server.id, enabled)
  upsertById(mcpServers.value, result.data)
}

async function toggleTool(tool: ToolDefinition, enabled: boolean) {
  const result = await api.setToolEnabled(tool.id, enabled)
  upsertById(tools.value, result.data)
}

async function scanDefaultSkills() {
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
}

async function refreshMCPTools(server: MCPServerConfig) {
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

function statusVariant(status?: string) {
  if (status === 'active' || status === 'online') return 'success' as const
  if (status === 'disabled' || status === 'offline') return 'muted' as const
  if (status === 'failed' || status === 'unavailable') return 'rose' as const
  return 'gold' as const
}

function roleLabel(role?: AgentRole) {
  if (!role) return t('agents.roles.default')
  return t(`agents.roles.${role.replace(/-/g, '_')}`)
}
</script>

<template>
  <div class="space-y-6">
    <section class="rounded-3xl border border-border bg-card/90 p-6 shadow-sm">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <p class="field-label text-xs uppercase tracking-[0.18em]">{{ t('agents.eyebrow') }}</p>
          <h1 class="mt-2 text-3xl font-semibold tracking-tight">{{ t('agents.title') }}</h1>
          <p class="mt-3 max-w-3xl text-sm leading-6 text-muted-foreground">{{ t('agents.description') }}</p>
        </div>
        <UiButton variant="outline" :disabled="loading" @click="refreshAll">
          <RefreshCw class="h-4 w-4" :class="loading && 'animate-spin'" />
          {{ t('actions.refresh') }}
        </UiButton>
      </div>
      <div class="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-2xl border border-border bg-muted/25 p-4">
          <p class="field-label">{{ t('agents.stats.agents') }}</p>
          <p class="mt-1 text-2xl font-semibold">{{ activeAgents }} / {{ agents.length }}</p>
        </div>
        <div class="rounded-2xl border border-border bg-muted/25 p-4">
          <p class="field-label">{{ t('agents.stats.skills') }}</p>
          <p class="mt-1 text-2xl font-semibold">{{ activeSkills }} / {{ skills.length }}</p>
        </div>
        <div class="rounded-2xl border border-border bg-muted/25 p-4">
          <p class="field-label">{{ t('agents.stats.mcp') }}</p>
          <p class="mt-1 text-2xl font-semibold">{{ activeMCPServers }} / {{ mcpServers.length }}</p>
        </div>
        <div class="rounded-2xl border border-border bg-muted/25 p-4">
          <p class="field-label">{{ t('agents.stats.tools') }}</p>
          <p class="mt-1 text-2xl font-semibold">{{ activeTools }} / {{ tools.length }}</p>
        </div>
      </div>
      <div v-if="errorMessage" class="mt-4 rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-300/25 dark:bg-rose-300/10 dark:text-rose-100">
        {{ errorMessage }}
      </div>
      <div v-if="successMessage" class="mt-4 rounded-2xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800 dark:border-emerald-300/25 dark:bg-emerald-300/10 dark:text-emerald-100">
        {{ successMessage }}
      </div>
    </section>

    <div class="grid gap-6 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
      <UiCard>
        <div class="flex items-center gap-3">
          <Bot class="h-5 w-5 text-muted-foreground" />
          <div>
            <h2 class="font-semibold">{{ t('agents.sections.agents') }}</h2>
            <p class="text-sm text-muted-foreground">{{ t('agents.sections.agentsDescription') }}</p>
          </div>
        </div>
        <form class="mt-5 grid gap-3" @submit.prevent="saveAgent">
          <UiInput v-model="agentForm.name" :placeholder="t('agents.placeholders.agentName')" />
          <UiInput v-model="agentForm.description" :placeholder="t('agents.placeholders.description')" />
          <UiSelect v-model="agentForm.role" :options="roleOptions" />
          <UiTextarea v-model="agentForm.system_prompt" :placeholder="t('agents.placeholders.systemPrompt')" />
          <UiSwitch v-model="agentForm.enabled" :label="t('agents.enabled')" />
          <div class="flex gap-2">
            <UiButton type="submit" :disabled="savingAgent">
              <Save class="h-4 w-4" />
              {{ agentForm.id ? t('actions.save') : t('agents.actions.createAgent') }}
            </UiButton>
            <UiButton type="button" variant="outline" @click="resetAgentForm">{{ t('actions.reset') }}</UiButton>
          </div>
        </form>
        <div class="mt-5 space-y-3">
          <div v-for="agent in agents" :key="agent.id" class="rounded-2xl border border-border bg-muted/25 p-4">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="font-medium">{{ agent.name }}</p>
                <p class="mt-1 text-xs text-muted-foreground">{{ roleLabel(agent.role) }} · {{ agent.model_id || t('common.emptyValue') }}</p>
              </div>
              <div class="flex items-center gap-2">
                <UiBadge :variant="agent.enabled ? 'success' : 'muted'">{{ agent.enabled ? t('agents.enabled') : t('agents.disabled') }}</UiBadge>
                <UiButton size="sm" variant="outline" @click="editAgent(agent)">{{ t('actions.edit') }}</UiButton>
              </div>
            </div>
            <p v-if="agent.description" class="mt-3 text-sm leading-6 text-muted-foreground">{{ agent.description }}</p>
          </div>
          <p v-if="agents.length === 0" class="rounded-2xl border border-dashed border-border p-4 text-sm text-muted-foreground">{{ t('agents.empty.agents') }}</p>
        </div>
      </UiCard>

      <UiCard>
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <Sparkles class="h-5 w-5 text-muted-foreground" />
            <div>
              <h2 class="font-semibold">{{ t('agents.sections.skills') }}</h2>
              <p class="text-sm text-muted-foreground">{{ t('agents.sections.skillsDescription') }}</p>
            </div>
          </div>
          <UiButton size="sm" variant="outline" @click="scanDefaultSkills">{{ t('agents.actions.scanSkills') }}</UiButton>
        </div>
        <form class="mt-5 grid gap-3" @submit.prevent="saveSkill">
          <UiInput v-model="skillForm.name" :placeholder="t('agents.placeholders.skillName')" />
          <UiInput v-model="skillForm.description" :placeholder="t('agents.placeholders.description')" />
          <UiTextarea v-model="skillForm.content" :placeholder="t('agents.placeholders.skillContent')" />
          <UiSwitch v-model="skillForm.enabled" :label="t('agents.enabled')" />
          <UiButton type="submit" :disabled="savingSkill">
            <Save class="h-4 w-4" />
            {{ t('agents.actions.createSkill') }}
          </UiButton>
        </form>
        <div class="mt-5 space-y-3">
          <div v-for="skill in skills" :key="skill.id" class="rounded-2xl border border-border bg-muted/25 p-4">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="font-medium">{{ skill.name }}</p>
                <p class="mt-1 text-xs text-muted-foreground">{{ skill.source_id }}</p>
              </div>
              <UiSwitch :model-value="skill.enabled" class="max-w-40" :label="skill.enabled ? t('agents.enabled') : t('agents.disabled')" @update:model-value="toggleSkill(skill, $event)" />
            </div>
            <p v-if="skill.description" class="mt-3 text-sm leading-6 text-muted-foreground">{{ skill.description }}</p>
          </div>
          <p v-if="skills.length === 0" class="rounded-2xl border border-dashed border-border p-4 text-sm text-muted-foreground">{{ t('agents.empty.skills') }}</p>
        </div>
      </UiCard>
    </div>

    <div class="grid gap-6 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
      <UiCard>
        <div class="flex items-center gap-3">
          <PlugZap class="h-5 w-5 text-muted-foreground" />
          <div>
            <h2 class="font-semibold">{{ t('agents.sections.mcp') }}</h2>
            <p class="text-sm text-muted-foreground">{{ t('agents.sections.mcpDescription') }}</p>
          </div>
        </div>
        <form class="mt-5 grid gap-3" @submit.prevent="saveMCPServer">
          <UiInput v-model="mcpForm.name" :placeholder="t('agents.placeholders.mcpName')" />
          <UiSelect v-model="mcpForm.transport" :options="mcpTransportOptions" />
          <UiInput v-if="mcpForm.transport === 'stdio'" v-model="mcpForm.command" :placeholder="t('agents.placeholders.command')" />
          <UiInput v-else v-model="mcpForm.url" :placeholder="t('agents.placeholders.url')" />
          <UiInput :model-value="argsText(mcpForm.args)" :placeholder="t('agents.placeholders.args')" @update:model-value="updateArgs" />
          <UiSwitch v-model="mcpForm.enabled" :label="t('agents.enabled')" />
          <div class="flex gap-2">
            <UiButton type="submit" :disabled="savingMCP">
              <Save class="h-4 w-4" />
              {{ mcpForm.id ? t('actions.save') : t('agents.actions.createMCP') }}
            </UiButton>
            <UiButton type="button" variant="outline" @click="resetMCPForm">{{ t('actions.reset') }}</UiButton>
          </div>
        </form>
        <div class="mt-5 space-y-3">
          <div v-for="server in mcpServers" :key="server.id" class="rounded-2xl border border-border bg-muted/25 p-4">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="font-medium">{{ server.name }}</p>
                <p class="mt-1 text-xs text-muted-foreground">{{ server.transport }} · {{ server.command || server.url || t('common.emptyValue') }}</p>
              </div>
              <div class="flex items-center gap-2">
                <UiBadge :variant="statusVariant(server.status)">{{ server.status }}</UiBadge>
                <UiButton size="sm" variant="outline" @click="editMCP(server)">{{ t('actions.edit') }}</UiButton>
                <UiButton size="sm" variant="outline" @click="refreshMCPTools(server)">{{ t('agents.actions.refreshTools') }}</UiButton>
              </div>
            </div>
            <UiSwitch :model-value="server.enabled" class="mt-3" :label="server.enabled ? t('agents.enabled') : t('agents.disabled')" @update:model-value="toggleMCP(server, $event)" />
          </div>
          <p v-if="mcpServers.length === 0" class="rounded-2xl border border-dashed border-border p-4 text-sm text-muted-foreground">{{ t('agents.empty.mcp') }}</p>
        </div>
      </UiCard>

      <UiCard>
        <div class="flex items-center gap-3">
          <Hammer class="h-5 w-5 text-muted-foreground" />
          <div>
            <h2 class="font-semibold">{{ t('agents.sections.tools') }}</h2>
            <p class="text-sm text-muted-foreground">{{ t('agents.sections.toolsDescription') }}</p>
          </div>
        </div>
        <div class="mt-5 space-y-3">
          <div v-for="tool in tools" :key="tool.id" class="rounded-2xl border border-border bg-muted/25 p-4">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="font-medium">{{ tool.display_name || tool.name }}</p>
                <p class="mt-1 text-xs text-muted-foreground">{{ tool.kind }} · {{ tool.id }}</p>
                <p v-if="tool.description" class="mt-2 text-sm leading-6 text-muted-foreground">{{ tool.description }}</p>
                <p v-if="tool.updated_at" class="mt-2 text-xs text-muted-foreground">{{ formatDateTime(tool.updated_at) }}</p>
              </div>
              <div class="flex shrink-0 flex-col items-end gap-2">
                <UiBadge :variant="statusVariant(tool.status)">{{ tool.status }}</UiBadge>
                <UiSwitch :model-value="tool.status === 'active'" class="w-36" :label="tool.status === 'active' ? t('agents.enabled') : t('agents.disabled')" @update:model-value="toggleTool(tool, $event)" />
              </div>
            </div>
          </div>
          <p v-if="tools.length === 0" class="rounded-2xl border border-dashed border-border p-4 text-sm text-muted-foreground">{{ t('agents.empty.tools') }}</p>
        </div>
      </UiCard>
    </div>
  </div>
</template>
