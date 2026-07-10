<script setup lang="ts">
import { BookOpenText, Bot, CheckCircle2, Hammer, Pencil, PlugZap, Plus, RefreshCw, ScanSearch, TestTube2, Trash2 } from '@lucide/vue'
import AgentConfigureDialog from '~/features/agent-configure/AgentConfigureDialog.vue'
import MCPConfigureDialog from '~/features/agent-configure/MCPConfigureDialog.vue'
import SkillConfigureDialog from '~/features/agent-configure/SkillConfigureDialog.vue'
import SettingsWorkspace from '~/widgets/settings-workspace/SettingsWorkspace.vue'
import { useAgentStore } from '~/entities/agent'
import { useModelStore } from '~/entities/model'
import type { AgentConfig, MCPServerConfig, Skill, ToolDefinition } from '~/lib/types'

type ResourceKind = 'agents' | 'skills' | 'mcp' | 'tools'
type DeleteTarget = { kind: Exclude<ResourceKind, 'tools'>; id: string; name: string }

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const agentStore = useAgentStore()
const modelStore = useModelStore()
const activeKind = ref<ResourceKind>('agents')
const loading = ref(false)
const loadError = ref('')
const saving = ref(false)
const pendingAction = ref('')
const search = ref('')
const agents = computed(() => agentStore.items)
const skills = ref<Skill[]>([])
const servers = ref<MCPServerConfig[]>([])
const tools = ref<ToolDefinition[]>([])
const models = computed(() => modelStore.items)
const agentDialogOpen = ref(false)
const skillDialogOpen = ref(false)
const mcpDialogOpen = ref(false)
const selectedAgent = ref<AgentConfig | null>(null)
const selectedSkill = ref<Skill | null>(null)
const selectedServer = ref<MCPServerConfig | null>(null)
const deleteTarget = ref<DeleteTarget | null>(null)
const deleteConfirmOpen = computed({ get: () => Boolean(deleteTarget.value), set: (value) => { if (!value) deleteTarget.value = null } })
const resourceTasks = computed(() => [
  { kind: 'agents' as const, label: t('settings.agents.resources.agents'), description: t('settings.agents.resourceDescriptions.agents'), count: agents.value.length, icon: Bot },
  { kind: 'skills' as const, label: t('settings.agents.resources.skills'), description: t('settings.agents.resourceDescriptions.skills'), count: skills.value.length, icon: BookOpenText },
  { kind: 'mcp' as const, label: t('settings.agents.resources.mcp'), description: t('settings.agents.resourceDescriptions.mcp'), count: servers.value.length, icon: PlugZap },
  { kind: 'tools' as const, label: t('settings.agents.resources.tools'), description: t('settings.agents.resourceDescriptions.tools'), count: tools.value.length, icon: Hammer }
])
const visibleAgents = computed(() => filter(agents.value, (agent) => [agent.id, agent.name, agent.description, agent.role, agent.model_id, agent.project_id, ...(agent.skill_ids || []), ...(agent.tool_ids || []), ...(agent.mcp_server_ids || [])]))
const visibleSkills = computed(() => filter(skills.value, (skill) => [skill.id, skill.name, skill.description, skill.source_id, skill.project_id, skill.path]))
const visibleServers = computed(() => filter(servers.value, (server) => [server.id, server.name, server.project_id, server.transport, server.status, server.command, server.url]))
const visibleTools = computed(() => filter(tools.value, (tool) => [tool.id, tool.name, tool.display_name, tool.description, tool.kind, tool.status, tool.project_id, tool.mcp_server_id, tool.source_id, tool.skill_id]))

onMounted(loadResources)

