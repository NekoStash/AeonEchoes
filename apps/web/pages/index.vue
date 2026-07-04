<script setup lang="ts">
import { ArrowRight, BookOpen, FolderKanban, PenLine, PlusCircle, RefreshCw } from '@lucide/vue'
import { storeToRefs } from 'pinia'
import { formatDateTime } from '~/lib/utils'

const { t } = useI18n()
const workspace = useWorkspaceStore()
const { health, projects, openedProjects, providers, errors, loading, indexJobs } = storeToRefs(workspace)
const activeDashboardTab = ref('continue')

onMounted(async () => {
  await workspace.loadDashboard()
  await workspace.loadIndexJobs()
})

const projectSummary = computed(() => {
  const readyProjects = projects.value.filter((item) => item.bible_status === 'ready').length
  return [
    {
      label: t('dashboard.projects'),
      value: projects.value.length,
      hint: t('dashboard.metricHints.readyProjects', { count: readyProjects }),
      icon: BookOpen
    },
    {
      label: t('dashboard.openedProjects'),
      value: openedProjects.value.length,
      hint: t('dashboard.metricHints.openedProjectsHint', { count: projects.value.length }),
      icon: FolderKanban
    },
    {
      label: t('dashboard.systemStatus'),
      value: health.value?.ok ? t('status.online') : t('status.offline'),
      hint: health.value?.lastHeartbeat ? formatDateTime(health.value.lastHeartbeat) : t('common.emptyValue'),
      icon: RefreshCw
    },
    {
      label: t('dashboard.providers'),
      value: providers.value.filter((item) => item.enabled).length,
      hint: t('dashboard.metricHints.enabledProviders', { count: providers.value.length }),
      icon: PenLine
    }
  ]
})

const dashboardTabs = computed(() => [
  { label: t('dashboard.tabs.continue'), value: 'continue', badge: String(continueProjects.value.length) },
  { label: t('dashboard.tabs.flow'), value: 'flow' },
  { label: t('dashboard.tabs.system'), value: 'system', badge: String(indexJobs.value.filter((job) => job.status === 'pending' || job.status === 'running').length) }
])
const continueProjects = computed(() => (openedProjects.value.length > 0 ? openedProjects.value : projects.value).slice(0, 4))
const newestProject = computed(() => projects.value[0] || null)
const activeIndexJobs = computed(() => indexJobs.value.filter((job) => job.status === 'pending' || job.status === 'running'))
</script>

