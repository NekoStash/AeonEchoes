<script setup lang="ts">
import { Save } from '@lucide/vue'
import { configurableAgentRoles } from '~/features/model-configure/model-form'
import { agentFormToConfig, createAgentForm, type AgentFormState } from './resource-forms'
import type { AgentConfig, ModelConfig } from '~/lib/types'

const props = defineProps<{ open: boolean; agent?: AgentConfig | null; models: ModelConfig[]; saving?: boolean }>()
const emit = defineEmits<{ 'update:open': [value: boolean]; save: [agent: AgentConfig, mode: 'create' | 'edit'] }>()
const { t } = useI18n()
const form = reactive<AgentFormState>(createAgentForm())
const errorMessage = ref('')
const mode = computed<'create' | 'edit'>(() => props.agent ? 'edit' : 'create')
const modelOptions = computed(() => props.models.map((model) => ({ label: model.display_name || model.name, value: model.id.includes(':') ? model.id : `${model.provider_id}:${model.name}`, description: model.kind })))
const roleOptions = computed(() => configurableAgentRoles.map((role) => ({ value: role, label: roleLabel(role) })))

watch(() => [props.open, props.agent] as const, ([open]) => { if (open) { Object.assign(form, createAgentForm(props.agent || undefined)); errorMessage.value = '' } }, { immediate: true })

function submit() {
  errorMessage.value = ''
  try { emit('save', agentFormToConfig(form, props.agent || undefined), mode.value) } catch (error) { errorMessage.value = error instanceof Error ? error.message : t('settings.agents.errors.invalidForm') }
}
function roleLabel(role: string) { const key = `agents.roles.${role.replace(/-/g, '_')}`; const value = t(key); return value === key ? role : value }
</script>

<template>
  <UiDialog :open="open" size="xl" :title="mode === 'edit' ? t('settings.agents.dialog.editTitle') : t('settings.agents.dialog.createTitle')" :description="t('settings.agents.dialog.description')" @update:open="emit('update:open', $event)">
    <div class="space-y-6">
      <UiAlert v-if="errorMessage" tone="danger" :title="t('settings.formError')" :description="errorMessage" />
      <section class="border border-border"><div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.agents.sections.identity') }}</h3></div><div class="grid gap-4 p-4 md:grid-cols-2">
        <label class="space-y-2"><span class="field-label">ID</span><UiInput v-model="form.id" :disabled="mode === 'edit'" /></label><label class="space-y-2"><span class="field-label">project_id</span><UiInput v-model="form.project_id" /></label><label class="space-y-2"><span class="field-label">{{ t('settings.fields.name') }}</span><UiInput v-model="form.name" /></label><label class="space-y-2"><span class="field-label">role</span><UiSelect v-model="form.role" :options="roleOptions" /></label><label class="space-y-2 md:col-span-2"><span class="field-label">model_id</span><UiSelect v-model="form.model_id" :options="modelOptions" :placeholder="t('settings.agents.inheritModel')" searchable /></label><label class="space-y-2 md:col-span-2"><span class="field-label">{{ t('settings.fields.description') }}</span><UiInput v-model="form.description" /></label><label class="space-y-2 md:col-span-2"><span class="field-label">system_prompt</span><UiTextarea v-model="form.system_prompt" :rows="7" /></label><UiSwitch v-model="form.enabled" class="md:col-span-2" :label="t('settings.fields.enabled')" />
      </div></section>
      <section class="border border-border"><div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.agents.sections.bindings') }}</h3></div><div class="grid gap-4 p-4 md:grid-cols-3"><label class="space-y-2"><span class="field-label">skill_ids</span><UiTextarea v-model="form.skillIdsText" :rows="6" /></label><label class="space-y-2"><span class="field-label">tool_ids</span><UiTextarea v-model="form.toolIdsText" :rows="6" /></label><label class="space-y-2"><span class="field-label">mcp_server_ids</span><UiTextarea v-model="form.mcpServerIdsText" :rows="6" /></label></div></section>
      <section class="border border-border"><div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.agents.sections.runtime') }}</h3></div><div class="grid gap-4 p-4 lg:grid-cols-3"><label class="space-y-2"><span class="field-label">memory_policy</span><UiTextarea v-model="form.memoryPolicyText" :rows="7" /></label><label class="space-y-2"><span class="field-label">runtime_options</span><UiTextarea v-model="form.runtimeOptionsText" :rows="7" /></label><label class="space-y-2"><span class="field-label">metadata</span><UiTextarea v-model="form.metadataText" :rows="7" /></label></div><p class="px-4 pb-4 text-xs text-muted-foreground">{{ t('settings.agents.jsonHelp') }}</p></section>
    </div>
    <template #footer><div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end"><UiButton variant="outline" :disabled="saving" @click="emit('update:open', false)">{{ t('actions.cancel') }}</UiButton><UiButton :loading="saving" @click="submit"><Save class="h-4 w-4" />{{ t('actions.saveConfig') }}</UiButton></div></template>
  </UiDialog>
</template>
