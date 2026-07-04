<script setup lang="ts">
import type { IndexJob } from '~/lib/types'
import { formatDateTime } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    jobs: IndexJob[]
    projectScoped?: boolean
  }>(),
  {
    projectScoped: false
  }
)

const { t } = useI18n()
const activeStatus = ref('all')

const statusTabs = computed(() => [
  { label: t('taskBoard.tabs.all'), value: 'all', badge: String(props.jobs.length) },
  { label: t('taskBoard.tabs.pending'), value: 'pending', badge: String(countByStatus('pending')) },
  { label: t('taskBoard.tabs.running'), value: 'running', badge: String(countByStatus('running')) },
  { label: t('taskBoard.tabs.failed'), value: 'failed', badge: String(countByStatus('failed')) },
  { label: t('taskBoard.tabs.completed'), value: 'completed', badge: String(countByStatus('completed')) }
])

const statCards = computed(() => [
  { key: 'all', label: t('taskBoard.stats.all'), value: props.jobs.length, variant: 'muted' as const },
  { key: 'pending', label: t('taskBoard.stats.pending'), value: countByStatus('pending'), variant: 'gold' as const },
  { key: 'running', label: t('taskBoard.stats.running'), value: countByStatus('running'), variant: 'violet' as const },
  { key: 'completed', label: t('taskBoard.stats.completed'), value: countByStatus('completed'), variant: 'success' as const },
  { key: 'failed', label: t('taskBoard.stats.failed'), value: countByStatus('failed'), variant: 'rose' as const }
])

const filteredJobs = computed(() => activeStatus.value === 'all'
  ? props.jobs
  : props.jobs.filter((job) => job.status === activeStatus.value))

function countByStatus(status: string) {
  return props.jobs.filter((job) => job.status === status).length
}

function statusLabel(status: string) {
  const key = `status.indexJob.${status}`
  const label = t(key)
  return label === key ? status : label
}

function statusVariant(status: string) {
  if (status === 'completed') return 'success' as const
  if (status === 'failed') return 'rose' as const
  if (status === 'running') return 'violet' as const
  if (status === 'pending') return 'gold' as const
  return 'muted' as const
}

function emptyValue(value?: string) {
  return value?.trim() || t('common.emptyValue')
}

function formatOptionalDate(value?: string) {
  return value ? formatDateTime(value) : t('common.emptyValue')
}

function payloadText(job: IndexJob) {
  if (!job.payload || Object.keys(job.payload).length === 0) return t('common.emptyValue')
  return JSON.stringify(job.payload, null, 2)
}
</script>

