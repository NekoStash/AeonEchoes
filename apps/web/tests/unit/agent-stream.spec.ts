import { describe, expect, it, vi } from 'vitest'
import { consumeAgentRunSse, streamAgentRun } from '../../entities/agent/stream'
import type { AgentRunStreamEvent } from '../../entities/agent/types'

const resolution = {
  route_key: 'writer',
  resolution_source: 'agent',
  provider_id: 'provider-1',
  provider_name: 'Provider',
  provider_type: 'openai',
  model_id: 'model-1',
  model_name: 'Model',
  model_kind: 'text'
}

function event(name: string, data: Record<string, unknown>, eol = '\r\n') {
  return `event: ${name}${eol}data: ${JSON.stringify(data)}${eol}${eol}`
}

function completedResult(content = '你好，世界') {
  return {
    run: { id: 'run-1', agent_id: 'agent-1', project_id: 'project-1', status: 'completed' },
    content,
    tool_trace: [],
    model_resolution: resolution
  }
}

function responseFromByteChunks(chunks: Uint8Array[]) {
  return new Response(new ReadableStream<Uint8Array>({
    start(controller) {
      chunks.forEach((chunk) => controller.enqueue(chunk))
      controller.close()
    }
  }), { headers: { 'content-type': 'text/event-stream; charset=utf-8' } })
}

function pendingResponse(source: string, cancel = vi.fn()) {
  const bytes = new TextEncoder().encode(source)
  return {
    cancel,
    response: new Response(new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(bytes)
      },
      cancel
    }), { headers: { 'content-type': 'text/event-stream; charset=utf-8' } })
  }
}

function splitBytes(bytes: Uint8Array, offsets: number[]) {
  const chunks: Uint8Array[] = []
  let start = 0
  for (const offset of offsets) {
    chunks.push(bytes.slice(start, offset))
    start = offset
  }
  chunks.push(bytes.slice(start))
  return chunks
}

function started(sequence = 1) {
  return event('run.started', { type: 'run.started', sequence, run_id: 'run-1', run: { id: 'run-1', agent_id: 'agent-1', status: 'running' } })
}

