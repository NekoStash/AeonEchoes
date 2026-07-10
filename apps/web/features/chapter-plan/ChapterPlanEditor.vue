<script setup lang="ts">
import { Plus, Trash2 } from '@lucide/vue'
import { computed, ref } from 'vue'
import { CHAPTER_STATUS_VALUES } from '~/entities/chapter'
import type { ChapterStatus, StoryBibleChapter } from '~/entities/story-bible'
import { createStoryBibleItemId } from '~/features/story-bible-edit'

const { t } = useI18n()
const props = defineProps<{
  modelValue: StoryBibleChapter[]
  disabled?: boolean
}>()
const emit = defineEmits<{
  'update:modelValue': [value: StoryBibleChapter[]]
}>()

const statusOptions = computed(() => CHAPTER_STATUS_VALUES.map((status) => ({ label: t(`status.chapter.${status}`), value: status })))
const pendingRemovalIndex = ref<number | null>(null)
const pendingRemoval = computed(() => pendingRemovalIndex.value === null ? null : props.modelValue[pendingRemovalIndex.value] || null)
const removeConfirmOpen = computed({
  get: () => pendingRemovalIndex.value !== null,
  set: (value: boolean) => {
    if (!value) pendingRemovalIndex.value = null
  }
})

function update(mutator: (draft: StoryBibleChapter[]) => void) {
  const next = JSON.parse(JSON.stringify(props.modelValue)) as StoryBibleChapter[]
  mutator(next)
  emit('update:modelValue', next)
}

function addPlan() {
  update((draft) => draft.push({
    id: createStoryBibleItemId('chapter-plan', draft.map((chapter) => chapter.id)),
    title: '',
    status: 'planned',
    summary: ''
  }))
}

function updateChapterStatus(index: number, status: string) {
  if (!CHAPTER_STATUS_VALUES.some((value) => value === status)) throw new Error(`Unsupported chapter status: ${status}`)
  updatePlan(index, { status: status as ChapterStatus })
}

function updatePlan(index: number, patch: Partial<StoryBibleChapter>) {
  update((draft) => {
    const current = draft[index]
    if (!current) throw new Error(`Chapter plan at index ${index} does not exist.`)
    draft[index] = { ...current, ...patch }
  })
}

function requestRemovePlan(index: number) {
  if (!props.modelValue[index]) throw new Error(`Chapter plan at index ${index} does not exist.`)
  pendingRemovalIndex.value = index
}

function confirmRemovePlan() {
  const index = pendingRemovalIndex.value
  if (index === null || !props.modelValue[index]) return
  update((draft) => draft.splice(index, 1))
  pendingRemovalIndex.value = null
}
</script>

<template>
  <section class="border-t-2 border-foreground pt-8" :aria-label="t('projectOverview.chapterPlan.title')">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <p class="text-xs font-bold uppercase tracking-[0.22em] text-muted-foreground">{{ t('projectOverview.chapterPlan.eyebrow') }}</p>
        <h2 class="mt-2 font-serif text-3xl font-semibold tracking-tight">{{ t('projectOverview.chapterPlan.title') }}</h2>
        <p class="mt-3 max-w-2xl text-sm leading-7 text-muted-foreground">{{ t('projectOverview.chapterPlan.description') }}</p>
      </div>
      <UiButton variant="outline" :disabled="disabled" @click="addPlan">
        <Plus class="h-4 w-4" aria-hidden="true" />
        {{ t('projectOverview.chapterPlan.add') }}
      </UiButton>
    </div>

    <p v-if="modelValue.length === 0" class="mt-6 border-y border-border py-7 text-sm text-muted-foreground">
      {{ t('projectOverview.chapterPlan.empty') }}
    </p>

    <ol v-else class="mt-6 divide-y divide-border border-y border-border">
      <li v-for="(chapter, index) in modelValue" :key="chapter.id || index" class="grid gap-5 py-7 lg:grid-cols-[5rem_minmax(0,1fr)_12rem_auto]">
        <div>
          <span class="font-serif text-4xl text-muted-foreground/60">{{ String(index + 1).padStart(2, '0') }}</span>
          <p class="mt-2 break-all text-[10px] uppercase tracking-[0.12em] text-muted-foreground">{{ chapter.id }}</p>
        </div>
        <div class="space-y-4">
          <label class="space-y-2">
            <span class="field-label">{{ t('projectOverview.fields.chapterTitle') }}</span>
            <UiInput
              :model-value="chapter.title"
              :disabled="disabled"
              @update:model-value="updatePlan(index, { title: $event })"
            />
          </label>
          <label class="space-y-2">
            <span class="field-label">{{ t('projectOverview.fields.chapterSummary') }}</span>
            <UiTextarea
              :model-value="chapter.summary"
              :disabled="disabled"
              :rows="4"
              @update:model-value="updatePlan(index, { summary: $event })"
            />
          </label>
        </div>
        <label class="space-y-2">
          <span class="field-label">{{ t('projectOverview.fields.chapterStatus') }}</span>
          <UiSelect
            :model-value="chapter.status"
            :disabled="disabled"
            :aria-label="t('projectOverview.fields.chapterStatus')"
            :options="statusOptions"
            @update:model-value="updateChapterStatus(index, $event)"
          />
        </label>
        <UiButton
          size="icon"
          variant="destructive"
          :disabled="disabled"
          :icon-label="t('projectOverview.actions.removeChapterPlanNamed', { name: chapter.title || index + 1 })"
          @click="requestRemovePlan(index)"
        >
          <Trash2 class="h-4 w-4" aria-hidden="true" />
        </UiButton>
      </li>
    </ol>
    <UiConfirm
      v-model:open="removeConfirmOpen"
      :title="t('actions.delete')"
      :description="pendingRemoval ? t('projectOverview.confirmRemove.chapterPlan', { name: pendingRemoval.title || (pendingRemovalIndex ?? 0) + 1 }) : ''"
      tone="danger"
      @confirm="confirmRemovePlan"
    />
  </section>
</template>
