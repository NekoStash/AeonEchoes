<script setup lang="ts">
import { Pencil, Plus, RefreshCw, RotateCcw, Save, Trash2 } from '@lucide/vue'
import ModelConfigureDialog from '~/features/model-configure/ModelConfigureDialog.vue'
import {
  buildRoutingOptions,
  cloneRouting,
  findRoutingReferences,
  isRoutingDirty,
  qualifiedModelId,
  ROUTING_KEYS,
  validateRouting,
  type RoutingEligibilityReason,
  type RoutingReferences,
  type RoutingValidationError
} from '~/features/model-routing/routing-state'
import SettingsWorkspace from '~/widgets/settings-workspace/SettingsWorkspace.vue'
import { useModelStore } from '~/entities/model'
import type { ModelConfig, ModelUsageKey, ModelUsageSettings, ProviderConfig } from '~/lib/types'

interface RoutingGroup {
  id: 'writing' | 'canon' | 'knowledge'
  keys: ModelUsageKey[]
}

type RoutingSaveState = 'saved' | 'dirty' | 'saving' | 'failed'

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const modelStore = useModelStore()
const providers = ref<ProviderConfig[]>([])
const models = computed(() => modelStore.items)
const routingBaseline = ref<ModelUsageSettings>(cloneRouting())
const routingDraft = ref<ModelUsageSettings>(cloneRouting())
const routingSaveState = ref<RoutingSaveState>('saved')
const routingError = ref('')
const routingReloadNotice = ref('')
const loading = ref(false)
const loadError = ref('')
const saving = ref(false)
const dialogOpen = ref(false)
const selectedModel = ref<ModelConfig | null>(null)
const deleteTarget = ref<ModelConfig | null>(null)
const blockedDelete = ref<{ model: ModelConfig, references: RoutingReferences } | null>(null)
const pendingAction = ref('')
const search = ref('')
const providerFilter = ref('')
const kindFilter = ref('')
const enabledFilter = ref('')
const refreshConfirmOpen = ref(false)
const pendingNavigation = ref<null | { resolve: (allow: boolean) => void }>(null)
const pendingLeaveCleanup = ref<null | (() => void)>(null)

const routingGroups: RoutingGroup[] = [
  { id: 'writing', keys: ['writer', 'editor', 'genesis-optimizer', 'plot-architect'] },
  { id: 'canon', keys: ['world-builder', 'character-keeper', 'continuity-auditor'] },
  { id: 'knowledge', keys: ['fact-extractor', 'graph-curator', 'embedding'] }
]

const deleteConfirmOpen = computed({
  get: () => Boolean(deleteTarget.value),
  set: (value) => { if (!value) deleteTarget.value = null }
})
const leaveConfirmOpen = computed({
  get: () => Boolean(pendingNavigation.value),
  set: (value) => {
    if (value || !pendingNavigation.value) return
    pendingNavigation.value.resolve(false)
    pendingNavigation.value = null
  }
})
const providerById = computed(() => new Map(providers.value.map((provider) => [provider.id, provider])))
const providerOptions = computed(() => providers.value.map((provider) => ({ label: provider.name || provider.id, value: provider.id })))
const routingDirty = computed(() => isRoutingDirty(routingBaseline.value, routingDraft.value))
const routingValidation = computed(() => validateRouting(routingDraft.value, models.value))
const routingValidationByKey = computed(() => new Map(routingValidation.value.map((error) => [error.key, error])))
const embeddingChanged = computed(() => routingBaseline.value.embedding !== routingDraft.value.embedding)
const hasFilters = computed(() => Boolean(search.value.trim() || providerFilter.value || kindFilter.value || enabledFilter.value))
const visibleModels = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return models.value.filter((model) => {
    if (providerFilter.value && model.provider_id !== providerFilter.value) return false
    if (kindFilter.value && model.kind !== kindFilter.value) return false
    if (enabledFilter.value === 'enabled' && !model.enabled) return false
    if (enabledFilter.value === 'disabled' && model.enabled) return false
    if (!query) return true
    return [model.id, model.name, model.display_name, model.provider_id, model.kind, ...(model.allowed_agent_roles || [])]
      .some((value) => String(value || '').toLocaleLowerCase().includes(query))
  })
})
const referencesByModel = computed(() => new Map(models.value.map((model) => [model.id, findRoutingReferences(model, routingBaseline.value, routingDraft.value)])))
const routingStatusText = computed(() => t(`settings.models.routingStates.${routingSaveState.value}`))

