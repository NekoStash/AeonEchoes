<script setup lang="ts">
import { AlertTriangle, CheckCircle2, DatabaseZap, Play, RefreshCw, RotateCcw, XCircle } from '@lucide/vue'
import { createMaintenanceState, failMaintenance, sortIndexJobs, startMaintenance, succeedMaintenance, type MaintenanceAction } from '~/features/index-maintenance/maintenance'
import SettingsWorkspace from '~/widgets/settings-workspace/SettingsWorkspace.vue'
import { useIndexJobStore } from '~/entities/index-job'
import type { IndexJob, RebuildVectorsResponse } from '~/lib/types'

const { t } = useI18n()
const toast = useToast()
const indexJobStore = useIndexJobStore()
const jobs = computed(() => indexJobStore.items)
const loading = computed(() => indexJobStore.listRequest.loading)
const loadError = ref('')
const state = reactive(createMaintenanceState())
const rebuildResult = ref<RebuildVectorsResponse | null>(null)
const projectId = ref('')
const statusFilter = ref('')
const limit = ref('20')
const confirmationAction = ref<MaintenanceAction | null>(null)
const confirmationJob = ref<IndexJob | null>(null)
const confirmOpen = computed({ get: () => Boolean(confirmationAction.value), set: (value) => { if (!value) { confirmationAction.value = null; confirmationJob.value = null } } })
const visibleJobs = computed(() => sortIndexJobs(jobs.value).filter((job) => !statusFilter.value || job.status === statusFilter.value))
const jobCounts = computed(() => jobs.value.reduce<Record<string, number>>((counts, job) => { counts[job.status] = (counts[job.status] || 0) + 1; return counts }, {}))
const parsedLimit = computed(() => {
  const value = Number(limit.value)
  return Number.isInteger(value) && value > 0 ? value : 20
})

onMounted(loadJobs)

async function loadJobs() {
  loadError.value = ''
  try {
    await indexJobStore.load({ projectId: projectId.value.trim() || undefined, status: statusFilter.value || undefined, limit: parsedLimit.value })
    indexJobStore.items = sortIndexJobs(indexJobStore.items)
  } catch (error) {
    console.error('[index-maintenance] Failed to load index jobs.', error)
    loadError.value = error instanceof Error ? error.message : t('settings.index.messages.loadFailed')
    toast.error(t('settings.index.messages.loadFailed'), loadError.value, error)
  }
}

function requestAction(action: MaintenanceAction, job?: IndexJob) {
  confirmationAction.value = action
  confirmationJob.value = job || null
}

async function runConfirmedAction() {
  const action = confirmationAction.value
  if (!action) return
  const job = confirmationJob.value
  confirmOpen.value = false
  startMaintenance(state, action)
  try {
    if (action === 'rebuild-vectors') {
      const result = await indexJobStore.rebuild()
      rebuildResult.value = result.data
      succeedMaintenance(state, t('settings.index.messages.rebuildCompleted', { count: result.data.job_count }))
    } else if (action === 'run-pending') {
      const result = await indexJobStore.runPending(projectId.value.trim() || undefined, parsedLimit.value)
      if (result.data.error) throw new Error(result.data.error)
      succeedMaintenance(state, t('settings.index.messages.pendingCompleted', { count: result.data.count }))
    } else {
      if (!job) throw new Error('run-job requires an index job.')
      const result = await indexJobStore.run(job.id)
      succeedMaintenance(state, t('settings.index.messages.jobCompleted', { id: result.data.id }))
    }
    toast.success(t('settings.index.messages.operationSucceeded'), state.message)
    await loadJobs()
  } catch (error) {
    failMaintenance(state, error)
    toast.error(t('settings.index.messages.operationFailed'), state.error, error)
  }
}

function confirmationTitle() {
  if (confirmationAction.value === 'rebuild-vectors') return t('settings.index.confirm.rebuildTitle')
  if (confirmationAction.value === 'run-pending') return t('settings.index.confirm.pendingTitle')
  return t('settings.index.confirm.jobTitle')
}

function confirmationDescription() {
  if (confirmationAction.value === 'rebuild-vectors') return t('settings.index.confirm.rebuildDescription')
  if (confirmationAction.value === 'run-pending') return t('settings.index.confirm.pendingDescription', { project: projectId.value || t('settings.index.allProjects'), limit: parsedLimit.value })
  return confirmationJob.value ? t('settings.index.confirm.jobDescription', { id: confirmationJob.value.id }) : ''
}

