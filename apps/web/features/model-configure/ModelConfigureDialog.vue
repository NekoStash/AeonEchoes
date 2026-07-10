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
const baselineSnapshot = ref('')
const errorMessage = ref('')
const discardConfirmOpen = ref(false)
const mode = computed(() => props.model ? 'edit' : 'create')
const providerOptions = computed(() => props.providers.map((provider) => ({ label: provider.name || provider.id, value: provider.id, description: provider.provider_type })))
const isDirty = computed(() => props.open && snapshotForm(form) !== baselineSnapshot.value)

watch(() => props.open, (open) => {
  if (!open) {
    discardConfirmOpen.value = false
    return
  }
  Object.assign(form, createModelForm(props.model || undefined, props.providers[0]?.id || ''))
  baselineSnapshot.value = snapshotForm(form)
  errorMessage.value = ''
}, { immediate: true })

function snapshotForm(value: ModelFormState) {
  return JSON.stringify({
    ...value,
    allowed_agent_roles: [...value.allowed_agent_roles].sort()
  })
}

function requestClose() {
  if (props.saving) return
  if (isDirty.value) {
    discardConfirmOpen.value = true
    return
  }
  emit('update:open', false)
}

function confirmDiscard() {
  discardConfirmOpen.value = false
  baselineSnapshot.value = snapshotForm(form)
  emit('update:open', false)
}

