<script setup lang="ts">
import { ArrowRight, FileText } from '@lucide/vue'
import type { Chapter } from '~/entities/chapter'

const { t } = useI18n()
defineProps<{
  projectId: string
  chapters: Chapter[]
  loading?: boolean
}>()
</script>

<template>
  <section class="border-t-2 border-foreground pt-8" :aria-label="t('projectOverview.realChapters.title')">
    <div>
      <p class="text-xs font-bold uppercase tracking-[0.22em] text-muted-foreground">{{ t('projectOverview.realChapters.eyebrow') }}</p>
      <h2 class="mt-2 font-serif text-3xl font-semibold tracking-tight">{{ t('projectOverview.realChapters.title') }}</h2>
      <p class="mt-3 max-w-2xl text-sm leading-7 text-muted-foreground">{{ t('projectOverview.realChapters.description') }}</p>
    </div>

    <div v-if="loading" class="mt-6 space-y-3" aria-busy="true">
      <div v-for="index in 3" :key="index" class="h-20 animate-pulse border-y border-border bg-muted/40" />
    </div>

    <div v-else-if="chapters.length === 0" class="mt-6 border-y border-border py-10 text-center">
      <FileText class="mx-auto h-8 w-8 text-muted-foreground" aria-hidden="true" />
      <p class="mt-3 font-serif text-xl font-semibold">{{ t('projectOverview.realChapters.emptyTitle') }}</p>
      <p class="mx-auto mt-2 max-w-lg text-sm leading-6 text-muted-foreground">{{ t('projectOverview.realChapters.emptyDescription') }}</p>
    </div>

    <ol v-else class="relative mt-6 border-y border-border before:absolute before:bottom-0 before:left-[1.65rem] before:top-0 before:w-px before:bg-border">
      <li v-for="chapter in chapters" :key="chapter.id" class="relative grid gap-4 border-b border-border py-5 pl-14 last:border-b-0 sm:grid-cols-[5rem_minmax(0,1fr)_auto] sm:items-center">
        <span class="absolute left-5 top-8 h-3 w-3 border-2 border-foreground bg-background" aria-hidden="true" />
        <p class="font-serif text-2xl text-muted-foreground">{{ String(chapter.number).padStart(2, '0') }}</p>
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <h3 class="truncate font-semibold">{{ chapter.title }}</h3>
            <span class="border border-border px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.14em] text-muted-foreground">{{ t(`status.chapter.${chapter.status}`) }}</span>
          </div>
          <p class="mt-1 line-clamp-2 text-sm leading-6 text-muted-foreground">{{ chapter.summary || t('common.emptyValue') }}</p>
        </div>
        <UiButton variant="outline" :to="`/projects/${projectId}/editor?chapter=${encodeURIComponent(chapter.id)}`">
          {{ t('projectOverview.openChapter') }}
          <ArrowRight class="h-4 w-4" aria-hidden="true" />
        </UiButton>
      </li>
    </ol>
  </section>
</template>
