<script setup lang="ts">
import {
  CheckCircle2,
  DatabaseZap,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Trash2,
  WifiOff
} from '@lucide/vue'
import { storeToRefs } from 'pinia'
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
import type { AgentRole, ModelConfig, ModelKind, ModelUsageKey, ModelUsageSettings, ProviderConfig, ProviderType } from '~/lib/types'
import { formatDateTime } from '~/lib/utils'

const { t } = useI18n()
const workspace = useWorkspaceStore()
const { providers, models, errors, loading, indexJobs } = storeToRefs(workspace)
const api = useApi()

const providerTypeValues: ProviderType[] = ['openai-responses', 'openai', 'anthropic', 'gemini']
const providerExampleKeyByType: Record<ProviderType, string> = {
  'openai-responses': 'openaiResponses',
  openai: 'openai',
  anthropic: 'anthropic',
  gemini: 'gemini'
}
const modelKindValues: ModelKind[] = ['text', 'embedding']
const agentRoles: AgentRole[] = [
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
const usageKeys: ModelUsageKey[] = [
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

type ModelViewMode = 'table' | 'grid' | 'list'
type ModelDensity = 'compact' | 'comfortable' | 'relaxed'
type ModelStatusFilter = '' | 'enabled' | 'disabled'
type ModelCapabilityFilter = '' | 'tools' | 'streaming' | 'default'
type ModelSortKey = 'name:asc' | 'name:desc' | 'provider:asc' | 'provider:desc' | 'context_window:desc' | 'context_window:asc' | 'routing_weight:desc' | 'routing_weight:asc' | 'activity:desc' | 'activity:asc'

const activeTab = ref('providers')
const modelSearchQuery = ref('')
const modelFilterProviderId = ref('')
const modelFilterKind = ref<ModelKind | ''>('')
const modelFilterEnabled = ref<ModelStatusFilter>('')
const modelFilterCapability = ref<ModelCapabilityFilter>('')
const modelFilterRole = ref<AgentRole | ''>('')
const modelSortKey = ref<ModelSortKey>('name:asc')
const modelViewMode = ref<ModelViewMode>('table')
const modelDensity = ref<ModelDensity>('comfortable')
const providerSaveState = ref<'idle' | 'saving' | 'saved' | 'failed'>('idle')
const modelSaveState = ref<'idle' | 'saving' | 'saved' | 'failed'>('idle')
const settingsSaveState = ref<'idle' | 'saving' | 'saved' | 'failed'>('idle')
const maintenanceState = ref<'idle' | 'running' | 'saved' | 'failed'>('idle')
const maintenanceAction = ref<'rebuild' | 'pending' | ''>('')
const providerMode = ref<'create' | 'edit'>('create')
const modelMode = ref<'create' | 'edit'>('create')
const providerDialogOpen = ref(false)
const modelDialogOpen = ref(false)
const confirmDialogOpen = ref(false)
const deleteTarget = ref<{ type: 'provider' | 'model'; id: string; name: string } | null>(null)

const usageSettings = reactive<ModelUsageSettings>({
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
})

const localProvider = reactive<ProviderConfig>(createProviderDraft())
const modelForm = reactive(createModelDraft())

const pageTabs = computed(() => [
  { label: t('models.tabs.providers'), value: 'providers', badge: String(providers.value.length) },
  { label: t('models.tabs.models'), value: 'models', badge: String(models.value.length) },
  { label: t('models.tabs.routing'), value: 'routing' },
  { label: t('models.tabs.indexJobs'), value: 'indexJobs', badge: String(indexJobs.value.length) }
])
const providerById = computed(() => new Map(providers.value.map((provider) => [provider.id, provider])))
const providerTypeOptions = computed(() => providerTypeValues.map((value) => ({ label: providerTypeLabel(value), value })))
const modelKindOptions = computed(() => modelKindValues.map((value) => ({ label: kindLabel(value), value })))
const providerSelectOptions = computed(() => providers.value.map((provider) => ({ label: providerOptionLabel(provider), value: provider.id })))
const providerFilterOptions = computed(() => [
  { label: t('models.allProviders'), value: '' },
  ...providerSelectOptions.value
])
const modelKindFilterOptions = computed(() => [
  { label: t('models.filters.allKinds'), value: '' },
  ...modelKindOptions.value
])
const modelEnabledFilterOptions = computed(() => [
  { label: t('models.filters.allStatuses'), value: '' },
  { label: t('status.enabled'), value: 'enabled' },
  { label: t('status.disabled'), value: 'disabled' }
])
const modelCapabilityFilterOptions = computed(() => [
  { label: t('models.filters.allCapabilities'), value: '' },
  { label: t('models.filters.capabilityTools'), value: 'tools', description: t('tooltips.supportsTools') },
  { label: t('models.filters.capabilityStreaming'), value: 'streaming', description: t('tooltips.streaming') },
  { label: t('models.filters.capabilityDefault'), value: 'default', description: t('tooltips.defaultForKind') }
])
const modelRoleFilterOptions = computed(() => [
  { label: t('models.filters.allRoles'), value: '' },
  ...agentRoles.map((role) => ({ label: roleLabel(role), value: role, description: t('models.filters.roleIncludesAllRoles') }))
])
const modelSortOptions = computed(() => [
  { label: t('models.sort.nameAsc'), value: 'name:asc' },
  { label: t('models.sort.nameDesc'), value: 'name:desc' },
  { label: t('models.sort.providerAsc'), value: 'provider:asc' },
  { label: t('models.sort.providerDesc'), value: 'provider:desc' },
  { label: t('models.sort.contextDesc'), value: 'context_window:desc' },
  { label: t('models.sort.contextAsc'), value: 'context_window:asc' },
  { label: t('models.sort.weightDesc'), value: 'routing_weight:desc' },
  { label: t('models.sort.weightAsc'), value: 'routing_weight:asc' },
  { label: t('models.sort.activityDesc'), value: 'activity:desc' },
  { label: t('models.sort.activityAsc'), value: 'activity:asc' }
])
const modelTableColumns = computed(() => [
  { key: 'model', label: t('models.table.model'), class: 'min-w-[260px]', headerClass: 'min-w-[260px]' },
  { key: 'provider', label: t('models.table.provider'), class: 'min-w-[180px]', headerClass: 'min-w-[180px]' },
  { key: 'kind', label: t('models.table.kind'), class: 'min-w-[120px]' },
  { key: 'status', label: t('models.table.status'), class: 'min-w-[120px]' },
  { key: 'context', label: t('models.table.context'), align: 'right' as const, class: 'min-w-[120px] tabular-nums' },
  { key: 'output', label: t('models.table.outputDimension'), class: 'min-w-[150px]' },
  { key: 'capabilities', label: t('models.table.capabilities'), class: 'min-w-[190px]' },
  { key: 'roles', label: t('models.table.roles'), class: 'min-w-[180px]' },
  { key: 'weight', label: t('models.table.routingWeight'), align: 'right' as const, class: 'min-w-[120px] tabular-nums' },
  { key: 'actions', label: t('models.table.actions'), align: 'right' as const, class: 'min-w-[150px]' }
])
const isProviderDraft = computed(() => providerMode.value === 'create')
const selectedProviderExampleKey = computed(() => providerExampleKeyByType[localProvider.provider_type || 'openai-responses'])
const maintenanceLoading = computed(() =>
  maintenanceState.value === 'running'
  || Object.keys(loading.value).some((key) => key.startsWith('index-jobs:'))
  || loading.value['index-run-pending:all']
)
const activeModelFilterCount = computed(() => [
  modelSearchQuery.value.trim(),
  modelFilterProviderId.value,
  modelFilterKind.value,
  modelFilterEnabled.value,
  modelFilterCapability.value,
  modelFilterRole.value
].filter(Boolean).length)
const hasActiveModelFilters = computed(() => activeModelFilterCount.value > 0)
const visibleModels = computed(() => {
  const filtered = models.value.filter((model) => modelMatchesCatalogFilters(model))
  return filtered.sort(compareModelsForCatalog)
})
const modelTableRows = computed<Array<Record<string, unknown>>>(() => visibleModels.value.map((model) => model as unknown as Record<string, unknown>))
const modelCatalogLoading = computed(() => Boolean(loading.value.providers) && models.value.length === 0)
const modelCatalogError = computed(() => {
  if (modelCatalogLoading.value || models.value.length > 0) return ''
  return errors.value.find((error) => /models|providers/i.test(error.endpoint))?.message || ''
})
const modelCatalogEmpty = computed(() => !modelCatalogLoading.value && !modelCatalogError.value && models.value.length === 0)
const modelCatalogNoResults = computed(() => !modelCatalogLoading.value && !modelCatalogError.value && models.value.length > 0 && visibleModels.value.length === 0)
const modelResultSummary = computed(() => t('models.filters.resultSummary', { visible: visibleModels.value.length, total: models.value.length }))
const modelPanelPadding = computed<'sm' | 'md'>(() => modelDensity.value === 'compact' ? 'sm' : 'md')
const modelCollectionDensity = computed<'compact' | 'comfortable'>(() => modelDensity.value === 'compact' ? 'compact' : 'comfortable')
const providerSummary = computed(() => ({
  total: providers.value.length,
  enabled: providers.value.filter((provider) => provider.enabled).length
}))
const modelSummary = computed(() => ({
  total: models.value.length,
  enabled: models.value.filter((model) => model.enabled).length,
  text: models.value.filter((model) => model.kind === 'text').length,
  embedding: models.value.filter((model) => model.kind === 'embedding').length
}))
const modelSelectionOptions = computed(() => {
  const options = [
    { label: t('models.inheritRouting'), value: '', description: t('models.inheritRoutingDescription') },
    ...models.value.map((model) => ({
      label: modelFriendlyLabel(model),
      description: modelOptionDescription(model),
      value: modelQualifiedId(model),
      disabled: !model.enabled,
      disabledReason: !model.enabled ? t('models.disabledModelReason') : undefined
    }))
  ]
  const knownValues = new Set(options.map((option) => option.value))
  Object.values(usageSettings)
    .filter(Boolean)
    .forEach((value) => {
      if (!knownValues.has(value)) {
        options.push({ label: t('models.unknownModel'), value, description: `${t('models.storedValue')}: ${value}` })
      }
    })
  return options
})
const usageGroups = computed(() => [
  {
    key: 'writing',
    title: t('models.routeGroups.writing.title'),
    description: t('models.routeGroups.writing.description'),
    keys: ['writer', 'editor', 'plot-architect'] as ModelUsageKey[]
  },
  {
    key: 'canon',
    title: t('models.routeGroups.canon.title'),
    description: t('models.routeGroups.canon.description'),
    keys: ['genesis-optimizer', 'world-builder', 'character-keeper', 'continuity-auditor'] as ModelUsageKey[]
  },
  {
    key: 'knowledge',
    title: t('models.routeGroups.knowledge.title'),
    description: t('models.routeGroups.knowledge.description'),
    keys: ['fact-extractor', 'graph-curator', 'embedding'] as ModelUsageKey[]
  }
])

onMounted(async () => {
  await Promise.all([workspace.loadProvidersAndModels(), loadModelUsageSettings(), workspace.loadIndexJobs()])
  resetModelForm(providers.value[0]?.id || '')
})

function createProviderDraft(): ProviderConfig {
  return {
    id: '',
    name: t('models.defaults.providerName'),
    provider_type: 'openai-responses',
    type: 'openai-responses',
    base_url: t('models.placeholders.providers.openaiResponses.baseUrl'),
    api_key: '',
    api_key_hint: undefined,
    trace_enabled: undefined,
    trace_retention_days: undefined,
    default_request_timeout_sec: undefined,
    default_model_id: '',
    metadata: undefined,
    created_at: undefined,
    updated_at: undefined,
    last_checked_at: undefined,
    last_model_refresh_at: undefined,
    streaming: true,
    enabled: true,
    status: 'unknown'
  }
}

function createModelDraft(providerId = '') {
  return {
    id: '',
    provider_id: providerId,
    name: '',
    display_name: '',
    kind: 'text' as ModelKind,
    context_window: t('models.placeholders.model.contextWindow'),
    max_output_tokens: t('models.placeholders.model.maxOutputTokens'),
    dimension: '',
    routing_weight: t('models.placeholders.model.routingWeight'),
    default_for_kind: false,
    enabled: true,
    supports_tools: true,
    supports_streaming: true,
    allowed_agent_roles: [] as AgentRole[],
    created_at: undefined as string | undefined
  }
}

function loadProviderIntoForm(provider: ProviderConfig) {
  Object.assign(localProvider, createProviderDraft(), provider, {
    type: provider.provider_type || provider.type,
    provider_type: provider.provider_type || provider.type || 'openai-responses',
    api_key: '',
    api_key_hint: provider.api_key_hint,
    default_model_id: provider.default_model_id || ''
  })
  providerMode.value = 'edit'
  providerSaveState.value = 'idle'
}

function openProviderDialog(provider?: ProviderConfig) {
  if (provider) {
    loadProviderIntoForm(provider)
  } else {
    providerMode.value = 'create'
    Object.assign(localProvider, createProviderDraft())
    providerSaveState.value = 'idle'
  }
  providerDialogOpen.value = true
}

function providerPayloadFromForm(): ProviderConfig {
  const id = localProvider.id.trim()
  const name = localProvider.name.trim()
  const baseUrl = localProvider.base_url.trim()
  if (!name) throw new Error(t('models.errors.providerNameRequired'))
  if (!baseUrl) throw new Error(t('models.errors.baseUrlRequired'))
  return {
    ...localProvider,
    id,
    name,
    base_url: baseUrl,
    api_key: localProvider.api_key?.trim() || undefined,
    default_model_id: localProvider.default_model_id?.trim() || undefined,
    type: localProvider.provider_type,
    created_at: isProviderDraft.value ? undefined : localProvider.created_at
  }
}

async function saveProvider() {
  providerSaveState.value = 'saving'
  try {
    const result = await api.saveProvider(providerPayloadFromForm(), providerMode.value)
    workspace.recordResult(t('models.resultScopes.providerSave'), result)
    const index = providers.value.findIndex((provider) => provider.id === result.data.id)
    if (index >= 0) providers.value[index] = result.data
    else providers.value.unshift(result.data)
    loadProviderIntoForm(result.data)
    if (modelMode.value === 'create') resetModelForm(result.data.id)
    providerSaveState.value = 'saved'
    providerDialogOpen.value = false
  } catch (error) {
    workspace.recordError(t('models.resultScopes.providerSave'), error)
    providerSaveState.value = 'failed'
  }
}

function requestDeleteProvider(provider: ProviderConfig) {
  deleteTarget.value = { type: 'provider', id: provider.id, name: provider.name }
  confirmDialogOpen.value = true
}

async function deleteProvider(providerId: string) {
  try {
    const result = await api.deleteProvider(providerId)
    workspace.recordResult(t('models.resultScopes.providerDelete'), result)
    providers.value = providers.value.filter((item) => item.id !== providerId)
    models.value = models.value.filter((model) => model.provider_id !== providerId)
    usageKeys.forEach((key) => {
      if (usageSettings[key].startsWith(`${providerId}:`)) usageSettings[key] = ''
    })
    if (modelFilterProviderId.value === providerId) modelFilterProviderId.value = ''
    if (modelForm.provider_id === providerId) resetModelForm(providers.value[0]?.id || '')
  } catch (error) {
    workspace.recordError(t('models.resultScopes.providerDelete'), error)
  }
}

async function refreshModels(providerId: string) {
  await workspace.refreshModels(providerId)
}

function resetModelForm(providerId = modelFilterProviderId.value || providers.value[0]?.id || '') {
  Object.assign(modelForm, createModelDraft(providerId))
  modelMode.value = 'create'
  modelSaveState.value = 'idle'
}

function openModelDialog(model?: ModelConfig, providerId = modelFilterProviderId.value || providers.value[0]?.id || '') {
  if (model) {
    Object.assign(modelForm, {
      id: model.id,
      provider_id: model.provider_id,
      name: model.name,
      display_name: model.display_name || model.name,
      kind: model.kind || 'text',
      context_window: String(model.context_window || 0),
      max_output_tokens: String(model.max_output_tokens || 0),
      dimension: model.dimension ? String(model.dimension) : '',
      routing_weight: String(model.routing_weight || 100),
      default_for_kind: Boolean(model.default_for_kind),
      enabled: Boolean(model.enabled),
      supports_tools: Boolean(model.supports_tools),
      supports_streaming: Boolean(model.supports_streaming),
      allowed_agent_roles: [...(model.allowed_agent_roles || [])],
      created_at: model.created_at
    })
    modelFilterProviderId.value = model.provider_id
    modelMode.value = 'edit'
  } else {
    resetModelForm(providerId)
  }
  modelSaveState.value = 'idle'
  modelDialogOpen.value = true
}

function parseModelNumber(value: string, fieldLabel: string) {
  const trimmed = value.trim()
  if (!trimmed) return 0
  const parsed = Number(trimmed)
  if (!Number.isFinite(parsed) || parsed < 0) {
    throw new Error(t('models.errors.invalidNumber', { field: fieldLabel }))
  }
  return Math.trunc(parsed)
}

function modelPayloadFromForm(): ModelConfig {
  const provider = providers.value.find((item) => item.id === modelForm.provider_id)
  if (!provider) throw new Error(t('models.errors.providerRequired'))
  const name = modelForm.name.trim()
  if (!name) throw new Error(t('models.errors.modelNameRequired'))
  const displayName = modelForm.display_name.trim() || name
  return {
    id: modelForm.id.trim(),
    provider_id: provider.id,
    provider_type: provider.provider_type,
    name,
    display_name: displayName,
    kind: modelForm.kind,
    context_window: parseModelNumber(modelForm.context_window, t('models.contextWindow')),
    max_output_tokens: parseModelNumber(modelForm.max_output_tokens, t('models.maxOutputTokens')),
    dimension: parseModelNumber(modelForm.dimension, t('models.dimension')),
    routing_weight: parseModelNumber(modelForm.routing_weight, t('models.routingWeight')),
    default_for_kind: modelForm.default_for_kind,
    enabled: modelForm.enabled,
    supports_tools: modelForm.supports_tools,
    supports_streaming: modelForm.supports_streaming,
    allowed_agent_roles: [...modelForm.allowed_agent_roles],
    created_at: modelForm.created_at
  }
}

async function saveModel() {
  modelSaveState.value = 'saving'
  try {
    const result = await api.saveModel(modelPayloadFromForm())
    workspace.recordResult(t('models.resultScopes.modelSave'), result)
    const index = models.value.findIndex((model) => model.id === result.data.id)
    if (index >= 0) models.value[index] = result.data
    else models.value.unshift(result.data)
    openModelDialog(result.data)
    modelSaveState.value = 'saved'
    modelDialogOpen.value = false
  } catch (error) {
    workspace.recordError(t('models.resultScopes.modelSave'), error)
    modelSaveState.value = 'failed'
  }
}

function requestDeleteModel(model: ModelConfig) {
  deleteTarget.value = { type: 'model', id: model.id, name: model.display_name || model.name }
  confirmDialogOpen.value = true
}

async function deleteModel(modelId: string) {
  try {
    const result = await api.deleteModel(modelId)
    workspace.recordResult(t('models.resultScopes.modelDelete'), result)
    models.value = models.value.filter((item) => item.id !== modelId)
    if (modelForm.id === modelId) resetModelForm(modelFilterProviderId.value || providers.value[0]?.id || '')
  } catch (error) {
    workspace.recordError(t('models.resultScopes.modelDelete'), error)
  }
}

async function confirmDelete() {
  const target = deleteTarget.value
  if (!target) return
  if (target.type === 'provider') await deleteProvider(target.id)
  else await deleteModel(target.id)
  confirmDialogOpen.value = false
  deleteTarget.value = null
}

async function loadModelUsageSettings() {
  try {
    const result = await api.getModelUsageSettings()
    workspace.recordResult(t('models.resultScopes.settingsLoad'), result)
    Object.assign(usageSettings, result.data)
  } catch (error) {
    workspace.recordError(t('models.resultScopes.settingsLoad'), error)
  }
}

async function saveModelUsageSettings() {
  settingsSaveState.value = 'saving'
  try {
    const result = await api.saveModelUsageSettings({ ...usageSettings })
    workspace.recordResult(t('models.resultScopes.settingsSave'), result)
    Object.assign(usageSettings, result.data)
    settingsSaveState.value = 'saved'
  } catch (error) {
    workspace.recordError(t('models.resultScopes.settingsSave'), error)
    settingsSaveState.value = 'failed'
  }
}

async function rebuildVectors() {
  maintenanceState.value = 'running'
  maintenanceAction.value = 'rebuild'
  try {
    const result = await api.rebuildVectors()
    workspace.recordResult(t('models.resultScopes.rebuildVectors'), result)
    await workspace.loadIndexJobs()
    maintenanceState.value = 'saved'
  } catch (error) {
    workspace.recordError(t('models.resultScopes.rebuildVectors'), error)
    maintenanceState.value = 'failed'
  }
}

async function runPendingIndexMaintenance() {
  maintenanceState.value = 'running'
  maintenanceAction.value = 'pending'
  try {
    const result = await workspace.runPendingIndexJobs(undefined, 20)
    workspace.recordResult(t('models.resultScopes.runPendingIndex'), result)
    await workspace.loadIndexJobs()
    maintenanceState.value = 'saved'
  } catch (error) {
    workspace.recordError(t('models.resultScopes.runPendingIndex'), error)
    maintenanceState.value = 'failed'
  }
}

function setModelRole(role: AgentRole, enabled: boolean) {
  if (enabled && !modelForm.allowed_agent_roles.includes(role)) {
    modelForm.allowed_agent_roles.push(role)
    return
  }
  if (!enabled) {
    modelForm.allowed_agent_roles = modelForm.allowed_agent_roles.filter((item) => item !== role)
  }
}

function modelRoleSelected(role: AgentRole) {
  return modelForm.allowed_agent_roles.includes(role)
}

function clearModelCatalogFilters() {
  modelSearchQuery.value = ''
  modelFilterProviderId.value = ''
  modelFilterKind.value = ''
  modelFilterEnabled.value = ''
  modelFilterCapability.value = ''
  modelFilterRole.value = ''
}

function normalizeSearch(value: unknown) {
  return String(value || '').trim().toLowerCase()
}

function modelFromRow(row: unknown) {
  return row as ModelConfig
}

function modelMatchesCatalogFilters(model: ModelConfig) {
  const query = normalizeSearch(modelSearchQuery.value)
  if (query && !modelSearchFields(model).some((field) => normalizeSearch(field).includes(query))) return false
  if (modelFilterProviderId.value && model.provider_id !== modelFilterProviderId.value) return false
  if (modelFilterKind.value && model.kind !== modelFilterKind.value) return false
  if (modelFilterEnabled.value === 'enabled' && !model.enabled) return false
  if (modelFilterEnabled.value === 'disabled' && model.enabled) return false
  if (!modelMatchesCapabilityFilter(model)) return false
  if (!modelMatchesRoleFilter(model)) return false
  return true
}

function modelSearchFields(model: ModelConfig) {
  const provider = providerById.value.get(model.provider_id)
  const roles = model.allowed_agent_roles || []
  return [
    model.display_name,
    model.name,
    model.id,
    model.provider_id,
    provider?.name,
    model.kind,
    kindLabel(model.kind),
    ...roles,
    ...roles.map(roleLabel)
  ].filter(Boolean)
}

function modelMatchesCapabilityFilter(model: ModelConfig) {
  if (modelFilterCapability.value === 'tools') return Boolean(model.supports_tools)
  if (modelFilterCapability.value === 'streaming') return Boolean(model.supports_streaming)
  if (modelFilterCapability.value === 'default') return Boolean(model.default_for_kind)
  return true
}

function modelMatchesRoleFilter(model: ModelConfig) {
  if (!modelFilterRole.value) return true
  const roles = model.allowed_agent_roles || []
  return roles.length === 0 || roles.includes(modelFilterRole.value)
}

function compareModelsForCatalog(left: ModelConfig, right: ModelConfig) {
  const [field, direction] = modelSortKey.value.split(':') as [ModelSortKey extends `${infer Field}:${string}` ? Field : string, 'asc' | 'desc']
  const multiplier = direction === 'asc' ? 1 : -1
  if (field === 'provider') {
    return compareText(modelProviderName(left), modelProviderName(right)) * multiplier || compareText(modelDisplayTitle(left), modelDisplayTitle(right))
  }
  if (field === 'context_window') {
    return (numericSortValue(left.context_window) - numericSortValue(right.context_window)) * multiplier || compareText(modelDisplayTitle(left), modelDisplayTitle(right))
  }
  if (field === 'routing_weight') {
    return (numericSortValue(left.routing_weight) - numericSortValue(right.routing_weight)) * multiplier || compareText(modelDisplayTitle(left), modelDisplayTitle(right))
  }
  if (field === 'activity') {
    return (modelActivityTimestamp(left) - modelActivityTimestamp(right)) * multiplier || compareText(modelDisplayTitle(left), modelDisplayTitle(right))
  }
  return compareText(modelDisplayTitle(left), modelDisplayTitle(right)) * multiplier || compareText(left.id, right.id)
}

function compareText(left: string, right: string) {
  return left.localeCompare(right, undefined, { numeric: true, sensitivity: 'base' })
}

function numericSortValue(value?: number) {
  return Number.isFinite(value) ? Number(value) : 0
}

function modelActivityTimestamp(model: ModelConfig) {
  const timestamp = Date.parse(model.updated_at || model.last_seen_at || model.created_at || '')
  return Number.isFinite(timestamp) ? timestamp : 0
}

function modelDisplayTitle(model: ModelConfig) {
  return model.display_name || model.name || model.id
}

function modelProviderName(model: ModelConfig) {
  return providerById.value.get(model.provider_id)?.name || model.provider_id
}

function modelProviderTypeLabel(model: ModelConfig) {
  const providerType = providerById.value.get(model.provider_id)?.provider_type || model.provider_type
  return providerTypeLabel(providerType)
}

function modelOutputMetrics(model: ModelConfig) {
  const metrics = [
    { key: 'output', label: t('models.outputShort'), value: formatInteger(model.max_output_tokens) }
  ]
  if (model.kind === 'embedding' || model.dimension) {
    metrics.push({ key: 'dimension', label: t('models.dimensionShort'), value: formatInteger(model.dimension) })
  }
  return metrics
}

function modelCapabilityItems(model: ModelConfig) {
  return [
    { key: 'default', label: t('models.defaultForKind'), active: Boolean(model.default_for_kind), variant: model.default_for_kind ? 'gold' as const : 'muted' as const },
    { key: 'streaming', label: t('models.capabilityLabels.streaming'), active: Boolean(model.supports_streaming), variant: model.supports_streaming ? 'default' as const : 'muted' as const },
    { key: 'tools', label: t('models.capabilityLabels.tools'), active: Boolean(model.supports_tools), variant: model.supports_tools ? 'default' as const : 'muted' as const }
  ]
}

function modelAllowedRoles(model: ModelConfig) {
  return model.allowed_agent_roles || []
}

function modelVisibleRoles(model: ModelConfig) {
  return modelAllowedRoles(model).slice(0, modelDensity.value === 'compact' ? 2 : 4)
}

function modelHiddenRoleCount(model: ModelConfig) {
  return Math.max(0, modelAllowedRoles(model).length - modelVisibleRoles(model).length)
}

function modelUpdatedLabel(model: ModelConfig) {
  const value = model.updated_at || model.last_seen_at || model.created_at
  return value ? formatDateTime(value) : t('common.emptyValue')
}

function statusVariant(status: ProviderConfig['status']) {
  if (status === 'online') return 'success'
  if (status === 'degraded') return 'gold'
  if (status === 'offline') return 'rose'
  return 'muted'
}

function providerStatusLabel(status: string) {
  return t(`status.provider.${status}`)
}

function enabledLabel(enabled: boolean) {
  return enabled ? t('status.enabled') : t('status.disabled')
}

function streamingLabel(streaming: boolean) {
  return streaming ? t('models.streamingOn') : t('models.streamingOff')
}

function providerTypeLabel(type?: ProviderType) {
  const labels: Record<ProviderType, string> = {
    'openai-responses': t('models.providerTypes.openaiResponses'),
    openai: t('models.providerTypes.openai'),
    anthropic: t('models.providerTypes.anthropic'),
    gemini: t('models.providerTypes.gemini')
  }
  return type ? labels[type] : t('common.emptyValue')
}

function kindLabel(kind?: string) {
  if (kind === 'text') return t('models.kinds.text')
  if (kind === 'embedding') return t('models.kinds.embedding')
  return prettifyToken(kind || '')
}

function roleLabel(role: string) {
  const key = role.replace(/-/g, '_')
  const messageKey = `models.roles.${key}`
  const label = t(messageKey)
  return label === messageKey ? prettifyToken(role) : label
}

function usageLabel(key: ModelUsageKey) {
  if (key === 'embedding') return t('models.usage.embedding')
  return roleLabel(key)
}

function prettifyToken(value: string) {
  if (!value) return t('common.emptyValue')
  return value
    .split(/[-_]/g)
    .filter(Boolean)
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join(' ')
}

function providerOptionLabel(provider: ProviderConfig) {
  return `${provider.name} · ${providerTypeLabel(provider.provider_type)}`
}

function providerName(providerId: string) {
  return providers.value.find((provider) => provider.id === providerId)?.name || providerId
}

function providerModelCount(providerId: string) {
  return models.value.filter((model) => model.provider_id === providerId).length
}

function modelQualifiedId(model: ModelConfig) {
  if (model.id.includes(':')) return model.id
  return `${model.provider_id}:${model.name || model.id}`
}

function modelFriendlyLabel(model: ModelConfig) {
  return `${model.display_name || model.name} · ${providerName(model.provider_id)}`
}

function modelOptionDescription(model: ModelConfig) {
  const hints = [
    kindLabel(model.kind),
    model.default_for_kind ? t('models.defaultForKind') : '',
    `${t('models.storedValue')}: ${modelQualifiedId(model)}`
  ].filter(Boolean)
  return hints.join(' · ')
}

function providerPlaceholderKey(type: ProviderType | undefined = localProvider.provider_type) {
  return `models.placeholders.providers.${providerExampleKeyByType[type || 'openai-responses']}`
}

function providerIdPlaceholder() {
  return t(`${providerPlaceholderKey()}.id`)
}

function providerNamePlaceholder() {
  return t(`${providerPlaceholderKey()}.name`)
}

function providerBaseUrlPlaceholder() {
  return t(`${providerPlaceholderKey()}.baseUrl`)
}

function modelPlaceholder(field: 'id' | 'upstreamModelId' | 'displayName' | 'contextWindow' | 'maxOutputTokens' | 'dimension') {
  const providerKey = selectedProviderExampleKey.value
  const kindKey = modelForm.kind === 'embedding' ? 'embedding' : 'text'
  const key = `models.placeholders.model.providers.${providerKey}.${kindKey}.${field}`
  const value = t(key)
  return value === key ? t(`models.placeholders.model.${field}`) : value
}

function routingWeightPlaceholder() {
  return t('models.placeholders.model.routingWeight')
}

function apiKeyConfigurationLabel(provider: ProviderConfig) {
  return provider.api_key_hint ? t('models.apiKeySavedConfigured') : t('models.noApiKeyHint')
}

function modelFeatureSummary(model: ModelConfig) {
  const features = [
    kindLabel(model.kind),
    model.supports_streaming ? t('models.supportsStreaming') : '',
    model.supports_tools ? t('models.supportsTools') : '',
    model.default_for_kind ? t('models.defaultForKind') : ''
  ].filter(Boolean)
  return features.join(t('common.pathSeparator'))
}

function providerModelDescription(providerId: string) {
  const textModels = models.value.filter((model) => model.provider_id === providerId && model.kind === 'text').length
  const embeddingModels = models.value.filter((model) => model.provider_id === providerId && model.kind === 'embedding').length
  return t('models.providerModelBreakdown', { text: textModels, embedding: embeddingModels })
}

function formatInteger(value?: number) {
  return Number(value || 0).toLocaleString()
}
</script>

<template>
  <PageShell density="normal">
    <PageHeader :title="t('models.title')" :description="t('models.description')">
      <template #actions>
        <UiButton variant="outline" :disabled="loading.providers" class="w-full sm:w-auto" @click="workspace.loadProvidersAndModels()">
          <RefreshCw :class="['h-4 w-4', loading.providers && 'animate-spin']" />
          {{ t('actions.reload') }}
        </UiButton>
      </template>
    </PageHeader>

    <StatusStack v-if="errors.length">
      <StatusAlert :errors="errors" />
    </StatusStack>

    <StatGrid columns="four">
      <StatCard :label="t('models.summary.providers')" :value="providerSummary.total" :hint="t('models.summary.enabledProviders', { count: providerSummary.enabled })" />
      <StatCard :label="t('models.summary.models')" :value="modelSummary.total" :hint="t('models.summary.enabledModels', { count: modelSummary.enabled })" tone="info" />
      <StatCard :label="t('models.summary.textModels')" :value="modelSummary.text" :hint="t('models.kinds.text')" tone="success" />
      <StatCard :label="t('models.summary.embeddingModels')" :value="modelSummary.embedding" :hint="t('models.kinds.embedding')" tone="warning" />
    </StatGrid>

    <UiTabs v-model="activeTab" :tabs="pageTabs" class="w-full" />

    <section v-if="activeTab === 'providers'" class="space-y-4">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div class="min-w-0">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('models.providerConnectionEyebrow') }}</p>
          <h2 class="mt-2 text-xl font-semibold">{{ t('models.providerConnectionTitle') }}</h2>
          <p class="mt-2 text-sm leading-7 text-muted-foreground">{{ t('models.providerConnectionDescription') }}</p>
        </div>
        <UiButton class="w-full sm:w-auto" @click="openProviderDialog()">
          <Plus class="h-4 w-4" />
          {{ t('models.addProvider') }}
        </UiButton>
      </div>

      <div v-if="providers.length === 0" class="rounded-2xl border border-border bg-muted/35 p-4 text-sm text-muted-foreground">
        {{ t('models.emptyProviders') }}
      </div>
      <div v-else class="grid gap-4 xl:grid-cols-2">
        <UiCard v-for="provider in providers" :key="provider.id" class="p-4 sm:p-5">
          <div class="flex min-w-0 flex-wrap items-start justify-between gap-4">
            <div class="min-w-0 flex-1">
              <h3 class="break-words font-semibold" :title="provider.name">{{ provider.name }}</h3>
              <p class="mt-1 break-words text-sm text-muted-foreground">{{ providerTypeLabel(provider.provider_type) }}</p>
              <p class="mt-3 break-all font-mono text-xs text-muted-foreground" :title="provider.base_url">{{ provider.base_url }}</p>
            </div>
            <UiBadge class="shrink-0" :variant="statusVariant(provider.status)">{{ providerStatusLabel(provider.status) }}</UiBadge>
          </div>

          <div class="mt-4 grid gap-3 sm:grid-cols-2">
            <div class="rounded-xl border border-border bg-muted/25 p-3">
              <p class="field-label text-xs">{{ t('models.apiKeyHint') }}<UiInfoTooltip :text="t('tooltips.apiKey')" /></p>
              <p class="mt-2 text-sm text-foreground">{{ apiKeyConfigurationLabel(provider) }}</p>
            </div>
            <div class="rounded-xl border border-border bg-muted/25 p-3">
              <p class="text-xs text-muted-foreground">{{ t('models.lastCheckedAt') }}</p>
              <p class="mt-2 text-sm text-foreground">{{ provider.last_checked_at ? formatDateTime(provider.last_checked_at) : t('models.notChecked') }}</p>
            </div>
          </div>

          <p class="mt-3 text-xs text-muted-foreground">{{ providerModelDescription(provider.id) }}</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <UiBadge :variant="provider.streaming ? 'success' : 'muted'">{{ streamingLabel(provider.streaming) }}</UiBadge>
            <UiBadge :variant="provider.enabled ? 'default' : 'muted'">{{ enabledLabel(provider.enabled) }}</UiBadge>
            <UiBadge variant="muted">{{ t('models.modelCount', { count: providerModelCount(provider.id) }) }}</UiBadge>
          </div>

          <div class="mt-5 flex flex-wrap gap-2">
            <UiButton size="sm" variant="outline" @click="openProviderDialog(provider)">
              <Pencil class="h-4 w-4" />
              {{ t('actions.edit') }}
            </UiButton>
            <UiButton size="sm" variant="outline" :disabled="loading[`models:${provider.id}`]" @click="refreshModels(provider.id)">
              <RefreshCw :class="['h-4 w-4', loading[`models:${provider.id}`] && 'animate-spin']" />
              {{ t('models.refreshModels') }}
            </UiButton>
            <UiButton size="sm" variant="destructive" @click="requestDeleteProvider(provider)">
              <Trash2 class="h-4 w-4" />
              {{ t('actions.delete') }}
            </UiButton>
          </div>
        </UiCard>
      </div>
    </section>

    <section v-else-if="activeTab === 'models'" class="space-y-4">
      <DataCollection
        :title="t('models.modelCatalogTitle')"
        :description="t('models.modelCatalogDescription')"
        :loading="modelCatalogLoading"
        :error="modelCatalogError"
        :empty="modelCatalogEmpty"
        :no-results="modelCatalogNoResults"
        :loading-title="t('models.states.loadingTitle')"
        :loading-description="t('models.states.loadingDescription')"
        :empty-title="t('models.states.emptyTitle')"
        :empty-description="t('models.states.emptyDescription')"
        :no-results-title="t('models.states.noResultsTitle')"
        :no-results-description="t('models.states.noResultsDescription')"
      >
        <template #toolbar>
          <Toolbar density="compact" class="w-full lg:w-auto">
            <template #start>
              <span class="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{{ modelResultSummary }}</span>
              <UiBadge v-if="hasActiveModelFilters" variant="muted">{{ t('models.filters.activeCount', { count: activeModelFilterCount }) }}</UiBadge>
            </template>
            <template #end>
              <ViewModeToggle v-model="modelViewMode" :modes="['table', 'grid']" :label="t('models.viewModeLabel')" />
              <DensityToggle v-model="modelDensity" :densities="['compact', 'comfortable']" :label="t('models.densityLabel')" />
              <UiButton class="w-full sm:w-auto" @click="openModelDialog()">
                <Plus class="h-4 w-4" />
                {{ t('models.newModel') }}
              </UiButton>
            </template>
          </Toolbar>
        </template>

        <template #filters>
          <DataFilterBar density="compact">
            <template #search>
              <SearchInput
                v-model="modelSearchQuery"
                :label="t('models.search.modelCatalogLabel')"
                :placeholder="t('models.search.modelCatalog')"
              />
            </template>
            <UiSelect
              v-model="modelFilterProviderId"
              :options="providerFilterOptions"
              searchable
              :search-placeholder="t('models.search.provider')"
              :empty-text="t('models.search.empty')"
              class="min-w-[180px] flex-1 sm:max-w-[260px]"
            />
            <UiSelect v-model="modelFilterKind" :options="modelKindFilterOptions" class="min-w-[150px] flex-1 sm:max-w-[190px]" />
            <UiSelect v-model="modelFilterEnabled" :options="modelEnabledFilterOptions" class="min-w-[150px] flex-1 sm:max-w-[190px]" />
            <UiSelect v-model="modelFilterCapability" :options="modelCapabilityFilterOptions" class="min-w-[170px] flex-1 sm:max-w-[220px]" />
            <UiSelect
              v-model="modelFilterRole"
              :options="modelRoleFilterOptions"
              searchable
              :search-placeholder="t('models.filters.roleSearch')"
              :empty-text="t('models.search.empty')"
              class="min-w-[170px] flex-1 sm:max-w-[220px]"
            />
            <template #actions>
              <SortSelect v-model="modelSortKey" :options="modelSortOptions" class="min-w-[190px]" />
              <UiButton v-if="hasActiveModelFilters" variant="outline" @click="clearModelCatalogFilters">
                {{ t('models.filters.clear') }}
              </UiButton>
            </template>
          </DataFilterBar>
        </template>

        <template #empty>
          <DataEmptyState :title="t('models.states.emptyTitle')" :description="t('models.states.emptyDescription')">
            <template #actions>
              <UiButton @click="openModelDialog()">
                <Plus class="h-4 w-4" />
                {{ t('models.newModel') }}
              </UiButton>
              <UiButton variant="outline" :disabled="loading.providers" @click="workspace.loadProvidersAndModels()">
                <RefreshCw :class="['h-4 w-4', loading.providers && 'animate-spin']" />
                {{ t('actions.reload') }}
              </UiButton>
            </template>
          </DataEmptyState>
        </template>

        <template #no-results>
          <DataNoResultsState :title="t('models.states.noResultsTitle')" :description="t('models.states.noResultsDescription')">
            <template #actions>
              <UiButton variant="outline" @click="clearModelCatalogFilters">{{ t('models.filters.clear') }}</UiButton>
            </template>
          </DataNoResultsState>
        </template>

        <DataTable
          v-if="modelViewMode === 'table'"
          :columns="modelTableColumns"
          :rows="modelTableRows"
          row-key="id"
          :density="modelCollectionDensity"
          :caption="t('models.table.caption')"
          class="hidden lg:block"
        >
          <template #cell="{ row, column }">
            <div v-if="column.key === 'model'" class="min-w-0 space-y-1">
              <p class="break-words font-medium text-foreground" :title="modelDisplayTitle(modelFromRow(row))">{{ modelDisplayTitle(modelFromRow(row)) }}</p>
              <p class="break-all font-mono text-[11px] text-muted-foreground" :title="modelFromRow(row).name">{{ modelFromRow(row).name }}</p>
              <p class="break-all font-mono text-[11px] text-muted-foreground" :title="modelQualifiedId(modelFromRow(row))">{{ modelQualifiedId(modelFromRow(row)) }}</p>
            </div>
            <div v-else-if="column.key === 'provider'" class="min-w-0 space-y-1">
              <p class="truncate font-medium" :title="modelProviderName(modelFromRow(row))">{{ modelProviderName(modelFromRow(row)) }}</p>
              <p class="truncate text-xs text-muted-foreground">{{ modelProviderTypeLabel(modelFromRow(row)) }}</p>
              <p class="truncate text-xs text-muted-foreground" :title="modelFromRow(row).provider_id">{{ modelFromRow(row).provider_id }}</p>
            </div>
            <UiBadge v-else-if="column.key === 'kind'" variant="muted">{{ kindLabel(modelFromRow(row).kind) }}</UiBadge>
            <UiBadge v-else-if="column.key === 'status'" :variant="modelFromRow(row).enabled ? 'success' : 'muted'">{{ enabledLabel(modelFromRow(row).enabled) }}</UiBadge>
            <span v-else-if="column.key === 'context'" class="font-mono text-sm">{{ formatInteger(modelFromRow(row).context_window) }}</span>
            <div v-else-if="column.key === 'output'" class="space-y-1 text-xs text-muted-foreground">
              <p v-for="metric in modelOutputMetrics(modelFromRow(row))" :key="metric.key"><span class="font-medium text-foreground">{{ metric.label }}:</span> {{ metric.value }}</p>
            </div>
            <div v-else-if="column.key === 'capabilities'" class="flex flex-wrap gap-1.5">
              <UiBadge v-for="capability in modelCapabilityItems(modelFromRow(row))" :key="capability.key" :variant="capability.variant">{{ capability.label }}</UiBadge>
            </div>
            <div v-else-if="column.key === 'roles'" class="flex max-w-[220px] flex-wrap gap-1.5">
              <UiBadge v-if="modelAllowedRoles(modelFromRow(row)).length === 0" variant="muted">{{ t('models.allRoles') }}</UiBadge>
              <template v-else>
                <UiBadge v-for="role in modelVisibleRoles(modelFromRow(row))" :key="role" variant="muted">{{ roleLabel(role) }}</UiBadge>
              </template>
              <UiBadge v-if="modelHiddenRoleCount(modelFromRow(row)) > 0" variant="muted">+{{ modelHiddenRoleCount(modelFromRow(row)) }}</UiBadge>
            </div>
            <span v-else-if="column.key === 'weight'" class="font-mono text-sm">{{ formatInteger(modelFromRow(row).routing_weight) }}</span>
            <div v-else-if="column.key === 'actions'" class="flex justify-end gap-2">
              <UiButton size="sm" variant="outline" @click.stop="openModelDialog(modelFromRow(row))">
                <Pencil class="h-4 w-4" />
                {{ t('actions.edit') }}
              </UiButton>
              <UiButton size="sm" variant="destructive" @click.stop="requestDeleteModel(modelFromRow(row))">
                <Trash2 class="h-4 w-4" />
                {{ t('actions.delete') }}
              </UiButton>
            </div>
          </template>
        </DataTable>

        <DataCardGrid :items="modelTableRows" :density="modelCollectionDensity" columns="two" :class="modelViewMode === 'table' ? 'lg:hidden' : ''">
          <template #default="{ item }">
            <Panel as="article" :padding="modelPanelPadding" interactive>
              <div class="flex min-w-0 flex-wrap items-start justify-between gap-4">
                <div class="min-w-0 flex-1">
                  <h3 class="break-words font-semibold" :title="modelDisplayTitle(modelFromRow(item))">{{ modelDisplayTitle(modelFromRow(item)) }}</h3>
                  <p class="mt-1 break-words text-xs text-muted-foreground">{{ modelProviderName(modelFromRow(item)) }} · {{ modelFeatureSummary(modelFromRow(item)) }}</p>
                  <p class="mt-1 break-all font-mono text-[11px] text-muted-foreground" :title="modelFromRow(item).name">{{ t('models.upstreamModelId') }}: {{ modelFromRow(item).name }}</p>
                  <p class="mt-1 break-all font-mono text-[11px] text-muted-foreground" :title="modelQualifiedId(modelFromRow(item))">{{ t('models.storedValue') }}: {{ modelQualifiedId(modelFromRow(item)) }}</p>
                </div>
                <UiBadge :variant="modelFromRow(item).enabled ? 'success' : 'muted'">{{ enabledLabel(modelFromRow(item).enabled) }}</UiBadge>
              </div>

              <div class="mt-4 grid grid-cols-2 gap-3 text-sm xl:grid-cols-4">
                <div class="rounded-xl bg-muted/35 p-3">
                  <p class="field-label text-xs">{{ t('models.contextWindow') }}<UiInfoTooltip :text="t('tooltips.contextWindow')" /></p>
                  <p class="mt-1 font-medium">{{ formatInteger(modelFromRow(item).context_window) }}</p>
                </div>
                <div class="rounded-xl bg-muted/35 p-3">
                  <p class="field-label text-xs">{{ t('models.maxOutputTokens') }}<UiInfoTooltip :text="t('tooltips.maxOutputTokens')" /></p>
                  <p class="mt-1 font-medium">{{ formatInteger(modelFromRow(item).max_output_tokens) }}</p>
                </div>
                <div class="rounded-xl bg-muted/35 p-3">
                  <p class="field-label text-xs">{{ t('models.dimension') }}<UiInfoTooltip :text="t('tooltips.dimension')" /></p>
                  <p class="mt-1 font-medium">{{ modelFromRow(item).kind === 'embedding' ? formatInteger(modelFromRow(item).dimension) : t('common.emptyValue') }}</p>
                </div>
                <div class="rounded-xl bg-muted/35 p-3">
                  <p class="field-label text-xs">{{ t('models.routingWeight') }}<UiInfoTooltip :text="t('tooltips.routingWeight')" /></p>
                  <p class="mt-1 font-medium">{{ formatInteger(modelFromRow(item).routing_weight) }}</p>
                </div>
              </div>

              <div class="mt-4 flex flex-wrap gap-2">
                <UiBadge v-for="capability in modelCapabilityItems(modelFromRow(item))" :key="capability.key" :variant="capability.variant">{{ capability.label }}</UiBadge>
                <UiBadge v-if="modelAllowedRoles(modelFromRow(item)).length === 0" variant="muted">{{ t('models.allRoles') }}</UiBadge>
              </div>

              <div v-if="modelAllowedRoles(modelFromRow(item)).length" class="mt-4">
                <p class="field-label text-xs uppercase tracking-[0.18em]">{{ t('models.allowedAgentRoles') }}<UiInfoTooltip :text="t('tooltips.allowedRoles')" /></p>
                <div class="mt-2 flex flex-wrap gap-2">
                  <UiBadge v-for="role in modelVisibleRoles(modelFromRow(item))" :key="role" variant="muted">{{ roleLabel(role) }}</UiBadge>
                  <UiBadge v-if="modelHiddenRoleCount(modelFromRow(item)) > 0" variant="muted">+{{ modelHiddenRoleCount(modelFromRow(item)) }}</UiBadge>
                </div>
              </div>

              <p class="mt-4 text-xs text-muted-foreground">{{ t('models.updatedAt') }}: {{ modelUpdatedLabel(modelFromRow(item)) }}</p>

              <div class="mt-5 flex flex-wrap gap-2">
                <UiButton size="sm" variant="outline" @click="openModelDialog(modelFromRow(item))">
                  <Pencil class="h-4 w-4" />
                  {{ t('actions.edit') }}
                </UiButton>
                <UiButton size="sm" variant="destructive" @click="requestDeleteModel(modelFromRow(item))">
                  <Trash2 class="h-4 w-4" />
                  {{ t('actions.delete') }}
                </UiButton>
              </div>
            </Panel>
          </template>
        </DataCardGrid>
      </DataCollection>
    </section>

    <section v-else-if="activeTab === 'routing'" class="space-y-4">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div class="min-w-0">
          <div class="field-label text-xs uppercase tracking-[0.18em]">
            {{ t('models.routingEyebrow') }}
            <UiInfoTooltip :text="t('tooltips.modelRouting')" />
          </div>
          <h2 class="mt-2 text-xl font-semibold">{{ t('models.routingTitle') }}</h2>
          <p class="mt-2 text-sm leading-7 text-muted-foreground">{{ t('models.routingDescription') }}</p>
        </div>
        <UiButton class="w-full sm:w-auto" :disabled="settingsSaveState === 'saving'" @click="saveModelUsageSettings">
          <Save class="h-4 w-4" />
          {{ settingsSaveState === 'saving' ? t('actions.saving') : t('models.saveUsageSettings') }}
        </UiButton>
      </div>

      <div class="grid gap-4 xl:grid-cols-3">
        <UiCard v-for="group in usageGroups" :key="group.key" class="p-4 sm:p-5">
          <h3 class="font-semibold">{{ group.title }}</h3>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ group.description }}</p>
          <div class="mt-4 space-y-4">
            <label v-for="usage in group.keys" :key="usage" class="block min-w-0 space-y-2 rounded-2xl border border-border bg-muted/20 p-3">
              <span class="field-label">
                {{ usageLabel(usage) }}
                <UiInfoTooltip :text="usage === 'embedding' ? t('tooltips.embeddingRoute') : t('tooltips.roleRoute')" />
              </span>
              <UiSelect
                v-model="usageSettings[usage]"
                :options="modelSelectionOptions"
                searchable
                :search-placeholder="t('models.search.model')"
                :empty-text="t('models.search.empty')"
              />
              <span class="block break-words text-[11px] text-muted-foreground">
                {{ t('models.currentStoredValue') }}: {{ usageSettings[usage] || t('models.inheritRouting') }}
              </span>
            </label>
          </div>
        </UiCard>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <UiBadge v-if="settingsSaveState === 'saved'" variant="success">
          <CheckCircle2 class="h-3 w-3" />
          {{ t('actions.saved') }}
        </UiBadge>
        <UiBadge v-if="settingsSaveState === 'failed'" variant="gold">
          <WifiOff class="h-3 w-3" />
          {{ t('apiError.saveFailed') }}
        </UiBadge>
      </div>
    </section>

    <section v-else class="space-y-4">
      <div class="grid gap-4 xl:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
        <UiCard class="p-4 sm:p-5">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('models.embeddingMaintenanceEyebrow') }}</p>
          <h2 class="mt-2 text-xl font-semibold">{{ t('models.embeddingMaintenanceTitle') }}</h2>
          <p class="mt-2 text-sm leading-7 text-muted-foreground">{{ t('models.embeddingMaintenanceDescription') }}</p>

          <div class="mt-4 grid gap-4">
            <div class="rounded-2xl border border-border bg-muted/25 p-4">
              <p class="text-sm font-medium text-foreground">{{ t('models.rebuildVectorsTitle') }}</p>
              <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ t('models.rebuildVectorsDescription') }}</p>
              <UiButton class="mt-4 w-full" :disabled="maintenanceLoading" @click="rebuildVectors">
                <Loader2 v-if="maintenanceState === 'running' && maintenanceAction === 'rebuild'" class="h-4 w-4 animate-spin" />
                <DatabaseZap v-else class="h-4 w-4" />
                {{ maintenanceState === 'running' && maintenanceAction === 'rebuild' ? t('models.maintenance.rebuildRunning') : t('models.rebuildVectorsAction') }}
              </UiButton>
            </div>
            <div class="rounded-2xl border border-border bg-muted/25 p-4">
              <p class="field-label text-sm text-foreground">{{ t('models.indexMaintenanceTitle') }}<UiInfoTooltip :text="t('tooltips.indexJobs')" /></p>
              <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ t('models.indexMaintenanceDescription') }}</p>
              <UiButton variant="archive" class="mt-4 w-full" :disabled="maintenanceLoading" @click="runPendingIndexMaintenance">
                <RefreshCw :class="['h-4 w-4', maintenanceState === 'running' && maintenanceAction === 'pending' && 'animate-spin']" />
                {{ maintenanceState === 'running' && maintenanceAction === 'pending' ? t('models.maintenance.pendingRunning') : t('models.runPendingIndex') }}
              </UiButton>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap items-center gap-3">
            <UiBadge v-if="maintenanceState === 'saved'" variant="success"><CheckCircle2 class="h-3 w-3" />{{ t('actions.saved') }}</UiBadge>
            <UiBadge v-if="maintenanceState === 'failed'" variant="gold"><WifiOff class="h-3 w-3" />{{ t('apiError.saveFailed') }}</UiBadge>
          </div>
        </UiCard>

        <UiCard class="p-4 sm:p-5">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p class="field-label text-xs uppercase tracking-[0.18em]">{{ t('models.indexJobsTitle') }}<UiInfoTooltip :text="t('tooltips.indexJobs')" /></p>
              <h2 class="mt-2 text-xl font-semibold">{{ t('models.tabs.indexJobs') }}</h2>
            </div>
            <UiButton variant="outline" size="sm" @click="workspace.loadIndexJobs()">
              <RefreshCw class="h-4 w-4" />
              {{ t('actions.refresh') }}
            </UiButton>
          </div>
          <AppTaskBoard class="mt-5" :jobs="indexJobs" />
        </UiCard>
      </div>
    </section>

    <UiDialog v-model:open="providerDialogOpen" size="lg" :title="isProviderDraft ? t('models.newProvider') : t('models.providerConfig')" :description="t('models.providerDialogDescription')">
      <div class="space-y-5">
        <div class="rounded-2xl border border-amber-300/40 bg-amber-50 px-4 py-3 text-sm leading-6 text-amber-900 dark:border-amber-300/20 dark:bg-amber-300/10 dark:text-amber-100">
          {{ t('models.protocolCompatibilityHint') }}
        </div>
        <div class="grid gap-4 md:grid-cols-2">
          <label class="space-y-2">
            <span class="field-label">{{ t('models.providerId') }}</span>
            <UiInput v-model="localProvider.id" :placeholder="providerIdPlaceholder()" :disabled="providerMode === 'edit'" />
          </label>
          <label class="space-y-2">
            <span class="field-label">{{ t('models.displayName') }}</span>
            <UiInput v-model="localProvider.name" :placeholder="providerNamePlaceholder()" />
          </label>
          <label class="space-y-2">
            <span class="field-label">{{ t('models.providerType') }}<UiInfoTooltip :text="t('tooltips.providerType')" /></span>
            <UiSelect v-model="localProvider.provider_type" :options="providerTypeOptions" />
          </label>
          <label class="space-y-2">
            <span class="field-label">{{ t('models.baseUrl') }}<UiInfoTooltip :text="t('tooltips.baseUrl')" /></span>
            <UiInput v-model="localProvider.base_url" :placeholder="providerBaseUrlPlaceholder()" />
          </label>
          <label class="space-y-2 md:col-span-2">
            <span class="field-label">{{ t('models.apiKey') }}<UiInfoTooltip :text="t('tooltips.apiKey')" /></span>
            <UiInput v-model="localProvider.api_key" type="password" :placeholder="t('models.apiKeyPlaceholder')" />
            <span class="block text-xs leading-5 text-muted-foreground">{{ t('models.apiKeyBlankKeepsExisting') }}</span>
          </label>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <UiSwitch v-model="localProvider.enabled" :label="t('models.enableProvider')" />
          <UiSwitch v-model="localProvider.streaming" :label="t('models.streaming')" :description="t('tooltips.streaming')" />
        </div>
        <div v-if="providerMode === 'edit'" class="rounded-2xl border border-border bg-muted/25 px-4 py-3 text-sm text-muted-foreground">
          {{ t('models.apiKeyHint') }}
          <p class="mt-1 text-foreground">{{ localProvider.api_key_hint ? t('models.apiKeySavedConfigured') : t('models.noApiKeyHint') }}</p>
        </div>
      </div>
      <template #footer>
        <div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:justify-end">
          <UiBadge v-if="providerSaveState === 'failed'" variant="gold"><WifiOff class="h-3 w-3" />{{ t('apiError.saveFailed') }}</UiBadge>
          <UiButton variant="outline" @click="providerDialogOpen = false">{{ t('actions.cancel') }}</UiButton>
          <UiButton :disabled="providerSaveState === 'saving'" @click="saveProvider">
            <Save class="h-4 w-4" />
            {{ providerSaveState === 'saving' ? t('actions.saving') : t('actions.saveConfig') }}
          </UiButton>
        </div>
      </template>
    </UiDialog>

    <UiDialog v-model:open="modelDialogOpen" size="xl" :title="modelMode === 'edit' ? t('models.editModel') : t('models.addModel')" :description="t('models.modelDialogDescription')">
      <div class="space-y-5">
        <div class="grid gap-4 md:grid-cols-2">
          <label class="space-y-2">
            <span class="field-label">{{ t('models.modelId') }}</span>
            <UiInput v-model="modelForm.id" :disabled="modelMode === 'edit'" :placeholder="modelPlaceholder('id')" />
          </label>
          <label class="space-y-2">
            <span class="field-label">{{ t('models.modelProvider') }}</span>
            <UiSelect v-model="modelForm.provider_id" :options="providerSelectOptions" :placeholder="t('models.selectProvider')" searchable :search-placeholder="t('models.search.provider')" :empty-text="t('models.search.empty')" />
          </label>
          <label class="space-y-2">
            <span class="field-label">{{ t('models.kind') }}<UiInfoTooltip :text="t('tooltips.modelKind')" /></span>
            <UiSelect v-model="modelForm.kind" :options="modelKindOptions" />
          </label>
          <label class="space-y-2">
            <span class="field-label">{{ t('models.upstreamModelId') }}</span>
            <UiInput v-model="modelForm.name" :placeholder="modelPlaceholder('upstreamModelId')" />
          </label>
          <label class="space-y-2">
            <span class="field-label">{{ t('models.displayName') }}</span>
            <UiInput v-model="modelForm.display_name" :placeholder="modelPlaceholder('displayName')" />
          </label>
          <label class="space-y-2">
            <span class="field-label">{{ t('models.contextWindow') }}<UiInfoTooltip :text="t('tooltips.contextWindow')" /></span>
            <UiInput v-model="modelForm.context_window" type="number" :placeholder="modelPlaceholder('contextWindow')" />
          </label>
          <label class="space-y-2">
            <span class="field-label">{{ t('models.maxOutputTokens') }}<UiInfoTooltip :text="t('tooltips.maxOutputTokens')" /></span>
            <UiInput v-model="modelForm.max_output_tokens" type="number" :placeholder="modelPlaceholder('maxOutputTokens')" />
          </label>
          <label class="space-y-2">
            <span class="field-label">{{ t('models.dimension') }}<UiInfoTooltip :text="t('tooltips.dimension')" /></span>
            <UiInput v-model="modelForm.dimension" type="number" :placeholder="modelPlaceholder('dimension')" />
          </label>
          <label class="space-y-2 md:col-span-2">
            <span class="field-label">{{ t('models.routingWeight') }}<UiInfoTooltip :text="t('tooltips.routingWeight')" /></span>
            <UiInput v-model="modelForm.routing_weight" type="number" :placeholder="routingWeightPlaceholder()" />
          </label>
        </div>

        <div class="grid gap-3 sm:grid-cols-2">
          <UiSwitch v-model="modelForm.enabled" :label="t('models.enabled')" />
          <UiSwitch v-model="modelForm.default_for_kind" :label="t('models.defaultForKind')" :description="t('tooltips.defaultForKind')" />
          <UiSwitch v-model="modelForm.supports_tools" :label="t('models.supportsTools')" :description="t('tooltips.supportsTools')" />
          <UiSwitch v-model="modelForm.supports_streaming" :label="t('models.supportsStreaming')" :description="t('tooltips.streaming')" />
        </div>

        <div class="rounded-2xl border border-border bg-muted/20 p-4">
          <p class="field-label text-sm font-medium text-foreground">{{ t('models.allowedAgentRoles') }}<UiInfoTooltip :text="t('tooltips.allowedRoles')" /></p>
          <p class="mt-1 text-xs text-muted-foreground">{{ t('models.allowedAgentRolesDescription') }}</p>
          <div class="mt-4 grid gap-2 sm:grid-cols-2">
            <UiSwitch v-for="role in agentRoles" :key="role" :model-value="modelRoleSelected(role)" :label="roleLabel(role)" class="py-2" @update:model-value="setModelRole(role, $event)" />
          </div>
        </div>
      </div>
      <template #footer>
        <div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:justify-end">
          <UiBadge v-if="modelSaveState === 'failed'" variant="gold"><WifiOff class="h-3 w-3" />{{ t('apiError.saveFailed') }}</UiBadge>
          <UiButton variant="outline" @click="modelDialogOpen = false">{{ t('actions.cancel') }}</UiButton>
          <UiButton :disabled="modelSaveState === 'saving'" @click="saveModel">
            <Save class="h-4 w-4" />
            {{ modelSaveState === 'saving' ? t('actions.saving') : t('models.saveModel') }}
          </UiButton>
        </div>
      </template>
    </UiDialog>

    <UiDialog v-model:open="confirmDialogOpen" size="sm" :title="t('models.confirmDeleteTitle')" :description="deleteTarget ? t(deleteTarget.type === 'provider' ? 'models.confirmDeleteProvider' : 'models.confirmDeleteModel', { name: deleteTarget.name }) : ''">
      <div class="rounded-2xl border border-destructive/30 bg-destructive/10 p-4 text-sm leading-6 text-destructive">
        {{ t('models.confirmDeleteWarning') }}
      </div>
      <template #footer>
        <div class="flex flex-col gap-3 sm:flex-row sm:justify-end">
          <UiButton variant="outline" @click="confirmDialogOpen = false">{{ t('actions.cancel') }}</UiButton>
          <UiButton variant="destructive" @click="confirmDelete">
            <Trash2 class="h-4 w-4" />
            {{ t('actions.delete') }}
          </UiButton>
        </div>
      </template>
    </UiDialog>
  </PageShell>
</template>
