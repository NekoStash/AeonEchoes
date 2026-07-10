<script setup lang="ts">
import { ArrowUpRight, BookOpenText, Filter, Search, X } from '@lucide/vue'
import { computed, reactive, ref } from 'vue'
import UiAsyncState from '~/components/ui/AsyncState.vue'
import UiBadge from '~/components/ui/Badge.vue'
import UiButton from '~/components/ui/Button.vue'
import UiEmptyState from '~/components/ui/EmptyState.vue'
import UiField from '~/components/ui/Field.vue'
import UiInlineNotice from '~/components/ui/InlineNotice.vue'
import UiInput from '~/components/ui/Input.vue'
import UiSelect from '~/components/ui/Select.vue'
import type { ProjectSummary } from '~/lib/types'
import { createProjectLibraryFilters, filterProjects, projectChapterCount } from './project-library'

const props = defineProps<{
  projects: ProjectSummary[]
  openedProjectIds: string[]
  loading?: boolean
  error?: string
}>()

const emit = defineEmits<{
  open: [project: ProjectSummary]
  retry: []
}>()

const { t } = useI18n()
const filters = reactive(createProjectLibraryFilters())
const advancedOpen = ref(false)

const recentIds = computed(() => new Set(props.openedProjectIds))
const visibleProjects = computed(() => filterProjects(props.projects, filters, recentIds.value))
const hasFilters = computed(() => Boolean(
  filters.query.trim()
  || filters.status
  || filters.storyBible !== 'all'
  || filters.recent !== 'all'
  || filters.sort !== 'updated'
))
const statusOptions = computed(() => [
  { label: t('projectLibrary.filters.allProjectStatuses'), value: '' },
  ...Array.from(new Set(props.projects.map((project) => project.status?.trim()).filter((status): status is string => Boolean(status))))
    .sort((left, right) => left.localeCompare(right))
    .map((status) => ({ label: statusLabel(status), value: status }))
])
const storyBibleOptions = computed(() => [
  { label: t('projectLibrary.filters.allStoryBibleStatuses'), value: 'all' },
  { label: t('status.projectBible.missing'), value: 'missing' },
  { label: t('status.projectBible.draft'), value: 'draft' },
  { label: t('status.projectBible.ready'), value: 'ready' }
])
const recentOptions = computed(() => [
  { label: t('projectLibrary.filters.allRecentStates'), value: 'all' },
  { label: t('projects.opened'), value: 'recent' },
  { label: t('projects.notOpened'), value: 'other' }
])
const sortOptions = computed(() => [
  { label: t('projectLibrary.sort.updated'), value: 'updated' },
  { label: t('projectLibrary.sort.created'), value: 'created' },
  { label: t('projectLibrary.sort.title'), value: 'title' }
])

function clearFilters() {
  Object.assign(filters, createProjectLibraryFilters())
}

function statusLabel(status?: string) {
  if (!status) return t('projects.statuses.unknown')
  const key = `projects.statuses.${status}`
  const translated = t(key)
  return translated === key ? status.replace(/[_-]+/g, ' ') : translated
}

function bibleStatusLabel(status: ProjectSummary['bible_status']) {
  return t(`status.projectBible.${status}`)
}

function bibleStatusTone(status: ProjectSummary['bible_status']) {
  if (status === 'ready') return 'success' as const
  if (status === 'draft') return 'warning' as const
  return 'muted' as const
}

function projectDate(value?: string) {
  if (!value) return t('common.emptyValue')
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return t('common.emptyValue')
  return new Intl.DateTimeFormat(undefined, { year: 'numeric', month: 'short', day: 'numeric' }).format(parsed)
}
</script>

