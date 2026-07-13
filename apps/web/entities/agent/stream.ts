import type { AgentRun, AgentRunRequest, AgentRunResult, ModelResolution, ToolTrace } from '~/lib/types'
import { ApiClientError, isRecord, normalizeApiBase } from '~/shared/api'
import type { AgentRunStreamEvent, AgentRunStreamEventName, AgentRunStreamOptions, AgentRunStreamTool } from './types'

const STREAM_ENDPOINT = 'runAgentStream'
const STREAM_EVENT_NAMES: AgentRunStreamEventName[] = [
  'run.started',
  'model.resolved',
  'tool.started',
  'tool.completed',
  'content.delta',
  'content.reset',
  'run.completed',
  'run.failed'
]
const ENVELOPE_FIELDS = new Set(['type', 'sequence', 'run_id', 'delta', 'run', 'result', 'model_resolution', 'tool', 'error'])

function streamFailure(message: string, cause?: unknown, field?: string, kind: 'transport' | 'response' | 'validation' = 'validation', status?: number): never {
  const error = new ApiClientError({
    endpoint: STREAM_ENDPOINT,
    field,
    kind,
    status,
    code: kind === 'validation' ? 'invalid_sse_stream' : 'agent_stream_failed',
    message,
    cause
  })
  console.error('[AeonEchoes Agent Stream] Invalid or failed stream.', error.state)
  throw error
}

function requireRecord(value: unknown, field: string): Record<string, unknown> {
  if (!isRecord(value)) streamFailure(`invalid_sse_stream: ${field} must be an object`, value, field)
  return value
}

function requireString(value: unknown, field: string, allowEmpty = false): string {
  if (typeof value !== 'string' || (!allowEmpty && !value.trim())) {
    streamFailure(`invalid_sse_stream: ${field} must be ${allowEmpty ? 'a string' : 'a non-empty string'}`, value, field)
  }
  return value
}

function requireSequence(value: unknown, field: string): number {
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < 1) {
    streamFailure(`invalid_sse_stream: ${field} must be a positive safe integer`, value, field)
  }
  return value
}

function optionalString(value: unknown, field: string): string | undefined {
  if (value === undefined || value === null) return undefined
  return requireString(value, field, true)
}

function decodeAgentRun(value: unknown, field: string): AgentRun {
  const run = requireRecord(value, field)
  return {
    ...(run as unknown as AgentRun),
    id: requireString(run.id, `${field}.id`),
    agent_id: requireString(run.agent_id, `${field}.agent_id`),
    status: requireString(run.status, `${field}.status`),
    project_id: optionalString(run.project_id, `${field}.project_id`),
    error: optionalString(run.error, `${field}.error`)
  }
}

function decodeModelResolution(value: unknown, field: string): ModelResolution {
  const resolution = requireRecord(value, field)
  return {
    route_key: requireString(resolution.route_key, `${field}.route_key`),
    resolution_source: requireString(resolution.resolution_source, `${field}.resolution_source`),
    provider_id: requireString(resolution.provider_id, `${field}.provider_id`),
    provider_name: requireString(resolution.provider_name, `${field}.provider_name`),
    provider_type: requireString(resolution.provider_type, `${field}.provider_type`) as ModelResolution['provider_type'],
    model_id: requireString(resolution.model_id, `${field}.model_id`),
    model_name: requireString(resolution.model_name, `${field}.model_name`),
    model_kind: requireString(resolution.model_kind, `${field}.model_kind`) as ModelResolution['model_kind']
  }
}

function optionalJSONObject(value: unknown, field: string): Record<string, unknown> | undefined {
  if (value === undefined || value === null) return undefined
  return requireRecord(value, field)
}

