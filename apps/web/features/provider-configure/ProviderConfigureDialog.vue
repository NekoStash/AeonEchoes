<script setup lang="ts">
import { Save } from '@lucide/vue'
import { createProviderForm, providerFormToConfig, type ProviderFormState } from './provider-form'
import type { ProviderConfig, ProviderType } from '~/lib/types'

const props = defineProps<{
  open: boolean
  provider?: ProviderConfig | null
  saving?: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  save: [provider: ProviderConfig, mode: 'create' | 'edit']
}>()

const { t } = useI18n()
const form = reactive<ProviderFormState>(createProviderForm())
const errorMessage = ref('')
const mode = computed<'create' | 'edit'>(() => props.provider ? 'edit' : 'create')
const providerTypes: Array<{ value: ProviderType; label: string }> = [
  { value: 'openai-responses', label: 'OpenAI Responses' },
  { value: 'openai', label: 'OpenAI Chat Completions' },
  { value: 'anthropic', label: 'Anthropic Messages' },
  { value: 'gemini', label: 'Gemini Generate Content' }
]

watch(() => [props.open, props.provider] as const, ([open]) => {
  if (!open) return
  Object.assign(form, createProviderForm(props.provider || undefined))
  errorMessage.value = ''
}, { immediate: true })

function submit() {
  errorMessage.value = ''
  try {
    emit('save', providerFormToConfig(form, props.provider || undefined), mode.value)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('settings.providers.errors.invalidForm')
  }
}
</script>

<template>
  <UiDialog :open="open" size="xl" :title="mode === 'edit' ? t('settings.providers.dialog.editTitle') : t('settings.providers.dialog.createTitle')" :description="t('settings.providers.dialog.description')" @update:open="emit('update:open', $event)">
    <div class="space-y-6">
      <UiAlert v-if="errorMessage" tone="danger" :title="t('settings.formError')" :description="errorMessage" />
      <section class="border border-border">
        <div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.providers.sections.identity') }}</h3></div>
        <div class="grid gap-4 p-4 md:grid-cols-2">
          <label class="space-y-2"><span class="field-label">ID</span><UiInput v-model="form.id" :disabled="mode === 'edit'" :placeholder="t('settings.providers.placeholders.id')" /></label>
          <label class="space-y-2"><span class="field-label">{{ t('settings.fields.name') }}</span><UiInput v-model="form.name" /></label>
          <label class="space-y-2"><span class="field-label">{{ t('settings.providers.fields.type') }}</span><UiSelect v-model="form.provider_type" :options="providerTypes" /></label>
          <label class="space-y-2"><span class="field-label">Base URL</span><UiInput v-model="form.base_url" placeholder="https://api.example.com/v1" /></label>
          <label class="space-y-2 md:col-span-2"><span class="field-label">API Key</span><UiInput v-model="form.api_key" type="password" :placeholder="mode === 'edit' ? t('settings.providers.placeholders.keepSecret') : ''" /><span class="block text-xs text-muted-foreground">{{ t('settings.providers.secretHelp') }}</span></label>
        </div>
      </section>

      <section class="border border-border">
        <div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.providers.sections.runtime') }}</h3></div>
        <div class="grid gap-3 p-4 md:grid-cols-2">
          <UiSwitch v-model="form.enabled" :label="t('settings.fields.enabled')" />
          <UiSwitch v-model="form.streaming" :label="t('settings.providers.fields.streaming')" />
          <UiSwitch v-model="form.trace_enabled" :label="t('settings.providers.fields.traceEnabled')" />
          <label class="space-y-2"><span class="field-label">{{ t('settings.providers.fields.traceRetention') }}</span><UiInput v-model="form.trace_retention_days" type="number" min="0" /></label>
          <label class="space-y-2"><span class="field-label">{{ t('settings.providers.fields.timeout') }}</span><UiInput v-model="form.default_request_timeout_sec" type="number" min="0" /></label>
          <label class="space-y-2"><span class="field-label">{{ t('settings.providers.fields.defaultModel') }}</span><UiInput v-model="form.default_model_id" /></label>
        </div>
      </section>

      <section class="border border-border">
        <div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">metadata</h3></div>
        <div class="p-4"><UiTextarea v-model="form.metadataText" :rows="6" placeholder="{&#10;  &quot;region&quot;: &quot;us&quot;&#10;}" /><p class="mt-2 text-xs text-muted-foreground">{{ t('settings.stringMapHelp') }}</p></div>
      </section>
    </div>
    <template #footer>
      <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end"><UiButton variant="outline" :disabled="saving" @click="emit('update:open', false)">{{ t('actions.cancel') }}</UiButton><UiButton :loading="saving" @click="submit"><Save class="h-4 w-4" />{{ t('actions.saveConfig') }}</UiButton></div>
    </template>
  </UiDialog>
</template>
