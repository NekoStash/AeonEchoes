import type { AgentConfig } from './types'

export function preferredAgent(agents: AgentConfig[], projectId: string): AgentConfig | undefined {
  const projectWriter = agents.find((agent) => agent.project_id === projectId && agent.role === 'writer')
  if (projectWriter) return projectWriter
  const globalWriter = agents.find((agent) => !agent.project_id && agent.role === 'writer')
  if (globalWriter) return globalWriter
  const projectAgent = agents.find((agent) => agent.project_id === projectId)
  if (projectAgent) return projectAgent
  return agents.find((agent) => !agent.project_id)
}
