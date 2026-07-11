import type { AgentRunStreamStatus } from '~/entities/agent'

const ACTIVE_AGENT_RUN_STATUSES = new Set<AgentRunStreamStatus>(['connecting', 'streaming', 'tool-running', 'finalizing'])
const CANCELLABLE_AGENT_RUN_STATUSES = new Set<AgentRunStreamStatus>(['connecting', 'streaming', 'tool-running'])

export function isAgentRunActive(status: AgentRunStreamStatus) {
  return ACTIVE_AGENT_RUN_STATUSES.has(status)
}

export function canCancelAgentRun(status: AgentRunStreamStatus) {
  return CANCELLABLE_AGENT_RUN_STATUSES.has(status)
}