onMounted(() => {
  void load()
  if (!import.meta.client) return
  const beforeUnload = (event: BeforeUnloadEvent) => {
    if (!routingDirty.value) return
    event.preventDefault()
    event.returnValue = ''
  }
  window.addEventListener('beforeunload', beforeUnload)
  pendingLeaveCleanup.value = () => window.removeEventListener('beforeunload', beforeUnload)
})

onBeforeUnmount(() => pendingLeaveCleanup.value?.())

onBeforeRouteLeave(() => {
  if (!routingDirty.value) return true
  return new Promise<boolean>((resolve) => {
    pendingNavigation.value?.resolve(false)
    pendingNavigation.value = { resolve }
  })
})

async function load(replaceDirtyRouting = false) {
  loading.value = true
  loadError.value = ''
  blockedDelete.value = null
  try {
    const [providerResult, , routingResult] = await Promise.all([api.listProviders(), modelStore.load(), modelStore.loadUsageSettings()])
    providers.value = providerResult.data
    if (!routingDirty.value || replaceDirtyRouting) {
      applyRoutingSnapshot(routingResult.data)
      routingReloadNotice.value = ''
    } else {
      routingReloadNotice.value = t('settings.models.refresh.draftKept')
    }
  } catch (error) {
    console.error('[model-settings] Failed to load model configuration.', error)
    loadError.value = error instanceof Error ? error.message : t('settings.models.messages.loadFailed')
    toast.error(t('settings.models.messages.loadFailed'), loadError.value, error)
  } finally {
    loading.value = false
  }
}

function applyRoutingSnapshot(settings: ModelUsageSettings) {
  const next = cloneRouting(settings)
  routingBaseline.value = cloneRouting(next)
  routingDraft.value = cloneRouting(next)
  routingSaveState.value = 'saved'
  routingError.value = ''
}

function requestReload() {
  if (loading.value || routingSaveState.value === 'saving') return
  if (routingDirty.value) {
    refreshConfirmOpen.value = true
    return
  }
  void load()
}

function confirmReload() {
  refreshConfirmOpen.value = false
  void load(true)
}

function openCreate() {
  selectedModel.value = null
  dialogOpen.value = true
}

function openEdit(model: ModelConfig) {
  selectedModel.value = model
  dialogOpen.value = true
}

async function saveModel(model: ModelConfig) {
  saving.value = true
  try {
    const result = await modelStore.save(model)
    dialogOpen.value = false
    toast.success(t('settings.models.messages.saved'), result.data.display_name)
  } catch (error) {
    console.error('[model-settings] Failed to save model.', error)
    toast.error(t('settings.models.messages.saveFailed'), error instanceof Error ? error.message : undefined, error)
  } finally {
    saving.value = false
  }
}

function updateRouting(key: ModelUsageKey, value: string) {
  routingDraft.value = cloneRouting({ ...routingDraft.value, [key]: value })
  routingSaveState.value = isRoutingDirty(routingBaseline.value, routingDraft.value) ? 'dirty' : 'saved'
}

function resetRouting() {
  if (routingSaveState.value === 'saving') return
  routingDraft.value = cloneRouting(routingBaseline.value)
  routingSaveState.value = 'saved'
  routingError.value = ''
}

async function saveRouting() {
  if (!routingDirty.value || routingSaveState.value === 'saving') return
  const [firstError] = routingValidation.value
  if (firstError) {
    const error = new Error(`Invalid model route: ${firstError.key} -> ${firstError.value}`)
    console.error('[model-settings] Model routing validation failed.', error, firstError)
    routingSaveState.value = 'failed'
    routingError.value = t('settings.models.routingValidation.summary')
    await nextTick()
    document.getElementById(routingFieldId(firstError.key))?.focus()
    return
  }

  routingSaveState.value = 'saving'
  routingError.value = ''
  try {
    const result = await modelStore.saveUsageSettings(cloneRouting(routingDraft.value))
    applyRoutingSnapshot(result.data)
    toast.success(t('settings.models.messages.routingSaved'))
  } catch (error) {
    console.error('[model-settings] Failed to save model routing.', error)
    routingSaveState.value = 'failed'
    routingError.value = error instanceof Error ? error.message : t('settings.models.messages.routingSaveFailed')
    toast.error(t('settings.models.messages.routingSaveFailed'), routingError.value, error)
  }
}

