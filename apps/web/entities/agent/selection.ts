import type { AgentConfig, AgentRole } from './types'

export function preferredAgent(
  agents: AgentConfig[],
  projectId: string,
  preferredRole?: AgentRole
): AgentConfig | undefined {
  if (preferredRole) {
    const projectPreferred = agents.find((agent) => agent.project_id === projectId && agent.role === preferredRole)
    if (projectPreferred) return projectPreferred
    const globalPreferred = agents.find((agent) => !agent.project_id && agent.role === preferredRole)
    if (globalPreferred) return globalPreferred
  }

  const projectWriter = agents.find((agent) => agent.project_id === projectId && agent.role === 'writer')
  if (projectWriter) return projectWriter
  const globalWriter = agents.find((agent) => !agent.project_id && agent.role === 'writer')
  if (globalWriter) return globalWriter
  const projectAgent = agents.find((agent) => agent.project_id === projectId)
  if (projectAgent) return projectAgent
  return agents.find((agent) => !agent.project_id)
}
