import type { IndexJob } from '~/lib/types'

export type MaintenanceAction = 'rebuild-vectors' | 'run-pending' | 'run-job'
export type MaintenancePhase = 'idle' | 'running' | 'succeeded' | 'failed'

export interface MaintenanceState {
  action: MaintenanceAction | null
  phase: MaintenancePhase
  startedAt: string
  finishedAt: string
  message: string
  error: string
}

export function createMaintenanceState(): MaintenanceState {
  return { action: null, phase: 'idle', startedAt: '', finishedAt: '', message: '', error: '' }
}

export function startMaintenance(state: MaintenanceState, action: MaintenanceAction, now = new Date()) {
  Object.assign(state, { action, phase: 'running', startedAt: now.toISOString(), finishedAt: '', message: '', error: '' })
}

export function succeedMaintenance(state: MaintenanceState, message: string, now = new Date()) {
  Object.assign(state, { phase: 'succeeded', finishedAt: now.toISOString(), message, error: '' })
}

export function failMaintenance(state: MaintenanceState, error: unknown, now = new Date()) {
  const message = error instanceof Error ? error.message : String(error || 'Unknown maintenance error')
  console.error('[index-maintenance] Maintenance operation failed.', error)
  Object.assign(state, { phase: 'failed', finishedAt: now.toISOString(), message: '', error: message })
}

export function sortIndexJobs(jobs: IndexJob[]): IndexJob[] {
  return [...jobs].sort((left, right) => Date.parse(right.updated_at || right.created_at) - Date.parse(left.updated_at || left.created_at))
}

export function mergeIndexJobs(current: IndexJob[], updates: IndexJob[]): IndexJob[] {
  const updateIds = new Set(updates.map((job) => job.id))
  return sortIndexJobs([...updates, ...current.filter((job) => !updateIds.has(job.id))])
}