<template>
  <div class="min-w-0 space-y-4">
    <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
      <div v-for="stat in statCards" :key="stat.key" class="rounded-2xl border border-border bg-muted/25 p-3">
        <div class="flex items-center justify-between gap-2">
          <p class="text-xs uppercase tracking-[0.16em] text-muted-foreground">{{ stat.label }}</p>
          <UiBadge :variant="stat.variant">{{ stat.value }}</UiBadge>
        </div>
      </div>
    </div>

    <UiTabs v-model="activeStatus" :tabs="statusTabs" />

    <div v-if="filteredJobs.length === 0" class="rounded-2xl border border-border bg-muted/30 p-4 text-sm leading-6 text-muted-foreground">
      {{ projectScoped ? t('taskBoard.emptyProject') : t('taskBoard.empty') }}
    </div>

    <div v-else class="grid gap-4 xl:grid-cols-2">
      <article v-for="job in filteredJobs" :key="job.id" class="min-w-0 rounded-2xl border border-border bg-card p-4 shadow-sm">
        <div class="flex min-w-0 flex-wrap items-start justify-between gap-3">
          <div class="min-w-0 flex-1">
            <div class="field-label">
              {{ t('taskBoard.fields.kind') }}
              <UiInfoTooltip :text="t('tooltips.indexJobKind')" />
            </div>
            <h3 class="mt-1 break-words font-semibold text-foreground">{{ job.kind }}</h3>
          </div>
          <UiBadge class="shrink-0" :variant="statusVariant(job.status)">{{ statusLabel(job.status) }}</UiBadge>
        </div>

        <dl class="mt-4 grid gap-3 text-sm sm:grid-cols-2">
          <div class="rounded-xl border border-border bg-muted/20 p-3">
            <dt class="field-label text-xs">
              {{ t('taskBoard.fields.scope') }}
              <UiInfoTooltip :text="t('tooltips.indexJobScope')" />
            </dt>
            <dd class="mt-2 space-y-1 break-all font-mono text-xs text-foreground">
              <p>{{ t('taskBoard.fields.project') }}: {{ emptyValue(job.project_id) }}</p>
              <p>{{ t('taskBoard.fields.chapter') }}: {{ emptyValue(job.chapter_id) }}</p>
              <p>{{ t('taskBoard.fields.chapterVersion') }}: {{ emptyValue(job.chapter_version_id) }}</p>
            </dd>
          </div>
          <div class="rounded-xl border border-border bg-muted/20 p-3">
            <dt class="field-label text-xs">
              {{ t('taskBoard.fields.attempts') }}
              <UiInfoTooltip :text="t('tooltips.indexJobAttempts')" />
            </dt>
            <dd class="mt-2 text-foreground">{{ job.attempts }}</dd>
          </div>
          <div class="rounded-xl border border-border bg-muted/20 p-3">
            <dt class="field-label text-xs">
              {{ t('taskBoard.fields.createdAt') }}
              <UiInfoTooltip :text="t('tooltips.indexJobCreatedAt')" />
            </dt>
            <dd class="mt-2 text-foreground">{{ formatDateTime(job.created_at) }}</dd>
          </div>
          <div class="rounded-xl border border-border bg-muted/20 p-3">
            <dt class="field-label text-xs">
              {{ t('taskBoard.fields.updatedAt') }}
              <UiInfoTooltip :text="t('tooltips.indexJobUpdatedAt')" />
            </dt>
            <dd class="mt-2 text-foreground">{{ formatDateTime(job.updated_at) }}</dd>
          </div>
        </dl>

        <div v-if="job.error" class="mt-4 rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm leading-6 text-destructive">
          <div class="field-label text-destructive">
            {{ t('taskBoard.fields.error') }}
            <UiInfoTooltip :text="t('tooltips.indexJobError')" />
          </div>
          <p class="mt-1 break-words">{{ job.error }}</p>
        </div>

        <details class="mt-4 rounded-xl border border-border bg-muted/20 p-3 text-sm">
          <summary class="cursor-pointer select-none font-medium text-foreground focus-ring rounded-lg px-1 py-0.5">
            {{ t('taskBoard.details') }}
          </summary>
          <dl class="mt-3 space-y-3 leading-6">
            <div>
              <dt class="field-label text-xs">{{ t('taskBoard.fields.id') }}<UiInfoTooltip :text="t('tooltips.indexJobRawId')" /></dt>
              <dd class="mt-1 break-all font-mono text-xs text-muted-foreground">{{ job.id }}</dd>
            </div>
            <div>
              <dt class="field-label text-xs">{{ t('taskBoard.fields.payload') }}<UiInfoTooltip :text="t('tooltips.indexJobPayload')" /></dt>
              <dd class="mt-1">
                <pre class="max-h-44 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border bg-background p-3 text-xs text-muted-foreground subtle-scrollbar">{{ payloadText(job) }}</pre>
              </dd>
            </div>
            <div class="grid gap-2 sm:grid-cols-3">
              <div>
                <dt class="text-xs text-muted-foreground">{{ t('taskBoard.fields.startedAt') }}</dt>
                <dd class="mt-1 text-xs text-foreground">{{ formatOptionalDate(job.started_at) }}</dd>
              </div>
              <div>
                <dt class="text-xs text-muted-foreground">{{ t('taskBoard.fields.completedAt') }}</dt>
                <dd class="mt-1 text-xs text-foreground">{{ formatOptionalDate(job.completed_at) }}</dd>
              </div>
              <div>
                <dt class="text-xs text-muted-foreground">{{ t('taskBoard.fields.scheduledAt') }}</dt>
                <dd class="mt-1 text-xs text-foreground">{{ formatOptionalDate(job.scheduled_at) }}</dd>
              </div>
            </div>
          </dl>
        </details>
      </article>
    </div>
  </div>
</template>