<template>
  <div class="space-y-6">
    <SectionHeader :title="t('dashboard.title')" :description="t('dashboard.description')">
      <template #actions>
        <UiButton variant="outline" :disabled="loading.dashboard" @click="workspace.loadDashboard()">
          <RefreshCw :class="['h-4 w-4', loading.dashboard && 'animate-spin']" />
          {{ t('actions.refresh') }}
        </UiButton>
        <UiButton to="/projects/new">
          <PlusCircle class="h-4 w-4" />
          {{ t('nav.newProject') }}
        </UiButton>
      </template>
    </SectionHeader>

    <StatusAlert :errors="errors" />

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <UiCard v-for="metric in projectSummary" :key="metric.label" class="p-4 sm:p-5">
        <div class="flex min-w-0 items-center justify-between gap-3">
          <div class="min-w-0">
            <p class="truncate text-sm text-muted-foreground">{{ metric.label }}</p>
            <p class="mt-2 truncate text-2xl font-semibold">{{ metric.value }}</p>
            <p class="mt-1 truncate text-xs text-muted-foreground" :title="String(metric.hint)">{{ metric.hint }}</p>
          </div>
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <component :is="metric.icon" class="h-5 w-5" />
          </div>
        </div>
      </UiCard>
    </div>

    <UiTabs v-model="activeDashboardTab" :tabs="dashboardTabs" />

    <UiCard v-if="activeDashboardTab === 'continue'" class="p-4 sm:p-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0">
          <h2 class="truncate text-lg font-semibold">{{ t('dashboard.recentProjects') }}</h2>
          <p class="mt-1 text-sm text-muted-foreground">{{ t('dashboard.recentProjectsDescription') }}</p>
        </div>
        <UiButton variant="outline" size="sm" to="/projects">{{ t('actions.viewAll') }}</UiButton>
      </div>
      <div v-if="continueProjects.length === 0" class="mt-5 rounded-xl border border-border bg-muted/35 p-4 text-sm text-muted-foreground">
        {{ t('dashboard.emptyRecentProjects') }}
      </div>
      <div v-else class="mt-5 grid gap-3 lg:grid-cols-2">
        <NuxtLink v-for="project in continueProjects" :key="project.id" :to="`/projects/${project.id}`" class="block min-w-0 rounded-xl border border-border p-4 transition-colors hover:bg-muted/60">
          <div class="flex min-w-0 items-center justify-between gap-3">
            <p class="truncate font-medium">{{ project.title }}</p>
            <UiBadge :variant="openedProjects.some((item) => item.id === project.id) ? 'success' : 'muted'">
              {{ openedProjects.some((item) => item.id === project.id) ? t('projects.opened') : t('projects.notOpened') }}
            </UiBadge>
          </div>
          <p class="mt-2 line-clamp-2 text-sm text-muted-foreground">{{ project.logline }}</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <UiBadge v-for="tag in project.tags.slice(0, 3)" :key="tag" variant="muted">{{ tag }}</UiBadge>
          </div>
        </NuxtLink>
      </div>
    </UiCard>

    <UiCard v-else-if="activeDashboardTab === 'flow'" class="p-4 sm:p-6">
      <div class="flex min-w-0 items-start justify-between gap-3">
        <div>
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('dashboard.primaryFlowEyebrow') }}</p>
          <h2 class="mt-2 text-lg font-semibold">{{ t('dashboard.primaryFlowTitle') }}</h2>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('dashboard.primaryFlowDescription') }}</p>
        </div>
        <ArrowRight class="mt-1 h-5 w-5 text-muted-foreground" />
      </div>

      <div class="mt-5 grid gap-4 md:grid-cols-3">
        <NuxtLink to="/projects/new" class="rounded-2xl border border-border p-4 transition-colors hover:bg-muted/60">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">1</p>
          <p class="mt-2 font-medium">{{ t('dashboard.flow.create.title') }}</p>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('dashboard.flow.create.description') }}</p>
        </NuxtLink>
        <NuxtLink :to="newestProject ? `/projects/${newestProject.id}` : '/projects'" class="rounded-2xl border border-border p-4 transition-colors hover:bg-muted/60">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">2</p>
          <p class="mt-2 font-medium">{{ t('dashboard.flow.prepare.title') }}</p>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('dashboard.flow.prepare.description') }}</p>
        </NuxtLink>
        <NuxtLink :to="continueProjects[0] ? `/projects/${continueProjects[0].id}/editor` : '/projects'" class="rounded-2xl border border-border p-4 transition-colors hover:bg-muted/60">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">3</p>
          <p class="mt-2 font-medium">{{ t('dashboard.flow.write.title') }}</p>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('dashboard.flow.write.description') }}</p>
        </NuxtLink>
      </div>
    </UiCard>

    <UiCard v-else class="p-4 sm:p-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 class="text-lg font-semibold">{{ t('dashboard.systemStatus') }}</h2>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('dashboard.systemStatusDescription') }}</p>
        </div>
        <UiButton variant="outline" size="sm" :disabled="loading.dashboard" @click="workspace.loadDashboard()">
          <RefreshCw :class="['h-4 w-4', loading.dashboard && 'animate-spin']" />
          {{ t('actions.refresh') }}
        </UiButton>
      </div>
      <div class="mt-5 grid gap-4 md:grid-cols-3">
        <div class="rounded-2xl border border-border bg-muted/30 p-4">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('dashboard.systemHealth') }}</p>
          <p class="mt-2 text-lg font-semibold">{{ health?.ok ? t('status.online') : t('status.offline') }}</p>
          <p class="mt-1 text-sm text-muted-foreground">{{ health?.lastHeartbeat ? formatDateTime(health.lastHeartbeat) : t('common.emptyValue') }}</p>
        </div>
        <div class="rounded-2xl border border-border bg-muted/30 p-4">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('dashboard.providers') }}</p>
          <p class="mt-2 text-lg font-semibold">{{ providers.filter((item) => item.enabled).length }} / {{ providers.length }}</p>
          <p class="mt-1 text-sm text-muted-foreground">{{ t('dashboard.metricHints.enabledProviders', { count: providers.length }) }}</p>
        </div>
        <div class="rounded-2xl border border-border bg-muted/30 p-4">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('models.tabs.indexJobs') }}</p>
          <p class="mt-2 text-lg font-semibold">{{ activeIndexJobs.length }}</p>
          <p class="mt-1 text-sm text-muted-foreground">{{ t('dashboard.activeIndexJobs') }}</p>
        </div>
      </div>
      <AppTaskBoard class="mt-5" :jobs="indexJobs" />
    </UiCard>
  </div>
</template>