function requestDelete(model: ModelConfig) {
  const references = modelReferences(model)
  if (references.all.length > 0) {
    blockedDelete.value = { model, references }
    const error = new Error(`Model ${qualifiedModelId(model)} is referenced by model routing.`)
    console.error('[model-settings] Refused to delete referenced model.', error, references)
    toast.warning(t('settings.models.deleteBlocked.title'), routeLabels(references.all))
    return
  }
  blockedDelete.value = null
  deleteTarget.value = model
}

async function confirmDelete() {
  const target = deleteTarget.value
  if (!target) return
  const references = modelReferences(target)
  if (references.all.length > 0) {
    blockedDelete.value = { model: target, references }
    deleteTarget.value = null
    const error = new Error(`Model ${qualifiedModelId(target)} became referenced before deletion.`)
    console.error('[model-settings] Refused to delete newly referenced model.', error, references)
    return
  }

  pendingAction.value = `delete:${target.id}`
  try {
    await modelStore.remove(target.id)
    deleteTarget.value = null
    toast.success(t('settings.models.messages.deleted'), target.display_name || target.name)
  } catch (error) {
    console.error('[model-settings] Failed to delete model.', error)
    toast.error(t('settings.models.messages.deleteFailed'), error instanceof Error ? error.message : undefined, error)
  } finally {
    pendingAction.value = ''
  }
}

function confirmLeave() {
  pendingNavigation.value?.resolve(true)
  pendingNavigation.value = null
}

function clearFilters() {
  search.value = ''
  providerFilter.value = ''
  kindFilter.value = ''
  enabledFilter.value = ''
}

function routingFieldId(key: ModelUsageKey) {
  return `model-routing-${key}`
}

function routingLabel(key: ModelUsageKey) {
  if (key === 'embedding') return t('models.usage.embedding')
  const messageKey = `models.roles.${key.replace(/-/g, '_')}`
  const value = t(messageKey)
  return value === messageKey ? key : value
}

function routeLabels(keys: ModelUsageKey[]) {
  return keys.map(routingLabel).join(t('settings.models.listSeparator'))
}

function routeValidationError(key: ModelUsageKey) {
  const error = routingValidationByKey.value.get(key)
  return error ? validationMessage(error) : ''
}

function validationMessage(error: RoutingValidationError) {
  return t(`settings.models.routingValidation.${error.reason}`, {
    model: error.model?.display_name || error.model?.name || error.value,
    role: routingLabel(error.key),
    expected: error.key === 'embedding' ? t('models.kinds.embedding') : t('models.kinds.text')
  })
}

function eligibilityReason(reason: RoutingEligibilityReason | undefined, key: ModelUsageKey, value: string, model?: ModelConfig) {
  if (!reason) return undefined
  return validationMessage({ key, value, reason, model })
}

function routingOptions(key: ModelUsageKey) {
  return buildRoutingOptions(models.value, key, routingDraft.value[key]).map((option) => ({
    label: option.model?.display_name || option.model?.name || option.value,
    value: option.value,
    description: option.model
      ? `${providerById.value.get(option.model.provider_id)?.name || option.model.provider_id} · ${option.model.kind || t('settings.models.catalog.unknownKind')}`
      : t('settings.models.routingValidation.unknownDescription'),
    disabled: option.disabled,
    disabledReason: eligibilityReason(option.reason, key, option.value, option.model)
  }))
}

function modelReferences(model: ModelConfig) {
  return referencesByModel.value.get(model.id) || findRoutingReferences(model, routingBaseline.value, routingDraft.value)
}

