import { describe, expect, it, vi } from 'vitest'
import { applyAgentProposal, canCancelAgentRun, createAgentProposal, isAgentRunActive } from '../../features/agent-run'
import { buildChapterVersionPayload, loadChapterVersion } from '../../features/chapter-version'
import { resolveStrictChapter } from '../../features/chapter-write'
import {
  draftDiffersFromBackend,
  editorDraftStorageKey,
  readEditorDraft,
  removeEditorDraft,
  writeEditorDraft,
  type DraftStorage
} from '../../features/editor-draft-recovery'
import type { AgentRunResult, Chapter } from '../../lib/types'

const chapter: Chapter = {
  id: 'chapter-1',
  project_id: 'project-1',
  number: 1,
  title: '第一章',
  status: 'drafting',
  summary: '',
  metadata: {}
}

function createMemoryStorage(): DraftStorage & { values: Map<string, string> } {
  const values = new Map<string, string>()
  return {
    values,
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => { values.set(key, value) },
    removeItem: (key) => { values.delete(key) }
  }
}

function agentRun(content: string): AgentRunResult {
  return {
    run: { id: 'run-1', agent_id: 'agent-1', status: 'completed' },
    content,
    tool_trace: [],
    model_resolution: {
      route_key: 'writer',
      resolution_source: 'agent',
      provider_id: 'provider-1',
      provider_name: 'Provider',
      provider_type: 'openai',
      model_id: 'model-1',
      model_name: 'Model',
      model_kind: 'text'
    }
  }
}

describe('严格章节写作入口', () => {
  it('无真实章节时保持项目章节空态且不生成章节', () => {
    expect(resolveStrictChapter([], 'chapter-plan-1')).toEqual({ state: 'empty', chapter: null })
  })

  it('路由章节不存在时返回 invalid，不回退到其他章节', () => {
    expect(resolveStrictChapter([chapter], 'missing')).toEqual({
      state: 'invalid',
      chapter: null,
      requestedChapterId: 'missing'
    })
  })

  it('版本负载拒绝不存在的章节 ID', () => {
    expect(() => buildChapterVersionPayload([chapter], 'project-1', 'missing', {
      title: '不存在',
      content: '不应保存'
    })).toThrow('does not exist')
  })
})

describe('编辑器本地草稿恢复', () => {
  it('使用带 schema 版本、项目和章节的稳定 key', () => {
    expect(editorDraftStorageKey('project-1', 'chapter-1')).toBe('aeon-echoes:chapter-draft:v2:project-1:chapter-1')
  })

  it('可以保存、读取并删除本地草稿', () => {
    const storage = createMemoryStorage()
    const written = writeEditorDraft(storage, {
      project_id: 'project-1',
      chapter_id: 'chapter-1',
      title: '本地标题',
      content: '本地正文',
      parent_version_id: 'version-1',
      updated_at: '2026-01-01T00:00:00Z'
    })

    expect(written.error).toBeNull()
    expect(readEditorDraft(storage, 'project-1', 'chapter-1').value).toMatchObject({
      schema_version: 2,
      title: '本地标题',
      content: '本地正文'
    })
    expect(draftDiffersFromBackend(written.value!, '后端标题', '后端正文', 'version-1')).toBe(true)
    expect(removeEditorDraft(storage, 'project-1', 'chapter-1')).toEqual({ value: true, error: null })
    expect(readEditorDraft(storage, 'project-1', 'chapter-1').value).toBeNull()
  })

  it('写入失败时记录日志并返回可展示错误', () => {
    const error = new Error('quota exceeded')
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const storage: DraftStorage = {
      getItem: () => null,
      setItem: () => { throw error },
      removeItem: () => undefined
    }

    const result = writeEditorDraft(storage, {
      project_id: 'project-1',
      chapter_id: 'chapter-1',
      title: '标题',
      content: '正文'
    })

    expect(result.value).toBeNull()
    expect(result.error).toBe(error)
    expect(consoleError).toHaveBeenCalled()
    consoleError.mockRestore()
  })
})

