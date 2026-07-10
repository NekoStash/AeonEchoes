import { defineStore } from 'pinia'
import type { AgentConfig, AgentListOptions, AgentRunRequest } from './types'
import { createApiRequestState, withApiRequestState } from '~/shared/store'

function mergeAgents(current: AgentConfig[], agent: AgentConfig) {
  return [...current.filter((item) => item.id !== agent.id), agent].sort((left, right) => left.id.localeCompare(right.id))
}

export const useAgentStore = defineStore('agent-domain', {
  state: () => ({
    items: [] as AgentConfig[],
    listRequest: createApiRequestState(),
    saveRequest: createApiRequestState(),
    deleteRequest: createApiRequestState(),
    runRequest: createApiRequestState()
  }),
  actions: {
    async load(options?: AgentListOptions) {
      return withApiRequestState(this.listRequest, 'agents.list', async () => {
        const result = await useApi().agent.listAgents(options)
        this.items = result.data
        return result
      })
    },
    async save(agent: AgentConfig, mode?: 'create' | 'edit') {
      return withApiRequestState(this.saveRequest, 'agents.save', async () => {
        const result = await useApi().agent.saveAgent(agent, mode)
        this.items = mergeAgents(this.items, result.data)
        return result
      })
    },
    async remove(id: string) {
      return withApiRequestState(this.deleteRequest, 'agents.delete', async () => {
        const result = await useApi().agent.deleteAgent(id)
        this.items = this.items.filter((item) => item.id !== id)
        return result
      })
    },
    async run(agentId: string, request: AgentRunRequest) {
      return withApiRequestState(this.runRequest, 'agents.run', async () => {
        return useApi().agent.runAgent(agentId, request)
      })
    }
  }
})