describe('Agent Run SSE parser', () => {
  it('跨 chunk、CRLF、heartbeat comment 和 UTF-8 多字节边界解析严格事件', async () => {
    const source = [
      started(1),
      ': heartbeat\r\n\r\n',
      event('model.resolved', { type: 'model.resolved', sequence: 2, run_id: 'run-1', model_resolution: resolution }),
      event('content.delta', { type: 'content.delta', sequence: 3, run_id: 'run-1', delta: '暂' }),
      event('content.reset', { type: 'content.reset', sequence: 4, run_id: 'run-1' }),
      event('tool.started', { type: 'tool.started', sequence: 5, run_id: 'run-1', tool: { call_id: 'call-1', name: 'character.search', status: 'started', arguments: { project_id: 'project-1', query: '林' } } }),
      event('tool.completed', { type: 'tool.completed', sequence: 6, run_id: 'run-1', tool: { call_id: 'call-1', name: 'character.search', status: 'completed', arguments: { project_id: 'project-1', query: '林' }, result: { count: 1 } } }),
      event('content.delta', { type: 'content.delta', sequence: 7, run_id: 'run-1', delta: '你好，' }),
      event('content.delta', { type: 'content.delta', sequence: 8, run_id: 'run-1', delta: '世界' }),
      event('run.completed', { type: 'run.completed', sequence: 9, run_id: 'run-1', result: completedResult() })
    ].join('')
    const bytes = new TextEncoder().encode(source)
    const chineseByte = bytes.findIndex((value) => value > 127)
    const heartbeatByte = source.indexOf('heartbeat') + 4
    const events: AgentRunStreamEvent[] = []

    const result = await consumeAgentRunSse(responseFromByteChunks(splitBytes(bytes, [1, 19, heartbeatByte, chineseByte + 1, chineseByte + 2, bytes.length - 7])), {
      onEvent: (streamEvent) => events.push(streamEvent)
    })

    expect(events.map((item) => item.type)).toEqual(['run.started', 'model.resolved', 'content.delta', 'content.reset', 'tool.started', 'tool.completed', 'content.delta', 'content.delta', 'run.completed'])
    expect(events.filter((item) => item.type === 'content.delta').map((item) => item.delta).join('')).toBe('暂你好，世界')
    const startedTool = events.find((item) => item.type === 'tool.started')?.tool
    expect(startedTool).toMatchObject({ call_id: 'call-1', name: 'character.search', status: 'started', arguments: { project_id: 'project-1', query: '林' } })
    const completedTool = events.find((item) => item.type === 'tool.completed')?.tool
    expect(completedTool).toMatchObject({ call_id: 'call-1', name: 'character.search', status: 'completed', arguments: { project_id: 'project-1', query: '林' }, result: { count: 1 } })
    expect(result).toEqual(completedResult())
  })


  it.each([
    ['未知事件', event('mystery.event', { type: 'mystery.event', sequence: 1, run_id: 'run-1' })],
    ['event 与 type 不一致', event('run.started', { type: 'content.delta', sequence: 1, run_id: 'run-1', run: { id: 'run-1', agent_id: 'agent-1', status: 'running' } })],
    ['sequence 从 0 开始', event('run.started', { type: 'run.started', sequence: 0, run_id: 'run-1', run: { id: 'run-1', agent_id: 'agent-1', status: 'running' } })],
    ['事件包含非法字段', `${started(1)}${event('content.delta', { type: 'content.delta', sequence: 2, run_id: 'run-1', delta: 'a', unexpected: true })}`],
    ['sequence 倒退', `${started(2)}${event('content.delta', { type: 'content.delta', sequence: 1, run_id: 'run-1', delta: 'a' })}`],
    ['tool.started 状态不匹配', `${started(1)}${event('tool.started', { type: 'tool.started', sequence: 2, run_id: 'run-1', tool: { call_id: 'call-1', name: 'search', status: 'completed' } })}`],
    ['tool call_id 为空', `${started(1)}${event('tool.started', { type: 'tool.started', sequence: 2, run_id: 'run-1', tool: { call_id: '', name: 'search', status: 'started' } })}`],
    ['tool call_id 仅空白', `${started(1)}${event('tool.started', { type: 'tool.started', sequence: 2, run_id: 'run-1', tool: { call_id: '   ', name: 'search', status: 'started' } })}`],
    ['tool name 为空', `${started(1)}${event('tool.started', { type: 'tool.started', sequence: 2, run_id: 'run-1', tool: { call_id: 'call-1', name: '', status: 'started' } })}`],
    ['tool name 仅空白', `${started(1)}${event('tool.started', { type: 'tool.started', sequence: 2, run_id: 'run-1', tool: { call_id: 'call-1', name: '  ', status: 'started' } })}`],
    ['tool arguments 不是对象', `${started(1)}${event('tool.started', { type: 'tool.started', sequence: 2, run_id: 'run-1', tool: { call_id: 'call-1', name: 'search', status: 'started', arguments: 'secret' } })}`],
    ['tool result 不是对象', `${started(1)}${event('tool.completed', { type: 'tool.completed', sequence: 2, run_id: 'run-1', tool: { call_id: 'call-1', name: 'search', status: 'completed', result: 'hidden' } })}`],
    ['run.failed error 不是字符串', `${started(1)}${event('run.failed', { type: 'run.failed', sequence: 2, run_id: 'run-1', error: { message: 'boom' } })}`],
    ['run.failed error 为空', `${started(1)}${event('run.failed', { type: 'run.failed', sequence: 2, run_id: 'run-1', error: '' })}`]
  ])('%s 时 fail-fast 并记录 console.error', async (_label, source) => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    await expect(consumeAgentRunSse(responseFromByteChunks([new TextEncoder().encode(source)]))).rejects.toThrow()
    expect(consoleError).toHaveBeenCalled()
    consoleError.mockRestore()
  })

  it('completed 后底层流永不 close 也立即返回并 cancel reader', async () => {
    const source = `${started(1)}${event('run.completed', { type: 'run.completed', sequence: 2, run_id: 'run-1', result: completedResult() })}`
    const controlled = pendingResponse(source)

    await expect(consumeAgentRunSse(controlled.response)).resolves.toEqual(completedResult())
    expect(controlled.cancel).toHaveBeenCalledWith('completed')
  })

  it('completed 是协议终止，忽略同 chunk 非法尾随且不会转失败', async () => {
    const source = `${started(1)}${event('run.completed', { type: 'run.completed', sequence: 2, run_id: 'run-1', result: completedResult() })}${event('mystery.event', { type: 'mystery.event', sequence: 3, run_id: 'run-1' })}`
    const controlled = pendingResponse(source)
    const events: AgentRunStreamEvent[] = []

    await expect(consumeAgentRunSse(controlled.response, { onEvent: value => events.push(value) })).resolves.toEqual(completedResult())
    expect(events.map(value => value.type)).toEqual(['run.started', 'run.completed'])
    expect(controlled.cancel).toHaveBeenCalledWith('completed')
  })

  it('run.failed 是失败终止并立即 cancel reader', async () => {
    const source = `${started(1)}${event('run.failed', { type: 'run.failed', sequence: 2, run_id: 'run-1', error: '模型失败' })}`
    const controlled = pendingResponse(source)
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined)

    await expect(consumeAgentRunSse(controlled.response)).rejects.toThrow('模型失败')
    expect(controlled.cancel).toHaveBeenCalledWith('failed')
    consoleError.mockRestore()
  })

  it('解析失败统一 cancel reader 并释放异常流', async () => {
    const controlled = pendingResponse('event: run.started\ndata: {bad json}\n\n')
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined)

    await expect(consumeAgentRunSse(controlled.response)).rejects.toThrow('not valid JSON')
    expect(controlled.cancel).toHaveBeenCalledWith('abnormal')
    consoleError.mockRestore()
  })

  it('AbortSignal 会取消 pending reader 并抛出 AbortError', async () => {
    const controlled = pendingResponse(started(1))
    const controller = new AbortController()
    const operation = consumeAgentRunSse(controlled.response, {
      signal: controller.signal,
      onEvent: () => controller.abort()
    })

    await expect(operation).rejects.toMatchObject({ name: 'AbortError' })
    expect(controlled.cancel).toHaveBeenCalledWith('aborted')
  })

  it('连接结束却没有 run.completed 时 fail-fast 并记录 console.error', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined)

    await expect(consumeAgentRunSse(responseFromByteChunks([new TextEncoder().encode(started(1))]))).rejects.toThrow('before run.completed')
    expect(consoleError).toHaveBeenCalled()
    consoleError.mockRestore()
  })

  it('typed fetch 使用 POST stream endpoint、同步 run JSON body 与 AbortSignal', async () => {
    const controller = new AbortController()
    const request = { project_id: 'project-1', task_type: 'generic', input: { instruction: '继续写' } }
    const source = [
      started(1).replaceAll('\r\n', '\n'),
      event('run.completed', { type: 'run.completed', sequence: 2, run_id: 'run-1', result: completedResult() }, '\n')
    ].join('')
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(responseFromByteChunks([new TextEncoder().encode(source)]))

    await streamAgentRun('http://api.test/api/v1/', 'agent/one', request, { signal: controller.signal })

    expect(fetchMock).toHaveBeenCalledWith('http://api.test/api/v1/agents/agent%2Fone/runs/stream', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify(request),
      signal: controller.signal,
      headers: { Accept: 'text/event-stream', 'Content-Type': 'application/json' }
    }))
    fetchMock.mockRestore()
  })
})
