import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import AssistantPanel from '../../widgets/assistant-panel/AssistantPanel.vue'
import type { AgentConfig, Chapter } from '../../lib/types'

const chapter: Chapter = {
  id: 'chapter-1',
  project_id: 'project-1',
  number: 1,
  title: '第一章',
  status: 'drafting',
  summary: '',
  metadata: {}
}

function renderPanel(agents: AgentConfig[], agentLoadError = '', overrides: Record<string, unknown> = {}) {
  return render(AssistantPanel, {
    props: {
      agents,
      projectId: 'project-1',
      agentLoadError,
      chapters: [chapter],
      chapter,
      bible: null,
      versions: [],
      proposals: [],
      prompt: '',
      selectedAgentId: '',
      contextState: { previousChapterCount: 0, includeCurrentChapter: true, includeWorldRules: true, characterIds: [] },
      selectedText: '',
      localDraft: null,
      loadingAgents: false,
      runningAgent: false,
      streamState: { status: 'idle', chapterId: '', runId: '', content: '', tools: [], modelResolution: null, error: '' },
      loadingVersions: false,
      diagnostics: { modelResolution: null, toolTrace: [] },
      ...overrides
    }
  })
}

describe('AssistantPanel Agent 选择', () => {
  it('Select 不启用搜索且展示当前项目/全局 scope 与 role 描述', async () => {
    const user = userEvent.setup()
    renderPanel([
      { id: 'project-writer', project_id: 'project-1', name: '项目写手', role: 'writer', enabled: true },
      { id: 'global-editor', name: '全局编辑', role: 'editor', enabled: true }
    ])

    await user.click(screen.getByRole('button', { name: 'editor.assistant.agentLabel' }))

    expect(screen.queryByRole('searchbox')).not.toBeInTheDocument()
    expect(screen.getByRole('option', { name: /项目写手/ })).toHaveTextContent('editor.assistant.scopeProject')
    expect(screen.getByRole('option', { name: /全局编辑/ })).toHaveTextContent('editor.assistant.scopeGlobal')
  })

  it('真正空列表显示持久设置入口，加载失败显示重试且不伪装为空态', () => {
    const empty = renderPanel([])
    expect(screen.getByText('editor.assistant.emptyAgentsTitle')).toBeVisible()
    expect(screen.getByRole('button', { name: 'editor.assistant.openAgentSettings' })).toBeVisible()
    empty.unmount()

    renderPanel([], '网络不可用')
    expect(screen.getByText('editor.assistant.loadFailedTitle')).toBeVisible()
    expect(screen.getByText('网络不可用')).toBeVisible()
    expect(screen.queryByText('editor.assistant.emptyAgentsTitle')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'common.retry' })).toBeVisible()
  })
})

describe('AssistantPanel 流式生成与覆盖提案', () => {
  it('实时展示增量正文、工具进度并允许取消', async () => {
    const user = userEvent.setup()
    const view = renderPanel([{ id: 'agent-1', name: '写手', role: 'writer', enabled: true }], '', {
      selectedAgentId: 'agent-1',
      prompt: '继续写',
      runningAgent: true,
      streamState: {
        status: 'tool-running',
        chapterId: 'chapter-1',
        runId: 'run-1',
        content: '首段增量',
        tools: [{ call_id: 'tool-1', name: '搜索设定', status: 'started' }],
        modelResolution: null,
        error: ''
      }
    })

    expect(screen.getByTestId('agent-stream-content')).toHaveTextContent('首段增量')
    expect(screen.getByText('搜索设定')).toBeVisible()
    expect(screen.getByText('editor.stream.toolStatus.started')).toBeVisible()
    expect(screen.getByRole('status')).toHaveTextContent('editor.stream.status.tool-running')
    await user.click(screen.getByRole('button', { name: 'editor.stream.cancel' }))
    expect(view.emitted('cancelRun')).toHaveLength(1)
  })

  it('finalizing 保持运行锁并隐藏取消按钮', () => {
    renderPanel([{ id: 'agent-1', name: '写手', role: 'writer', enabled: true }], '', {
      selectedAgentId: 'agent-1',
      prompt: '继续写',
      runningAgent: true,
      streamState: {
        status: 'finalizing',
        chapterId: 'chapter-1',
        runId: 'run-finalizing',
        content: '完整结果正在整理',
        tools: [],
        modelResolution: null,
        error: ''
      }
    })

    expect(screen.getByRole('status')).toHaveTextContent('editor.stream.status.finalizing')
    expect(screen.getByRole('button', { name: 'editor.actions.runAgent' })).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'editor.stream.cancel' })).not.toBeInTheDocument()
  })

  it('失败卡保留部分文本但不提供提案应用按钮', () => {
    renderPanel([], '', {
      streamState: {
        status: 'failed',
        chapterId: 'chapter-1',
        runId: 'run-failed',
        content: '仍可查看的部分正文',
        tools: [],
        modelResolution: null,
        error: '模型连接中断'
      }
    })

    expect(screen.getByText('仍可查看的部分正文')).toBeVisible()
    expect(screen.getByText('模型连接中断')).toBeVisible()
    expect(screen.queryByRole('button', { name: 'editor.proposals.overwrite' })).not.toBeInTheDocument()
  })

  it.each(['failed', 'cancelled'] as const)('%s 且正文为空时不显示 waiting 文案', (status) => {
    renderPanel([], '', {
      streamState: {
        status,
        chapterId: 'chapter-1',
        runId: 'run-empty',
        content: '',
        tools: [],
        modelResolution: null,
        error: '已终止'
      }
    })

    expect(screen.queryByText('editor.stream.waiting')).not.toBeInTheDocument()
  })

  it('完成后的同 run 提案不重复显示流卡并可请求覆盖正文', async () => {
    const user = userEvent.setup()
    const result = {
      run: { id: 'run-1', agent_id: 'agent-1', status: 'completed' },
      content: '完整提案',
      tool_trace: [],
      model_resolution: {
        route_key: 'writer', resolution_source: 'agent', provider_id: 'provider-1', provider_name: 'Provider',
        provider_type: 'openai', model_id: 'model-1', model_name: 'Model', model_kind: 'text'
      }
    }
    const view = renderPanel([], '', {
      proposals: [{ id: 'proposal:run-1', agentId: 'agent-1', runId: 'run-1', content: '完整提案', status: 'pending', createdAt: '2026-01-01T00:00:00Z', result }],
      streamState: { status: 'completed', chapterId: 'chapter-1', runId: 'run-1', content: '完整提案', tools: [], modelResolution: result.model_resolution, error: '' }
    })

    expect(screen.queryByTestId('agent-stream-card')).not.toBeInTheDocument()
    expect(screen.getAllByText('完整提案')).toHaveLength(1)
    await user.click(screen.getByRole('button', { name: 'editor.proposals.overwrite' }))
    expect(view.emitted('overwrite')).toEqual([['proposal:run-1']])
  })
})
