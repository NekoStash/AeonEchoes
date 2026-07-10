<script setup lang="ts">
import { Save } from '@lucide/vue'
import { createSkillForm, skillFormToConfig, type SkillFormState } from './resource-forms'
import type { Skill } from '~/lib/types'

const props = defineProps<{ open: boolean; skill?: Skill | null; saving?: boolean }>()
const emit = defineEmits<{ 'update:open': [value: boolean]; save: [skill: Skill, mode: 'create' | 'edit'] }>()
const { t } = useI18n()
const form = reactive<SkillFormState>(createSkillForm())
const errorMessage = ref('')
const mode = computed<'create' | 'edit'>(() => props.skill ? 'edit' : 'create')
watch(() => [props.open, props.skill] as const, ([open]) => { if (open) { Object.assign(form, createSkillForm(props.skill || undefined)); errorMessage.value = '' } }, { immediate: true })
function submit() { errorMessage.value = ''; try { emit('save', skillFormToConfig(form, props.skill || undefined), mode.value) } catch (error) { errorMessage.value = error instanceof Error ? error.message : t('settings.agents.errors.invalidForm') } }
</script>

<template>
  <UiDialog :open="open" size="lg" :title="mode === 'edit' ? t('settings.agents.skillDialog.editTitle') : t('settings.agents.skillDialog.createTitle')" :description="t('settings.agents.skillDialog.description')" @update:open="emit('update:open', $event)">
    <div class="space-y-4"><UiAlert v-if="errorMessage" tone="danger" :title="t('settings.formError')" :description="errorMessage" /><div class="grid gap-4 md:grid-cols-2"><label class="space-y-2"><span class="field-label">ID</span><UiInput v-model="form.id" :disabled="mode === 'edit'" /></label><label class="space-y-2"><span class="field-label">project_id</span><UiInput v-model="form.project_id" /></label><label class="space-y-2"><span class="field-label">source_id</span><UiInput v-model="form.source_id" /></label><label class="space-y-2"><span class="field-label">{{ t('settings.fields.name') }}</span><UiInput v-model="form.name" /></label><label class="space-y-2 md:col-span-2"><span class="field-label">path</span><UiInput v-model="form.path" /></label><label class="space-y-2 md:col-span-2"><span class="field-label">{{ t('settings.fields.description') }}</span><UiInput v-model="form.description" /></label><label class="space-y-2 md:col-span-2"><span class="field-label">content</span><UiTextarea v-model="form.content" :rows="10" /></label><label class="space-y-2 md:col-span-2"><span class="field-label">metadata</span><UiTextarea v-model="form.metadataText" :rows="5" /></label><UiSwitch v-model="form.enabled" class="md:col-span-2" :label="t('settings.fields.enabled')" /></div></div>
    <template #footer><div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end"><UiButton variant="outline" :disabled="saving" @click="emit('update:open', false)">{{ t('actions.cancel') }}</UiButton><UiButton :loading="saving" @click="submit"><Save class="h-4 w-4" />{{ t('actions.saveConfig') }}</UiButton></div></template>
  </UiDialog>
</template>