function statusTone(status: string) {
  if (status === 'completed') return 'success' as const
  if (status === 'failed') return 'danger' as const
  if (status === 'running') return 'info' as const
  if (status === 'pending') return 'warning' as const
  return 'muted' as const
}
</script>

<template>
  <SettingsWorkspace :title="t('settings.index.title')" :description="t('settings.index.description')">
    <div class="space-y-8">
      <UiInlineNotice v-if="loadError" tone="danger" :title="t('settings.index.messages.loadFailed')" :description="loadError"><template #actions><UiButton variant="outline" size="sm" :loading="loading" @click="loadJobs">{{ t('common.retry') }}</UiButton></template></UiInlineNotice>
      <section class="grid border border-border lg:grid-cols-2">
        <article class="border-b border-border p-5 lg:border-b-0 lg:border-r"><div class="flex items-start gap-3"><DatabaseZap class="mt-1 h-5 w-5" /><div><p class="text-xs font-black uppercase tracking-[0.16em] text-muted-foreground">{{ t('settings.index.rebuild.eyebrow') }}</p><h2 class="mt-2 text-2xl font-black">{{ t('settings.index.rebuild.title') }}</h2><p class="mt-3 text-sm leading-6 text-muted-foreground">{{ t('settings.index.rebuild.description') }}</p></div></div><UiButton class="mt-5" :disabled="state.phase === 'running'" @click="requestAction('rebuild-vectors')"><RotateCcw class="h-4 w-4" />{{ t('settings.index.rebuild.action') }}</UiButton></article>
        <article class="p-5"><div class="flex items-start gap-3"><Play class="mt-1 h-5 w-5" /><div><p class="text-xs font-black uppercase tracking-[0.16em] text-muted-foreground">{{ t('settings.index.pending.eyebrow') }}</p><h2 class="mt-2 text-2xl font-black">{{ t('settings.index.pending.title') }}</h2><p class="mt-3 text-sm leading-6 text-muted-foreground">{{ t('settings.index.pending.description') }}</p></div></div><div class="mt-5 grid gap-3 sm:grid-cols-[minmax(0,1fr)_8rem_auto]"><UiInput v-model="projectId" placeholder="project_id" /><UiInput v-model="limit" type="number" min="1" /><UiButton :disabled="state.phase === 'running'" @click="requestAction('run-pending')"><Play class="h-4 w-4" />{{ t('settings.index.pending.action') }}</UiButton></div></article>
      </section>

      <section v-if="state.phase !== 'idle'" :class="['border p-5', state.phase === 'failed' ? 'border-state-danger-border bg-state-danger-surface text-state-danger-foreground' : state.phase === 'succeeded' ? 'border-state-success-border bg-state-success-surface text-state-success-foreground' : 'border-state-info-border bg-state-info-surface text-state-info-foreground']">
        <div class="flex items-start gap-3"><XCircle v-if="state.phase === 'failed'" class="mt-0.5 h-5 w-5" /><CheckCircle2 v-else-if="state.phase === 'succeeded'" class="mt-0.5 h-5 w-5" /><RefreshCw v-else class="mt-0.5 h-5 w-5 animate-spin" /><div><p class="font-black">{{ t(`settings.index.phases.${state.phase}`) }}</p><p class="mt-1 text-sm">{{ state.error || state.message || t('settings.index.messages.running') }}</p><p class="mt-2 font-mono text-[11px] opacity-75">{{ state.action }} · {{ state.startedAt }}<template v-if="state.finishedAt"> → {{ state.finishedAt }}</template></p></div></div>
      </section>

      <section v-if="rebuildResult" class="border border-border bg-surface-muted p-5"><p class="text-xs font-black uppercase tracking-[0.16em] text-muted-foreground">{{ t('settings.index.rebuild.result') }}</p><dl class="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-5"><div><dt class="text-xs text-muted-foreground">embedding_model</dt><dd class="mt-1 break-all font-mono text-xs font-bold">{{ rebuildResult.embedding_model_name }} ({{ rebuildResult.embedding_model_id }})</dd></div><div><dt class="text-xs text-muted-foreground">dimension</dt><dd class="mt-1 text-xl font-black">{{ rebuildResult.embedding_dimension }}</dd></div><div><dt class="text-xs text-muted-foreground">projects</dt><dd class="mt-1 text-xl font-black">{{ rebuildResult.project_count }}</dd></div><div><dt class="text-xs text-muted-foreground">versions</dt><dd class="mt-1 text-xl font-black">{{ rebuildResult.chapter_version_count }}</dd></div><div><dt class="text-xs text-muted-foreground">jobs</dt><dd class="mt-1 text-xl font-black">{{ rebuildResult.job_count }}</dd></div></dl></section>

      <section>
        <div class="flex flex-col gap-4 border-b border-border pb-5 sm:flex-row sm:items-end sm:justify-between"><div><p class="text-xs font-black uppercase tracking-[0.16em] text-muted-foreground">{{ t('settings.index.jobs.eyebrow') }}</p><h2 class="mt-2 text-2xl font-black">{{ t('settings.index.jobs.title') }}</h2><div class="mt-3 flex flex-wrap gap-2"><UiBadge tone="warning">{{ t('status.indexJob.pending') }} {{ jobCounts.pending || 0 }}</UiBadge><UiBadge tone="info">{{ t('status.indexJob.running') }} {{ jobCounts.running || 0 }}</UiBadge><UiBadge tone="danger">{{ t('status.indexJob.failed') }} {{ jobCounts.failed || 0 }}</UiBadge><UiBadge tone="success">{{ t('status.indexJob.completed') }} {{ jobCounts.completed || 0 }}</UiBadge></div></div><div class="flex gap-2"><UiSelect v-model="statusFilter" class="w-40" :options="['pending','running','failed','completed','superseded'].map((value) => ({ value, label: t(`status.indexJob.${value}`) }))" :placeholder="t('settings.index.jobs.allStatuses')" /><UiButton variant="outline" :loading="loading" @click="loadJobs"><RefreshCw class="h-4 w-4" />{{ t('actions.refresh') }}</UiButton></div></div>
        <UiAlert v-if="indexJobStore.listRequest.error && jobs.length === 0" tone="danger" :title="t('settings.index.messages.loadFailed')" :description="indexJobStore.listRequest.error.message" />
        <div v-else-if="loading && jobs.length === 0" class="py-14 text-center text-sm font-bold text-muted-foreground">{{ t('settings.index.messages.loading') }}</div>
        <div v-else-if="visibleJobs.length === 0" class="py-14 text-center text-sm text-muted-foreground">{{ t('settings.index.jobs.empty') }}</div>
        <div v-else class="divide-y divide-border border-b border-border">
          <article v-for="job in visibleJobs" :key="job.id" class="grid gap-4 py-4 xl:grid-cols-[minmax(0,1fr)_14rem_10rem] xl:items-center"><div><div class="flex flex-wrap items-center gap-2"><h3 class="font-black">{{ job.kind }}</h3><UiBadge :tone="statusTone(job.status)">{{ t(`status.indexJob.${job.status}`) }}</UiBadge></div><p class="mt-2 break-all font-mono text-[11px] text-muted-foreground">{{ job.id }} · project={{ job.project_id }} · chapter={{ job.chapter_id || '—' }}</p><p v-if="job.error" class="mt-2 flex gap-2 text-sm text-state-danger-foreground"><AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" />{{ job.error }}</p></div><dl class="grid grid-cols-2 gap-3 text-xs"><div><dt class="text-muted-foreground">attempts</dt><dd class="mt-1 font-mono font-bold">{{ job.attempts }}</dd></div><div><dt class="text-muted-foreground">updated</dt><dd class="mt-1 font-mono text-[11px] font-bold">{{ job.updated_at }}</dd></div></dl><div class="xl:text-right"><UiButton size="sm" variant="outline" :disabled="state.phase === 'running' || job.status === 'completed' || job.status === 'running'" @click="requestAction('run-job', job)"><Play class="h-4 w-4" />{{ t('settings.index.jobs.run') }}</UiButton></div></article>
        </div>
      </section>
    </div>

    <UiConfirm v-model:open="confirmOpen" :title="confirmationTitle()" :description="confirmationDescription()" :loading="state.phase === 'running'" tone="danger" @confirm="runConfirmedAction" />
  </SettingsWorkspace>
</template>
