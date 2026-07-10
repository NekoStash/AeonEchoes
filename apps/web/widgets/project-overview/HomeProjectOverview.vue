<script setup lang="ts">
import { ArrowRight, BookOpenText, FilePlus2, LibraryBig, Plus } from '@lucide/vue'
import UiButton from '~/components/ui/Button.vue'
import type { ProjectSummary } from '~/lib/types'
import { projectChapterCount } from '~/features/project-library/project-library'

const props = defineProps<{
  projects: ProjectSummary[]
}>()

const emit = defineEmits<{
  open: [project: ProjectSummary]
}>()

const { t } = useI18n()
const readyCount = computed(() => props.projects.filter((project) => project.bible_status === 'ready').length)
const knownChapterCount = computed(() => props.projects.reduce((total, project) => total + (projectChapterCount(project) ?? 0), 0))
const unknownChapterCount = computed(() => props.projects.filter((project) => projectChapterCount(project) === null).length)
const needsStoryBible = computed(() => props.projects.filter((project) => project.bible_status !== 'ready').slice(0, 3))
const zeroChapterProjects = computed(() => props.projects.filter((project) => projectChapterCount(project) === 0).slice(0, 3))
</script>

<template>
  <section aria-labelledby="project-overview-heading" class="py-8">
    <div class="grid gap-8 lg:grid-cols-[minmax(0,0.7fr)_minmax(0,1.3fr)]">
      <div>
        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t('projectOverviewWidget.eyebrow') }}</p>
        <h2 id="project-overview-heading" class="mt-2 text-2xl font-semibold tracking-tight">{{ t('projectOverviewWidget.title') }}</h2>
        <p class="mt-3 max-w-xl text-sm leading-6 text-muted-foreground">{{ t('projectOverviewWidget.description') }}</p>

        <dl class="mt-7 divide-y divide-border border-y border-border">
          <div class="flex items-baseline justify-between gap-4 py-4">
            <dt class="text-sm text-muted-foreground">{{ t('projectOverviewWidget.totalProjects') }}</dt>
            <dd class="font-mono text-2xl font-semibold">{{ projects.length }}</dd>
          </div>
          <div class="flex items-baseline justify-between gap-4 py-4">
            <dt class="text-sm text-muted-foreground">{{ t('projectOverviewWidget.readyStoryBibles') }}</dt>
            <dd class="font-mono text-2xl font-semibold">{{ readyCount }}</dd>
          </div>
          <div class="flex items-baseline justify-between gap-4 py-4">
            <dt class="text-sm text-muted-foreground">{{ t('projectOverviewWidget.knownChapters') }}</dt>
            <dd class="text-right">
              <span class="font-mono text-2xl font-semibold">{{ knownChapterCount }}</span>
              <span v-if="unknownChapterCount" class="ml-2 block text-xs text-muted-foreground sm:inline">{{ t('projectOverviewWidget.unknownChapterProjects', { count: unknownChapterCount }) }}</span>
            </dd>
          </div>
        </dl>
      </div>

      <div class="grid gap-6 sm:grid-cols-2">
        <article class="border-t-4 border-foreground bg-surface-muted p-5">
          <LibraryBig class="h-5 w-5" aria-hidden="true" />
          <h3 class="mt-4 text-lg font-semibold">{{ t('projectOverviewWidget.setupQueue') }}</h3>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('projectOverviewWidget.setupQueueDescription') }}</p>
          <ol v-if="needsStoryBible.length" class="mt-5 divide-y divide-border">
            <li v-for="project in needsStoryBible" :key="project.id" class="py-3 first:pt-0">
              <button type="button" class="focus-ring flex w-full items-center justify-between gap-3 text-left" @click="emit('open', project)">
                <span class="min-w-0 truncate text-sm font-semibold">{{ project.title }}</span>
                <ArrowRight class="h-4 w-4 shrink-0" />
              </button>
            </li>
          </ol>
          <p v-else class="mt-5 text-sm text-muted-foreground">{{ t('projectOverviewWidget.setupQueueEmpty') }}</p>
        </article>

        <article class="border-t-4 border-foreground bg-surface-muted p-5">
          <FilePlus2 class="h-5 w-5" aria-hidden="true" />
          <h3 class="mt-4 text-lg font-semibold">{{ t('projectOverviewWidget.firstChapterQueue') }}</h3>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('projectOverviewWidget.firstChapterQueueDescription') }}</p>
          <ol v-if="zeroChapterProjects.length" class="mt-5 divide-y divide-border">
            <li v-for="project in zeroChapterProjects" :key="project.id" class="py-3 first:pt-0">
              <UiButton variant="ghost" class="h-auto w-full justify-between px-0 py-0" :to="`/projects/${project.id}?createChapter=1`">
                <span class="min-w-0 truncate text-sm">{{ project.title }}</span>
                <Plus class="h-4 w-4 shrink-0" />
              </UiButton>
            </li>
          </ol>
          <p v-else class="mt-5 text-sm text-muted-foreground">{{ t('projectOverviewWidget.firstChapterQueueEmpty') }}</p>
        </article>
      </div>
    </div>

    <div class="mt-8 flex flex-col gap-3 border-t border-border pt-5 sm:flex-row">
      <UiButton to="/projects/new" class="w-full sm:w-auto"><Plus class="h-4 w-4" />{{ t('actions.createProject') }}</UiButton>
      <UiButton variant="outline" to="/projects" class="w-full sm:w-auto"><BookOpenText class="h-4 w-4" />{{ t('projectOverviewWidget.openLibrary') }}</UiButton>
    </div>
  </section>
</template>
