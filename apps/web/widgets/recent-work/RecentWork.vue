<script setup lang="ts">
import { ArrowRight, BookOpenText, FilePlus2, LibraryBig, Plus } from '@lucide/vue'
import UiBadge from '~/components/ui/Badge.vue'
import UiButton from '~/components/ui/Button.vue'
import UiEmptyState from '~/components/ui/EmptyState.vue'
import type { ProjectSummary } from '~/lib/types'
import { projectChapterCount } from '~/features/project-library/project-library'

const props = defineProps<{
  projects: ProjectSummary[]
}>()

const emit = defineEmits<{
  open: [project: ProjectSummary]
}>()

const { t } = useI18n()
const leadProject = computed(() => props.projects[0] || null)
const otherProjects = computed(() => props.projects.slice(1, 4))

function nextAction(project: ProjectSummary) {
  if (project.bible_status !== 'ready') {
    return {
      label: t('recentWork.actions.completeStoryBible'),
      description: t('recentWork.actions.completeStoryBibleDescription'),
      to: `/projects/${project.id}?section=story`,
      icon: LibraryBig
    }
  }
  const count = projectChapterCount(project)
  if (count === 0) {
    return {
      label: t('recentWork.actions.createFirstChapter'),
      description: t('recentWork.actions.createFirstChapterDescription'),
      to: `/projects/${project.id}?createChapter=1`,
      icon: FilePlus2
    }
  }
  return {
    label: t('recentWork.actions.continueProject'),
    description: t('recentWork.actions.continueProjectDescription'),
    to: `/projects/${project.id}`,
    icon: BookOpenText
  }
}
</script>

<template>
  <section aria-labelledby="recent-work-heading" class="border-y border-border">
    <div class="grid lg:grid-cols-[minmax(0,1.45fr)_minmax(18rem,0.55fr)]">
      <div class="min-w-0 px-0 py-6 lg:border-r lg:border-border lg:pr-8">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t('recentWork.eyebrow') }}</p>
            <h2 id="recent-work-heading" class="mt-2 text-2xl font-semibold tracking-tight">{{ t('recentWork.title') }}</h2>
          </div>
          <UiButton variant="ghost" size="sm" to="/projects">{{ t('actions.viewAll') }}<ArrowRight class="h-4 w-4" /></UiButton>
        </div>

        <div v-if="leadProject" class="mt-8">
          <div class="flex flex-wrap gap-2">
            <UiBadge :tone="leadProject.bible_status === 'ready' ? 'success' : leadProject.bible_status === 'draft' ? 'warning' : 'muted'">
              {{ t(`status.projectBible.${leadProject.bible_status}`) }}
            </UiBadge>
            <UiBadge v-if="projectChapterCount(leadProject) !== null" tone="neutral">
              {{ t('projects.chapterCountValue', { count: projectChapterCount(leadProject) }) }}
            </UiBadge>
          </div>
          <button type="button" class="focus-ring mt-4 block w-full text-left" @click="emit('open', leadProject)">
            <span class="block break-words text-3xl font-semibold tracking-[-0.03em] sm:text-4xl">{{ leadProject.title }}</span>
            <span class="mt-3 block max-w-3xl text-base leading-7 text-muted-foreground">{{ leadProject.logline }}</span>
          </button>
          <div class="mt-7 border-l-4 border-foreground bg-surface-muted p-5">
            <p class="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">{{ t('recentWork.nextAction') }}</p>
            <div class="mt-3 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div class="min-w-0">
                <p class="font-semibold">{{ nextAction(leadProject).label }}</p>
                <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ nextAction(leadProject).description }}</p>
              </div>
              <UiButton :to="nextAction(leadProject).to" class="w-full sm:w-auto">
                <component :is="nextAction(leadProject).icon" class="h-4 w-4" />
                {{ nextAction(leadProject).label }}
              </UiButton>
            </div>
          </div>
        </div>

        <UiEmptyState v-else class="mt-6" :title="t('recentWork.emptyTitle')" :description="t('recentWork.emptyDescription')">
          <template #icon><div class="flex h-10 w-10 items-center justify-center border border-border bg-surface"><Plus class="h-5 w-5" /></div></template>
          <template #actions><UiButton to="/projects/new">{{ t('actions.createProject') }}</UiButton></template>
        </UiEmptyState>
      </div>

      <aside class="min-w-0 py-6 lg:pl-8">
        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t('recentWork.moreRecent') }}</p>
        <ol v-if="otherProjects.length" class="mt-4 divide-y divide-border">
          <li v-for="project in otherProjects" :key="project.id" class="py-4 first:pt-0">
            <button type="button" class="focus-ring group block w-full text-left" @click="emit('open', project)">
              <span class="flex items-center justify-between gap-3">
                <span class="min-w-0 break-words font-semibold">{{ project.title }}</span>
                <ArrowRight class="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-1" />
              </span>
              <span class="mt-2 line-clamp-2 block text-sm leading-6 text-muted-foreground">{{ project.logline }}</span>
            </button>
          </li>
        </ol>
        <p v-else class="mt-4 text-sm leading-6 text-muted-foreground">{{ t('recentWork.noMoreRecent') }}</p>
      </aside>
    </div>
  </section>
</template>
