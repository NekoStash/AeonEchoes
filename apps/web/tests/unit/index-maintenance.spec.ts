import { describe, expect, it } from 'vitest'
import { createMaintenanceState, failMaintenance, mergeIndexJobs, startMaintenance, succeedMaintenance } from '../../features/index-maintenance/maintenance'
import type { IndexJob } from '../../lib/types'

function job(id: string, status: string, updatedAt: string): IndexJob {
  return { id, project_id: 'p', kind: 'chapter', status, attempts: 0, created_at: updatedAt, updated_at: updatedAt }
}

describe('index maintenance state', () => {
  it('记录运行、成功和失败阶段', () => {
    const state = createMaintenanceState()
    startMaintenance(state, 'run-pending', new Date('2026-01-01T00:00:00Z'))
    expect(state.phase).toBe('running')
    succeedMaintenance(state, 'done', new Date('2026-01-01T00:01:00Z'))
    expect(state).toMatchObject({ phase: 'succeeded', message: 'done' })
    failMaintenance(state, new Error('boom'), new Date('2026-01-01T00:02:00Z'))
    expect(state).toMatchObject({ phase: 'failed', error: 'boom' })
  })

  it('按任务 ID 合并后端最新状态', () => {
    const merged = mergeIndexJobs([job('a', 'pending', '2026-01-01T00:00:00Z')], [job('a', 'completed', '2026-01-02T00:00:00Z')])
    expect(merged).toHaveLength(1)
    expect(merged[0]?.status).toBe('completed')
  })
})
