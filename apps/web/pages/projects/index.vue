<script setup lang="ts">
import { ArrowRight, BookOpen, FolderPlus, RefreshCw } from '@lucide/vue'
import { storeToRefs } from 'pinia'
import type { ProjectSummary } from '~/lib/types'

const { t } = useI18n()
const workspace = useWorkspaceStore()
const { projects, errors, loading } = storeToRefs(workspace)

onMounted(() => workspace.loadDashboard())

function projectStatusLabel(status: ProjectSummary['bible_status']) {
  return t(`status.projectBible.${status}`)
}

function openProject(project: ProjectSummary) {
  workspace.openProject(project)
  return navigateTo(`/projects/${project.id}`)
}
</script>

<template>
  <div class="space-y-6">
    <SectionHeader :title="t('projects.title')" :description="t('projects.description')">
      <template #actions>
        <UiButton variant="outline" :disabled="loading.dashboard" @click="workspace.loadDashboard()">
          <RefreshCw :class="['h-4 w-4', loading.dashboard && 'animate-spin']" />
          {{ t('actions.refresh') }}
        </UiButton>
        <UiButton to="/projects/new">
          <FolderPlus class="h-4 w-4" />
          {{ t('actions.createProject') }}
        </UiButton>
      </template>
    </SectionHeader>

    <StatusAlert :errors="errors" />

    <UiCard v-if="projects.length === 0" class="p-5 text-center sm:p-8">
      <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-muted text-muted-foreground">
        <BookOpen class="h-6 w-6" />
      </div>
      <h2 class="mt-4 text-lg font-semibold">{{ t('projects.emptyTitle') }}</h2>
      <p class="mx-auto mt-2 max-w-xl text-sm leading-6 text-muted-foreground">{{ t('projects.emptyDescription') }}</p>
      <UiButton class="mt-5" to="/projects/new">{{ t('actions.createProject') }}</UiButton>
    </UiCard>

    <div v-else class="grid gap-4 lg:grid-cols-2 2xl:grid-cols-3">
      <UiCard v-for="project in projects" :key="project.id" class="flex min-w-0 flex-col p-4 sm:p-5">
        <div class="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
          <div class="min-w-0">
            <h2 class="break-words text-lg font-semibold leading-7">{{ project.title }}</h2>
            <p class="mt-2 line-clamp-3 text-sm leading-6 text-muted-foreground">{{ project.logline }}</p>
          </div>
          <UiBadge class="w-fit shrink-0" :variant="project.bible_status === 'ready' ? 'success' : project.bible_status === 'draft' ? 'gold' : 'muted'">
            {{ projectStatusLabel(project.bible_status) }}
          </UiBadge>
        </div>

        <div class="mt-5 flex flex-wrap gap-2">
          <UiBadge v-for="tag in project.tags" :key="tag" variant="muted">{{ tag }}</UiBadge>
          <span v-if="project.tags.length === 0" class="text-sm text-muted-foreground">{{ t('projects.noTags') }}</span>
        </div>

        <div class="mt-5 grid gap-3 border-t border-border pt-5 sm:grid-cols-2">
          <div>
            <p class="text-xs text-muted-foreground">{{ t('projects.chapterCount') }}</p>
            <p class="mt-1 font-medium">{{ t('projects.chapterCountValue', { count: project.chapter_count }) }}</p>
          </div>
          <div>
            <p class="text-xs text-muted-foreground">{{ t('projects.storyBibleStatus') }}</p>
            <p class="mt-1 font-medium">{{ projectStatusLabel(project.bible_status) }}</p>
          </div>
        </div>

        <div class="mt-6 flex flex-col gap-3 pt-1 sm:flex-row sm:items-center sm:justify-between">
          <UiBadge v-if="workspace.isProjectOpen(project.id)" variant="success">{{ t('projects.opened') }}</UiBadge>
          <span v-else class="text-sm text-muted-foreground">{{ t('projects.notOpened') }}</span>
          <UiButton size="sm" class="w-full sm:w-auto" @click="openProject(project)">
            {{ t('actions.open') }}
            <ArrowRight class="h-4 w-4" />
          </UiButton>
        </div>
      </UiCard>
    </div>
  </div>
</template>