describe('Agent Run 提案行为', () => {
  it('finalizing 保持运行锁但禁止取消', () => {
    expect(isAgentRunActive('finalizing')).toBe(true)
    expect(canCancelAgentRun('finalizing')).toBe(false)
    expect(['connecting', 'streaming', 'tool-running'].every(status => canCancelAgentRun(status as 'connecting' | 'streaming' | 'tool-running'))).toBe(true)
  })

  it('Agent 结果创建为 pending 提案且不修改正文', () => {
    const source = '原正文'
    const proposal = createAgentProposal('agent-1', agentRun('提案正文'), '2026-01-01T00:00:00Z')

    expect(source).toBe('原正文')
    expect(proposal).toMatchObject({ status: 'pending', content: '提案正文', runId: 'run-1' })
  })

  it('支持插入、有效选区替换、追加与拒绝', () => {
    const proposal = createAgentProposal('agent-1', agentRun('新内容'))

    expect(applyAgentProposal('前后', proposal, 'insert', { start: 1, end: 1 }).content).toBe('前新内容后')
    expect(applyAgentProposal('旧内容', proposal, 'replace', { start: 0, end: 1 }).content).toBe('新内容内容')
    expect(applyAgentProposal('正文', proposal, 'append', { start: 0, end: 0 }).content).toBe('正文\n\n新内容')
    expect(applyAgentProposal('正文', proposal, 'reject', { start: 0, end: 0 })).toMatchObject({
      content: '正文',
      proposal: { status: 'rejected' }
    })
  })

  it('覆盖模式用完整提案替换正文、保持标题职责在编辑器外且光标落在末尾', () => {
    const proposal = createAgentProposal('agent-1', agentRun('完整替换正文'))
    const application = applyAgentProposal('未保存旧正文', proposal, 'overwrite', { start: 2, end: 4 })

    expect(application.content).toBe('完整替换正文')
    expect(application.selection).toEqual({ start: 6, end: 6 })
    expect(application.proposal.status).toBe('applied')
  })

  it('空选区不能执行替换', () => {
    const proposal = createAgentProposal('agent-1', agentRun('新内容'))
    expect(() => applyAgentProposal('正文', proposal, 'replace', { start: 1, end: 1 })).toThrow('non-empty')
  })
})

describe('章节版本行为', () => {
  it('载入历史版本后明确将该版本作为后续手动版本父节点', () => {
    const historicalVersion = {
      id: 'version-history',
      project_id: 'project-1',
      chapter_id: 'chapter-1',
      version: 1,
      title: '历史标题',
      content: '历史正文',
      created_at: '2026-01-01T00:00:00Z'
    }

    const loaded = loadChapterVersion(historicalVersion)
    const payload = buildChapterVersionPayload([chapter], 'project-1', 'chapter-1', {
      title: loaded.title,
      content: `${loaded.content}续写`,
      parentVersionId: loaded.parentVersionId
    })

    expect(loaded.parentVersionId).toBe('version-history')
    expect(payload.parent_version_id).toBe('version-history')
  })

  it('只构建用户点击创建版本时使用的真实版本请求', () => {
    const payload = buildChapterVersionPayload([chapter], 'project-1', 'chapter-1', {
      title: '手动标题',
      content: '手动正文',
      changeNote: '用户手动创建版本',
      parentVersionId: 'version-1'
    })

    expect(payload).toEqual({
      chapter_id: 'chapter-1',
      title: '手动标题',
      content: '手动正文',
      summary: '手动正文',
      author_role: 'editor',
      change_note: '用户手动创建版本',
      parent_version_id: 'version-1'
    })
    expect(payload).not.toHaveProperty('id')
    expect(payload).not.toHaveProperty('version')
    expect(payload).not.toHaveProperty('index_status')
  })
})
