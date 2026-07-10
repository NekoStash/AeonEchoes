<script setup lang="ts">
import { Maximize2, Minimize2, PanelRightOpen, Save } from '@lucide/vue'
import UiBadge from '~/components/ui/Badge.vue'
import UiButton from '~/components/ui/Button.vue'
import UiSelect from '~/components/ui/Select.vue'
import type { Chapter } from '~/entities/chapter'
import type { TextSelection } from '~/features/chapter-write'

const props = defineProps<{
  chapter: Chapter
  chapters: Chapter[]
  title: string
  content: string
  selectedChapterId: string
  characters: number
  paragraphs: number
  dirty: boolean
  saving: boolean
  fullscreen: boolean
}>()

const emit = defineEmits<{
  'update:title': [value: string]
  'update:content': [value: string]
  'update:selectedChapterId': [value: string]
  'update:fullscreen': [value: boolean]
  selection: [value: TextSelection]
  save: []
  assistant: []
}>()

const { t } = useI18n()
const textarea = ref<HTMLTextAreaElement | null>(null)
const chapterOptions = computed(() => props.chapters.map((chapter) => ({
  label: chapter.title || t('editor.chapterFallbackTitle', { number: chapter.number }),
  value: chapter.id,
  description: t('editor.chapterOptionDescription', { number: chapter.number, status: t(`status.chapter.${chapter.status}`) })
})))

function emitSelection() {
  const element = textarea.value
  if (!element) return
  emit('selection', { start: element.selectionStart, end: element.selectionEnd })
}

defineExpose({
  focus: () => textarea.value?.focus(),
  setSelection: (selection: TextSelection) => {
    nextTick(() => {
      textarea.value?.focus()
      textarea.value?.setSelectionRange(selection.start, selection.end)
    })
  }
})
</script>

<template>
  <section
    data-testid="writing-workspace"
    :class="[
      'writing-workspace relative min-w-0 overflow-hidden border border-border bg-surface text-surface-foreground',
      fullscreen && 'fixed inset-0 z-40 h-[100dvh] border-0'
    ]"
  >
    <div class="flex min-h-16 flex-col gap-3 border-b border-current/15 px-4 py-3 sm:flex-row sm:items-center sm:justify-between lg:px-6">
      <div class="flex min-w-0 flex-1 items-center gap-3">
        <div class="hidden h-9 w-1 bg-current/80 sm:block" />
        <UiSelect
          :model-value="selectedChapterId"
          :options="chapterOptions"
          :aria-label="t('editor.chapterSelector.label')"
          class="max-w-sm"
          @update:model-value="emit('update:selectedChapterId', $event)"
        />
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <UiBadge tone="muted">{{ t('editor.metrics.words', { count: characters }) }}</UiBadge>
        <UiBadge tone="muted">{{ t('editor.metrics.paragraphs', { count: paragraphs }) }}</UiBadge>
        <UiBadge :tone="dirty ? 'warning' : 'success'">
          {{ dirty ? t('editor.workspace.unsaved') : t('editor.workspace.savedLocally') }}
        </UiBadge>
        <UiButton size="sm" variant="outline" @click="emit('assistant')">
          <PanelRightOpen class="h-4 w-4" />
          {{ t('editor.actions.openAssistant') }}
        </UiButton>
        <UiButton size="icon" variant="ghost" :icon-label="fullscreen ? t('editor.actions.exitFullscreen') : t('editor.actions.fullscreen')" @click="emit('update:fullscreen', !fullscreen)">
          <Minimize2 v-if="fullscreen" class="h-4 w-4" />
          <Maximize2 v-else class="h-4 w-4" />
        </UiButton>
        <UiButton size="sm" :loading="saving" :disabled="!dirty" @click="emit('save')">
          <Save class="h-4 w-4" />
          {{ t('editor.actions.createVersion') }}
        </UiButton>
      </div>
    </div>

    <div class="mx-auto flex min-h-[calc(100dvh-15rem)] w-full max-w-[52rem] flex-col px-5 py-8 sm:px-10 lg:px-14 lg:py-12">
      <label class="block">
        <span class="sr-only">{{ t('editor.fields.title') }}</span>
        <input
          :value="title"
          class="w-full border-0 bg-transparent font-serif text-3xl font-semibold tracking-tight text-current outline-none placeholder:text-current/35 sm:text-4xl"
          :placeholder="chapter.title || t('editor.fields.titlePlaceholder')"
          @input="emit('update:title', ($event.target as HTMLInputElement).value)"
        >
      </label>
      <div class="mt-5 flex items-center gap-3 text-xs uppercase tracking-[0.2em] text-current/55">
        <span>{{ t('editor.workspace.chapterNumber', { number: chapter.number }) }}</span>
        <span class="h-px flex-1 bg-current/20" />
      </div>
      <textarea
        ref="textarea"
        :value="content"
        data-testid="chapter-content"
        class="mt-8 min-h-[62vh] w-full flex-1 resize-none border-0 bg-transparent font-serif text-[1.08rem] leading-[2.05] text-current outline-none placeholder:text-current/35 sm:text-[1.15rem]"
        :placeholder="t('editor.fields.contentPlaceholder')"
        @input="emit('update:content', ($event.target as HTMLTextAreaElement).value); emitSelection()"
        @select="emitSelection"
        @keyup="emitSelection"
        @click="emitSelection"
      />
    </div>
  </section>
</template>
