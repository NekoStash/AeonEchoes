import { defineStore } from 'pinia'
import type { ApiRequestState } from '~/shared/api'
import type { AgentConfig, AgentListOptions, AgentRunRequest } from './types'
import { createApiRequestState, withApiRequestState } from '~/shared/store'

interface AgentQueryScope {
  options: AgentListOptions
  items: AgentConfig[]
  request: ApiRequestState
}

function normalizeOptions(options?: AgentListOptions): AgentListOptions {
  const projectId = options?.projectId?.trim()
  return {
    projectId: projectId || undefined,
    enabled: options?.enabled,
    limit: options?.limit
  }
}

export function agentQueryScopeKey(options?: AgentListOptions): string {
  const normalized = normalizeOptions(options)
  return JSON.stringify({
    projectId: normalized.projectId || '',
    enabled: normalized.enabled === undefined ? 'all' : normalized.enabled,
    limit: normalized.limit ?? 'all'
  })
}

function agentScopeRank(agent: AgentConfig, projectId?: string) {
  if (projectId) {
    if (agent.project_id === projectId) return 0
    return agent.project_id ? 2 : 1
  }
  return agent.project_id ? 0 : 1
}

function compareAgents(left: AgentConfig, right: AgentConfig, projectId?: string) {
  return agentScopeRank(left, projectId) - agentScopeRank(right, projectId)
    || left.name.localeCompare(right.name)
    || left.id.localeCompare(right.id)
}

function matchesScope(agent: AgentConfig, options: AgentListOptions) {
  if (options.projectId && agent.project_id && agent.project_id !== options.projectId) return false
  if (options.enabled !== undefined && agent.enabled !== options.enabled) return false
  return true
}

function mergeAgent(scope: AgentQueryScope, agent: AgentConfig) {
  const items = scope.items.filter((item) => item.id !== agent.id)
  if (matchesScope(agent, scope.options)) items.push(agent)
  scope.items = items.sort((left, right) => compareAgents(left, right, scope.options.projectId))
}

export const useAgentStore = defineStore('agent-domain', {
  state: () => ({
    scopes: {} as Record<string, AgentQueryScope>,
    saveRequest: createApiRequestState(),
    deleteRequest: createApiRequestState(),
    runRequest: createApiRequestState()
  }),
  actions: {
    itemsFor(options?: AgentListOptions) {
      return this.scopes[agentQueryScopeKey(options)]?.items || []
    },
    requestFor(options?: AgentListOptions) {
      return this.scopes[agentQueryScopeKey(options)]?.request || null
    },
    async load(options?: AgentListOptions) {
      const normalized = normalizeOptions(options)
      const key = agentQueryScopeKey(normalized)
      const scope = this.scopes[key] || {
        options: normalized,
        items: [],
        request: createApiRequestState()
      }
      this.scopes[key] = scope
      return withApiRequestState(scope.request, 'agents.list', async () => {
        const result = await useApi().agent.listAgents(normalized)
        scope.items = [...result.data].sort((left, right) => compareAgents(left, right, normalized.projectId))
        return result
      })
    },
    async save(agent: AgentConfig, mode?: 'create' | 'edit') {
      return withApiRequestState(this.saveRequest, 'agents.save', async () => {
        const result = await useApi().agent.saveAgent(agent, mode)
        Object.values(this.scopes).forEach((scope) => mergeAgent(scope, result.data))
        return result
      })
    },
    async remove(id: string) {
      return withApiRequestState(this.deleteRequest, 'agents.delete', async () => {
        const result = await useApi().agent.deleteAgent(id)
        Object.values(this.scopes).forEach((scope) => {
          scope.items = scope.items.filter((item) => item.id !== id)
        })
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
