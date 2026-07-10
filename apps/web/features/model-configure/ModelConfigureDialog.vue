<script setup lang="ts">
import { Save } from '@lucide/vue'
import { configurableAgentRoles, createModelForm, modelFormToConfig, toggleRole, type ModelFormState } from './model-form'
import type { ModelConfig, ProviderConfig } from '~/lib/types'

const props = defineProps<{
  open: boolean
  model?: ModelConfig | null
  providers: ProviderConfig[]
  saving?: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  save: [model: ModelConfig]
}>()

const { t } = useI18n()
const form = reactive<ModelFormState>(createModelForm())
const errorMessage = ref('')
const mode = computed(() => props.model ? 'edit' : 'create')
const providerOptions = computed(() => props.providers.map((provider) => ({ label: provider.name || provider.id, value: provider.id, description: provider.provider_type })))

watch(() => [props.open, props.model, props.providers] as const, ([open]) => {
  if (!open) return
  Object.assign(form, createModelForm(props.model || undefined, props.providers[0]?.id || ''))
  errorMessage.value = ''
}, { immediate: true })

function submit() {
  errorMessage.value = ''
  try {
    emit('save', modelFormToConfig(form, props.model || undefined))
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('settings.models.errors.invalidForm')
  }
}

function roleLabel(role: string) {
  const key = `models.roles.${role.replace(/-/g, '_')}`
  const value = t(key)
  return value === key ? role : value
}
</script>

<template>
  <UiDialog :open="open" size="xl" :title="mode === 'edit' ? t('settings.models.dialog.editTitle') : t('settings.models.dialog.createTitle')" :description="t('settings.models.dialog.description')" @update:open="emit('update:open', $event)">
    <div class="space-y-6">
      <UiAlert v-if="errorMessage" tone="danger" :title="t('settings.formError')" :description="errorMessage" />
      <section class="border border-border">
        <div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.models.sections.identity') }}</h3></div>
        <div class="grid gap-4 p-4 md:grid-cols-2">
          <label class="space-y-2"><span class="field-label">ID</span><UiInput v-model="form.id" :disabled="mode === 'edit'" /></label>
          <label class="space-y-2"><span class="field-label">{{ t('settings.models.fields.provider') }}</span><UiSelect v-model="form.provider_id" :options="providerOptions" searchable /></label>
          <label class="space-y-2"><span class="field-label">{{ t('settings.models.fields.upstreamId') }}</span><UiInput v-model="form.name" /></label>
          <label class="space-y-2"><span class="field-label">{{ t('settings.fields.name') }}</span><UiInput v-model="form.display_name" /></label>
          <label class="space-y-2"><span class="field-label">{{ t('settings.models.fields.kind') }}</span><UiSelect v-model="form.kind" :options="[{ value: 'text', label: t('models.kinds.text') }, { value: 'embedding', label: t('models.kinds.embedding') }]" /></label>
        </div>
      </section>

      <section class="border border-border">
        <div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.models.sections.limits') }}</h3></div>
        <div class="grid gap-4 p-4 md:grid-cols-2 xl:grid-cols-3">
          <label class="space-y-2"><span class="field-label">context_window</span><UiInput v-model="form.context_window" type="number" min="0" /></label>
          <label class="space-y-2"><span class="field-label">max_output_tokens</span><UiInput v-model="form.max_output_tokens" type="number" min="0" /></label>
          <label class="space-y-2"><span class="field-label">dimension</span><UiInput v-model="form.dimension" type="number" min="0" /></label>
          <label class="space-y-2"><span class="field-label">cost_input_per_mtok</span><UiInput v-model="form.cost_input_per_mtok" type="number" min="0" step="0.000001" /></label>
          <label class="space-y-2"><span class="field-label">cost_output_per_mtok</span><UiInput v-model="form.cost_output_per_mtok" type="number" min="0" step="0.000001" /></label>
          <label class="space-y-2"><span class="field-label">routing_weight</span><UiInput v-model="form.routing_weight" type="number" min="0" /></label>
        </div>
      </section>

      <section class="border border-border">
        <div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.models.sections.capabilities') }}</h3></div>
        <div class="grid gap-3 p-4 md:grid-cols-2">
          <UiSwitch v-model="form.enabled" :label="t('settings.fields.enabled')" />
          <UiSwitch v-model="form.default_for_kind" :label="t('models.defaultForKind')" />
          <UiSwitch v-model="form.supports_tools" :label="t('models.supportsTools')" />
          <UiSwitch v-model="form.supports_streaming" :label="t('models.supportsStreaming')" />
        </div>
      </section>

      <section class="border border-border">
        <div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">allowed_agent_roles</h3></div>
        <div class="grid gap-2 p-4 sm:grid-cols-2 lg:grid-cols-3">
          <button v-for="role in configurableAgentRoles" :key="role" type="button" :aria-pressed="form.allowed_agent_roles.includes(role)" :class="['focus-ring border px-3 py-3 text-left text-sm font-bold', form.allowed_agent_roles.includes(role) ? 'border-foreground bg-foreground text-background' : 'border-border bg-background hover:bg-muted']" @click="toggleRole(form, role)">{{ roleLabel(role) }}</button>
        </div>
      </section>

      <section class="border border-border"><div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">metadata</h3></div><div class="p-4"><UiTextarea v-model="form.metadataText" :rows="5" /><p class="mt-2 text-xs text-muted-foreground">{{ t('settings.stringMapHelp') }}</p></div></section>
    </div>
    <template #footer><div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end"><UiButton variant="outline" :disabled="saving" @click="emit('update:open', false)">{{ t('actions.cancel') }}</UiButton><UiButton :loading="saving" @click="submit"><Save class="h-4 w-4" />{{ t('actions.saveConfig') }}</UiButton></div></template>
  </UiDialog>
</template>
