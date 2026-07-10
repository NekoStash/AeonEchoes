import { defineStore } from 'pinia'
import type { IndexJob, IndexJobListOptions } from './types'
import { createApiRequestState, withApiRequestState } from '~/shared/store'

function mergeJobs(current: IndexJob[], updates: IndexJob[]): IndexJob[] {
  const updatedIds = new Set(updates.map((job) => job.id))
  return [...updates, ...current.filter((job) => !updatedIds.has(job.id))]
}

export const useIndexJobStore = defineStore('index-job-domain', {
  state: () => ({
    items: [] as IndexJob[],
    listRequest: createApiRequestState(),
    runRequest: createApiRequestState(),
    rebuildRequest: createApiRequestState()
  }),
  actions: {
    async load(options?: string | IndexJobListOptions) {
      return withApiRequestState(this.listRequest, 'index-jobs.list', async () => {
        const result = await useApi().indexJob.listIndexJobs(options)
        this.items = result.data
        return result
      })
    },
    async runPending(projectId?: string, limit = 10) {
      return withApiRequestState(this.runRequest, 'index-jobs.run-pending', async () => {
        const result = await useApi().indexJob.runPendingIndexJobs(projectId, limit)
        this.items = mergeJobs(this.items, result.data.processed)
        if (result.data.error) throw new Error(result.data.error)
        return result
      })
    },
    async run(id: string) {
      return withApiRequestState(this.runRequest, 'index-jobs.run', async () => {
        const result = await useApi().indexJob.runIndexJob(id)
        this.items = mergeJobs(this.items, [result.data])
        return result
      })
    },
    async rebuild() {
      return withApiRequestState(this.rebuildRequest, 'index-jobs.rebuild-vectors', async () => {
        return useApi().indexJob.rebuildVectors()
      })
    }
  }
})
