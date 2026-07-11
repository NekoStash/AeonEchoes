import { expect, test, type Page, type Route } from '@playwright/test'

const now = '2026-01-01T00:00:00Z'

function envelope(data: unknown) {
  return JSON.stringify({ data, meta: { request_id: 'editor-e2e' } })
}

async function fulfill(route: Route, data: unknown, status = 200) {
  await route.fulfill({ status, contentType: 'application/json', body: envelope(data) })
}

async function mockEditorApi(page: Page, options?: { chapters?: unknown[]; versions?: unknown[]; agents?: unknown[]; agentContent?: string; agentListStatus?: number; chapterListStatus?: number; chapterUpdateStatus?: number }) {
  page.on('pageerror', (error) => console.error('[writing-editor pageerror]', error))
  page.on('console', (message) => {
    if (message.type() === 'error') console.error('[writing-editor console]', message.text())
  })
  const chapters = options?.chapters ?? [{
    id: 'chapter-1',
    project_id: 'project-1',
    number: 1,
    title: '第一章',
    status: 'drafting',
    summary: '从雨夜开始。',
    metadata: {}
  }]
  const versions = options?.versions ?? [{
    id: 'version-1',
    project_id: 'project-1',
    chapter_id: 'chapter-1',
    version: 1,
    title: '第一章',
    content: '后端正文',
    author_role: 'editor',
    index_status: 'completed',
    metadata: { change_note: '初始版本' },
    created_at: now
  }]

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname

    if (path.endsWith('/projects/project-1/story-bibles/current')) {
      return fulfill(route, {
        id: 'bible-1',
        project_id: 'project-1',
        premise: '',
        themes: [],
        world_rules: ['时间不可逆'],
        characters: [],
        foreshadows: [],
        chapter_plan: []
      })
    }
    if (path.endsWith('/projects/project-1/chapters') && request.method() === 'GET') {
      if (options?.chapterListStatus && options.chapterListStatus !== 200) {
        return route.fulfill({ status: options.chapterListStatus, contentType: 'application/json', body: JSON.stringify({ error: { code: 'chapter_list_failed', message: '章节列表加载失败' }, meta: { request_id: 'editor-e2e-error' } }) })
      }
      return fulfill(route, chapters)
    }
    if (path.endsWith('/projects/project-1/chapters/chapter-1') && request.method() === 'PUT') {
      if (options?.chapterUpdateStatus && options.chapterUpdateStatus !== 200) {
        return route.fulfill({ status: options.chapterUpdateStatus, contentType: 'application/json', body: JSON.stringify({ error: { code: 'chapter_update_failed', message: '章节标题更新失败' }, meta: { request_id: 'editor-e2e-error' } }) })
      }
      const body = request.postDataJSON()
      return fulfill(route, { ...(chapters[0] as Record<string, unknown>), ...body, updated_at: now })
    }
    if (path.endsWith('/projects/project-1/chapters/chapter-1/versions') && request.method() === 'GET') return fulfill(route, versions)
    if (path.endsWith('/projects/project-1/chapters/chapter-1/versions') && request.method() === 'POST') {
      const body = request.postDataJSON()
      return fulfill(route, {
        chapter_version: {
          id: 'version-2',
          project_id: 'project-1',
          chapter_id: 'chapter-1',
          version: 2,
          title: body.title,
          content: body.content,
          author_role: 'editor',
          index_status: 'pending',
          metadata: body.metadata || {},
          created_at: now
        },
        index_job: {
          id: 'job-1',
          project_id: 'project-1',
          chapter_id: 'chapter-1',
          chapter_version_id: 'version-2',
          kind: 'chapter-version',
          status: 'pending',
          attempts: 0,
          created_at: now,
          updated_at: now
        }
      }, 201)
    }
    if (path.endsWith('/agents') && request.method() === 'GET') {
      if (options?.agentListStatus && options.agentListStatus !== 200) {
        return route.fulfill({ status: options.agentListStatus, contentType: 'application/json', body: JSON.stringify({ error: { code: 'agent_list_failed', message: 'Agent 列表加载失败' }, meta: { request_id: 'editor-agent-error' } }) })
      }
      return fulfill(route, options?.agents ?? [{ id: 'agent-1', project_id: 'project-1', name: '写作 Agent', role: 'writer', enabled: true }])
    }
    if (path.endsWith('/agents/agent-1/runs')) {
      return fulfill(route, {
        run: { id: 'run-1', agent_id: 'agent-1', project_id: 'project-1', status: 'completed', input: {}, output: {}, created_at: now },
        content: options?.agentContent || 'AI 提案正文',
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
      })
    }
    return fulfill(route, [])
  })
}

