import { describe, expect, it, vi } from 'vitest'
import {
  createApiClient,
  decodeChapterVersionResponse,
  decodeGraphExpansionResponse,
  decodeProjectSummaryResponse,
  decodeStoryBibleResponse,
  decodeWorkflowResponse
} from '../../lib/api'
import { ApiClientError } from '../../shared/api/error'
import {
  optionalApiArray,
  requireApiNumber,
  requireApiString
} from '../../shared/api/validation'

describe('Agent API 契约', () => {
  it('enabled=true/false 都作为布尔查询参数发送', async () => {
    const requests: string[] = []
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      requests.push(input instanceof Request ? input.url : String(input))
      return new Response(JSON.stringify({ data: [], meta: { request_id: 'agents-query' } }), { status: 200, headers: { 'content-type': 'application/json' } })
    })
    const client = createApiClient('http://api.test/api/v1')

    await client.listAgents({ projectId: 'project-1', enabled: true })
    await client.listAgents({ projectId: 'project-1', enabled: false })

    expect(new URL(requests[0]!).searchParams.get('enabled')).toBe('true')
    expect(new URL(requests[1]!).searchParams.get('enabled')).toBe('false')
    expect(new URL(requests[0]!).searchParams.get('project_id')).toBe('project-1')
    fetchMock.mockRestore()
  })

  it('严格校验 Agent 必填字段和允许的 role', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: [{ id: 'agent-1', name: 'Writer', enabled: true, role: 'writer', project_id: '' }], meta: { request_id: 'valid-agent' } }), { status: 200, headers: { 'content-type': 'application/json' } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: [{ id: 'agent-2', name: 'Broken', enabled: true, role: 'mystery-role' }], meta: { request_id: 'invalid-role' } }), { status: 200, headers: { 'content-type': 'application/json' } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: [{ id: 'agent-3', name: 'Broken', enabled: 'true', role: 'writer' }], meta: { request_id: 'invalid-enabled' } }), { status: 200, headers: { 'content-type': 'application/json' } }))
    const client = createApiClient('http://api.test/api/v1')

    await expect(client.listAgents({ enabled: true })).resolves.toMatchObject({ data: [{ id: 'agent-1', role: 'writer', enabled: true, project_id: '' }] })
    await expect(client.listAgents({ enabled: true })).rejects.toMatchObject({ state: { field: 'agents[0].role' } })
    await expect(client.listAgents({ enabled: true })).rejects.toMatchObject({ state: { field: 'agents[0].enabled' } })
    fetchMock.mockRestore()
  })
})

