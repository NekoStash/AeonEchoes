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

function renderPanel(agents: AgentConfig[], agentLoadError = '') {
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
      loadingVersions: false,
      diagnostics: { modelResolution: null, toolTrace: [] }
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