async function loadResources() {
  loading.value = true
  loadError.value = ''
  try {
    const [, skillResult, serverResult, toolResult] = await Promise.all([
      agentStore.load({ limit: 100 }), api.listSkills({ limit: 100 }), api.listMCPServers({ limit: 100 }), api.listToolCatalog({ limit: 200 }), modelStore.load()
    ])
    skills.value = skillResult.data
    servers.value = serverResult.data
    tools.value = toolResult.data
  } catch (error) {
    console.error('[agent-settings] Failed to load resources.', error)
    loadError.value = error instanceof Error ? error.message : t('settings.agents.messages.loadFailed')
    toast.error(t('settings.agents.messages.loadFailed'), loadError.value, error)
  } finally {
    loading.value = false
  }
}

function openAgent(agent?: AgentConfig) { selectedAgent.value = agent || null; agentDialogOpen.value = true }
function openSkill(skill?: Skill) { selectedSkill.value = skill || null; skillDialogOpen.value = true }
function openMCP(server?: MCPServerConfig) { selectedServer.value = server || null; mcpDialogOpen.value = true }

async function saveAgent(agent: AgentConfig, mode: 'create' | 'edit') {
  saving.value = true
  try { const result = await agentStore.save(agent, mode); agentDialogOpen.value = false; toast.success(t('settings.agents.messages.agentSaved'), result.data.name) }
  catch (error) { console.error('[agent-settings] Failed to save Agent.', error); toast.error(t('settings.agents.messages.agentSaveFailed'), error instanceof Error ? error.message : undefined, error) }
  finally { saving.value = false }
}

async function saveSkill(skill: Skill, mode: 'create' | 'edit') {
  saving.value = true
  try { const result = await api.saveSkill(skill, mode); skills.value = merge(skills.value, result.data); skillDialogOpen.value = false; toast.success(t('settings.agents.messages.skillSaved'), result.data.name) }
  catch (error) { console.error('[agent-settings] Failed to save Skill.', error); toast.error(t('settings.agents.messages.skillSaveFailed'), error instanceof Error ? error.message : undefined, error) }
  finally { saving.value = false }
}

async function saveMCP(server: MCPServerConfig, mode: 'create' | 'edit') {
  saving.value = true
  try { const result = await api.saveMCPServer(server, mode); servers.value = merge(servers.value, result.data); mcpDialogOpen.value = false; toast.success(t('settings.agents.messages.mcpSaved'), result.data.name) }
  catch (error) { console.error('[agent-settings] Failed to save MCP Server.', error); toast.error(t('settings.agents.messages.mcpSaveFailed'), error instanceof Error ? error.message : undefined, error) }
  finally { saving.value = false }
}

async function toggleAgent(agent: AgentConfig, enabled: boolean) {
  pendingAction.value = `agent:${agent.id}`
  try { await agentStore.save({ ...agent, enabled }, 'edit') }
  catch (error) { console.error('[agent-settings] Failed to toggle Agent.', error); toast.error(t('settings.agents.messages.toggleFailed'), error instanceof Error ? error.message : undefined, error) }
  finally { pendingAction.value = '' }
}

async function toggleSkill(skill: Skill, enabled: boolean) {
  pendingAction.value = `skill:${skill.id}`
  try { const result = await api.setSkillEnabled(skill.id, enabled); skills.value = merge(skills.value, result.data) }
  catch (error) { console.error('[agent-settings] Failed to toggle Skill.', error); toast.error(t('settings.agents.messages.toggleFailed'), error instanceof Error ? error.message : undefined, error) }
  finally { pendingAction.value = '' }
}

async function toggleMCP(server: MCPServerConfig, enabled: boolean) {
  pendingAction.value = `mcp:${server.id}`
  try { const result = await api.setMCPServerEnabled(server.id, enabled); servers.value = merge(servers.value, result.data) }
  catch (error) { console.error('[agent-settings] Failed to toggle MCP.', error); toast.error(t('settings.agents.messages.toggleFailed'), error instanceof Error ? error.message : undefined, error) }
  finally { pendingAction.value = '' }
}

