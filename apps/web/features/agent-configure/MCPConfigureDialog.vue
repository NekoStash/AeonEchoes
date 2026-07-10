<script setup lang="ts">
import { Save } from '@lucide/vue'
import { createMCPForm, mcpFormToConfig, type MCPFormState } from './resource-forms'
import type { MCPServerConfig } from '~/lib/types'

const props = defineProps<{ open: boolean; server?: MCPServerConfig | null; saving?: boolean }>()
const emit = defineEmits<{ 'update:open': [value: boolean]; save: [server: MCPServerConfig, mode: 'create' | 'edit'] }>()
const { t } = useI18n()
const form = reactive<MCPFormState>(createMCPForm())
const errorMessage = ref('')
const mode = computed<'create' | 'edit'>(() => props.server ? 'edit' : 'create')
watch(() => [props.open, props.server] as const, ([open]) => { if (open) { Object.assign(form, createMCPForm(props.server || undefined)); errorMessage.value = '' } }, { immediate: true })
function submit() { errorMessage.value = ''; try { emit('save', mcpFormToConfig(form, props.server || undefined), mode.value) } catch (error) { errorMessage.value = error instanceof Error ? error.message : t('settings.agents.errors.invalidForm') } }
</script>

<template>
  <UiDialog :open="open" size="xl" :title="mode === 'edit' ? t('settings.agents.mcpDialog.editTitle') : t('settings.agents.mcpDialog.createTitle')" :description="t('settings.agents.mcpDialog.description')" @update:open="emit('update:open', $event)">
    <div class="space-y-6"><UiAlert v-if="errorMessage" tone="danger" :title="t('settings.formError')" :description="errorMessage" />
      <section class="border border-border"><div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.agents.sections.identity') }}</h3></div><div class="grid gap-4 p-4 md:grid-cols-2"><label class="space-y-2"><span class="field-label">ID</span><UiInput v-model="form.id" :disabled="mode === 'edit'" /></label><label class="space-y-2"><span class="field-label">project_id</span><UiInput v-model="form.project_id" /></label><label class="space-y-2"><span class="field-label">{{ t('settings.fields.name') }}</span><UiInput v-model="form.name" /></label><label class="space-y-2"><span class="field-label">transport</span><UiSelect v-model="form.transport" :options="[{ value: 'stdio', label: 'stdio' }, { value: 'streamable_http', label: 'streamable HTTP' }, { value: 'sse', label: 'SSE' }]" /></label><label v-if="form.transport === 'stdio'" class="space-y-2 md:col-span-2"><span class="field-label">command</span><UiInput v-model="form.command" /></label><label v-else class="space-y-2 md:col-span-2"><span class="field-label">url</span><UiInput v-model="form.url" /></label><label class="space-y-2 md:col-span-2"><span class="field-label">args</span><UiTextarea v-model="form.argsText" :rows="4" /></label><label class="space-y-2"><span class="field-label">timeout_sec</span><UiInput v-model="form.timeoutSec" type="number" min="0" /></label><UiSwitch v-model="form.enabled" :label="t('settings.fields.enabled')" /></div></section>
      <section class="border border-border"><div class="border-b border-border bg-surface-muted px-4 py-3"><h3 class="text-xs font-black uppercase tracking-[0.16em]">{{ t('settings.agents.sections.environment') }}</h3></div><div class="grid gap-4 p-4 md:grid-cols-2"><label class="space-y-2"><span class="field-label">headers</span><UiTextarea v-model="form.headersText" :rows="6" /></label><label class="space-y-2"><span class="field-label">secret_headers</span><UiTextarea v-model="form.secretHeadersText" :rows="6" /><span class="block text-xs text-muted-foreground">{{ t('settings.agents.secretBlankHelp') }}</span></label><label class="space-y-2"><span class="field-label">env</span><UiTextarea v-model="form.envText" :rows="6" /></label><label class="space-y-2"><span class="field-label">secret_env</span><UiTextarea v-model="form.secretEnvText" :rows="6" /><span class="block text-xs text-muted-foreground">{{ t('settings.agents.secretBlankHelp') }}</span></label><label class="space-y-2 md:col-span-2"><span class="field-label">metadata</span><UiTextarea v-model="form.metadataText" :rows="5" /></label></div><p class="px-4 pb-4 text-xs text-muted-foreground">{{ t('settings.stringMapHelp') }}</p></section>
    </div>
    <template #footer><div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end"><UiButton variant="outline" :disabled="saving" @click="emit('update:open', false)">{{ t('actions.cancel') }}</UiButton><UiButton :loading="saving" @click="submit"><Save class="h-4 w-4" />{{ t('actions.saveConfig') }}</UiButton></div></template>
  </UiDialog>
</template>