test('无真实章节时显示项目章节空态且不触发创建请求', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  const writes: string[] = []
  await mockEditorApi(page, { chapters: [], versions: [] })
  page.on('request', (request) => {
    const path = new URL(request.url()).pathname
    if (request.method() !== 'GET' && path.includes('/api/v1/')) writes.push(`${request.method()} ${path}`)
  })

  await page.goto('/projects/project-1/editor?chapter=chapter-plan-1')
  await expect(page.getByTestId('editor-empty-chapters')).toBeVisible()
  await expect(page.getByText('项目还没有真实章节')).toBeVisible()
  expect(writes).toEqual([])
})

test('章节列表 500 显式报错并可重试，不伪装为 0 章节', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await mockEditorApi(page, { chapterListStatus: 500 })
  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  await expect(page.getByTestId('editor-chapter-load-error')).toContainText('章节列表加载失败')
  await expect(page.getByTestId('editor-empty-chapters')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '重试' })).toBeVisible()
})

test('只有全局 enabled writer 时编辑器仍有选项并自动选择', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  const agentRequests: URL[] = []
  await mockEditorApi(page, { agents: [{ id: 'global-writer', name: '全局写手', role: 'writer', enabled: true }] })
  page.on('request', (request) => {
    const url = new URL(request.url())
    if (url.pathname.endsWith('/agents')) agentRequests.push(url)
  })

  await page.goto('/projects/project-1/editor?chapter=chapter-1')

  await expect(page.getByRole('button', { name: '选择 Agent' }).first()).toContainText('全局写手')
  expect(agentRequests[0]?.searchParams.get('project_id')).toBe('project-1')
  expect(agentRequests[0]?.searchParams.get('enabled')).toBe('true')
  await page.getByRole('button', { name: '选择 Agent' }).first().click()
  await expect(page.getByRole('option', { name: /全局写手/ })).toContainText('全局')
  await expect(page.getByRole('searchbox')).toHaveCount(0)
})

test('项目 writer 优先且禁用与其他项目 Agent 不出现在编辑器', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await mockEditorApi(page, { agents: [
    { id: 'global-writer', name: '全局写手', role: 'writer', enabled: true },
    { id: 'project-writer', project_id: 'project-1', name: '项目写手', role: 'writer', enabled: true },
    { id: 'disabled-writer', project_id: 'project-1', name: '禁用写手', role: 'writer', enabled: false },
    { id: 'other-writer', project_id: 'project-2', name: '其他项目写手', role: 'writer', enabled: true }
  ] })

  await page.goto('/projects/project-1/editor?chapter=chapter-1')

  const select = page.getByRole('button', { name: '选择 Agent' }).first()
  await expect(select).toContainText('项目写手')
  await select.click()
  await expect(page.getByRole('option', { name: /项目写手/ })).toContainText('当前项目')
  await expect(page.getByRole('option', { name: /全局写手/ })).toBeVisible()
  await expect(page.getByRole('option', { name: /禁用写手/ })).toHaveCount(0)
  await expect(page.getByRole('option', { name: /其他项目写手/ })).toHaveCount(0)
})

