<script setup lang="ts">
import { ArrowRight, BookOpen, CheckCircle2, FilePenLine, FolderOpen, FolderPlus, GitFork, RefreshCw } from '@lucide/vue'
import { storeToRefs } from 'pinia'
import DataCardGrid from '~/components/data/DataCardGrid.vue'
import DataCollection from '~/components/data/DataCollection.vue'
import DataEmptyState from '~/components/data/EmptyState.vue'
import DataErrorState from '~/components/data/ErrorState.vue'
import FilterBar from '~/components/data/FilterBar.vue'
import DataNoResultsState from '~/components/data/NoResultsState.vue'
import DataTable from '~/components/data/DataTable.vue'
import DensityToggle from '~/components/data/DensityToggle.vue'
import SearchInput from '~/components/data/SearchInput.vue'
import SortSelect from '~/components/data/SortSelect.vue'
import ViewModeToggle from '~/components/data/ViewModeToggle.vue'
import Panel from '~/components/ds/Panel.vue'
import StatCard from '~/components/ds/StatCard.vue'
import StatusStack from '~/components/ds/StatusStack.vue'
import PageHeader from '~/components/layout/PageHeader.vue'
import PageShell from '~/components/layout/PageShell.vue'
import Toolbar from '~/components/layout/Toolbar.vue'
import type { ProjectSummary } from '~/lib/types'
import { formatDateTime } from '~/lib/utils'

type ProjectViewMode = 'table' | 'grid'
type ProjectDensity = 'compact' | 'comfortable'
type ProjectSortKey = 'updated_at:desc' | 'created_at:desc' | 'title:asc' | 'target_chapters:desc'
type ActiveBibleFilter = '' | 'with' | 'without'
type RecentFilter = '' | 'recent' | 'not_recent'
type ProjectRouteTarget = 'overview' | 'editor' | 'graph'
type ProjectRow = { id: string; project: ProjectSummary } & Record<string, unknown>

const { t } = useI18n()
const workspace = useWorkspaceStore()
const { projects, openedProjects, errors, loading } = storeToRefs(workspace)

const searchQuery = ref('')
const projectStatusFilter = ref('')
const bibleStatusFilter = ref<ProjectSummary['bible_status'] | ''>('')
const activeBibleFilter = ref<ActiveBibleFilter>('')
const recentFilter = ref<RecentFilter>('')
const sortKey = ref<ProjectSortKey>('updated_at:desc')
const viewMode = ref<ProjectViewMode>('table')
const density = ref<ProjectDensity>('comfortable')

const bibleStatusValues: ProjectSummary['bible_status'][] = ['missing', 'draft', 'ready']

onMounted(() => workspace.loadDashboard())

const openedProjectIds = computed(() => new Set(openedProjects.value.map((project) => project.id)))
const activeBibleCount = computed(() => projects.value.filter(hasActiveStoryBible).length)
const totalTargetChapters = computed(() => projects.value.reduce((total, project) => total + targetChapterCount(project), 0))
const collectionDensity = computed<'compact' | 'comfortable'>(() => density.value === 'compact' ? 'compact' : 'comfortable')
const panelPadding = computed<'sm' | 'md'>(() => density.value === 'compact' ? 'sm' : 'md')

const statusItems = computed(() => errors.value.map((error, index) => ({
  id: `${error.endpoint}:${index}:${error.message}`,
  tone: 'warning' as const,
  title: `${t('apiError.title')} · ${error.endpoint}`,
  description: error.message
})))

const projectStats = computed(() => [
  {
    key: 'total',
    label: t('projects.stats.total'),
    value: projects.value.length,
    hint: t('projects.stats.readyBibleHint', { count: projects.value.filter((project) => project.bible_status === 'ready').length }),
    tone: 'info' as const,
    icon: BookOpen
  },
  {
    key: 'activeBible',
    label: t('projects.stats.activeBible'),
    value: activeBibleCount.value,
    hint: t('projects.stats.activeBibleHint', { count: projects.value.length }),
    tone: 'success' as const,
    icon: CheckCircle2
  },
  {
    key: 'recent',
    label: t('projects.stats.recentlyOpened'),
    value: openedProjects.value.length,
    hint: t('projects.stats.openedHint', { count: visibleProjects.value.length }),
    tone: 'warning' as const,
    icon: FolderOpen
  },
  {
    key: 'chapters',
    label: t('projects.stats.targetChapters'),
    value: totalTargetChapters.value,
    hint: t('projects.stats.targetChaptersHint', { count: projects.value.length ? Math.round(totalTargetChapters.value / projects.value.length) : 0 }),
    tone: 'neutral' as const,
    icon: FilePenLine
  }
])

