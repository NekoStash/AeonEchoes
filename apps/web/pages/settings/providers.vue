<script setup lang="ts">
import { KeyRound, Pencil, Plus, RefreshCw, Trash2, Waves } from '@lucide/vue'
import ProviderConfigureDialog from '~/features/provider-configure/ProviderConfigureDialog.vue'
import SettingsWorkspace from '~/widgets/settings-workspace/SettingsWorkspace.vue'
import type { ProviderConfig } from '~/lib/types'

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const providers = ref<ProviderConfig[]>([])
const loading = ref(false)
const loadError = ref('')
const saving = ref(false)
const pendingAction = ref('')
const dialogOpen = ref(false)
const selectedProvider = ref<ProviderConfig | null>(null)
const deleteTarget = ref<ProviderConfig | null>(null)
const deleteConfirmOpen = computed({ get: () => Boolean(deleteTarget.value), set: (value) => { if (!value) deleteTarget.value = null } })
const query = ref('')
const visibleProviders = computed(() => {
  const normalized = query.value.trim().toLocaleLowerCase()
  if (!normalized) return providers.value
  return providers.value.filter((provider) => [provider.id, provider.name, provider.provider_type, provider.base_url, provider.status].some((value) => String(value || '').toLocaleLowerCase().includes(normalized)))
})

onMounted(loadProviders)

