import { expect, test, type Page, type Route } from '@playwright/test'

const now = '2026-01-01T00:00:00Z'

function envelope(data: unknown) {
  return JSON.stringify({ data, meta: { request_id: 'editor-e2e' } })
}

async function fulfill(route: Route, data: unknown, status = 200) {
  await route.fulfill({ status, contentType: 'application/json', body: envelope(data) })
}

async function mockEditorApi(page: Page, options?: { chapters?: unknown[]; versions?: unknown[]; agentContent?: string; chapterListStatus?: number; chapterUpdateStatus?: number }) {
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
      return fulfill(route, [{ id: 'agent-1', project_id: 'project-1', name: '写作 Agent', role: 'writer', enabled: true }])
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

test('移动端正文全屏且 AI 使用 Sheet', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile-chromium')
  await mockEditorApi(page)
  await page.goto('/projects/project-1/editor?chapter=chapter-1')

  await page.getByRole('button', { name: 'AI 助手' }).first().click()
  await expect(page.getByRole('dialog', { name: 'AI 助手' })).toBeVisible()
  await page.keyboard.press('Escape')
  await page.getByRole('button', { name: '正文全屏' }).click()
  await expect(page.getByTestId('writing-workspace')).toHaveClass(/fixed/)
})
