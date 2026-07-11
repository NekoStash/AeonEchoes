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
const componentId = useId()
const workspaceHeadingId = `${componentId}-workspace-heading`
const titleInputId = `${componentId}-title-input`
const titleLabelId = `${componentId}-title-label`
const chapterMetaId = `${componentId}-chapter-meta`
const contentInputId = `${componentId}-content-input`
const contentLabelId = `${componentId}-content-label`
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
    :aria-labelledby="workspaceHeadingId"
    :class="[
      'writing-workspace relative min-w-0 border border-border bg-surface-muted text-surface-foreground',
      fullscreen
        ? 'fixed inset-0 z-50 min-h-[100dvh] overflow-y-auto border-0'
        : 'overflow-hidden'
    ]"
  >
    <header class="flex min-h-12 flex-col gap-2 border-b border-current/15 bg-surface px-3 py-2 sm:flex-row sm:items-center sm:justify-between sm:px-4 lg:px-5">
      <h2 :id="workspaceHeadingId" class="sr-only">{{ t('editor.eyebrow') }}</h2>
      <div class="flex min-w-0 flex-1 items-center gap-2">
        <span class="hidden h-8 w-1 bg-current/80 sm:block" aria-hidden="true" />
        <UiSelect
          :model-value="selectedChapterId"
          :options="chapterOptions"
          :aria-label="t('editor.chapterSelector.label')"
          class="max-w-sm"
          @update:model-value="emit('update:selectedChapterId', $event)"
        />
      </div>
      <div class="flex flex-wrap items-center gap-1.5">
        <UiBadge tone="muted">{{ t('editor.metrics.writing', { characters, paragraphs }) }}</UiBadge>
        <UiBadge :tone="dirty ? 'warning' : 'success'">
          {{ dirty ? t('editor.workspace.unsaved') : t('editor.workspace.savedLocally') }}
        </UiBadge>
        <UiButton :class="fullscreen ? '' : 'xl:hidden'" size="sm" variant="outline" @click="emit('assistant')">
          <PanelRightOpen class="h-4 w-4" aria-hidden="true" />
          {{ t('editor.actions.openAssistant') }}
        </UiButton>
        <UiButton size="icon" variant="ghost" :icon-label="fullscreen ? t('editor.actions.exitFullscreen') : t('editor.actions.fullscreen')" @click="emit('update:fullscreen', !fullscreen)">
          <Minimize2 v-if="fullscreen" class="h-4 w-4" aria-hidden="true" />
          <Maximize2 v-else class="h-4 w-4" aria-hidden="true" />
        </UiButton>
        <UiButton size="sm" :loading="saving" :disabled="!dirty" @click="emit('save')">
          <Save class="h-4 w-4" aria-hidden="true" />
          {{ t('editor.actions.createVersion') }}
        </UiButton>
      </div>
    </header>

    <div
      data-testid="writing-surface"
      class="bg-surface-muted px-3 py-3 sm:px-4 sm:py-4 lg:px-5"
    >
      <article
        data-testid="writing-paper"
        :aria-labelledby="titleLabelId"
        class="mx-auto flex w-full max-w-[48rem] flex-col gap-3 sm:gap-4"
      >
        <header data-testid="chapter-title-surface" class="border border-border bg-surface-raised p-4 sm:p-5 lg:p-6">
          <label
            :id="titleLabelId"
            :for="titleInputId"
            class="mb-2 block text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground"
          >
            {{ t('editor.fields.title') }}
          </label>
          <input
            :id="titleInputId"
            :value="title"
            :aria-describedby="chapterMetaId"
            class="w-full rounded-none border-0 bg-transparent font-serif text-3xl font-semibold tracking-tight text-current outline-none placeholder:text-current/35 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring sm:text-4xl"
            :placeholder="chapter.title || t('editor.fields.titlePlaceholder')"
            @input="emit('update:title', ($event.target as HTMLInputElement).value)"
          >
          <div
            :id="chapterMetaId"
            data-testid="chapter-meta"
            class="mt-3 flex items-center gap-3 text-xs uppercase tracking-[0.2em] text-muted-foreground"
          >
            <span>{{ t('editor.workspace.chapterNumber', { number: chapter.number }) }}</span>
            <span class="h-px flex-1 bg-current/25" aria-hidden="true" />
          </div>
        </header>

        <section
          data-testid="chapter-content-surface"
          :aria-labelledby="contentLabelId"
          class="border border-border bg-surface-elevated p-4 sm:p-5 lg:p-6"
        >
          <label
            :id="contentLabelId"
            :for="contentInputId"
            class="block text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground"
          >
            {{ t('editor.content') }}
          </label>
          <textarea
            :id="contentInputId"
            ref="textarea"
            :value="content"
            data-testid="chapter-content"
            class="mt-3 min-h-[clamp(18rem,48dvh,36rem)] w-full resize-none rounded-none border-0 bg-transparent font-serif text-[1.08rem] [line-height:1.9] text-current outline-none placeholder:text-current/35 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring sm:min-h-[clamp(24rem,52dvh,40rem)] sm:text-[1.15rem] lg:min-h-[clamp(28rem,58dvh,42rem)]"
            :placeholder="t('editor.fields.contentPlaceholder')"
            @input="emit('update:content', ($event.target as HTMLTextAreaElement).value); emitSelection()"
            @select="emitSelection"
            @keyup="emitSelection"
            @click="emitSelection"
          />
        </section>
      </article>
    </div>
  </section>
</template>