function modelReferenceDescription(model: ModelConfig) {
  const references = modelReferences(model)
  const parts: string[] = []
  if (references.baseline.length) parts.push(t('settings.models.references.saved', { roles: routeLabels(references.baseline) }))
  const draftOnly = references.draft.filter((key) => !references.baseline.includes(key))
  if (draftOnly.length) parts.push(t('settings.models.references.draft', { roles: routeLabels(draftOnly) }))
  return parts.join(' ')
}

function blockedDeleteDescription() {
  if (!blockedDelete.value) return ''
  return t('settings.models.deleteBlocked.description', {
    name: blockedDelete.value.model.display_name || blockedDelete.value.model.name,
    roles: routeLabels(blockedDelete.value.references.all)
  })
}

function roleQualification(model: ModelConfig) {
  if (model.kind === 'embedding') return t('settings.models.catalog.rolesNotApplicable')
  const roles = model.allowed_agent_roles || []
  return roles.length === 0
    ? t('settings.models.catalog.allRolesEligible')
    : t('settings.models.catalog.roleCount', { count: roles.length })
}

function formatNumber(value: number | undefined) {
  return new Intl.NumberFormat().format(value || 0)
}
</script>

<template>
  <SettingsWorkspace layout="viewport" :title="t('settings.models.title')" :description="t('settings.models.description')">
    <div class="grid gap-5 lg:h-full lg:min-h-0 lg:grid-cols-[minmax(0,1fr)_21rem] lg:gap-6" data-testid="model-settings-workspace">
      <aside class="order-1 min-h-0 lg:order-2 lg:col-start-2 lg:row-start-1 lg:h-full" :aria-labelledby="'model-routing-title'" data-testid="model-routing-panel">
        <section class="flex min-h-0 flex-col border border-border bg-surface-muted lg:h-full">
          <header class="shrink-0 border-b border-border p-4">
            <p class="text-xs font-black uppercase tracking-[0.16em] text-muted-foreground">{{ t('settings.models.routingEyebrow') }}</p>
            <h2 id="model-routing-title" class="mt-2 text-xl font-black">{{ t('settings.models.routingTitle') }}</h2>
            <p class="mt-2 text-xs leading-5 text-muted-foreground">{{ t('settings.models.routingDescription') }}</p>
            <div class="mt-3 flex items-center justify-between gap-3 border-t border-border pt-3">
              <span class="text-xs font-bold text-muted-foreground">{{ t('settings.models.routingStatus') }}</span>
              <UiBadge :tone="routingSaveState === 'failed' ? 'danger' : routingSaveState === 'dirty' ? 'warning' : routingSaveState === 'saving' ? 'info' : 'success'" data-testid="routing-save-state">
                <span aria-live="polite">{{ routingStatusText }}</span>
              </UiBadge>
            </div>
          </header>

          <UiInlineNotice v-if="routingError" class="shrink-0 border-b border-border" tone="danger" :title="t('settings.models.messages.routingSaveFailed')" :description="routingError" data-testid="routing-error-notice" />
          <UiInlineNotice v-if="routingReloadNotice" class="shrink-0 border-b border-border" tone="warning" :title="t('settings.models.refresh.draftKeptTitle')" :description="routingReloadNotice" />

          <div class="space-y-6 p-4 lg:min-h-0 lg:flex-1 lg:overflow-y-auto lg:overscroll-contain subtle-scrollbar" data-testid="model-routing-scroll">
            <section v-for="group in routingGroups" :key="group.id" :aria-labelledby="`routing-group-${group.id}`">
              <div class="border-b border-border pb-3">
                <h3 :id="`routing-group-${group.id}`" class="text-sm font-black">{{ t(`models.routeGroups.${group.id}.title`) }}</h3>
                <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ t(`models.routeGroups.${group.id}.description`) }}</p>
              </div>
              <div class="divide-y divide-border">
                <div v-for="key in group.keys" :key="key" class="py-4">
                  <div class="mb-2 flex items-start justify-between gap-3">
                    <div class="min-w-0">
                      <p class="text-sm font-bold">{{ routingLabel(key) }}</p>
                      <p class="mt-0.5 break-all font-mono text-[10px] text-muted-foreground">{{ key }}</p>
                    </div>
                    <UiBadge :tone="routingDraft[key] ? 'info' : 'muted'">{{ routingDraft[key] ? t('settings.models.routeState.explicit') : t('settings.models.routeState.inherited') }}</UiBadge>
                  </div>
                  <UiField
                    :id="routingFieldId(key)"
                    :label="t('settings.models.routeFieldLabel', { role: routingLabel(key) })"
                    :description="routingDraft[key] ? t('settings.models.routeState.explicitDescription') : t('settings.models.routeState.inheritedDescription')"
                    :error="routeValidationError(key)"
                    class="space-y-1.5"
                    v-slot="field"
                  >
                    <UiSelect
                      :id="field.id"
                      :model-value="routingDraft[key]"
                      :options="routingOptions(key)"
                      :placeholder="t('settings.models.inheritRouting')"
                      :disabled="routingSaveState === 'saving' || loading"
                      :invalid="field.invalid"
                      :aria-describedby="field.describedby"
                      :aria-label="t('settings.models.routeFieldLabel', { role: routingLabel(key) })"
                      :search-label="t('settings.models.routeSearchLabel', { role: routingLabel(key) })"
                      searchable
                      @update:model-value="updateRouting(key, $event)"
                    />
                  </UiField>
                </div>
              </div>
            </section>
            <UiInlineNotice v-if="embeddingChanged" tone="warning" :title="t('settings.models.embeddingChange.title')" :description="t('settings.models.embeddingChange.description')" />
          </div>

          <footer class="grid shrink-0 grid-cols-2 gap-2 border-t border-border bg-surface p-3" data-testid="model-routing-actions">
            <UiButton variant="outline" :disabled="!routingDirty || routingSaveState === 'saving' || loading" @click="resetRouting"><RotateCcw class="h-4 w-4" />{{ t('actions.reset') }}</UiButton>
            <UiButton :disabled="!routingDirty || loading" :loading="routingSaveState === 'saving'" @click="saveRouting"><Save class="h-4 w-4" />{{ t('settings.models.saveRouting') }}</UiButton>
          </footer>
        </section>
      </aside>

      <section class="order-2 flex min-w-0 flex-col lg:order-1 lg:col-start-1 lg:row-start-1 lg:h-full lg:min-h-0" aria-labelledby="model-catalog-title" data-testid="model-catalog">
        <header class="shrink-0 border-b border-border pb-4">
          <div class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
            <div>
              <p class="text-xs font-black uppercase tracking-[0.16em] text-muted-foreground">{{ t('settings.models.catalogEyebrow') }}</p>
              <h2 id="model-catalog-title" class="mt-2 text-2xl font-black">{{ t('settings.models.catalogTitle') }}</h2>
              <p class="mt-2 text-sm text-muted-foreground" aria-live="polite">{{ t('settings.models.catalog.resultCount', { visible: visibleModels.length, total: models.length }) }}</p>
            </div>
            <div class="flex flex-wrap gap-2">
              <UiButton variant="outline" :loading="loading" :disabled="routingSaveState === 'saving'" @click="requestReload"><RefreshCw class="h-4 w-4" />{{ t('actions.refresh') }}</UiButton>
              <UiButton @click="openCreate"><Plus class="h-4 w-4" />{{ t('settings.models.add') }}</UiButton>
            </div>
          </div>

          <div class="mt-4 grid gap-3 sm:grid-cols-2 2xl:grid-cols-[minmax(13rem,1fr)_minmax(10rem,0.65fr)_minmax(9rem,0.55fr)_minmax(9rem,0.55fr)_auto]">
            <UiField :label="t('settings.models.catalog.searchLabel')" v-slot="field"><UiInput v-model="search" :id="field.id" :aria-describedby="field.describedby" type="search" :placeholder="t('settings.models.searchPlaceholder')" /></UiField>
            <UiField :label="t('settings.models.catalog.providerFilterLabel')" v-slot="field"><UiSelect v-model="providerFilter" :id="field.id" :aria-describedby="field.describedby" :aria-label="t('settings.models.catalog.providerFilterLabel')" :options="providerOptions" :placeholder="t('settings.models.allProviders')" /></UiField>
            <UiField :label="t('settings.models.catalog.kindFilterLabel')" v-slot="field"><UiSelect v-model="kindFilter" :id="field.id" :aria-describedby="field.describedby" :aria-label="t('settings.models.catalog.kindFilterLabel')" :options="[{ value: 'text', label: t('models.kinds.text') }, { value: 'embedding', label: t('models.kinds.embedding') }]" :placeholder="t('models.filters.allKinds')" /></UiField>
            <UiField :label="t('settings.models.catalog.statusFilterLabel')" v-slot="field"><UiSelect v-model="enabledFilter" :id="field.id" :aria-describedby="field.describedby" :aria-label="t('settings.models.catalog.statusFilterLabel')" :options="[{ value: 'enabled', label: t('settings.models.catalog.enabledOnly') }, { value: 'disabled', label: t('settings.models.catalog.disabledOnly') }]" :placeholder="t('settings.models.catalog.allStatuses')" /></UiField>
            <div class="flex items-end"><UiButton v-if="hasFilters" class="w-full" variant="ghost" @click="clearFilters">{{ t('settings.models.catalog.clearFilters') }}</UiButton></div>
          </div>
        </header>

        <div class="shrink-0 space-y-3 py-3">
          <UiInlineNotice v-if="loadError" tone="danger" :title="t('settings.models.messages.loadFailed')" :description="loadError"><template #actions><UiButton variant="outline" size="sm" :loading="loading" @click="requestReload">{{ t('common.retry') }}</UiButton></template></UiInlineNotice>
          <UiInlineNotice v-if="blockedDelete" tone="warning" :title="t('settings.models.deleteBlocked.title')" :description="blockedDeleteDescription()"><template #actions><UiButton variant="ghost" size="sm" @click="blockedDelete = null">{{ t('actions.dismiss') }}</UiButton></template></UiInlineNotice>
        </div>

        <div class="lg:min-h-0 lg:flex-1 lg:overflow-y-auto lg:overscroll-contain subtle-scrollbar" :aria-label="t('settings.models.catalog.listLabel')" data-testid="model-list-scroll">
          <UiAlert v-if="modelStore.listRequest.error && models.length === 0" tone="danger" :title="t('settings.models.messages.loadFailed')" :description="modelStore.listRequest.error.message" />
          <div v-else-if="loading && models.length === 0" class="border-y border-border py-16 text-center text-sm font-bold text-muted-foreground">{{ t('settings.models.messages.loading') }}</div>
          <div v-else-if="visibleModels.length === 0" class="border-y border-dashed border-border p-10 text-center"><h3 class="text-xl font-black">{{ t('settings.models.emptyTitle') }}</h3><p class="mt-2 text-sm text-muted-foreground">{{ t('settings.models.emptyDescription') }}</p></div>
          <div v-else class="divide-y divide-border border-y border-border">
            <article v-for="model in visibleModels" :key="model.id" class="space-y-4 py-5 pr-1" :data-testid="`model-item-${model.id}`">
              <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <h3 class="break-words text-lg font-black">{{ model.display_name || model.name }}</h3>
                    <UiBadge :tone="model.enabled ? 'success' : 'muted'">{{ model.enabled ? t('status.enabled') : t('status.disabled') }}</UiBadge>
                    <UiBadge tone="neutral">{{ model.kind ? t(`models.kinds.${model.kind}`) : t('settings.models.catalog.unknownKind') }}</UiBadge>
                    <UiBadge v-if="model.default_for_kind" tone="warning">{{ t('models.defaultForKind') }}</UiBadge>
                  </div>
                  <p class="mt-2 break-all font-mono text-xs text-muted-foreground">{{ qualifiedModelId(model) }}</p>
                  <p class="mt-1 text-sm font-semibold text-muted-foreground">{{ providerById.get(model.provider_id)?.name || model.provider_id }}</p>
                </div>
                <div class="flex shrink-0 flex-wrap gap-2 sm:justify-end">
                  <UiButton size="sm" variant="outline" @click="openEdit(model)"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton>
                  <UiButton size="sm" variant="destructive" :disabled="modelReferences(model).all.length > 0" :aria-describedby="modelReferences(model).all.length ? `model-references-${model.id}` : undefined" @click="requestDelete(model)"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton>
                </div>
              </div>

              <p v-if="!model.enabled" class="border-l-4 border-state-warning-border bg-state-warning-surface px-3 py-2 text-xs font-semibold text-state-warning-foreground">{{ t('settings.models.catalog.disabledRouting') }}</p>
              <p v-if="modelReferences(model).all.length" :id="`model-references-${model.id}`" class="border-l-4 border-state-info-border bg-state-info-surface px-3 py-2 text-xs font-semibold text-state-info-foreground">{{ modelReferenceDescription(model) }}</p>

              <dl class="grid gap-3 sm:grid-cols-2 2xl:grid-cols-4">
                <template v-if="model.kind === 'embedding'">
                  <div class="border border-border bg-surface-muted px-3 py-2"><dt class="text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">{{ t('settings.models.catalog.dimension') }}</dt><dd class="mt-1 font-mono text-sm font-black">{{ formatNumber(model.dimension) }}</dd></div>
                </template>
                <template v-else>
                  <div class="border border-border bg-surface-muted px-3 py-2"><dt class="text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">{{ t('settings.models.catalog.context') }}</dt><dd class="mt-1 font-mono text-sm font-black">{{ formatNumber(model.context_window) }}</dd></div>
                  <div class="border border-border bg-surface-muted px-3 py-2"><dt class="text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">{{ t('settings.models.catalog.output') }}</dt><dd class="mt-1 font-mono text-sm font-black">{{ formatNumber(model.max_output_tokens) }}</dd></div>
                  <div class="border border-border bg-surface-muted px-3 py-2"><dt class="text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">{{ t('settings.models.catalog.tools') }}</dt><dd class="mt-1 text-sm font-black">{{ model.supports_tools ? t('settings.models.catalog.supported') : t('settings.models.catalog.notSupported') }}</dd></div>
                  <div class="border border-border bg-surface-muted px-3 py-2"><dt class="text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">{{ t('settings.models.catalog.streaming') }}</dt><dd class="mt-1 text-sm font-black">{{ model.supports_streaming ? t('settings.models.catalog.supported') : t('settings.models.catalog.notSupported') }}</dd></div>
                </template>
              </dl>

              <dl class="grid gap-x-5 gap-y-3 border-t border-border pt-3 text-xs sm:grid-cols-2 xl:grid-cols-4">
                <div><dt class="text-muted-foreground">{{ t('settings.models.catalog.inputCost') }}</dt><dd class="mt-1 font-mono font-bold">{{ model.cost_input_per_mtok ?? 0 }}</dd></div>
                <div><dt class="text-muted-foreground">{{ t('settings.models.catalog.outputCost') }}</dt><dd class="mt-1 font-mono font-bold">{{ model.cost_output_per_mtok ?? 0 }}</dd></div>
                <div><dt class="text-muted-foreground">{{ t('settings.models.catalog.weight') }}</dt><dd class="mt-1 font-mono font-bold">{{ model.routing_weight ?? 0 }}</dd></div>
                <div><dt class="text-muted-foreground">{{ t('settings.models.catalog.roleEligibility') }}</dt><dd class="mt-1 font-bold">{{ roleQualification(model) }}</dd></div>
              </dl>
            </article>
          </div>
        </div>
      </section>
    </div>

    <ModelConfigureDialog v-model:open="dialogOpen" :model="selectedModel" :providers="providers" :saving="saving" @save="saveModel" />
    <UiConfirm v-model:open="deleteConfirmOpen" :title="t('settings.models.deleteTitle')" :description="deleteTarget ? t('settings.models.deleteDescription', { name: deleteTarget.display_name || deleteTarget.name }) : ''" :loading="Boolean(deleteTarget && pendingAction === `delete:${deleteTarget.id}`)" @confirm="confirmDelete" />
    <UiConfirm v-model:open="refreshConfirmOpen" :title="t('settings.models.refresh.title')" :description="t('settings.models.refresh.description')" :confirm-label="t('settings.models.refresh.confirm')" tone="danger" @confirm="confirmReload" />
    <UiConfirm v-model:open="leaveConfirmOpen" :title="t('settings.models.leave.title')" :description="t('settings.models.leave.description')" :confirm-label="t('settings.models.leave.confirm')" tone="danger" @confirm="confirmLeave" />
  </SettingsWorkspace>
</template>