function decodeTool(value: unknown, field: string, expectedStatus: AgentRunStreamTool['status']): AgentRunStreamTool {
  const tool = requireRecord(value, field)
  const allowed = new Set(['call_id', 'name', 'status', 'arguments', 'result'])
  for (const key of Object.keys(tool)) {
    if (!allowed.has(key)) streamFailure(`invalid_sse_stream: unsupported ${field} field ${key}`, tool, `${field}.${key}`)
  }
  const status = requireString(tool.status, `${field}.status`)
  if (status !== expectedStatus) {
    streamFailure(`invalid_sse_stream: ${field}.status must be ${expectedStatus} for this event`, status, `${field}.status`)
  }
  const decoded: AgentRunStreamTool = {
    call_id: requireString(tool.call_id, `${field}.call_id`),
    name: requireString(tool.name, `${field}.name`),
    status: expectedStatus
  }
  const args = optionalJSONObject(tool.arguments, `${field}.arguments`)
  if (args) decoded.arguments = args
  const result = optionalJSONObject(tool.result, `${field}.result`)
  if (result) decoded.result = result
  if (expectedStatus === 'completed' && tool.result !== undefined && tool.result !== null && !result) {
    streamFailure(`invalid_sse_stream: ${field}.result must be an object when present`, tool.result, `${field}.result`)
  }
  return decoded
}


function decodeError(value: unknown, field: string): string {
  return requireString(value, field)
}

function decodeToolTrace(value: unknown, field: string): ToolTrace[] | undefined {
  if (value === undefined || value === null) return undefined
  if (!Array.isArray(value)) streamFailure(`invalid_sse_stream: ${field} must be an array`, value, field)
  return value.map((item, index) => {
    if (typeof item === 'string') return requireString(item, `${field}[${index}]`)
    return requireRecord(item, `${field}[${index}]`) as ToolTrace
  })
}

function decodeAgentRunResult(value: unknown, field: string): AgentRunResult {
  const result = requireRecord(value, field)
  return {
    run: decodeAgentRun(result.run, `${field}.run`),
    content: requireString(result.content, `${field}.content`, true),
    model_resolution: decodeModelResolution(result.model_resolution, `${field}.model_resolution`),
    tool_trace: decodeToolTrace(result.tool_trace, `${field}.tool_trace`)
  }
}

function assertEnvelopeFields(payload: Record<string, unknown>, expectedField: string) {
  for (const key of Object.keys(payload)) {
    if (!ENVELOPE_FIELDS.has(key)) streamFailure(`invalid_sse_stream: unsupported payload field ${key}`, payload, key)
  }
  for (const field of ['delta', 'run', 'result', 'model_resolution', 'tool', 'error']) {
    if (field !== expectedField && payload[field] !== undefined) {
      streamFailure(`invalid_sse_stream: ${field} is not valid for ${String(payload.type)}`, payload[field], field)
    }
  }
}

export function decodeAgentRunStreamEvent(eventName: string, value: unknown): AgentRunStreamEvent {
  if (!STREAM_EVENT_NAMES.includes(eventName as AgentRunStreamEventName)) {
    streamFailure(`invalid_sse_stream: unsupported event ${eventName || '(empty)'}`, value, 'event')
  }
  const payload = requireRecord(value, 'data')
  const type = requireString(payload.type, 'type')
  if (type !== eventName) streamFailure(`invalid_sse_stream: event ${eventName} does not match data.type ${type}`, payload, 'type')
  const base = {
    type: eventName as AgentRunStreamEventName,
    sequence: requireSequence(payload.sequence, 'sequence'),
    run_id: requireString(payload.run_id, 'run_id')
  }

  switch (eventName) {
    case 'run.started': {
      assertEnvelopeFields(payload, 'run')
      const run = decodeAgentRun(payload.run, 'run')
      if (run.id !== base.run_id) streamFailure('invalid_sse_stream: run.id must match run_id', payload.run, 'run.id')
      return { ...base, type: eventName, run }
    }
    case 'model.resolved':
      assertEnvelopeFields(payload, 'model_resolution')
      return { ...base, type: eventName, model_resolution: decodeModelResolution(payload.model_resolution, 'model_resolution') }
    case 'tool.started':
      assertEnvelopeFields(payload, 'tool')
      return { ...base, type: eventName, tool: decodeTool(payload.tool, 'tool', 'started') }
    case 'tool.completed':
      assertEnvelopeFields(payload, 'tool')
      return { ...base, type: eventName, tool: decodeTool(payload.tool, 'tool', 'completed') }
    case 'content.delta':
      assertEnvelopeFields(payload, 'delta')
      return { ...base, type: eventName, delta: requireString(payload.delta, 'delta', true) }
    case 'content.reset':
      assertEnvelopeFields(payload, '')
      return { ...base, type: eventName }
    case 'run.completed': {
      assertEnvelopeFields(payload, 'result')
      const result = decodeAgentRunResult(payload.result, 'result')
      if (result.run.id !== base.run_id) streamFailure('invalid_sse_stream: result.run.id must match run_id', payload.result, 'result.run.id')
      return { ...base, type: eventName, result }
    }
    case 'run.failed':
      assertEnvelopeFields(payload, 'error')
      return { ...base, type: eventName, error: decodeError(payload.error, 'error') }
    default:
      return streamFailure(`invalid_sse_stream: unsupported event ${eventName}`, payload, 'event')
  }
}

