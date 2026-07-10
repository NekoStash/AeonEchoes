import type { ApiResult } from '~/shared/api'
import type { AgentConfig, AgentListOptions, AgentRun, AgentRunListOptions, AgentRunRequest, AgentRunResult } from './types'

export interface AgentApi {
  listAgents(options?: AgentListOptions): Promise<ApiResult<AgentConfig[]>>
  saveAgent(agent: AgentConfig, mode?: 'create' | 'edit'): Promise<ApiResult<AgentConfig>>
  deleteAgent(id: string): Promise<ApiResult<{ status: string }>>
  runAgent(agentId: string, request: AgentRunRequest): Promise<ApiResult<AgentRunResult>>
  listAgentRuns(options?: AgentRunListOptions): Promise<ApiResult<AgentRun[]>>
}