const projectStatusOptions = computed(() => [
  { label: t('projects.filters.allStatuses'), value: '' },
  ...uniqueTokens(projects.value.map((project) => project.status)).map((status) => ({
    label: projectStatusLabel(status),
    value: status
  }))
])

const bibleStatusOptions = computed(() => [
  { label: t('projects.filters.allBibleStatuses'), value: '' },
  ...bibleStatusValues.map((status) => ({
    label: projectBibleStatusLabel(status),
    value: status
  }))
])

const activeBibleOptions = computed(() => [
  { label: t('projects.filters.allActiveBible'), value: '' },
  { label: t('projects.filters.withActiveBible'), value: 'with' },
  { label: t('projects.filters.withoutActiveBible'), value: 'without' }
])

const recentOptions = computed(() => [
  { label: t('projects.filters.allRecent'), value: '' },
  { label: t('projects.filters.recentlyOpened'), value: 'recent' },
  { label: t('projects.filters.notRecentlyOpened'), value: 'not_recent' }
])

const sortOptions = computed(() => [
  { label: t('projects.sort.updatedDesc'), value: 'updated_at:desc' },
  { label: t('projects.sort.createdDesc'), value: 'created_at:desc' },
  { label: t('projects.sort.titleAsc'), value: 'title:asc' },
  { label: t('projects.sort.targetChaptersDesc'), value: 'target_chapters:desc' }
])

const projectTableColumns = computed(() => [
  { key: 'project', label: t('projects.table.project'), class: 'min-w-[280px]', headerClass: 'min-w-[280px]' },
  { key: 'status', label: t('projects.table.status'), class: 'min-w-[170px]' },
  { key: 'storyBible', label: t('projects.table.storyBible'), class: 'min-w-[220px]' },
  { key: 'targetChapters', label: t('projects.table.targetChapters'), align: 'right' as const, class: 'min-w-[140px] tabular-nums' },
  { key: 'updatedAt', label: t('projects.table.updatedAt'), class: 'min-w-[170px]' },
  { key: 'actions', label: t('projects.table.actions'), align: 'right' as const, class: 'min-w-[260px]' }
])

const activeFilterCount = computed(() => [
  searchQuery.value.trim(),
  projectStatusFilter.value,
  bibleStatusFilter.value,
  activeBibleFilter.value,
  recentFilter.value
].filter(Boolean).length)
const hasActiveFilters = computed(() => activeFilterCount.value > 0)

const visibleProjects = computed(() => sortProjects(projects.value.filter((project) => {
  if (projectStatusFilter.value && project.status !== projectStatusFilter.value) return false
  if (bibleStatusFilter.value && project.bible_status !== bibleStatusFilter.value) return false
  if (activeBibleFilter.value === 'with' && !hasActiveStoryBible(project)) return false
  if (activeBibleFilter.value === 'without' && hasActiveStoryBible(project)) return false
  if (recentFilter.value === 'recent' && !isRecentlyOpened(project)) return false
  if (recentFilter.value === 'not_recent' && isRecentlyOpened(project)) return false
  return matchesProjectSearch(project)
})))

const projectRows = computed<ProjectRow[]>(() => visibleProjects.value.map((project) => ({ id: project.id, project })))
const projectResultSummary = computed(() => t('projects.filters.resultSummary', { visible: visibleProjects.value.length, total: projects.value.length }))
const projectLoadError = computed(() => errors.value.find((error) => error.endpoint === 'projects' || error.endpoint.includes('/projects')))
const collectionLoading = computed(() => Boolean(loading.value.dashboard && projects.value.length === 0))
const collectionError = computed(() => !collectionLoading.value && projects.value.length === 0 ? projectLoadError.value?.message || '' : '')
const collectionEmpty = computed(() => !collectionLoading.value && !collectionError.value && projects.value.length === 0)
const collectionNoResults = computed(() => !collectionLoading.value && !collectionError.value && projects.value.length > 0 && visibleProjects.value.length === 0)

