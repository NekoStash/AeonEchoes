import type { AgentRunResult } from '~/entities/agent'
import type { TextSelection } from '~/features/chapter-write'
import { normalizeTextSelection } from '~/features/chapter-write'

export type ProposalStatus = 'pending' | 'applied' | 'rejected'
export type ProposalApplyMode = 'insert' | 'replace' | 'append' | 'overwrite' | 'reject'

export interface AgentProposal {
  id: string
  agentId: string
  runId: string
  content: string
  status: ProposalStatus
  createdAt: string
  result: AgentRunResult
}

export interface ProposalApplication {
  content: string
  proposal: AgentProposal
  selection: TextSelection
}

export function createAgentProposal(agentId: string, result: AgentRunResult, now = new Date().toISOString()): AgentProposal {
  const content = result.content.trim()
  if (!content) throw new Error('Agent Run returned an empty proposal.')
  if (!result.run.id) throw new Error('Agent Run result is missing its run ID.')
  return {
    id: `proposal:${result.run.id}`,
    agentId,
    runId: result.run.id,
    content,
    status: 'pending',
    createdAt: now,
    result
  }
}

export function applyAgentProposal(
  source: string,
  proposal: AgentProposal,
  mode: ProposalApplyMode,
  selection: TextSelection
): ProposalApplication {
  if (proposal.status !== 'pending') throw new Error('Only pending proposals can be applied or rejected.')
  const normalized = normalizeTextSelection(selection, source.length)

  if (mode === 'reject') {
    return { content: source, proposal: { ...proposal, status: 'rejected' }, selection: normalized }
  }
  if (mode === 'replace' && normalized.start === normalized.end) {
    throw new Error('Replacing text requires a non-empty editor selection.')
  }

  let content = source
  let caret = normalized.start
  if (mode === 'insert') {
    content = `${source.slice(0, normalized.start)}${proposal.content}${source.slice(normalized.start)}`
    caret = normalized.start + proposal.content.length
  } else if (mode === 'replace') {
    content = `${source.slice(0, normalized.start)}${proposal.content}${source.slice(normalized.end)}`
    caret = normalized.start + proposal.content.length
  } else if (mode === 'overwrite') {
    content = proposal.content
    caret = content.length
  } else {
    const separator = source.trim() ? '\n\n' : ''
    content = `${source}${separator}${proposal.content}`
    caret = content.length
  }

  return {
    content,
    proposal: { ...proposal, status: 'applied' },
    selection: { start: caret, end: caret }
  }
}
