<script setup lang="ts">
import { ArrowRight, Plus } from '@lucide/vue'
import { storeToRefs } from 'pinia'
import UiButton from '~/components/ui/Button.vue'
import UiInlineNotice from '~/components/ui/InlineNotice.vue'
import type { ProjectSummary } from '~/lib/types'
import { useProjectStore } from '~/entities/project'
import RecentWork from '~/widgets/recent-work/RecentWork.vue'

const { t } = useI18n()
const workspace = useWorkspaceStore()
const projectStore = useProjectStore()
const { openedProjects } = storeToRefs(workspace)
const { items: projects, listRequest } = storeToRefs(projectStore)

const recentProjects = computed(() => {
  const seen = new Set<string>()
  return [...openedProjects.value, ...projects.value]
    .filter((project) => {
      if (seen.has(project.id)) return false
      seen.add(project.id)
      return true
    })
    .slice(0, 4)
})

onMounted(() => {
  void loadProjects()
})

async function loadProjects() {
  try {
    await projectStore.load()
    workspace.syncOpenedProjects(projectStore.items)
  } catch (error) {
    console.error('[AeonEchoes Home] Failed to load projects.', error)
  }
}

async function openProject(project: ProjectSummary) {
  workspace.openProject(project)
  await navigateTo(`/projects/${project.id}`)
}
</script>

<template>
  <div class="mx-auto w-full max-w-[var(--layout-width-page)] px-[var(--layout-gutter)] py-6 sm:py-10">
    <header class="grid gap-8 pb-8 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.42fr)] lg:items-end">
      <div class="min-w-0">
        <p class="text-xs font-semibold uppercase tracking-[0.24em] text-muted-foreground">{{ t('home.eyebrow') }}</p>
        <h1 class="mt-3 max-w-4xl text-4xl font-semibold tracking-[-0.045em] sm:text-5xl lg:text-6xl">{{ t('home.title') }}</h1>
        <p class="mt-5 max-w-2xl text-base leading-7 text-muted-foreground">{{ t('home.description') }}</p>
      </div>
      <div class="border-l-4 border-foreground bg-surface-muted p-5">
        <p class="text-sm font-semibold">{{ t('home.newProjectTitle') }}</p>
        <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('home.newProjectDescription') }}</p>
        <UiButton to="/projects/new" class="mt-5 w-full justify-between">
          <span class="flex items-center gap-2"><Plus class="h-4 w-4" />{{ t('actions.createProject') }}</span>
          <ArrowRight class="h-4 w-4" />
        </UiButton>
      </div>
    </header>

    <UiInlineNotice v-if="listRequest.error" tone="danger" :title="t('home.loadErrorTitle')" :description="listRequest.error.message" class="mb-6">
      <template #actions><UiButton variant="outline" size="sm" :loading="listRequest.loading" @click="loadProjects">{{ t('common.retry') }}</UiButton></template>
    </UiInlineNotice>

    <RecentWork :projects="recentProjects" @open="openProject" />
  </div>
</template>