function uniqueTokens(values: Array<string | undefined>) {
  return Array.from(new Set(values.map((value) => value?.trim()).filter((value): value is string => Boolean(value)))).sort((left, right) => left.localeCompare(right))
}

function normalizeSearchValue(value: unknown) {
  return String(value ?? '').trim().toLowerCase()
}

function matchesProjectSearch(project: ProjectSummary) {
  const terms = normalizeSearchValue(searchQuery.value).split(/\s+/).filter(Boolean)
  if (terms.length === 0) return true

  const seed = project.seed
  const corpus = [
    project.title,
    project.id,
    project.slug,
    project.status,
    project.logline,
    seed?.premise,
    seed?.genre,
    seed?.tone,
    seed?.audience,
    project.active_story_bible_id,
    ...project.tags
  ].map(normalizeSearchValue).join('\n')

  return terms.every((term) => corpus.includes(term))
}

function sortProjects(items: ProjectSummary[]) {
  return [...items].sort((left, right) => {
    if (sortKey.value === 'title:asc') return left.title.localeCompare(right.title)
    if (sortKey.value === 'created_at:desc') return dateSortValue(right.created_at) - dateSortValue(left.created_at)
    if (sortKey.value === 'target_chapters:desc') return targetChapterCount(right) - targetChapterCount(left)
    return dateSortValue(right.updated_at) - dateSortValue(left.updated_at)
  })
}

function dateSortValue(value?: string) {
  if (!value) return 0
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) ? timestamp : 0
}

function formatProjectDate(value?: string) {
  if (!dateSortValue(value)) return t('common.emptyValue')
  return formatDateTime(value as string)
}

function targetChapterCount(project: ProjectSummary) {
  return project.target_chapters ?? project.seed?.target_chapters ?? project.chapter_count ?? 0
}

function targetChapterLabel(project: ProjectSummary) {
  return t('projects.targetChapterCountValue', { count: targetChapterCount(project) })
}

function hasActiveStoryBible(project: ProjectSummary) {
  return Boolean(project.active_story_bible_id)
}

function isRecentlyOpened(project: ProjectSummary) {
  return openedProjectIds.value.has(project.id)
}

function projectBibleStatusLabel(status: ProjectSummary['bible_status']) {
  return t(`status.projectBible.${status}`)
}

function projectBibleStatusVariant(status: ProjectSummary['bible_status']) {
  if (status === 'ready') return 'success'
  if (status === 'draft') return 'gold'
  return 'muted'
}

function projectStatusLabel(status?: string) {
  const value = status?.trim()
  if (!value) return t('projects.statuses.unknown')
  const labels: Record<string, string> = {
    active: t('projects.statuses.active'),
    draft: t('projects.statuses.draft'),
    planning: t('projects.statuses.planning'),
    paused: t('projects.statuses.paused'),
    archived: t('projects.statuses.archived'),
    completed: t('projects.statuses.completed'),
    unknown: t('projects.statuses.unknown')
  }
  return labels[value] || value.replace(/[_-]+/g, ' ')
}

function projectStatusVariant(status?: string) {
  const value = status?.trim().toLowerCase()
  if (value === 'active' || value === 'completed') return 'success'
  if (value === 'draft' || value === 'planning') return 'gold'
  return 'muted'
}

function seedProfileItems(project: ProjectSummary) {
  const seed = project.seed
  return [
    { key: 'genre', label: t('projects.seedGenre'), value: seed?.genre },
    { key: 'tone', label: t('projects.seedTone'), value: seed?.tone },
    { key: 'audience', label: t('projects.seedAudience'), value: seed?.audience }
  ].filter((item) => item.value?.trim())
}

function projectFromRow(row: Record<string, unknown>) {
  return (row as ProjectRow).project
}

function clearProjectFilters() {
  searchQuery.value = ''
  projectStatusFilter.value = ''
  bibleStatusFilter.value = ''
  activeBibleFilter.value = ''
  recentFilter.value = ''
}

function openProjectRoute(project: ProjectSummary, target: ProjectRouteTarget = 'overview') {
  workspace.openProject(project)
  const suffix = target === 'overview' ? '' : `/${target}`
  return navigateTo(`/projects/${project.id}${suffix}`)
}
</script>

