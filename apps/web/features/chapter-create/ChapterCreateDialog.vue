<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { CHAPTER_STATUS_VALUES, type Chapter } from '~/entities/chapter'
import type { StoryBibleChapter } from '~/entities/story-bible'
import { createChapterDraft, toCreateChapterRequest, type ChapterCreateDraft } from './model'

const { t } = useI18n()
const props = defineProps<{
  open: boolean
  chapters: Chapter[]
  plans: StoryBibleChapter[]
  loading?: boolean
  error?: string
}>()
const emit = defineEmits<{
  'update:open': [value: boolean]
  confirm: [request: ReturnType<typeof toCreateChapterRequest>]
}>()

const draft = ref<ChapterCreateDraft>(createChapterDraft())
const validationError = ref('')
const statusOptions = computed(() => CHAPTER_STATUS_VALUES.map((status) => ({ label: t(`status.chapter.${status}`), value: status })))
const planOptions = computed(() => props.plans.map((plan, index) => ({
  label: plan.title || t('projectOverview.chapterCreate.untitledPlan', { number: index + 1 }),
  value: plan.id,
  description: plan.summary
})))

watch(() => props.open, (open) => {
  if (!open) return
  draft.value = createChapterDraft()
  validationError.value = ''
})

function applyPlan(planId: string) {
  const plan = props.plans.find((item) => item.id === planId)
  draft.value = createChapterDraft(plan)
}

function confirm() {
  validationError.value = ''
  try {
    emit('confirm', toCreateChapterRequest(draft.value, props.chapters))
  } catch (error) {
    console.error('[AeonEchoes Chapter Create] Invalid chapter draft.', error)
    validationError.value = t('projectOverview.chapterCreate.titleRequired')
  }
}
</script>

<template>
  <UiDialog
    :open="open"
    :title="t('projectOverview.chapterCreate.title')"
    :description="t('projectOverview.chapterCreate.description')"
    size="md"
    :close-on-backdrop="!loading"
    @update:open="!loading && emit('update:open', $event)"
  >
    <form class="space-y-5" @submit.prevent="confirm">
      <label v-if="plans.length" class="block space-y-2">
        <span class="field-label">{{ t('projectOverview.chapterCreate.fromPlan') }}</span>
        <UiSelect
          :model-value="draft.planId"
          :options="planOptions"
          :disabled="loading"
          :placeholder="t('projectOverview.chapterCreate.noPlan')"
          searchable
          :aria-label="t('projectOverview.chapterCreate.fromPlan')"
          @update:model-value="applyPlan"
        />
      </label>

      <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_12rem]">
        <label class="space-y-2">
          <span class="field-label">{{ t('projectOverview.fields.chapterTitle') }}</span>
          <UiInput v-model="draft.title" :disabled="loading" :invalid="Boolean(validationError)" autofocus />
        </label>
        <label class="space-y-2">
          <span class="field-label">{{ t('projectOverview.fields.chapterStatus') }}</span>
          <UiSelect v-model="draft.status" :disabled="loading" :options="statusOptions" :aria-label="t('projectOverview.fields.chapterStatus')" />
        </label>
      </div>

      <label class="block space-y-2">
        <span class="field-label">{{ t('projectOverview.fields.chapterSummary') }}</span>
        <UiTextarea v-model="draft.summary" :disabled="loading" :rows="5" />
      </label>

      <p v-if="validationError || error" role="alert" class="border-l-4 border-destructive bg-destructive/10 px-4 py-3 text-sm text-destructive">
        {{ validationError || error }}
      </p>
    </form>

    <template #footer>
      <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <UiButton variant="outline" :disabled="loading" @click="emit('update:open', false)">
          {{ t('actions.cancel') }}
        </UiButton>
        <UiButton :loading="loading" :loading-label="t('projectOverview.chapterCreate.creating')" @click="confirm">
          {{ loading ? t('projectOverview.chapterCreate.creating') : t('projectOverview.chapterCreate.confirm') }}
        </UiButton>
      </div>
    </template>
  </UiDialog>
</template>