<template>
  <section aria-labelledby="project-library-heading" class="border-t border-border">
    <div class="grid gap-5 border-b border-border py-5 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
      <div>
        <p class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{{ t('projectLibrary.eyebrow') }}</p>
        <h2 id="project-library-heading" class="mt-2 text-2xl font-semibold tracking-tight">{{ t('projectLibrary.title') }}</h2>
        <p class="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">{{ t('projectLibrary.description') }}</p>
      </div>
      <div class="text-sm text-muted-foreground">{{ t('projectLibrary.resultCount', { visible: visibleProjects.length, total: projects.length }) }}</div>
    </div>

    <div class="border-b border-border py-4">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
        <label class="relative min-w-0 flex-1">
          <span class="sr-only">{{ t('projects.search.label') }}</span>
          <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
          <UiInput v-model="filters.query" type="search" class="pl-10" :placeholder="t('projectLibrary.searchPlaceholder')" />
        </label>
        <UiButton variant="outline" class="w-full sm:w-auto" :aria-expanded="advancedOpen" aria-controls="project-library-advanced-filters" @click="advancedOpen = !advancedOpen">
          <Filter class="h-4 w-4" />
          {{ t('projectLibrary.advancedFilters') }}
        </UiButton>
        <UiButton v-if="hasFilters" variant="ghost" class="w-full sm:w-auto" @click="clearFilters">
          <X class="h-4 w-4" />
          {{ t('projects.filters.clear') }}
        </UiButton>
      </div>

      <div v-if="advancedOpen" id="project-library-advanced-filters" class="mt-4 grid gap-4 border-l-4 border-foreground bg-surface-muted p-4 sm:grid-cols-2 xl:grid-cols-4">
        <UiField :label="t('projectLibrary.filters.projectStatus')">
          <template #default="slotProps">
            <UiSelect v-model="filters.status" :options="statusOptions" :aria-label="t('projectLibrary.filters.projectStatus')" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" />
          </template>
        </UiField>
        <UiField :label="t('projectLibrary.filters.storyBibleStatus')">
          <template #default="slotProps">
            <UiSelect v-model="filters.storyBible" :options="storyBibleOptions" :aria-label="t('projectLibrary.filters.storyBibleStatus')" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" />
          </template>
        </UiField>
        <UiField :label="t('projectLibrary.filters.recent')">
          <template #default="slotProps">
            <UiSelect v-model="filters.recent" :options="recentOptions" :aria-label="t('projectLibrary.filters.recent')" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" />
          </template>
        </UiField>
        <UiField :label="t('projectLibrary.filters.sort')">
          <template #default="slotProps">
            <UiSelect v-model="filters.sort" :options="sortOptions" :aria-label="t('projectLibrary.filters.sort')" v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby }" />
          </template>
        </UiField>
      </div>
    </div>

    <UiAsyncState
      :status="loading && projects.length === 0 ? 'loading' : error && projects.length === 0 ? 'error' : projects.length === 0 ? 'empty' : 'ready'"
      :error="error"
      :loading-title="t('projects.states.loadingTitle')"
      :loading-description="t('projects.states.loadingDescription')"
      :error-title="t('projects.states.errorTitle')"
      :error-description="error"
      :empty-title="t('projects.states.emptyTitle')"
      :empty-description="t('projects.states.emptyDescription')"
      class="py-5"
    >
      <template #error>
        <UiInlineNotice tone="danger" :title="t('projects.states.errorTitle')" :description="error">
          <template #actions><UiButton variant="outline" size="sm" @click="emit('retry')">{{ t('common.retry') }}</UiButton></template>
        </UiInlineNotice>
      </template>
      <template #empty>
        <UiEmptyState :title="t('projects.states.emptyTitle')" :description="t('projects.states.emptyDescription')">
          <template #actions><UiButton to="/projects/new">{{ t('actions.createProject') }}</UiButton></template>
        </UiEmptyState>
      </template>

      <UiEmptyState v-if="visibleProjects.length === 0" :title="t('projects.states.noResultsTitle')" :description="t('projects.states.noResultsDescription')">
        <template #actions><UiButton variant="outline" @click="clearFilters">{{ t('projects.filters.clear') }}</UiButton></template>
      </UiEmptyState>

      <ol v-else class="divide-y divide-border" aria-label="project-library-list">
        <li v-for="(project, index) in visibleProjects" :key="project.id">
          <article class="group grid gap-4 py-5 sm:grid-cols-[3rem_minmax(0,1fr)] lg:grid-cols-[3rem_minmax(0,1fr)_minmax(14rem,20rem)] lg:items-center">
            <p class="font-mono text-xs text-muted-foreground">{{ String(index + 1).padStart(2, '0') }}</p>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <h3 class="break-words text-lg font-semibold tracking-tight">{{ project.title }}</h3>
                <UiBadge :tone="bibleStatusTone(project.bible_status)">{{ bibleStatusLabel(project.bible_status) }}</UiBadge>
                <UiBadge v-if="recentIds.has(project.id)" tone="neutral">{{ t('projects.opened') }}</UiBadge>
              </div>
              <p class="mt-2 line-clamp-2 max-w-3xl text-sm leading-6 text-muted-foreground">{{ project.logline || t('projectLibrary.missingLogline') }}</p>
              <div class="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <span>{{ t('projects.updatedAt') }} · {{ projectDate(project.updated_at) }}</span>
                <span v-if="projectChapterCount(project) !== null">{{ t('projects.chapterCountValue', { count: projectChapterCount(project) }) }}</span>
                <span v-if="project.status">{{ statusLabel(project.status) }}</span>
              </div>
            </div>
            <div class="flex flex-col gap-3 sm:col-start-2 lg:col-start-auto">
              <div v-if="project.tags.length" class="flex flex-wrap gap-2">
                <UiBadge v-for="tag in project.tags.slice(0, 4)" :key="tag" tone="muted">{{ tag }}</UiBadge>
              </div>
              <UiButton variant="outline" class="w-full justify-between" @click="emit('open', project)">
                <span><BookOpenText class="mr-2 inline h-4 w-4" />{{ t('projectLibrary.openProject') }}</span>
                <ArrowUpRight class="h-4 w-4" />
              </UiButton>
            </div>
          </article>
        </li>
      </ol>
    </UiAsyncState>
  </section>
</template>