<template>
  <PageShell density="normal">
    <PageHeader :eyebrow="t('projects.eyebrow')" :title="t('projects.title')" :description="t('projects.description')">
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
    </PageHeader>

    <StatusStack v-if="statusItems.length" :items="statusItems" />

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <StatCard v-for="stat in projectStats" :key="stat.key" :label="stat.label" :value="stat.value" :hint="stat.hint" :tone="stat.tone">
        <template #icon>
          <component :is="stat.icon" class="h-5 w-5" />
        </template>
      </StatCard>
    </div>

    <DataCollection
      :title="t('projects.libraryTitle')"
      :description="t('projects.libraryDescription')"
      :loading="collectionLoading"
      :error="collectionError"
      :empty="collectionEmpty"
      :no-results="collectionNoResults"
      :loading-title="t('projects.states.loadingTitle')"
      :loading-description="t('projects.states.loadingDescription')"
      :empty-title="t('projects.states.emptyTitle')"
      :empty-description="t('projects.states.emptyDescription')"
      :no-results-title="t('projects.states.noResultsTitle')"
      :no-results-description="t('projects.states.noResultsDescription')"
    >
      <template #toolbar>
        <Toolbar density="compact" class="w-full lg:w-auto">
          <template #start>
            <span class="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{{ projectResultSummary }}</span>
            <UiBadge v-if="hasActiveFilters" variant="muted">{{ t('projects.filters.activeCount', { count: activeFilterCount }) }}</UiBadge>
          </template>
          <template #end>
            <ViewModeToggle v-model="viewMode" :modes="['table', 'grid']" :label="t('projects.viewModeLabel')" />
            <DensityToggle v-model="density" :densities="['compact', 'comfortable']" :label="t('projects.densityLabel')" />
            <UiButton class="w-full sm:w-auto" to="/projects/new">
              <FolderPlus class="h-4 w-4" />
              {{ t('actions.createProject') }}
            </UiButton>
          </template>
        </Toolbar>
      </template>

      <template #filters>
        <FilterBar density="compact">
          <template #search>
            <SearchInput
              v-model="searchQuery"
              :label="t('projects.search.label')"
              :placeholder="t('projects.search.placeholder')"
            />
          </template>
          <UiSelect v-model="projectStatusFilter" :options="projectStatusOptions" class="min-w-[150px] flex-1 sm:max-w-[190px]" />
          <UiSelect v-model="bibleStatusFilter" :options="bibleStatusOptions" class="min-w-[170px] flex-1 sm:max-w-[210px]" />
          <UiSelect v-model="activeBibleFilter" :options="activeBibleOptions" class="min-w-[190px] flex-1 sm:max-w-[240px]" />
          <UiSelect v-model="recentFilter" :options="recentOptions" class="min-w-[170px] flex-1 sm:max-w-[220px]" />
          <template #actions>
            <SortSelect v-model="sortKey" :options="sortOptions" class="min-w-[210px]" />
            <UiButton v-if="hasActiveFilters" variant="outline" @click="clearProjectFilters">
              {{ t('projects.filters.clear') }}
            </UiButton>
          </template>
        </FilterBar>
      </template>

      <template #empty>
        <DataEmptyState :title="t('projects.states.emptyTitle')" :description="t('projects.states.emptyDescription')">
          <template #actions>
            <UiButton to="/projects/new">
              <FolderPlus class="h-4 w-4" />
              {{ t('actions.createProject') }}
            </UiButton>
            <UiButton variant="outline" :disabled="loading.dashboard" @click="workspace.loadDashboard()">
              <RefreshCw :class="['h-4 w-4', loading.dashboard && 'animate-spin']" />
              {{ t('actions.refresh') }}
            </UiButton>
          </template>
        </DataEmptyState>
      </template>

      <template #error>
        <DataErrorState :title="t('projects.states.errorTitle')" :description="collectionError">
          <template #actions>
            <UiButton variant="outline" :disabled="loading.dashboard" @click="workspace.loadDashboard()">
              <RefreshCw :class="['h-4 w-4', loading.dashboard && 'animate-spin']" />
              {{ t('common.retry') }}
            </UiButton>
          </template>
        </DataErrorState>
      </template>

      <template #no-results>
        <DataNoResultsState :title="t('projects.states.noResultsTitle')" :description="t('projects.states.noResultsDescription')">
          <template #actions>
            <UiButton variant="outline" @click="clearProjectFilters">{{ t('projects.filters.clear') }}</UiButton>
          </template>
        </DataNoResultsState>
      </template>

      <DataTable
        v-if="viewMode === 'table'"
        :columns="projectTableColumns"
        :rows="projectRows"
        row-key="id"
        :density="collectionDensity"
        :caption="t('projects.table.caption')"
        class="hidden lg:block"
        @row-click="openProjectRoute(projectFromRow($event), 'overview')"
      >
        <template #cell="{ row, column }">
          <div v-if="column.key === 'project'" class="min-w-0 space-y-1">
            <p class="break-words font-medium text-foreground" :title="projectFromRow(row).title">{{ projectFromRow(row).title }}</p>
            <p class="break-all font-mono text-[11px] text-muted-foreground" :title="projectFromRow(row).id">{{ t('projects.projectId') }}: {{ projectFromRow(row).id }}</p>
            <p v-if="projectFromRow(row).slug" class="break-all font-mono text-[11px] text-muted-foreground" :title="projectFromRow(row).slug">{{ t('projects.slug') }}: {{ projectFromRow(row).slug }}</p>
            <p class="line-clamp-2 text-xs leading-5 text-muted-foreground">{{ projectFromRow(row).logline }}</p>
          </div>
          <div v-else-if="column.key === 'status'" class="space-y-2">
            <UiBadge :variant="projectStatusVariant(projectFromRow(row).status)">{{ projectStatusLabel(projectFromRow(row).status) }}</UiBadge>
            <UiBadge :variant="isRecentlyOpened(projectFromRow(row)) ? 'success' : 'muted'">
              {{ isRecentlyOpened(projectFromRow(row)) ? t('projects.opened') : t('projects.notOpened') }}
            </UiBadge>
          </div>
          <div v-else-if="column.key === 'storyBible'" class="min-w-0 space-y-2">
            <UiBadge :variant="projectBibleStatusVariant(projectFromRow(row).bible_status)">{{ projectBibleStatusLabel(projectFromRow(row).bible_status) }}</UiBadge>
            <p class="break-all font-mono text-[11px] text-muted-foreground">
              {{ projectFromRow(row).active_story_bible_id || t('projects.missingActiveStoryBible') }}
            </p>
          </div>
          <span v-else-if="column.key === 'targetChapters'" class="font-mono text-sm">{{ targetChapterCount(projectFromRow(row)) }}</span>
          <div v-else-if="column.key === 'updatedAt'" class="space-y-1 text-xs text-muted-foreground">
            <p><span class="font-medium text-foreground">{{ t('projects.updatedAt') }}:</span> {{ formatProjectDate(projectFromRow(row).updated_at) }}</p>
            <p><span class="font-medium text-foreground">{{ t('projects.createdAt') }}:</span> {{ formatProjectDate(projectFromRow(row).created_at) }}</p>
          </div>
          <div v-else-if="column.key === 'actions'" class="flex justify-end gap-2">
            <UiButton size="sm" @click.stop="openProjectRoute(projectFromRow(row), 'overview')">
              <ArrowRight class="h-4 w-4" />
              {{ t('projects.actions.overview') }}
            </UiButton>
            <UiButton size="sm" variant="outline" @click.stop="openProjectRoute(projectFromRow(row), 'editor')">
              <FilePenLine class="h-4 w-4" />
              {{ t('projects.actions.editor') }}
            </UiButton>
            <UiButton size="sm" variant="outline" @click.stop="openProjectRoute(projectFromRow(row), 'graph')">
              <GitFork class="h-4 w-4" />
              {{ t('projects.actions.graph') }}
            </UiButton>
          </div>
        </template>
      </DataTable>

      <DataCardGrid :items="projectRows" :density="collectionDensity" columns="two" :class="viewMode === 'table' ? 'lg:hidden' : ''">
        <template #default="{ item }">
          <Panel as="article" :padding="panelPadding" interactive>
            <div class="flex min-w-0 flex-wrap items-start justify-between gap-4">
              <div class="min-w-0 flex-1">
                <h3 class="break-words text-lg font-semibold" :title="projectFromRow(item).title">{{ projectFromRow(item).title }}</h3>
                <p class="mt-1 break-all font-mono text-[11px] text-muted-foreground" :title="projectFromRow(item).id">{{ projectFromRow(item).id }}</p>
                <p v-if="projectFromRow(item).slug" class="mt-1 break-all font-mono text-[11px] text-muted-foreground" :title="projectFromRow(item).slug">{{ projectFromRow(item).slug }}</p>
              </div>
              <div class="flex shrink-0 flex-col items-end gap-2">
                <UiBadge :variant="projectStatusVariant(projectFromRow(item).status)">{{ projectStatusLabel(projectFromRow(item).status) }}</UiBadge>
                <UiBadge :variant="isRecentlyOpened(projectFromRow(item)) ? 'success' : 'muted'">
                  {{ isRecentlyOpened(projectFromRow(item)) ? t('projects.opened') : t('projects.notOpened') }}
                </UiBadge>
              </div>
            </div>

            <p :class="['mt-4 text-sm leading-6 text-muted-foreground', density === 'compact' ? 'line-clamp-2' : 'line-clamp-3']">
              {{ projectFromRow(item).logline }}
            </p>

            <div class="mt-4 flex flex-wrap gap-2">
              <UiBadge v-for="tag in projectFromRow(item).tags" :key="tag" variant="muted">{{ tag }}</UiBadge>
              <span v-if="projectFromRow(item).tags.length === 0" class="text-sm text-muted-foreground">{{ t('projects.noTags') }}</span>
            </div>

            <div v-if="seedProfileItems(projectFromRow(item)).length" class="mt-4 rounded-xl bg-muted/35 p-3">
              <p class="field-label text-xs">{{ t('projects.seedProfile') }}</p>
              <div class="mt-2 flex flex-wrap gap-2">
                <UiBadge v-for="profile in seedProfileItems(projectFromRow(item))" :key="profile.key" variant="muted">
                  {{ profile.label }}: {{ profile.value }}
                </UiBadge>
              </div>
            </div>

            <div class="mt-4 grid gap-3 text-sm sm:grid-cols-3">
              <div class="rounded-xl bg-muted/35 p-3">
                <p class="field-label text-xs">{{ t('projects.storyBibleStatus') }}</p>
                <p class="mt-1 font-medium">{{ projectBibleStatusLabel(projectFromRow(item).bible_status) }}</p>
              </div>
              <div class="rounded-xl bg-muted/35 p-3">
                <p class="field-label text-xs">{{ t('projects.targetChapters') }}</p>
                <p class="mt-1 font-medium">{{ targetChapterLabel(projectFromRow(item)) }}</p>
              </div>
              <div class="rounded-xl bg-muted/35 p-3">
                <p class="field-label text-xs">{{ t('projects.updatedAt') }}</p>
                <p class="mt-1 font-medium">{{ formatProjectDate(projectFromRow(item).updated_at) }}</p>
              </div>
            </div>

            <div class="mt-4 min-w-0 rounded-xl border border-border bg-muted/20 p-3 text-xs text-muted-foreground">
              <p class="field-label text-xs">{{ t('projects.activeStoryBible') }}</p>
              <p class="mt-1 break-all font-mono">{{ projectFromRow(item).active_story_bible_id || t('projects.missingActiveStoryBible') }}</p>
            </div>

            <div class="mt-5 flex flex-wrap gap-2">
              <UiButton size="sm" @click="openProjectRoute(projectFromRow(item), 'overview')">
                <ArrowRight class="h-4 w-4" />
                {{ t('projects.actions.overview') }}
              </UiButton>
              <UiButton size="sm" variant="outline" @click="openProjectRoute(projectFromRow(item), 'editor')">
                <FilePenLine class="h-4 w-4" />
                {{ t('projects.actions.editor') }}
              </UiButton>
              <UiButton size="sm" variant="outline" @click="openProjectRoute(projectFromRow(item), 'graph')">
                <GitFork class="h-4 w-4" />
                {{ t('projects.actions.graph') }}
              </UiButton>
            </div>
          </Panel>
        </template>
      </DataCardGrid>
    </DataCollection>
  </PageShell>
</template>