async function toggleTool(tool: ToolDefinition, enabled: boolean) {
  pendingAction.value = `tool:${tool.id}`
  try { const result = await api.setToolEnabled(tool.id, enabled); tools.value = merge(tools.value, result.data) }
  catch (error) { console.error('[agent-settings] Failed to toggle Tool.', error); toast.error(t('settings.agents.messages.toggleFailed'), error instanceof Error ? error.message : undefined, error) }
  finally { pendingAction.value = '' }
}

async function scanSkills() {
  pendingAction.value = 'scan'
  try { const result = await api.scanDefaultSkillSource(); await reloadSkills(); toast.success(t('settings.agents.messages.scanComplete'), t('settings.agents.messages.scanSummary', { created: result.data.created, updated: result.data.updated, deleted: result.data.deleted, unchanged: result.data.unchanged })) }
  catch (error) { console.error('[agent-settings] Failed to scan default skills.', error); toast.error(t('settings.agents.messages.scanFailed'), error instanceof Error ? error.message : undefined, error) }
  finally { pendingAction.value = '' }
}

async function testServer(server: MCPServerConfig) {
  pendingAction.value = `test:${server.id}`
  try { const result = await api.testMCPServer(server.id); servers.value = merge(servers.value, result.data.server); result.data.ok ? toast.success(t('settings.agents.messages.testPassed'), server.name) : toast.warning(t('settings.agents.messages.testFailed'), server.name) }
  catch (error) { console.error('[agent-settings] Failed to test MCP.', error); toast.error(t('settings.agents.messages.testFailed'), error instanceof Error ? error.message : undefined, error) }
  finally { pendingAction.value = '' }
}

async function refreshTools(server: MCPServerConfig) {
  pendingAction.value = `refresh:${server.id}`
  try { const result = await api.refreshMCPTools(server.id); tools.value = [...result.data.tools, ...tools.value.filter((tool) => tool.mcp_server_id !== server.id)]; toast.success(t('settings.agents.messages.toolsRefreshed'), t('settings.agents.messages.toolsRefreshSummary', { count: result.data.count, unavailable: result.data.unavailable })) }
  catch (error) { console.error('[agent-settings] Failed to refresh MCP tools.', error); toast.error(t('settings.agents.messages.toolsRefreshFailed'), error instanceof Error ? error.message : undefined, error) }
  finally { pendingAction.value = '' }
}

async function confirmDelete() {
  const target = deleteTarget.value
  if (!target) return
  pendingAction.value = `delete:${target.kind}:${target.id}`
  try {
    if (target.kind === 'agents') { await agentStore.remove(target.id) }
    else if (target.kind === 'skills') { await api.deleteSkill(target.id); skills.value = skills.value.filter((item) => item.id !== target.id) }
    else { await api.deleteMCPServer(target.id); servers.value = servers.value.filter((item) => item.id !== target.id) }
    deleteTarget.value = null
    toast.success(t('settings.agents.messages.deleted'), target.name)
  } catch (error) { console.error('[agent-settings] Failed to delete resource.', error); toast.error(t('settings.agents.messages.deleteFailed'), error instanceof Error ? error.message : undefined, error) }
  finally { pendingAction.value = '' }
}

async function reloadSkills() { skills.value = (await api.listSkills({ limit: 100 })).data }
function requestDelete(kind: DeleteTarget['kind'], item: { id: string; name?: string; display_name?: string }) { deleteTarget.value = { kind, id: item.id, name: item.display_name || item.name || item.id } }
function merge<T extends { id: string }>(items: T[], item: T) { return [...items.filter((current) => current.id !== item.id), item].sort((a, b) => a.id.localeCompare(b.id)) }
function filter<T>(items: T[], values: (item: T) => unknown[]) { const query = search.value.trim().toLocaleLowerCase(); if (!query) return items; return items.filter((item) => values(item).some((value) => String(value || '').toLocaleLowerCase().includes(query))) }
function statusTone(status: string) { if (status === 'active' || status === 'online') return 'success' as const; if (status === 'failed' || status === 'offline' || status === 'unavailable') return 'danger' as const; if (status === 'unknown') return 'warning' as const; return 'muted' as const }
</script>

