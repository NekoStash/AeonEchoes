<script setup lang="ts">
import { Check, Pencil, Plus, RefreshCw, Save, Trash2 } from '@lucide/vue'
import ModelConfigureDialog from '~/features/model-configure/ModelConfigureDialog.vue'
import { configurableAgentRoles } from '~/features/model-configure/model-form'
import SettingsWorkspace from '~/widgets/settings-workspace/SettingsWorkspace.vue'
import { useModelStore } from '~/entities/model'
import type { ModelConfig, ModelUsageKey, ModelUsageSettings, ProviderConfig } from '~/lib/types'

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const modelStore = useModelStore()
const providers = ref<ProviderConfig[]>([])
const models = computed(() => modelStore.items)
const routing = reactive<ModelUsageSettings>(emptyRouting())
const loading = ref(false)
const loadError = ref('')
const saving = ref(false)
const routingSaving = ref(false)
const dialogOpen = ref(false)
const selectedModel = ref<ModelConfig | null>(null)
const deleteTarget = ref<ModelConfig | null>(null)
const pendingAction = ref('')
const search = ref('')
const providerFilter = ref('')
const kindFilter = ref('')
const deleteConfirmOpen = computed({ get: () => Boolean(deleteTarget.value), set: (value) => { if (!value) deleteTarget.value = null } })
const providerById = computed(() => new Map(providers.value.map((provider) => [provider.id, provider])))
const providerOptions = computed(() => providers.value.map((provider) => ({ label: provider.name || provider.id, value: provider.id })))
const modelOptions = computed(() => models.value.filter((model) => model.enabled).map((model) => ({ label: model.display_name || model.name, value: qualifiedModelId(model), description: `${providerById.value.get(model.provider_id)?.name || model.provider_id} · ${model.kind}` })))
const routingKeys: ModelUsageKey[] = [...configurableAgentRoles, 'embedding']
const visibleModels = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return models.value.filter((model) => {
    if (providerFilter.value && model.provider_id !== providerFilter.value) return false
    if (kindFilter.value && model.kind !== kindFilter.value) return false
    if (!query) return true
    return [model.id, model.name, model.display_name, model.provider_id, model.kind, ...(model.allowed_agent_roles || [])].some((value) => String(value || '').toLocaleLowerCase().includes(query))
  })
})

onMounted(load)

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const [providerResult, , routingResult] = await Promise.all([api.listProviders(), modelStore.load(), modelStore.loadUsageSettings()])
    providers.value = providerResult.data
    Object.assign(routing, routingResult.data)
  } catch (error) {
    console.error('[model-settings] Failed to load model configuration.', error)
    loadError.value = error instanceof Error ? error.message : t('settings.models.messages.loadFailed')
    toast.error(t('settings.models.messages.loadFailed'), loadError.value, error)
  } finally {
    loading.value = false
  }
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

async function saveRouting() {
  routingSaving.value = true
  try {
    const result = await modelStore.saveUsageSettings({ ...routing })
    Object.assign(routing, result.data)
    toast.success(t('settings.models.messages.routingSaved'))
  } catch (error) {
    console.error('[model-settings] Failed to save model routing.', error)
    toast.error(t('settings.models.messages.routingSaveFailed'), error instanceof Error ? error.message : undefined, error)
  } finally {
    routingSaving.value = false
  }
}