test('Agent 空态与加载失败态保持区分', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await mockEditorApi(page, { agents: [] })
  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  await expect(page.getByText('没有可用的已启用 Agent').first()).toBeVisible()
  await expect(page.getByRole('button', { name: '打开 Agent 设置' }).first()).toBeVisible()
})

test('Agent 加载失败显示重试而不是空态', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await mockEditorApi(page, { agentListStatus: 500 })
  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  await expect(page.getByText('Agent 加载失败').first()).toBeVisible()
  await expect(page.getByText('没有可用的已启用 Agent')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '重试' }).first()).toBeVisible()
})

test('Tab 聚焦标题和正文时纸面显示无跳动的直角焦点环', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await mockEditorApi(page)
  await page.goto('/projects/project-1/editor?chapter=chapter-1')

  const paper = page.getByTestId('writing-paper')
  const title = page.locator('input[placeholder="第一章"]')
  const editor = page.getByTestId('chapter-content')
  const paperBoxBeforeFocus = await paper.boundingBox()
  const titleBoxBeforeFocus = await title.boundingBox()
  const editorBoxBeforeFocus = await editor.boundingBox()

  await page.getByRole('button', { name: '正文全屏' }).focus()
  await page.keyboard.press('Tab')
  await expect(title).toBeFocused()
  const titleFocusStyle = await title.evaluate((element) => {
    const style = getComputedStyle(element)
    return { borderRadius: style.borderRadius, outlineStyle: style.outlineStyle, backgroundColor: style.backgroundColor }
  })
  const paperFocusStyle = await paper.evaluate((element) => {
    const style = getComputedStyle(element)
    return { borderRadius: style.borderRadius, boxShadow: style.boxShadow }
  })
  expect(titleFocusStyle.borderRadius).toBe('0px')
  expect(titleFocusStyle.outlineStyle).not.toBe('none')
  expect(titleFocusStyle.backgroundColor).toBe('rgba(0, 0, 0, 0)')
  expect(paperFocusStyle.borderRadius).toBe('0px')
  expect(paperFocusStyle.boxShadow).not.toBe('none')
  expect(await paper.boundingBox()).toEqual(paperBoxBeforeFocus)
  expect(await title.boundingBox()).toEqual(titleBoxBeforeFocus)

  await page.keyboard.press('Tab')
  await expect(editor).toBeFocused()
  const editorFocusStyle = await editor.evaluate((element) => {
    const style = getComputedStyle(element)
    return { borderRadius: style.borderRadius, outlineStyle: style.outlineStyle, backgroundColor: style.backgroundColor }
  })
  expect(editorFocusStyle.borderRadius).toBe('0px')
  expect(editorFocusStyle.outlineStyle).not.toBe('none')
  expect(editorFocusStyle.backgroundColor).toBe('rgba(0, 0, 0, 0)')
  expect(await paper.boundingBox()).toEqual(paperBoxBeforeFocus)
  expect(await editor.boundingBox()).toEqual(editorBoxBeforeFocus)
})