<template>
  <SettingsWorkspace :title="t('settings.agents.title')" :description="t('settings.agents.description')">
    <div class="space-y-6">
      <UiInlineNotice v-if="loadError" tone="danger" :title="t('settings.agents.messages.loadFailed')" :description="loadError"><template #actions><UiButton variant="outline" size="sm" :loading="loading" @click="loadResources">{{ t('common.retry') }}</UiButton></template></UiInlineNotice>
      <div class="grid border border-border sm:grid-cols-2 xl:grid-cols-4">
        <button v-for="task in resourceTasks" :key="task.kind" type="button" :aria-pressed="activeKind === task.kind" :class="['focus-ring flex min-h-32 gap-3 border-b border-border p-4 text-left last:border-b-0 sm:border-r xl:border-b-0', activeKind === task.kind ? 'bg-foreground text-background' : 'bg-background hover:bg-surface-muted']" @click="activeKind = task.kind; search = ''"><component :is="task.icon" class="mt-1 h-5 w-5 shrink-0" /><span class="min-w-0"><span class="flex items-center justify-between gap-3"><strong class="text-lg">{{ task.label }}</strong><span class="font-mono text-sm">{{ task.count }}</span></span><span :class="['mt-2 block text-xs leading-5', activeKind === task.kind ? 'text-background/60' : 'text-muted-foreground']">{{ task.description }}</span></span></button>
      </div>

      <div class="flex flex-col gap-3 border-b border-border pb-5 sm:flex-row sm:items-end sm:justify-between"><label class="block min-w-0 flex-1 space-y-2"><span class="field-label">{{ t('settings.agents.search') }}</span><UiInput v-model="search" :placeholder="t('settings.agents.searchPlaceholder')" /></label><div class="flex flex-wrap gap-2"><UiButton variant="outline" :loading="loading" @click="loadResources"><RefreshCw class="h-4 w-4" />{{ t('actions.refresh') }}</UiButton><UiButton v-if="activeKind === 'agents'" @click="openAgent()"><Plus class="h-4 w-4" />{{ t('settings.agents.addAgent') }}</UiButton><template v-else-if="activeKind === 'skills'"><UiButton variant="outline" :loading="pendingAction === 'scan'" @click="scanSkills"><ScanSearch class="h-4 w-4" />{{ t('settings.agents.scanSkills') }}</UiButton><UiButton @click="openSkill()"><Plus class="h-4 w-4" />{{ t('settings.agents.addSkill') }}</UiButton></template><UiButton v-else-if="activeKind === 'mcp'" @click="openMCP()"><Plus class="h-4 w-4" />{{ t('settings.agents.addMCP') }}</UiButton></div></div>

      <UiAlert v-if="agentStore.listRequest.error && activeKind === 'agents' && agents.length === 0" tone="danger" :title="t('settings.agents.messages.loadFailed')" :description="agentStore.listRequest.error.message" />
      <div v-else-if="loading && agents.length + skills.length + servers.length + tools.length === 0" class="py-16 text-center text-sm font-bold text-muted-foreground">{{ t('settings.agents.messages.loading') }}</div>

      <div v-else-if="activeKind === 'agents'" class="divide-y divide-border border-y border-border">
        <article v-for="agent in visibleAgents" :key="agent.id" class="grid gap-5 py-5 xl:grid-cols-[minmax(0,1fr)_18rem_14rem] xl:items-center"><div class="min-w-0"><div class="flex flex-wrap gap-2"><h2 class="truncate text-lg font-black">{{ agent.name || agent.id }}</h2><UiBadge :tone="agent.enabled ? 'success' : 'muted'">{{ agent.enabled ? t('status.enabled') : t('status.disabled') }}</UiBadge><UiBadge tone="muted">{{ agent.role || '—' }}</UiBadge></div><p class="mt-2 break-all font-mono text-xs text-muted-foreground">{{ agent.id }}{{ agent.project_id ? ` · ${agent.project_id}` : '' }}</p><p class="mt-2 line-clamp-2 text-sm leading-6 text-muted-foreground">{{ agent.description || t('settings.agents.noDescription') }}</p></div><dl class="grid grid-cols-2 gap-3 text-xs"><div><dt class="text-muted-foreground">model_id</dt><dd class="mt-1 truncate font-mono font-bold">{{ agent.model_id || '—' }}</dd></div><div><dt class="text-muted-foreground">bindings</dt><dd class="mt-1 font-bold">{{ (agent.skill_ids?.length || 0) + (agent.tool_ids?.length || 0) + (agent.mcp_server_ids?.length || 0) }}</dd></div></dl><div class="flex flex-wrap gap-2 xl:justify-end"><UiButton size="sm" variant="outline" @click="openAgent(agent)"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton><UiButton size="sm" variant="outline" :loading="pendingAction === `agent:${agent.id}`" @click="toggleAgent(agent, !agent.enabled)">{{ agent.enabled ? t('settings.agents.disable') : t('settings.agents.enable') }}</UiButton><UiButton size="sm" variant="destructive" @click="requestDelete('agents', agent)"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton></div></article>
        <p v-if="visibleAgents.length === 0" class="py-10 text-center text-sm text-muted-foreground">{{ t('settings.agents.noResults') }}</p>
      </div>

      <div v-else-if="activeKind === 'skills'" class="divide-y divide-border border-y border-border">
        <article v-for="skill in visibleSkills" :key="skill.id" class="grid gap-5 py-5 xl:grid-cols-[minmax(0,1fr)_16rem_14rem] xl:items-center"><div><div class="flex flex-wrap gap-2"><h2 class="text-lg font-black">{{ skill.name || skill.id }}</h2><UiBadge :tone="skill.enabled ? 'success' : 'muted'">{{ skill.enabled ? t('status.enabled') : t('status.disabled') }}</UiBadge></div><p class="mt-2 break-all font-mono text-xs text-muted-foreground">{{ skill.id }} · source={{ skill.source_id || '—' }}</p><p class="mt-2 text-sm text-muted-foreground">{{ skill.description || skill.path || t('settings.agents.noDescription') }}</p></div><div class="text-xs text-muted-foreground"><p>project_id: <strong class="font-mono text-foreground">{{ skill.project_id || '—' }}</strong></p><p class="mt-2">path: <strong class="font-mono text-foreground">{{ skill.path || '—' }}</strong></p></div><div class="flex flex-wrap gap-2 xl:justify-end"><UiButton size="sm" variant="outline" @click="openSkill(skill)"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton><UiButton size="sm" variant="outline" :loading="pendingAction === `skill:${skill.id}`" @click="toggleSkill(skill, !skill.enabled)">{{ skill.enabled ? t('settings.agents.disable') : t('settings.agents.enable') }}</UiButton><UiButton size="sm" variant="destructive" @click="requestDelete('skills', skill)"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton></div></article><p v-if="visibleSkills.length === 0" class="py-10 text-center text-sm text-muted-foreground">{{ t('settings.agents.noResults') }}</p>
      </div>

      <div v-else-if="activeKind === 'mcp'" class="divide-y divide-border border-y border-border">
        <article v-for="server in visibleServers" :key="server.id" class="grid gap-5 py-5 xl:grid-cols-[minmax(0,1fr)_16rem_20rem] xl:items-center"><div><div class="flex flex-wrap gap-2"><h2 class="text-lg font-black">{{ server.name || server.id }}</h2><UiBadge :tone="statusTone(server.status)">{{ server.status }}</UiBadge><UiBadge :tone="server.enabled ? 'success' : 'muted'">{{ server.enabled ? t('status.enabled') : t('status.disabled') }}</UiBadge></div><p class="mt-2 break-all font-mono text-xs text-muted-foreground">{{ server.id }} · {{ server.transport }}</p><p class="mt-2 break-all text-sm text-muted-foreground">{{ server.transport === 'stdio' ? [server.command, ...(server.args || [])].filter(Boolean).join(' ') : server.url }}</p></div><div class="text-xs text-muted-foreground"><p>timeout_sec: <strong class="font-mono text-foreground">{{ server.timeout_sec ?? '—' }}</strong></p><p class="mt-2">secrets: <strong class="text-foreground">{{ (server.secret_headers_hint?.length || 0) + (server.secret_env_hint?.length || 0) }}</strong></p></div><div class="flex flex-wrap gap-2 xl:justify-end"><UiButton size="sm" variant="outline" @click="openMCP(server)"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton><UiButton size="sm" variant="outline" :loading="pendingAction === `test:${server.id}`" @click="testServer(server)"><TestTube2 class="h-4 w-4" />{{ t('settings.agents.test') }}</UiButton><UiButton size="sm" variant="outline" :loading="pendingAction === `refresh:${server.id}`" @click="refreshTools(server)"><RefreshCw class="h-4 w-4" />{{ t('settings.agents.refreshTools') }}</UiButton><UiButton size="sm" variant="outline" :loading="pendingAction === `mcp:${server.id}`" @click="toggleMCP(server, !server.enabled)">{{ server.enabled ? t('settings.agents.disable') : t('settings.agents.enable') }}</UiButton><UiButton size="sm" variant="destructive" @click="requestDelete('mcp', server)"><Trash2 class="h-4 w-4" /></UiButton></div></article><p v-if="visibleServers.length === 0" class="py-10 text-center text-sm text-muted-foreground">{{ t('settings.agents.noResults') }}</p>
      </div>

      <div v-else class="divide-y divide-border border-y border-border">
        <article v-for="tool in visibleTools" :key="tool.id" class="grid gap-5 py-5 xl:grid-cols-[minmax(0,1fr)_18rem_12rem] xl:items-center"><div><div class="flex flex-wrap gap-2"><h2 class="text-lg font-black">{{ tool.display_name || tool.name || tool.id }}</h2><UiBadge :tone="statusTone(tool.status)">{{ tool.status }}</UiBadge><UiBadge tone="muted">{{ tool.kind }}</UiBadge></div><p class="mt-2 break-all font-mono text-xs text-muted-foreground">{{ tool.id }}</p><p class="mt-2 text-sm text-muted-foreground">{{ tool.description || t('settings.agents.noDescription') }}</p></div><div class="text-xs text-muted-foreground"><p>mcp_server_id: <strong class="font-mono text-foreground">{{ tool.mcp_server_id || '—' }}</strong></p><p class="mt-2">source / skill: <strong class="font-mono text-foreground">{{ tool.source_id || tool.skill_id || '—' }}</strong></p></div><div class="xl:text-right"><UiButton size="sm" variant="outline" :loading="pendingAction === `tool:${tool.id}`" @click="toggleTool(tool, tool.status !== 'active')">{{ tool.status === 'active' ? t('settings.agents.disable') : t('settings.agents.enable') }}</UiButton></div></article><p v-if="visibleTools.length === 0" class="py-10 text-center text-sm text-muted-foreground">{{ t('settings.agents.noResults') }}</p>
      </div>
    </div>

    <AgentConfigureDialog v-model:open="agentDialogOpen" :agent="selectedAgent" :models="models" :saving="saving" @save="saveAgent" />
    <SkillConfigureDialog v-model:open="skillDialogOpen" :skill="selectedSkill" :saving="saving" @save="saveSkill" />
    <MCPConfigureDialog v-model:open="mcpDialogOpen" :server="selectedServer" :saving="saving" @save="saveMCP" />
    <UiConfirm v-model:open="deleteConfirmOpen" :title="t('settings.agents.deleteTitle')" :description="deleteTarget ? t('settings.agents.deleteDescription', { name: deleteTarget.name }) : ''" :loading="Boolean(deleteTarget && pendingAction === `delete:${deleteTarget.kind}:${deleteTarget.id}`)" @confirm="confirmDelete" />
  </SettingsWorkspace>
</template>
