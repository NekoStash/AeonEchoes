<script setup lang="ts">
import { ArrowRight, Plus, RefreshCw } from '@lucide/vue'
import { storeToRefs } from 'pinia'
import UiButton from '~/components/ui/Button.vue'
import type { ProjectSummary } from '~/lib/types'
import ProjectLibrary from '~/features/project-library/ProjectLibrary.vue'
import { useProjectStore } from '~/entities/project'

const { t } = useI18n()
const workspace = useWorkspaceStore()
const projectStore = useProjectStore()
const { openedProjects } = storeToRefs(workspace)
const { items: projects, listRequest } = storeToRefs(projectStore)

const projectLoadError = computed(() => listRequest.value.error?.message || '')

onMounted(() => {
  void loadProjects()
})

async function loadProjects() {
  try {
    await projectStore.load()
    workspace.syncOpenedProjects(projectStore.items)
  } catch (error) {
    console.error('[AeonEchoes Project Library] Failed to load projects.', error)
  }
}

async function openProject(project: ProjectSummary) {
  workspace.openProject(project)
  await navigateTo(`/projects/${project.id}`)
}
</script>

<template>
  <div class="mx-auto w-full max-w-[var(--layout-width-page)] px-[var(--layout-gutter)] py-6 sm:py-10">
    <header class="grid gap-7 pb-8 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
      <div>
        <p class="text-xs font-semibold uppercase tracking-[0.24em] text-muted-foreground">{{ t('projectLibrary.pageEyebrow') }}</p>
        <h1 class="mt-3 text-4xl font-semibold tracking-[-0.045em] sm:text-5xl">{{ t('projectLibrary.pageTitle') }}</h1>
        <p class="mt-4 max-w-2xl text-base leading-7 text-muted-foreground">{{ t('projectLibrary.pageDescription') }}</p>
      </div>
      <div class="flex flex-col gap-3 sm:flex-row">
        <UiButton variant="outline" :loading="listRequest.loading" :loading-label="t('actions.refresh')" @click="loadProjects">
          <RefreshCw class="h-4 w-4" />{{ t('actions.refresh') }}
        </UiButton>
        <UiButton to="/projects/new" class="justify-between"><span class="flex items-center gap-2"><Plus class="h-4 w-4" />{{ t('actions.createProject') }}</span><ArrowRight class="h-4 w-4" /></UiButton>
      </div>
    </header>

    <ProjectLibrary
      :projects="projects"
      :opened-project-ids="openedProjects.map((project) => project.id)"
      :loading="listRequest.loading"
      :error="projectLoadError"
      @open="openProject"
      @retry="loadProjects"
    />
  </div>
</template>
