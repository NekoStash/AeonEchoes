<script setup lang="ts">
import { GitFork, ListTree, Sparkles, UsersRound } from '@lucide/vue'
import type { Chapter } from '~/entities/chapter'
import type { StoryBible } from '~/entities/story-bible'

const { t } = useI18n()
defineProps<{
  bible: StoryBible
  chapters: Chapter[]
}>()
</script>

<template>
  <section class="border-b-2 border-foreground pb-8" :aria-label="t('projectOverview.summary.title')">
    <div class="grid gap-8 xl:grid-cols-[minmax(0,1.4fr)_minmax(18rem,.6fr)]">
      <div>
        <p class="text-xs font-bold uppercase tracking-[0.24em] text-muted-foreground">{{ t('projectOverview.summary.eyebrow') }}</p>
        <h1 class="mt-3 max-w-4xl font-serif text-4xl font-semibold leading-tight tracking-tight sm:text-6xl">{{ bible.title || t('nav.project') }}</h1>
        <p class="mt-5 max-w-3xl text-base leading-8 text-muted-foreground">{{ bible.premise || t('common.emptyValue') }}</p>
        <div class="mt-6 flex flex-wrap gap-2">
          <span v-for="theme in bible.themes" :key="theme" class="border border-foreground px-3 py-1 text-xs font-bold uppercase tracking-[0.14em]">{{ theme }}</span>
        </div>
      </div>

      <dl class="grid grid-cols-2 border-l-4 border-foreground bg-muted/45">
        <div class="border-b border-r border-border p-5">
          <dt class="flex items-center gap-2 text-xs font-bold uppercase tracking-[0.16em] text-muted-foreground"><Sparkles class="h-4 w-4" aria-hidden="true" />{{ t('projectOverview.summary.foreshadows') }}</dt>
          <dd class="mt-3 font-serif text-4xl">{{ bible.foreshadows.length }}</dd>
        </div>
        <div class="border-b border-border p-5">
          <dt class="flex items-center gap-2 text-xs font-bold uppercase tracking-[0.16em] text-muted-foreground"><ListTree class="h-4 w-4" aria-hidden="true" />{{ t('projectOverview.summary.realChapters') }}</dt>
          <dd class="mt-3 font-serif text-4xl">{{ chapters.length }}</dd>
        </div>
        <div class="border-r border-border p-5">
          <dt class="flex items-center gap-2 text-xs font-bold uppercase tracking-[0.16em] text-muted-foreground"><UsersRound class="h-4 w-4" aria-hidden="true" />{{ t('projectOverview.characters') }}</dt>
          <dd class="mt-3 font-serif text-4xl">{{ bible.characters.length }}</dd>
        </div>
        <div class="p-5">
          <dt class="flex items-center gap-2 text-xs font-bold uppercase tracking-[0.16em] text-muted-foreground"><GitFork class="h-4 w-4" aria-hidden="true" />{{ t('projectOverview.worldRules') }}</dt>
          <dd class="mt-3 font-serif text-4xl">{{ bible.world_rules.length }}</dd>
        </div>
      </dl>
    </div>
  </section>
</template>
