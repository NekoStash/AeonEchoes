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

const activeTab = ref('providers')
const modelFilterProviderId = ref('')
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
const providerTypeOptions = computed(() => providerTypeValues.map((value) => ({ label: providerTypeLabel(value), value })))
const modelKindOptions = computed(() => modelKindValues.map((value) => ({ label: kindLabel(value), value })))
const providerSelectOptions = computed(() => providers.value.map((provider) => ({ label: providerOptionLabel(provider), value: provider.id })))
const providerFilterOptions = computed(() => [
  { label: t('models.allProviders'), value: '' },
  ...providerSelectOptions.value
])
const isProviderDraft = computed(() => providerMode.value === 'create')
const selectedProviderExampleKey = computed(() => providerExampleKeyByType[localProvider.provider_type || 'openai-responses'])
const maintenanceLoading = computed(() =>
  maintenanceState.value === 'running'
  || Object.keys(loading.value).some((key) => key.startsWith('index-jobs:'))
  || loading.value['index-run-pending:all']
)
const visibleModels = computed(() => {
  const source = !modelFilterProviderId.value
    ? models.value
    : models.value.filter((model) => model.provider_id === modelFilterProviderId.value)
  return [...source].sort((left, right) => (left.display_name || left.name).localeCompare(right.display_name || right.name))
})
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
  <div class="space-y-6">
    <SectionHeader :title="t('models.title')" :description="t('models.description')">
      <template #actions>
        <UiButton variant="outline" :disabled="loading.providers" class="w-full sm:w-auto" @click="workspace.loadProvidersAndModels()">
          <RefreshCw :class="['h-4 w-4', loading.providers && 'animate-spin']" />
          {{ t('actions.reload') }}
        </UiButton>
      </template>
    </SectionHeader>

    <StatusAlert :errors="errors" />

    <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <UiCard class="p-4">
        <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('models.summary.providers') }}</p>
        <p class="mt-3 text-2xl font-semibold">{{ providerSummary.total }}</p>
        <p class="mt-1 text-xs text-muted-foreground">{{ t('models.summary.enabledProviders', { count: providerSummary.enabled }) }}</p>
      </UiCard>
      <UiCard class="p-4">
        <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('models.summary.models') }}</p>
        <p class="mt-3 text-2xl font-semibold">{{ modelSummary.total }}</p>
        <p class="mt-1 text-xs text-muted-foreground">{{ t('models.summary.enabledModels', { count: modelSummary.enabled }) }}</p>
      </UiCard>
      <UiCard class="p-4">
        <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('models.summary.textModels') }}</p>
        <p class="mt-3 text-2xl font-semibold">{{ modelSummary.text }}</p>
        <p class="mt-1 text-xs text-muted-foreground">{{ t('models.kinds.text') }}</p>
      </UiCard>
      <UiCard class="p-4">
        <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('models.summary.embeddingModels') }}</p>
        <p class="mt-3 text-2xl font-semibold">{{ modelSummary.embedding }}</p>
        <p class="mt-1 text-xs text-muted-foreground">{{ t('models.kinds.embedding') }}</p>
      </UiCard>
    </div>

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
      <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div class="min-w-0">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('models.modelCatalogEyebrow') }}</p>
          <h2 class="mt-2 text-xl font-semibold">{{ t('models.modelCatalogTitle') }}</h2>
          <p class="mt-2 text-sm leading-7 text-muted-foreground">{{ t('models.modelCatalogDescription') }}</p>
        </div>
        <div class="flex w-full flex-col gap-3 sm:w-auto sm:flex-row sm:items-center">
          <UiSelect
            v-model="modelFilterProviderId"
            :options="providerFilterOptions"
            searchable
            :search-placeholder="t('models.search.provider')"
            :empty-text="t('models.search.empty')"
            class="w-full sm:min-w-[240px]"
          />
          <UiButton class="w-full sm:w-auto" @click="openModelDialog()">
            <Plus class="h-4 w-4" />
            {{ t('models.newModel') }}
          </UiButton>
        </div>
      </div>

      <div v-if="visibleModels.length === 0" class="rounded-2xl border border-border bg-muted/35 p-4 text-sm text-muted-foreground">
        {{ t('models.emptyModels') }}
      </div>
      <div v-else class="grid gap-4 xl:grid-cols-2">
        <UiCard v-for="model in visibleModels" :key="model.id" class="p-4 sm:p-5">
          <div class="flex min-w-0 flex-wrap items-start justify-between gap-4">
            <div class="min-w-0 flex-1">
              <h3 class="break-words font-semibold" :title="model.display_name || model.name">{{ model.display_name || model.name }}</h3>
              <p class="mt-1 break-words text-xs text-muted-foreground">{{ providerName(model.provider_id) }} · {{ modelFeatureSummary(model) }}</p>
              <p class="mt-1 break-words text-xs text-muted-foreground" :title="model.name">{{ t('models.upstreamModelId') }}: <span class="font-mono text-[11px]">{{ model.name }}</span></p>
              <p class="mt-1 break-words text-xs text-muted-foreground" :title="modelQualifiedId(model)">{{ t('models.storedValue') }}: <span class="font-mono text-[11px]">{{ modelQualifiedId(model) }}</span></p>
            </div>
            <UiBadge :variant="model.enabled ? 'success' : 'muted'">{{ enabledLabel(model.enabled) }}</UiBadge>
          </div>

          <div class="mt-4 grid grid-cols-2 gap-3 text-sm xl:grid-cols-4">
            <div class="rounded-xl bg-muted/35 p-3">
              <p class="field-label text-xs">{{ t('models.contextWindow') }}<UiInfoTooltip :text="t('tooltips.contextWindow')" /></p>
              <p class="mt-1 font-medium">{{ formatInteger(model.context_window) }}</p>
            </div>
            <div class="rounded-xl bg-muted/35 p-3">
              <p class="field-label text-xs">{{ t('models.maxOutputTokens') }}<UiInfoTooltip :text="t('tooltips.maxOutputTokens')" /></p>
              <p class="mt-1 font-medium">{{ formatInteger(model.max_output_tokens) }}</p>
            </div>
            <div class="rounded-xl bg-muted/35 p-3">
              <p class="field-label text-xs">{{ t('models.dimension') }}<UiInfoTooltip :text="t('tooltips.dimension')" /></p>
              <p class="mt-1 font-medium">{{ model.kind === 'embedding' ? formatInteger(model.dimension) : t('common.emptyValue') }}</p>
            </div>
            <div class="rounded-xl bg-muted/35 p-3">
              <p class="field-label text-xs">{{ t('models.routingWeight') }}<UiInfoTooltip :text="t('tooltips.routingWeight')" /></p>
              <p class="mt-1 font-medium">{{ formatInteger(model.routing_weight) }}</p>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <UiBadge :variant="model.default_for_kind ? 'gold' : 'muted'">{{ t('models.defaultForKind') }}</UiBadge>
            <UiBadge :variant="model.supports_streaming ? 'default' : 'muted'">{{ t('models.supportsStreaming') }}</UiBadge>
            <UiBadge :variant="model.supports_tools ? 'default' : 'muted'">{{ t('models.supportsTools') }}</UiBadge>
            <UiBadge v-if="!model.allowed_agent_roles?.length" variant="muted">{{ t('models.allRoles') }}</UiBadge>
          </div>

          <div v-if="model.allowed_agent_roles?.length" class="mt-4">
            <p class="field-label text-xs uppercase tracking-[0.18em]">{{ t('models.allowedAgentRoles') }}<UiInfoTooltip :text="t('tooltips.allowedRoles')" /></p>
            <div class="mt-2 flex flex-wrap gap-2">
              <UiBadge v-for="role in model.allowed_agent_roles" :key="role" variant="muted">{{ roleLabel(role) }}</UiBadge>
            </div>
          </div>

          <div class="mt-5 flex flex-wrap gap-2">
            <UiButton size="sm" variant="outline" @click="openModelDialog(model)">
              <Pencil class="h-4 w-4" />
              {{ t('actions.edit') }}
            </UiButton>
            <UiButton size="sm" variant="destructive" @click="requestDeleteModel(model)">
              <Trash2 class="h-4 w-4" />
              {{ t('actions.delete') }}
            </UiButton>
          </div>
        </UiCard>
      </div>
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
  </div>
</template>