function submit() {
  errorMessage.value = ''
  try {
    emit('save', modelFormToConfig(form, props.model || undefined))
  } catch (error) {
    console.error('[model-configure] Failed to submit model form.', error)
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
  <UiDialog
    :open="open"
    size="xl"
    :title="mode === 'edit' ? t('settings.models.dialog.editTitle') : t('settings.models.dialog.createTitle')"
    :description="t('settings.models.dialog.description')"
    :close-on-backdrop="!saving"
    data-testid="model-configure-dialog"
    @update:open="requestClose"
  >
    <div class="space-y-6">
      <UiAlert v-if="errorMessage" tone="danger" :title="t('settings.formError')" :description="errorMessage" />
      <section class="border border-border">
        <div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.models.sections.identity') }}</h3></div>
        <div class="grid gap-4 p-4 md:grid-cols-2">
          <UiField label="ID" v-slot="field"><UiInput v-model="form.id" :id="field.id" :aria-describedby="field.describedby" :disabled="mode === 'edit'" /></UiField>
          <UiField :label="t('settings.models.fields.provider')" v-slot="field"><UiSelect v-model="form.provider_id" :id="field.id" :aria-describedby="field.describedby" :aria-label="t('settings.models.fields.provider')" :options="providerOptions" searchable /></UiField>
          <UiField :label="t('settings.models.fields.upstreamId')" v-slot="field"><UiInput v-model="form.name" :id="field.id" :aria-describedby="field.describedby" /></UiField>
          <UiField :label="t('settings.fields.name')" v-slot="field"><UiInput v-model="form.display_name" :id="field.id" :aria-describedby="field.describedby" /></UiField>
          <UiField :label="t('settings.models.fields.kind')" v-slot="field"><UiSelect v-model="form.kind" :id="field.id" :aria-describedby="field.describedby" :aria-label="t('settings.models.fields.kind')" :options="[{ value: 'text', label: t('models.kinds.text') }, { value: 'embedding', label: t('models.kinds.embedding') }]" /></UiField>
        </div>
      </section>

      <section class="border border-border">
        <div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.models.sections.limits') }}</h3></div>
        <div class="grid gap-4 p-4 md:grid-cols-2 xl:grid-cols-3">
          <UiField v-if="form.kind === 'text'" label="context_window" v-slot="field"><UiInput v-model="form.context_window" :id="field.id" :aria-describedby="field.describedby" type="number" min="0" /></UiField>
          <UiField v-if="form.kind === 'text'" label="max_output_tokens" v-slot="field"><UiInput v-model="form.max_output_tokens" :id="field.id" :aria-describedby="field.describedby" type="number" min="0" /></UiField>
          <UiField v-if="form.kind === 'embedding'" label="dimension" v-slot="field"><UiInput v-model="form.dimension" :id="field.id" :aria-describedby="field.describedby" type="number" min="0" /></UiField>
          <UiField label="cost_input_per_mtok" v-slot="field"><UiInput v-model="form.cost_input_per_mtok" :id="field.id" :aria-describedby="field.describedby" type="number" min="0" step="0.000001" /></UiField>
          <UiField label="cost_output_per_mtok" v-slot="field"><UiInput v-model="form.cost_output_per_mtok" :id="field.id" :aria-describedby="field.describedby" type="number" min="0" step="0.000001" /></UiField>
          <UiField label="routing_weight" v-slot="field"><UiInput v-model="form.routing_weight" :id="field.id" :aria-describedby="field.describedby" type="number" min="0" /></UiField>
        </div>
      </section>

      <section class="border border-border">
        <div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.models.sections.capabilities') }}</h3></div>
        <div class="grid gap-3 p-4 md:grid-cols-2">
          <UiSwitch v-model="form.enabled" :label="t('settings.fields.enabled')" />
          <UiSwitch v-model="form.default_for_kind" :label="t('models.defaultForKind')" />
          <UiSwitch v-if="form.kind === 'text'" v-model="form.supports_tools" :label="t('models.supportsTools')" />
          <UiSwitch v-if="form.kind === 'text'" v-model="form.supports_streaming" :label="t('models.supportsStreaming')" />
        </div>
      </section>

      <section v-if="form.kind === 'text'" class="border border-border">
        <div class="border-b border-border bg-surface-muted px-4 py-3">
          <h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.models.roleEligibility.title') }}</h3>
          <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ t('settings.models.roleEligibility.description') }}</p>
        </div>
        <div class="grid gap-2 p-4 sm:grid-cols-2 lg:grid-cols-3">
          <button v-for="role in configurableAgentRoles" :key="role" type="button" :aria-pressed="form.allowed_agent_roles.includes(role)" :class="['focus-ring border px-3 py-3 text-left text-sm font-bold', form.allowed_agent_roles.includes(role) ? 'border-foreground bg-foreground text-background' : 'border-border bg-background hover:bg-muted']" @click="toggleRole(form, role)">{{ roleLabel(role) }}</button>
        </div>
        <p class="border-t border-border px-4 py-3 text-xs leading-5 text-muted-foreground">{{ t('settings.models.roleEligibility.emptyMeansAll') }}</p>
      </section>
      <UiInlineNotice v-else tone="neutral" :title="t('settings.models.roleEligibility.embeddingTitle')" :description="t('settings.models.roleEligibility.embeddingDescription')" />

      <section class="border border-border"><div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">metadata</h3></div><div class="p-4"><UiTextarea v-model="form.metadataText" :rows="5" /><p class="mt-2 text-xs text-muted-foreground">{{ t('settings.stringMapHelp') }}</p></div></section>
    </div>
    <template #footer>
      <div class="flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p class="text-xs text-muted-foreground" aria-live="polite">{{ isDirty ? t('settings.models.dialog.unsaved') : t('settings.models.dialog.savedState') }}</p>
        <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <UiButton variant="outline" :disabled="saving" @click="requestClose">{{ t('actions.cancel') }}</UiButton>
          <UiButton :loading="saving" @click="submit"><Save class="h-4 w-4" />{{ t('actions.saveConfig') }}</UiButton>
        </div>
      </div>
    </template>
  </UiDialog>

  <UiConfirm
    v-model:open="discardConfirmOpen"
    :title="t('settings.models.dialog.discardTitle')"
    :description="t('settings.models.dialog.discardDescription')"
    :confirm-label="t('settings.models.dialog.discardAction')"
    tone="danger"
    @confirm="confirmDiscard"
  />
</template>