test('桌面写作纸面、内边距与助手栏采用紧凑布局', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await mockEditorApi(page)
  await page.goto('/projects/project-1/editor?chapter=chapter-1')

  const pageShell = page.getByTestId('editor-page')
  const layout = page.getByTestId('editor-layout')
  const workspace = page.getByTestId('writing-workspace')
  const assistant = page.getByTestId('editor-assistant')
  const paper = page.getByTestId('writing-paper')
  const title = page.locator('input[placeholder="第一章"]')
  const editor = page.getByTestId('chapter-content')

  const colors = await Promise.all([pageShell, workspace, paper].map(locator => locator.evaluate(element => getComputedStyle(element).backgroundColor)))
  expect(colors[2]).not.toBe(colors[0])
  expect(colors[2]).not.toBe(colors[1])

  const paperStyle = await paper.evaluate((element) => {
    const style = getComputedStyle(element)
    return {
      paddingInline: parseFloat(style.paddingLeft),
      paddingBlock: parseFloat(style.paddingTop),
      maxWidth: parseFloat(style.maxWidth),
      borderRadius: style.borderRadius
    }
  })
  expect(paperStyle.paddingInline).toBeGreaterThanOrEqual(34)
  expect(paperStyle.paddingInline).toBeLessThanOrEqual(38)
  expect(paperStyle.paddingBlock).toBeGreaterThanOrEqual(26)
  expect(paperStyle.paddingBlock).toBeLessThanOrEqual(30)
  expect(paperStyle.maxWidth).toBe(768)
  expect(paperStyle.borderRadius).toBe('0px')

  const titleBox = await title.boundingBox()
  const editorBox = await editor.boundingBox()
  expect(titleBox).not.toBeNull()
  expect(editorBox).not.toBeNull()
  expect(editorBox!.y - (titleBox!.y + titleBox!.height)).toBeLessThanOrEqual(58)

  const editorStyle = await editor.evaluate((element) => {
    const style = getComputedStyle(element)
    return { minHeight: parseFloat(style.minHeight), lineHeight: parseFloat(style.lineHeight), fontSize: parseFloat(style.fontSize) }
  })
  expect(editorStyle.minHeight).toBeGreaterThanOrEqual(540)
  expect(editorStyle.minHeight).toBeLessThanOrEqual(548)
  expect(editorStyle.minHeight).not.toBeCloseTo(page.viewportSize()!.height * 0.62, 0)
  expect(editorStyle.lineHeight / editorStyle.fontSize).toBeCloseTo(1.9, 1)

  const layoutBox = await layout.boundingBox()
  const workspaceBox = await workspace.boundingBox()
  const assistantBox = await assistant.boundingBox()
  expect(layoutBox).not.toBeNull()
  expect(workspaceBox).not.toBeNull()
  expect(assistantBox).not.toBeNull()
  expect(assistantBox!.width).toBeGreaterThanOrEqual(334)
  expect(assistantBox!.width).toBeLessThanOrEqual(338)
  expect(assistantBox!.x - (workspaceBox!.x + workspaceBox!.width)).toBeCloseTo(16, 0)
  expect(workspaceBox!.width / assistantBox!.width).toBeGreaterThan(1.6)
})

test('标题变化先更新真实 Chapter，再创建版本并同步章节选择器', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  const calls: string[] = []
  await mockEditorApi(page)
  page.on('request', (request) => {
    const path = new URL(request.url()).pathname
    if (request.method() === 'PUT' && path.endsWith('/projects/project-1/chapters/chapter-1')) calls.push(`PUT:${request.postDataJSON().title}`)
    if (request.method() === 'POST' && path.endsWith('/projects/project-1/chapters/chapter-1/versions')) calls.push(`POST:${request.postDataJSON().title}`)
  })

  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  await page.locator('input[placeholder="第一章"]').fill('雨夜档案')
  await page.getByRole('button', { name: '创建版本' }).click()
  await expect.poll(() => calls).toEqual(['PUT:雨夜档案', 'POST:雨夜档案'])
  await expect(page.getByRole('button', { name: '当前章节' })).toContainText('雨夜档案')
})

test('章节标题更新失败时错误可见且不会创建版本', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  let versionPosts = 0
  await mockEditorApi(page, { chapterUpdateStatus: 500 })
  page.on('request', (request) => {
    if (request.method() === 'POST' && new URL(request.url()).pathname.endsWith('/projects/project-1/chapters/chapter-1/versions')) versionPosts += 1
  })
  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  await page.locator('input[placeholder="第一章"]').fill('失败标题')
  await page.getByRole('button', { name: '创建版本' }).click()
  await expect(page.getByRole('alert')).toContainText('章节标题更新失败')
  expect(versionPosts).toBe(0)
})

