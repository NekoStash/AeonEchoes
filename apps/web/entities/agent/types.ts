import type { AgentRunStreamEvent as GeneratedAgentRunStreamEvent, AgentRunStreamTool as GeneratedAgentRunStreamTool } from '~/lib/generated/api/types.gen'
import type { AgentRun, AgentRunResult, ModelResolution } from '~/lib/types'

export type { AgentConfig, AgentListOptions, AgentRole, AgentRun, AgentRunListOptions, AgentRunRequest, AgentRunResult } from '~/lib/types'

export type AgentRunStreamEventName = GeneratedAgentRunStreamEvent['type']

interface AgentRunStreamEventBase<TType extends AgentRunStreamEventName> {
  type: TType
  sequence: number
  run_id: string
}

export type AgentRunStreamToolStatus = GeneratedAgentRunStreamTool['status']
export type AgentRunStreamTool = GeneratedAgentRunStreamTool

export type AgentRunStreamEvent =
  | (AgentRunStreamEventBase<'run.started'> & { run: AgentRun })
  | (AgentRunStreamEventBase<'model.resolved'> & { model_resolution: ModelResolution })
  | (AgentRunStreamEventBase<'tool.started'> & { tool: AgentRunStreamTool })
  | (AgentRunStreamEventBase<'tool.completed'> & { tool: AgentRunStreamTool })
  | (AgentRunStreamEventBase<'content.delta'> & { delta: string })
  | AgentRunStreamEventBase<'content.reset'>
  | (AgentRunStreamEventBase<'run.completed'> & { result: AgentRunResult })
  | (AgentRunStreamEventBase<'run.failed'> & { error: string })

export interface AgentRunStreamOptions {
  signal?: AbortSignal
  onEvent?: (event: AgentRunStreamEvent) => void
}

export type AgentRunStreamStatus = 'idle' | 'connecting' | 'streaming' | 'tool-running' | 'finalizing' | 'completed' | 'failed' | 'cancelled'

export interface AgentRunStreamState {
  status: AgentRunStreamStatus
  chapterId: string
  runId: string
  content: string
  tools: AgentRunStreamTool[]
  modelResolution: ModelResolution | null
  error: string
}