async function loadProviders() {
  loading.value = true
  loadError.value = ''
  try {
    const result = await api.listProviders()
    providers.value = result.data
  } catch (error) {
    console.error('[provider-settings] Failed to load providers.', error)
    loadError.value = error instanceof Error ? error.message : t('settings.providers.messages.loadFailed')
    toast.error(t('settings.providers.messages.loadFailed'), loadError.value, error)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  selectedProvider.value = null
  dialogOpen.value = true
}

function openEdit(provider: ProviderConfig) {
  selectedProvider.value = provider
  dialogOpen.value = true
}

async function saveProvider(provider: ProviderConfig, mode: 'create' | 'edit') {
  saving.value = true
  try {
    const result = await api.saveProvider(provider, mode)
    providers.value = [...providers.value.filter((item) => item.id !== result.data.id), result.data].sort((a, b) => a.name.localeCompare(b.name))
    dialogOpen.value = false
    toast.success(t('settings.providers.messages.saved'), result.data.name)
  } catch (error) {
    console.error('[provider-settings] Failed to save provider.', error)
    toast.error(t('settings.providers.messages.saveFailed'), error instanceof Error ? error.message : undefined, error)
  } finally {
    saving.value = false
  }
}

async function refreshModels(provider: ProviderConfig) {
  pendingAction.value = `refresh:${provider.id}`
  try {
    const result = await api.refreshModels(provider.id)
    await loadProviders()
    toast.success(t('settings.providers.messages.modelsRefreshed'), t('settings.providers.messages.modelsRefreshedDescription', { count: result.data.length }))
  } catch (error) {
    console.error('[provider-settings] Failed to refresh provider models.', error)
    toast.error(t('settings.providers.messages.refreshFailed'), error instanceof Error ? error.message : undefined, error)
  } finally {
    pendingAction.value = ''
  }
}

async function confirmDelete() {
  const target = deleteTarget.value
  if (!target) return
  pendingAction.value = `delete:${target.id}`
  try {
    await api.deleteProvider(target.id)
    providers.value = providers.value.filter((item) => item.id !== target.id)
    deleteTarget.value = null
    toast.success(t('settings.providers.messages.deleted'), target.name)
  } catch (error) {
    console.error('[provider-settings] Failed to delete provider.', error)
    toast.error(t('settings.providers.messages.deleteFailed'), error instanceof Error ? error.message : undefined, error)
  } finally {
    pendingAction.value = ''
  }
}

function statusTone(status: ProviderConfig['status']) {
  if (status === 'online') return 'success' as const
  if (status === 'degraded') return 'warning' as const
  if (status === 'offline') return 'danger' as const
  return 'muted' as const
}
</script>

<template>
  <SettingsWorkspace :title="t('settings.providers.title')" :description="t('settings.providers.description')">
    <div class="space-y-6">
      <UiInlineNotice v-if="loadError" tone="danger" :title="t('settings.providers.messages.loadFailed')" :description="loadError"><template #actions><UiButton variant="outline" size="sm" :loading="loading" @click="loadProviders">{{ t('common.retry') }}</UiButton></template></UiInlineNotice>
      <div class="flex flex-col gap-3 border-b border-border pb-5 sm:flex-row sm:items-end sm:justify-between">
        <label class="block min-w-0 flex-1 space-y-2"><span class="field-label">{{ t('settings.providers.search') }}</span><UiInput v-model="query" :placeholder="t('settings.providers.searchPlaceholder')" /></label>
        <div class="flex gap-2"><UiButton variant="outline" :loading="loading" @click="loadProviders"><RefreshCw class="h-4 w-4" />{{ t('actions.refresh') }}</UiButton><UiButton @click="openCreate"><Plus class="h-4 w-4" />{{ t('settings.providers.add') }}</UiButton></div>
      </div>

      <div v-if="loading && providers.length === 0" class="py-16 text-center text-sm font-bold text-muted-foreground">{{ t('settings.providers.messages.loading') }}</div>
      <div v-else-if="visibleProviders.length === 0" class="border border-dashed border-border p-10 text-center"><h2 class="text-xl font-black">{{ t('settings.providers.emptyTitle') }}</h2><p class="mt-2 text-sm text-muted-foreground">{{ t('settings.providers.emptyDescription') }}</p></div>
      <div v-else class="divide-y divide-border border-y border-border">
        <article v-for="provider in visibleProviders" :key="provider.id" class="grid gap-5 py-5 xl:grid-cols-[minmax(0,1fr)_16rem_15rem] xl:items-center">
          <div class="min-w-0"><div class="flex flex-wrap items-center gap-2"><h2 class="truncate text-xl font-black">{{ provider.name || provider.id }}</h2><UiBadge :tone="statusTone(provider.status)">{{ t(`status.provider.${provider.status}`) }}</UiBadge><UiBadge :tone="provider.enabled ? 'success' : 'muted'">{{ provider.enabled ? t('status.enabled') : t('status.disabled') }}</UiBadge></div><p class="mt-2 break-all font-mono text-xs text-muted-foreground">{{ provider.id }} · {{ provider.provider_type }}</p><p class="mt-2 break-all text-sm text-muted-foreground">{{ provider.base_url }}</p></div>
          <dl class="grid grid-cols-2 gap-x-4 gap-y-3 text-sm"><div><dt class="text-xs text-muted-foreground">API Key</dt><dd class="mt-1 flex items-center gap-2 font-bold"><KeyRound class="h-4 w-4" />{{ provider.api_key_hint || t('settings.providers.noSecret') }}</dd></div><div><dt class="text-xs text-muted-foreground">{{ t('settings.providers.fields.streaming') }}</dt><dd class="mt-1 flex items-center gap-2 font-bold"><Waves class="h-4 w-4" />{{ provider.streaming ? t('status.enabled') : t('status.disabled') }}</dd></div><div><dt class="text-xs text-muted-foreground">{{ t('settings.providers.fields.timeout') }}</dt><dd class="mt-1 font-mono font-bold">{{ provider.default_request_timeout_sec ?? '—' }}</dd></div><div><dt class="text-xs text-muted-foreground">{{ t('settings.providers.fields.defaultModel') }}</dt><dd class="mt-1 truncate font-mono text-xs font-bold">{{ provider.default_model_id || '—' }}</dd></div></dl>
          <div class="flex flex-wrap justify-start gap-2 xl:justify-end"><UiButton size="sm" variant="outline" @click="openEdit(provider)"><Pencil class="h-4 w-4" />{{ t('actions.edit') }}</UiButton><UiButton size="sm" variant="outline" :loading="pendingAction === `refresh:${provider.id}`" @click="refreshModels(provider)"><RefreshCw class="h-4 w-4" />{{ t('settings.providers.refreshModels') }}</UiButton><UiButton size="sm" variant="destructive" @click="deleteTarget = provider"><Trash2 class="h-4 w-4" />{{ t('actions.delete') }}</UiButton></div>
        </article>
      </div>
    </div>

    <ProviderConfigureDialog v-model:open="dialogOpen" :provider="selectedProvider" :saving="saving" @save="saveProvider" />
    <UiConfirm v-model:open="deleteConfirmOpen" :title="t('settings.providers.deleteTitle')" :description="deleteTarget ? t('settings.providers.deleteDescription', { name: deleteTarget.name }) : ''" :loading="Boolean(deleteTarget && pendingAction === `delete:${deleteTarget.id}`)" @confirm="confirmDelete" />
  </SettingsWorkspace>
</template>