describe('API 响应校验', () => {
  it('关键字符串字段缺失时抛出类型化 validation error', () => {
    expect(() => requireApiString(undefined, 'storyBible', 'id')).toThrowError(ApiClientError)

    try {
      requireApiString(undefined, 'storyBible', 'id')
    } catch (error) {
      expect(error).toBeInstanceOf(ApiClientError)
      expect((error as ApiClientError).state).toMatchObject({
        endpoint: 'storyBible',
        field: 'id',
        kind: 'validation',
        code: 'invalid_api_response'
      })
    }
  })

  it('不会为缺失的可选领域数组生成伪数据', () => {
    expect(optionalApiArray(undefined, 'storyBible', 'characters', () => 'unused')).toEqual([])
    expect(optionalApiArray(undefined, 'storyBible', 'foreshadows', () => 'unused')).toEqual([])
    expect(optionalApiArray(undefined, 'storyBible', 'chapter_plan', () => 'unused')).toEqual([])
  })

  it('关键数字字段不接受字符串兜底', () => {
    expect(() => requireApiNumber('1', 'chapter', 'number')).toThrowError(ApiClientError)
    expect(requireApiNumber(1, 'chapter', 'number')).toBe(1)
  })

  it('Story Bible 仅暴露章节规划且零真实章节不会被补齐', () => {
    const bible = decodeStoryBibleResponse({
      id: 'bible-1',
      project_id: 'project-1',
      premise: '',
      themes: [],
      world_rules: [],
      characters: [],
      foreshadows: [],
      chapter_plan: []
    })

    expect(bible.chapter_plan).toEqual([])
    expect('chapters' in bible).toBe(false)
  })

  it('Story Bible 响应的新 ID 会被保留', () => {
    const bible = decodeStoryBibleResponse({
      id: 'bible-new',
      project_id: 'project-1',
      premise: 'premise',
      themes: [],
      world_rules: [],
      characters: [],
      foreshadows: [],
      chapter_plan: []
    }, 'updateStoryBible')

    expect(bible.id).toBe('bible-new')
  })

  it('项目响应没有真实章节计数时保持 unknown', () => {
    const project = decodeProjectSummaryResponse({
      id: 'project-1',
      title: 'Project',
      status: 'draft',
      updated_at: '2026-01-01T00:00:00Z',
      seed: { target_chapters: 12 }
    })

    expect(project.chapter_count).toBeUndefined()
    expect(project.target_chapters).toBe(12)
  })

  it('图谱派生 depth/timeline 缺失时合法并保持 undefined', () => {
    const now = '2026-01-01T00:00:00Z'
    const entity = {
      id: 'entity-1', project_id: 'project-1', name: '林烬', type: 'character', summary: '', importance: 1,
      status: 'stable', created_at: now, updated_at: now
    }
    const edge = {
      id: 'edge-1', project_id: 'project-1', source_entity_id: 'entity-1', target_entity_id: 'entity-1', type: 'knows', label: '', weight: 1,
      created_at: now, updated_at: now
    }
    const response = {
      project_id: 'project-1', depth: 1, entities: [entity], edges: [edge], facts: [], generated_at: now
    }

    expect(() => decodeGraphExpansionResponse(response)).not.toThrow(/invalid_api_response/)
    const valid = decodeGraphExpansionResponse(response)
    expect(valid).toMatchObject({ project_id: 'project-1', depth: 1, generated_at: now })
    expect(valid.nodes[0]).toMatchObject({ importance: 1 })
    expect(valid.nodes[0]?.depth).toBeUndefined()
    expect(valid.nodes[0]?.timeline).toBeUndefined()
    expect(valid.edges[0]?.timeline).toBeUndefined()
  })

  it('图谱派生 metadata 数字存在但为空或非有限值时抛类型化错误', () => {
    const now = '2026-01-01T00:00:00Z'
    const entity = {
      id: 'entity-1', project_id: 'project-1', name: '林烬', type: 'character', summary: '', importance: 1,
      status: 'stable', metadata: { depth: '1', timeline: '2' }, created_at: now, updated_at: now
    }
    const response = (patch: Record<string, unknown> = {}) => ({
      project_id: 'project-1', depth: 1, entities: [entity], edges: [], facts: [], generated_at: now, ...patch
    })

    expect(() => decodeGraphExpansionResponse(response({ entities: [{ ...entity, metadata: { depth: '', timeline: '2' } }] }))).toThrowError(ApiClientError)
    expect(() => decodeGraphExpansionResponse(response({ entities: [{ ...entity, metadata: { depth: '1', timeline: 'Infinity' } }] }))).toThrowError(ApiClientError)
    expect(() => decodeGraphExpansionResponse(response({ edges: [{
      id: 'edge-1', project_id: 'project-1', source_entity_id: 'entity-1', target_entity_id: 'entity-1', type: 'knows', label: '', weight: 1,
      metadata: { timeline: 'not-a-number' }, created_at: now, updated_at: now
    }] }))).toThrowError(ApiClientError)
  })

  it('图谱响应拒绝未知类型、状态和悬空边', () => {
    const now = '2026-01-01T00:00:00Z'
    const entity = {
      id: 'entity-1', project_id: 'project-1', name: '林烬', type: 'character', summary: '', importance: 1,
      status: 'stable', created_at: now, updated_at: now
    }
    const response = (patch: Record<string, unknown> = {}) => ({
      project_id: 'project-1', depth: 1, entities: [entity], edges: [], facts: [], generated_at: now, ...patch
    })

    expect(() => decodeGraphExpansionResponse(response({ entities: [{ ...entity, type: 'mystery' }] }))).toThrowError(ApiClientError)
    expect(() => decodeGraphExpansionResponse(response({ entities: [{ ...entity, status: 'unknown' }] }))).toThrowError(ApiClientError)
    expect(() => decodeGraphExpansionResponse(response({ edges: [{
      id: 'edge-1', project_id: 'project-1', source_entity_id: 'entity-1', target_entity_id: 'missing', type: 'knows', label: '', weight: 1,
      created_at: now, updated_at: now
    }] }))).toThrowError(ApiClientError)
  })

  it('图谱实体与边的 OpenAPI 必填字段仍严格校验', () => {
    const now = '2026-01-01T00:00:00Z'
    const entity = {
      id: 'entity-1', project_id: 'project-1', name: '林烬', type: 'character', summary: '', importance: 1,
      status: 'stable', created_at: now, updated_at: now
    }
    const edge = {
      id: 'edge-1', project_id: 'project-1', source_entity_id: 'entity-1', target_entity_id: 'entity-1', type: 'knows', label: '', weight: 1,
      created_at: now, updated_at: now
    }
    const response = { project_id: 'project-1', depth: 1, entities: [entity], edges: [edge], facts: [], generated_at: now }

    expect(() => decodeGraphExpansionResponse({ ...response, entities: [{ ...entity, importance: undefined }] })).toThrowError(ApiClientError)
    expect(() => decodeGraphExpansionResponse({ ...response, edges: [{ ...edge, updated_at: undefined }] })).toThrowError(ApiClientError)
  })

  it('图谱响应严格要求顶层契约字段', () => {
    const now = '2026-01-01T00:00:00Z'
    const entity = {
      id: 'entity-1', project_id: 'project-1', name: '林烬', type: 'character', summary: '', importance: 1,
      status: 'stable', metadata: { depth: '1', timeline: '2' }, created_at: now, updated_at: now
    }
    const valid = { project_id: 'project-1', depth: 1, entities: [entity], edges: [], facts: [], generated_at: now }

    expect(() => decodeGraphExpansionResponse({ ...valid, project_id: undefined })).toThrowError(ApiClientError)
    expect(() => decodeGraphExpansionResponse({ ...valid, depth: '1' })).toThrowError(ApiClientError)
    expect(() => decodeGraphExpansionResponse({ ...valid, depth: 0 })).toThrowError(ApiClientError)
    expect(() => decodeGraphExpansionResponse({ ...valid, generated_at: undefined })).toThrowError(ApiClientError)
    expect(() => decodeGraphExpansionResponse({ ...valid, generated_at: 123 })).toThrowError(ApiClientError)
  })

  it('图谱响应拒绝跨项目实体和边', () => {
    const now = '2026-01-01T00:00:00Z'
    const entity = {
      id: 'entity-1', project_id: 'project-1', name: '林烬', type: 'character', summary: '', importance: 1,
      status: 'stable', metadata: { depth: '1', timeline: '2' }, created_at: now, updated_at: now
    }
    const base = { project_id: 'project-1', depth: 1, entities: [entity], edges: [], facts: [], generated_at: now }

    expect(() => decodeGraphExpansionResponse({ ...base, entities: [{ ...entity, project_id: 'project-2' }] })).toThrowError(ApiClientError)
    expect(() => decodeGraphExpansionResponse({ ...base, edges: [{
      id: 'edge-1', project_id: 'project-2', source_entity_id: 'entity-1', target_entity_id: 'entity-1', type: 'knows', label: '', weight: 1,
      metadata: { timeline: '2' }, created_at: now, updated_at: now
    }] })).toThrowError(ApiClientError)
  })

  it('图谱客户端拒绝响应项目或 depth 与请求不一致', async () => {
    const now = '2026-01-01T00:00:00Z'
    const response = (projectId: string, depth: number) => ({
      project_id: projectId,
      depth,
      entities: [],
      edges: [],
      facts: [],
      generated_at: now
    })
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: response('project-2', 2), meta: { request_id: 'wrong-project' } }), { status: 200, headers: { 'content-type': 'application/json' } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: response('project-1', 3), meta: { request_id: 'wrong-depth' } }), { status: 200, headers: { 'content-type': 'application/json' } }))
    const client = createApiClient('http://api.test/api/v1')

    await expect(client.expandGraph({ project_id: 'project-1', depth: 2 })).rejects.toMatchObject({ state: { field: 'project_id' } })
    await expect(client.expandGraph({ project_id: 'project-1', depth: 2 })).rejects.toMatchObject({ state: { field: 'depth' } })
    fetchMock.mockRestore()
  })

  it('畸形 workflow 和 chapter version 响应会失败而非补运行结果', () => {
    expect(() => decodeWorkflowResponse({
      id: 'workflow-1',
      project_id: 'project-1',
      status: 'running',
      steps: []
    })).toThrowError(ApiClientError)

    expect(() => decodeChapterVersionResponse({
      id: 'version-1',
      project_id: 'project-1',
      chapter_id: 'chapter-1',
      version: 1,
      title: '',
      content: '',
      created_at: '2026-01-01T00:00:00Z',
      author_role: 'unsupported-role'
    })).toThrowError(ApiClientError)
  })
})