interface SseMessage {
  event: string
  data: string[]
}

export async function consumeAgentRunSse(response: Response, options: AgentRunStreamOptions = {}): Promise<AgentRunResult> {
  if (!response.ok) {
    let body = ''
    try {
      body = await response.text()
    } catch (cause) {
      console.error('[AeonEchoes Agent Stream] Failed to read the error response body.', cause)
    }
    streamFailure(`agent_stream_failed (${response.status}): ${body || response.statusText || 'request failed'}`, body, undefined, 'response', response.status)
  }
  const contentType = response.headers.get('content-type')?.toLowerCase() || ''
  if (!contentType.includes('text/event-stream')) {
    streamFailure(`invalid_sse_stream: expected text/event-stream but received ${contentType || 'no content type'}`, contentType, 'content-type')
  }
  if (!response.body) streamFailure('invalid_sse_stream: response body is missing', response, 'body')

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let message: SseMessage = { event: '', data: [] }
  let lastSequence = 0
  let runId = ''
  let started = false
  let terminal: 'none' | 'completed' | 'failed' = 'none'
  let completedResult: AgentRunResult | null = null
  let failedMessage = ''
  let aborted = false
  let cleanupKind: 'completed' | 'failed' | 'aborted' | 'abnormal' = 'abnormal'
  let cancelPromise: Promise<void> | null = null
  const terminalState = () => terminal as 'none' | 'completed' | 'failed'

  const cancelReader = (reason: string) => {
    if (!cancelPromise) {
      cancelPromise = reader.cancel(reason).catch((cause) => {
        console.error('[AeonEchoes Agent Stream] Failed to cancel the SSE reader.', { reason, cause })
      })
    }
    return cancelPromise
  }
  const createAbortError = () => {
    const reason = options.signal?.reason
    if (reason instanceof Error && reason.name === 'AbortError') return reason
    const error = new Error('The Agent stream was aborted.')
    error.name = 'AbortError'
    return error
  }
  const handleAbort = () => {
    aborted = true
    void cancelReader('aborted')
  }

  const dispatch = () => {
    if (!message.event && message.data.length === 0) return
    if (!message.event) streamFailure('invalid_sse_stream: SSE message is missing event', message, 'event')
    if (message.data.length === 0) streamFailure(`invalid_sse_stream: ${message.event} is missing data`, message, 'data')
    let parsed: unknown
    try {
      parsed = JSON.parse(message.data.join('\n'))
    } catch (cause) {
      streamFailure(`invalid_sse_stream: ${message.event} data is not valid JSON`, cause, 'data')
    }
    const event = decodeAgentRunStreamEvent(message.event, parsed)
    if (!started && event.type !== 'run.started') streamFailure('invalid_sse_stream: first event must be run.started', event, 'event')
    if (started && event.type === 'run.started') streamFailure('invalid_sse_stream: run.started can only appear once', event, 'event')
    if (event.sequence <= lastSequence) {
      streamFailure(`invalid_sse_stream: sequence ${event.sequence} must be greater than ${lastSequence}`, event, 'sequence')
    }
    if (runId && event.run_id !== runId) streamFailure('invalid_sse_stream: run_id changed during the stream', event, 'run_id')
    lastSequence = event.sequence
    runId = event.run_id
    if (event.type === 'run.started') started = true
    options.onEvent?.(event)
    if (event.type === 'run.completed') {
      completedResult = event.result
      terminal = 'completed'
    } else if (event.type === 'run.failed') {
      failedMessage = event.error
      terminal = 'failed'
    }
  }

  const consumeLine = (line: string) => {
    if (line === '') {
      dispatch()
      message = { event: '', data: [] }
      return
    }
    if (line.startsWith(':')) return
    const colon = line.indexOf(':')
    const field = colon < 0 ? line : line.slice(0, colon)
    let value = colon < 0 ? '' : line.slice(colon + 1)
    if (value.startsWith(' ')) value = value.slice(1)
    if (field === 'event') {
      if (message.event) streamFailure('invalid_sse_stream: duplicate event field', line, 'event')
      message.event = value
      return
    }
    if (field === 'data') {
      message.data.push(value)
      return
    }
    streamFailure(`invalid_sse_stream: unsupported SSE field ${field || '(empty)'}`, line, field || 'field')
  }

  const drainBuffer = (done: boolean) => {
    while (buffer.length > 0 && terminalState() === 'none') {
      const lineBreak = buffer.search(/[\r\n]/)
      if (lineBreak < 0) break
      const delimiter = buffer[lineBreak]
      if (!done && delimiter === '\r' && lineBreak === buffer.length - 1) break
      const width = delimiter === '\r' && buffer[lineBreak + 1] === '\n' ? 2 : 1
      const line = buffer.slice(0, lineBreak)
      buffer = buffer.slice(lineBreak + width)
      consumeLine(line)
    }
    if (done && terminalState() === 'none' && buffer.length > 0) {
      consumeLine(buffer)
      buffer = ''
    }
  }

  options.signal?.addEventListener('abort', handleAbort, { once: true })
  if (options.signal?.aborted) handleAbort()
  try {
    while (terminalState() === 'none' && !aborted) {
      const { value, done } = await reader.read()
      if (aborted) throw createAbortError()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      drainBuffer(false)
    }
    if (aborted) throw createAbortError()
    if (terminalState() === 'completed' && completedResult) {
      cleanupKind = 'completed'
      return completedResult
    }
    if (terminalState() === 'failed') {
      cleanupKind = 'failed'
      streamFailure(`agent_stream_failed: ${failedMessage}`, failedMessage, 'error', 'response')
    }

    buffer += decoder.decode()
    drainBuffer(true)
    if (terminalState() === 'completed' && completedResult) {
      cleanupKind = 'completed'
      return completedResult
    }
    if (terminalState() === 'failed') {
      cleanupKind = 'failed'
      streamFailure(`agent_stream_failed: ${failedMessage}`, failedMessage, 'error', 'response')
    }
    streamFailure('invalid_sse_stream: connection ended before run.completed', { run_id: runId, sequence: lastSequence }, 'event')
  } catch (cause) {
    if (aborted || (cause instanceof Error && cause.name === 'AbortError')) cleanupKind = 'aborted'
    throw cause
  } finally {
    options.signal?.removeEventListener('abort', handleAbort)
    await cancelReader(cleanupKind)
    try {
      reader.releaseLock()
    } catch (cause) {
      console.error('[AeonEchoes Agent Stream] Failed to release the SSE reader lock.', cause)
    }
  }
  return streamFailure('invalid_sse_stream: parser exited without a terminal result', { run_id: runId, sequence: lastSequence }, 'event')
}

export async function streamAgentRun(
  rawBaseUrl: string,
  agentId: string,
  request: AgentRunRequest,
  options: AgentRunStreamOptions = {}
): Promise<AgentRunResult> {
  const baseUrl = normalizeApiBase(rawBaseUrl)
  const endpoint = `${baseUrl}/agents/${encodeURIComponent(agentId)}/runs/stream`
  let response: Response
  try {
    response = await fetch(endpoint, {
      method: 'POST',
      headers: { Accept: 'text/event-stream', 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
      signal: options.signal
    })
  } catch (cause) {
    if (cause instanceof Error && cause.name === 'AbortError') throw cause
    streamFailure('agent_stream_failed: unable to connect to the Agent stream', cause, undefined, 'transport')
  }
  return consumeAgentRunSse(response, options)
}