async function confirmDelete() {
  const target = deleteTarget.value
  if (!target) return
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

function qualifiedModelId(model: ModelConfig) {
  return model.id.includes(':') ? model.id : `${model.provider_id}:${model.name}`
}

function routingLabel(key: ModelUsageKey) {
  if (key === 'embedding') return t('models.usage.embedding')
  const messageKey = `models.roles.${key.replace(/-/g, '_')}`
  const value = t(messageKey)
  return value === messageKey ? key : value
}

function emptyRouting(): ModelUsageSettings {
  return Object.fromEntries([...configurableAgentRoles, 'embedding'].map((key) => [key, ''])) as ModelUsageSettings
}
</script>

<template>
  <SettingsWorkspace :title="t('settings.models.title')" :description="t('settings.models.description')">
    <div class="space-y-10">
      <UiInlineNotice v-if="loadError" tone="danger" :title="t('settings.models.messages.loadFailed')" :description="loadError"><template #actions><UiButton variant="outline" size="sm" :loading="loading" @click="load">{{ t('common.retry') }}</UiButton></template></UiInlineNotice>
      <section class="space-y-5">
        <div class="flex flex-col gap-3 border-b border-border pb-5 xl:flex-row xl:items-end xl:justify-between">
          <div><p class="text-xs font-black uppercase tracking-[0.16em] text-muted-foreground">{{ t('settings.models.catalogEyebrow') }}</p><h2 class="mt-2 text-2xl font-black">{{ t('settings.models.catalogTitle') }}</h2></div>
          <div class="flex flex-col gap-2 sm:flex-row"><UiInput v-model="search" class="sm:w-64" :placeholder="t('settings.models.searchPlaceholder')" /><UiSelect v-model="providerFilter" class="sm:w-48" :options="providerOptions" :placeholder="t('settings.models.allProviders')" /><UiSelect v-model="kindFilter" class="sm:w-40" :options="[{ value: 'text', label: t('models.kinds.text') }, { value: 'embedding', label: t('models.kinds.embedding') }]" :placeholder="t('models.filters.allKinds')" /><UiButton variant="outline" :loading="loading" @click="load"><RefreshCw class="h-4 w-4" />{{ t('actions.refresh') }}</UiButton><UiButton @click="openCreate"><Plus class="h-4 w-4" />{{ t('settings.models.add') }}</UiButton></div>
        </div>

        <UiAlert v-if="modelStore.listRequest.error && models.length === 0" tone="danger" :title="t('settings.models.messages.loadFailed')" :description="modelStore.listRequest.error.message" />
        <div v-else-if="loading && models.length === 0" class="py-16 text-center text-sm font-bold text-muted-foreground">{{ t('settings.models.messages.loading') }}</div>
        <div v-else-if="visibleModels.length === 0" class="border border-dashed border-border p-10 text-center"><h3 class="text-xl font-black">{{ t('settings.models.emptyTitle') }}</h3><p class="mt-2 text-sm text-muted-foreground">{{ t('settings.models.emptyDescription') }}</p></div>
        <div v-else class="divide-y divide-border border-y border-border">
          <article v-for="model in visibleModels" :key="model.id" class="grid gap-5 py-5 xl:grid-cols-[minmax(0,1fr)_18rem_13rem] xl:items-center">
            <div class="min-w-0"><div class="flex flex-wrap items-center gap-2"><h3 class="truncate text-lg font-black">{{ model.display_name || model.name }}</h3><UiBadge :tone="model.enabled ? 'success' : 'muted'">{{ model.enabled ? t('status.enabled') : t('status.disabled') }}</UiBadge><UiBadge tone="muted">{{ model.kind }}</UiBadge><UiBadge v-if="model.default_for_kind" tone="warning">{{ t('models.defaultForKind') }}</UiBadge></div><p class="mt-2 break-all font-mono text-xs text-muted-foreground">{{ qualifiedModelId(model) }}</p><p class="mt-2 text-sm text-muted-foreground">{{ providerById.get(model.provider_id)?.name || model.provider_id }}</p></div>
            <dl class="grid grid-cols-2 gap-3 text-xs"><div><dt class="text-muted-foreground">context</dt><dd class="mt-1 font-mono font-bold">{{ model.context_window || 0 }}</dd></div><div><dt class="text-muted-foreground">output</dt><dd class="mt-1 font-mono font-bold">{{ model.max_output_tokens || 0 }}</dd></div><div><dt class="text-muted-foreground">tools / stream</dt><dd class="mt-1 font-bold">{{ model.supports_tools ? '✓' : '—' }} / {{ model.supports_streaming ? '✓' : '—' }}</dd></div><div><dt class="text-muted-foreground">weight</dt><dd class="mt-1 font-mono font-bold">{{ model.routing_weight || 0 }}</dd></div><div><dt class="text-muted-foreground">input / MTok</dt><dd class="mt-1 font-mono font-bold">{{ model.cost_input_per_mtok ?? 0 }}</dd></div><div><dt class="text-muted-foreground">output / MTok</dt><dd class="mt-1 font-mono font-bold">{{ model.cost_output_per_mtok ?? 0 }}</dd></div></dl>
            <div class="flex gap-2 xl:justify-end"><UiButton size="sm" variant="outline" @click="openEdit(model)"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton><UiButton size="sm" variant="destructive" @click="deleteTarget = model"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton></div>
          </article>
        </div>
      </section>

      <section class="border-t border-border pt-8">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between"><div><p class="text-xs font-black uppercase tracking-[0.16em] text-muted-foreground">{{ t('settings.models.routingEyebrow') }}</p><h2 class="mt-2 text-2xl font-black">{{ t('settings.models.routingTitle') }}</h2><p class="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">{{ t('settings.models.routingDescription') }}</p></div><UiButton :loading="routingSaving" @click="saveRouting"><Save class="h-4 w-4" />{{ t('settings.models.saveRouting') }}</UiButton></div>
        <div class="mt-5 divide-y divide-border border-y border-border">
          <label v-for="key in routingKeys" :key="key" class="grid gap-3 py-4 md:grid-cols-[15rem_minmax(0,1fr)] md:items-center"><span><strong class="block text-sm">{{ routingLabel(key) }}</strong><span class="mt-1 block font-mono text-[11px] text-muted-foreground">{{ key }}</span></span><UiSelect v-model="routing[key]" :options="modelOptions" :placeholder="t('settings.models.inheritRouting')" searchable /></label>
        </div>
      </section>
    </div>

    <ModelConfigureDialog v-model:open="dialogOpen" :model="selectedModel" :providers="providers" :saving="saving" @save="saveModel" />
    <UiConfirm v-model:open="deleteConfirmOpen" :title="t('settings.models.deleteTitle')" :description="deleteTarget ? t('settings.models.deleteDescription', { name: deleteTarget.display_name || deleteTarget.name }) : ''" :loading="Boolean(deleteTarget && pendingAction === `delete:${deleteTarget.id}`)" @confirm="confirmDelete" />
  </SettingsWorkspace>
</template>