test('Agent Run 结果只进入提案区，用户追加后仍需手动创建版本', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  const versionPosts: unknown[] = []
  await mockEditorApi(page)
  page.on('request', (request) => {
    const path = new URL(request.url()).pathname
    if (request.method() === 'POST' && path.endsWith('/projects/project-1/chapters/chapter-1/versions')) {
      versionPosts.push(request.postDataJSON())
    }
  })

  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  const editor = page.getByTestId('chapter-content')
  await expect(editor).toHaveValue('后端正文')

  await page.getByPlaceholder('说明你希望 Agent 提供什么写作提案……').fill('继续这一幕')
  await page.getByRole('button', { name: '运行 Agent' }).click()
  await expect(page.getByText('AI 提案正文')).toBeVisible()
  await expect(editor).toHaveValue('后端正文')
  expect(versionPosts).toHaveLength(0)

  await page.getByRole('button', { name: '追加', exact: true }).click()
  await expect(editor).toHaveValue('后端正文\n\nAI 提案正文')
  expect(versionPosts).toHaveLength(0)

  await page.getByRole('button', { name: '创建版本' }).click()
  await expect.poll(() => versionPosts.length).toBe(1)
  expect(versionPosts[0]).toMatchObject({
    content: '后端正文\n\nAI 提案正文',
    parent_version_id: 'version-1'
  })
})

test('移动端保持安全文字边距、正文高度、全屏与 AI Sheet', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile-chromium')
  await mockEditorApi(page)
  await page.goto('/projects/project-1/editor?chapter=chapter-1')

  const workspace = page.getByTestId('writing-workspace')
  const paper = page.getByTestId('writing-paper')
  const title = page.locator('input[placeholder="第一章"]')
  const editor = page.getByTestId('chapter-content')
  const viewport = page.viewportSize()!
  const paperStyle = await paper.evaluate((element) => {
    const style = getComputedStyle(element)
    return { paddingInline: parseFloat(style.paddingLeft), paddingBlock: parseFloat(style.paddingTop) }
  })
  expect(paperStyle.paddingInline).toBe(16)
  expect(paperStyle.paddingBlock).toBe(20)

  const titleBox = await title.boundingBox()
  const editorBox = await editor.boundingBox()
  expect(titleBox).not.toBeNull()
  expect(editorBox).not.toBeNull()
  expect(titleBox!.x).toBeGreaterThanOrEqual(29)
  expect(titleBox!.x).toBeLessThanOrEqual(32)
  expect(viewport.width - titleBox!.x - titleBox!.width).toBeGreaterThanOrEqual(29)
  expect(viewport.width - titleBox!.x - titleBox!.width).toBeLessThanOrEqual(32)
  expect(editorBox!.y - (titleBox!.y + titleBox!.height)).toBeLessThanOrEqual(58)

  const editorMinHeight = await editor.evaluate(element => parseFloat(getComputedStyle(element).minHeight))
  expect(editorMinHeight).toBeCloseTo(viewport.height * 0.42, 0)
  expect(editorMinHeight).not.toBeCloseTo(viewport.height * 0.62, 0)
  const colors = await Promise.all([workspace, paper].map(locator => locator.evaluate(element => getComputedStyle(element).backgroundColor)))
  expect(colors[1]).not.toBe(colors[0])

  await page.getByRole('button', { name: 'AI 助手' }).first().click()
  await expect(page.getByRole('dialog', { name: 'AI 助手' })).toBeVisible()
  await page.keyboard.press('Escape')
  await page.getByRole('button', { name: '正文全屏' }).click()
  await expect(workspace).toHaveClass(/fixed/)

  const fullscreenTitleBox = await title.boundingBox()
  expect(fullscreenTitleBox).not.toBeNull()
  expect(fullscreenTitleBox!.x).toBeGreaterThanOrEqual(29)
  expect(fullscreenTitleBox!.x).toBeLessThanOrEqual(32)
})
