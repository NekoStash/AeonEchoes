import { defineStore } from 'pinia'
import type { ProjectSummary } from './types'
import { createApiRequestState, withApiRequestState } from '~/shared/store'

export const useProjectStore = defineStore('project-domain', {
  state: () => ({
    items: [] as ProjectSummary[],
    listRequest: createApiRequestState(),
    createRequest: createApiRequestState()
  }),
  actions: {
    async load() {
      return withApiRequestState(this.listRequest, 'projects.list', async () => {
        const result = await useApi().project.listProjects()
        this.items = result.data
        return result
      })
    },
    async initialize(seed: Parameters<ReturnType<typeof useApi>['project']['initializeProjectFull']>[0]) {
      return withApiRequestState(this.createRequest, 'projects.initialize', async () => {
        const result = await useApi().project.initializeProjectFull(seed)
        return result
      })
    },
    upsert(project: ProjectSummary) {
      this.items = [project, ...this.items.filter((item) => item.id !== project.id)]
    }
  }
})
