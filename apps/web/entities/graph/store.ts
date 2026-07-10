import { defineStore } from 'pinia'
import type { GraphExpandRequest, GraphExpandResponse } from './types'
import { createApiRequestState, withApiRequestState } from '~/shared/store'

export const useGraphStore = defineStore('graph-domain', {
  state: () => ({
    byProjectId: {} as Record<string, GraphExpandResponse>,
    expandRequest: createApiRequestState()
  }),
  actions: {
    async expand(request: GraphExpandRequest) {
      return withApiRequestState(this.expandRequest, 'graph.expand', async () => {
        const result = await useApi().graph.expandGraph(request)
        this.byProjectId[request.project_id] = result.data
        return result
      })
    }
  }
})
