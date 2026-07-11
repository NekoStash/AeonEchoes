import { expect, test, type Page, type Route } from '@playwright/test'

const now = '2026-01-01T00:00:00Z'
const pageErrors = new WeakMap<Page, string[]>()

test.beforeEach(async ({ page }) => {
  const errors: string[] = []
  pageErrors.set(page, errors)
  page.on('pageerror', (error) => {
    errors.push(`${error.name}: ${error.message}`)
    console.error('[writing-editor pageerror]', error)
  })
})

test.afterEach(async ({ page }) => {
  const producerErrors = await page.evaluate(() => {
    return (window as Window & { __agentStreamProducerErrors?: string[] }).__agentStreamProducerErrors || []
  })
  expect(producerErrors, 'stream producer errors').toEqual([])
  expect(pageErrors.get(page) || [], 'unhandled page errors').toEqual([])
})

function envelope(data: unknown) {
  return JSON.stringify({ data, meta: { request_id: 'editor-e2e' } })
}

async function fulfill(route: Route, data: unknown, status = 200) {
  await route.fulfill({ status, contentType: 'application/json', body: envelope(data) })
}

function streamEvent(name: string, data: Record<string, unknown>) {
  return `event: ${name}\r\ndata: ${JSON.stringify(data)}\r\n\r\n`
}

function agentResult(content: string, runId = 'run-1') {
  return {
    run: { id: runId, agent_id: 'agent-1', project_id: 'project-1', status: 'completed', input: {}, output: {}, created_at: now },
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

function completedStream(content: string, runId = 'run-1') {
  return [
    streamEvent('run.started', { type: 'run.started', sequence: 1, run_id: runId, run: { id: runId, agent_id: 'agent-1', project_id: 'project-1', status: 'running' } }),
    streamEvent('content.delta', { type: 'content.delta', sequence: 2, run_id: runId, delta: content }),
    streamEvent('run.completed', { type: 'run.completed', sequence: 3, run_id: runId, result: agentResult(content, runId) })
  ].join('')
}

async function installChunkedAgentStream(
  page: Page,
  chunks: Array<{ text: string; delay?: number }>,
  ignoreAbort = false,
  cancelDelay = 0
) {
  await page.addInitScript(({ serializedChunks, shouldIgnoreAbort, readerCancelDelay }) => {
    const testWindow = window as Window & { __agentStreamFetchCount?: number; __agentStreamProducerErrors?: string[] }
    testWindow.__agentStreamFetchCount = 0
    testWindow.__agentStreamProducerErrors = []
    const nativeFetch = window.fetch.bind(window)
    window.fetch = async (input, init) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url
      if (!url.endsWith('/agents/agent-1/runs/stream')) return nativeFetch(input, init)
      testWindow.__agentStreamFetchCount = (testWindow.__agentStreamFetchCount || 0) + 1
      const encoder = new TextEncoder()
      const signal = init?.signal
      let consumerCancelled = false
      return new Response(new ReadableStream({
        start(controller) {
          let signalAborted = false
          if (!shouldIgnoreAbort && signal) {
            signal.addEventListener('abort', () => {
              signalAborted = true
              controller.error(new DOMException('Aborted', 'AbortError'))
            }, { once: true })
          }
          void (async () => {
            for (const chunk of serializedChunks) {
              if (chunk.delay) await new Promise(resolve => setTimeout(resolve, chunk.delay))
              if (signalAborted || consumerCancelled) return
              controller.enqueue(encoder.encode(chunk.text))
            }
            if (!signalAborted && !consumerCancelled && readerCancelDelay === 0) controller.close()
          })().catch((error) => {
            const message = error instanceof Error ? `${error.name}: ${error.message}` : String(error)
            testWindow.__agentStreamProducerErrors?.push(message)
            console.error('[writing-editor stream producer]', error)
          })
        },
        async cancel() {
          consumerCancelled = true
          if (readerCancelDelay > 0) await new Promise(resolve => setTimeout(resolve, readerCancelDelay))
        }
      }), { status: 200, headers: { 'content-type': 'text/event-stream; charset=utf-8' } })
    }
  }, { serializedChunks: chunks, shouldIgnoreAbort: ignoreAbort, readerCancelDelay: cancelDelay })
}

async function mockEditorApi(page: Page, options?: { chapters?: unknown[]; versions?: unknown[]; agents?: unknown[]; agentContent?: string; agentListStatus?: number; chapterListStatus?: number; chapterUpdateStatus?: number }) {
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
    if (path.endsWith('/agents/agent-1/runs/stream')) {
      return route.fulfill({
        status: 200,
        contentType: 'text/event-stream; charset=utf-8',
        body: completedStream(options?.agentContent || 'AI 提案正文')
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

test('标题与正文使用独立非透明 surface，并仅在当前字段显示单一焦点提示', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await mockEditorApi(page)
  await page.goto('/projects/project-1/editor?chapter=chapter-1')

  const titleSurface = page.getByTestId('chapter-title-surface')
  const contentSurface = page.getByTestId('chapter-content-surface')
  const title = page.getByRole('textbox', { name: '章节标题' })
  const editor = page.getByRole('textbox', { name: '正文' })
  const titleSurfaceBox = await titleSurface.boundingBox()
  const contentSurfaceBox = await contentSurface.boundingBox()
  const surfaceStyles = await Promise.all([titleSurface, contentSurface].map(locator => locator.evaluate((element) => {
    const style = getComputedStyle(element)
    return { backgroundColor: style.backgroundColor, boxShadow: style.boxShadow, borderRadius: style.borderRadius }
  })))
  const titleSurfaceStyle = surfaceStyles[0]!
  const contentSurfaceStyle = surfaceStyles[1]!

  expect(titleSurfaceStyle.backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
  expect(contentSurfaceStyle.backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
  expect(titleSurfaceStyle.backgroundColor).not.toBe(contentSurfaceStyle.backgroundColor)
  expect(titleSurfaceStyle.borderRadius).toBe('0px')
  expect(contentSurfaceStyle.borderRadius).toBe('0px')

  await page.getByRole('button', { name: '正文全屏' }).focus()
  await page.keyboard.press('Tab')
  await expect(title).toBeFocused()
  expect(await title.evaluate(element => getComputedStyle(element).outlineStyle)).not.toBe('none')
  expect(await contentSurface.evaluate(element => getComputedStyle(element).boxShadow)).toBe(contentSurfaceStyle.boxShadow)
  expect(await titleSurface.boundingBox()).toEqual(titleSurfaceBox)

  await page.keyboard.press('Tab')
  await expect(editor).toBeFocused()
  expect(await editor.evaluate(element => getComputedStyle(element).outlineStyle)).not.toBe('none')
  expect(await titleSurface.evaluate(element => getComputedStyle(element).boxShadow)).toBe(titleSurfaceStyle.boxShadow)
  expect(await contentSurface.boundingBox()).toEqual(contentSurfaceBox)
})

test('桌面独立写作 surface、正文高度与助手栏采用紧凑布局', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await mockEditorApi(page)
  await page.goto('/projects/project-1/editor?chapter=chapter-1')

  const layout = page.getByTestId('editor-layout')
  const workspace = page.getByTestId('writing-workspace')
  const writingSurface = page.getByTestId('writing-surface')
  const assistant = page.getByTestId('editor-assistant')
  const paper = page.getByTestId('writing-paper')
  const titleSurface = page.getByTestId('chapter-title-surface')
  const contentSurface = page.getByTestId('chapter-content-surface')
  const editor = page.getByRole('textbox', { name: '正文' })

  const surfacePadding = await Promise.all([titleSurface, contentSurface].map(locator => locator.evaluate((element) => {
    const style = getComputedStyle(element)
    return { inline: parseFloat(style.paddingLeft), block: parseFloat(style.paddingTop), background: style.backgroundColor }
  })))
  const titleSurfacePadding = surfacePadding[0]!
  const contentSurfacePadding = surfacePadding[1]!
  expect(titleSurfacePadding.inline).toBe(24)
  expect(contentSurfacePadding.inline).toBe(24)
  expect(titleSurfacePadding.block).toBe(24)
  expect(contentSurfacePadding.block).toBe(24)
  expect(titleSurfacePadding.background).not.toBe(contentSurfacePadding.background)
  expect(await paper.evaluate(element => parseFloat(getComputedStyle(element).maxWidth))).toBe(768)
  expect(await writingSurface.evaluate(element => parseFloat(getComputedStyle(element).paddingLeft))).toBe(20)

  const editorStyle = await editor.evaluate((element) => {
    const style = getComputedStyle(element)
    return { minHeight: parseFloat(style.minHeight), lineHeight: parseFloat(style.lineHeight), fontSize: parseFloat(style.fontSize) }
  })
  const expectedMinHeight = Math.min(672, Math.max(448, page.viewportSize()!.height * 0.58))
  expect(editorStyle.minHeight).toBeCloseTo(expectedMinHeight, 0)
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

test('多 chunk 首段实时显示、完成原位转 pending，覆盖只写本地草稿且手动保存', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  const writes: Array<{ method: string; path: string; body: unknown }> = []
  const runId = 'run-stream-1'
  const result = agentResult('首段续写完成', runId)
  await installChunkedAgentStream(page, [
    { text: streamEvent('run.started', { type: 'run.started', sequence: 1, run_id: runId, run: { id: runId, agent_id: 'agent-1', status: 'running' } }) },
    { text: streamEvent('content.delta', { type: 'content.delta', sequence: 2, run_id: runId, delta: '首段' }) },
    { text: streamEvent('content.delta', { type: 'content.delta', sequence: 3, run_id: runId, delta: '续写完成' }), delay: 700 },
    { text: `${streamEvent('run.completed', { type: 'run.completed', sequence: 4, run_id: runId, result })}${streamEvent('mystery.event', { type: 'mystery.event', sequence: 5, run_id: runId })}` }
  ])
  await mockEditorApi(page)
  page.on('request', (request) => {
    const path = new URL(request.url()).pathname
    if ((request.method() === 'PUT' || request.method() === 'POST') && path.includes('/projects/project-1/chapters/')) {
      writes.push({ method: request.method(), path, body: request.postDataJSON() })
    }
  })

  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  await expect(page.getByTestId('assistant-panel')).toHaveCount(1)
  const editor = page.getByTestId('chapter-content')
  const title = page.locator('input[placeholder="第一章"]')
  await expect(editor).toHaveValue('后端正文')

  await page.getByPlaceholder('说明你希望 Agent 提供什么写作提案……').fill('继续这一幕')
  await page.getByRole('button', { name: '运行 Agent' }).click()
  await expect(page.getByTestId('agent-stream-content')).toHaveText('首段')
  await expect(page.getByTestId('agent-stream-content')).not.toContainText('续写完成')
  await expect(editor).toHaveValue('后端正文')

  await expect(page.getByText('首段续写完成')).toBeVisible()
  await expect(page.getByText('生成失败', { exact: true })).toHaveCount(0)
  await expect(page.getByTestId('agent-stream-card')).toHaveCount(0)
  await page.getByRole('button', { name: '覆盖当前正文' }).click()
  const confirm = page.getByRole('dialog', { name: '覆盖当前正文？' })
  await expect(confirm).toContainText('不改标题')
  await expect(confirm).toContainText('不会自动创建版本')
  await confirm.getByRole('button', { name: '确认覆盖正文' }).click()

  await expect(editor).toHaveValue('首段续写完成')
  await expect(confirm).toHaveCount(0)
  await expect.poll(() => page.evaluate(() => (document.activeElement as HTMLElement | null)?.dataset.testid || '')).toBe('chapter-content')
  await expect(title).toHaveValue('第一章')
  expect(writes).toHaveLength(0)
  await expect.poll(() => page.evaluate(() => JSON.parse(localStorage.getItem('aeon-echoes:chapter-draft:v2:project-1:chapter-1') || 'null')?.content)).toBe('首段续写完成')

  await page.getByRole('button', { name: '创建版本' }).click()
  await expect.poll(() => writes.length).toBe(1)
  expect(writes[0]).toMatchObject({
    method: 'POST',
    body: { content: '首段续写完成', parent_version_id: 'version-1' }
  })
})

test('completed 已到但 reader.cancel 延迟期间保持运行锁，Promise 返回并提交 proposal 后才解锁', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  const runId = 'run-finalizing'
  const result = agentResult('等待清理完成', runId)
  await installChunkedAgentStream(page, [
    { text: streamEvent('run.started', { type: 'run.started', sequence: 1, run_id: runId, run: { id: runId, agent_id: 'agent-1', status: 'running' } }) },
    { text: streamEvent('content.delta', { type: 'content.delta', sequence: 2, run_id: runId, delta: '等待清理完成' }) },
    { text: streamEvent('run.completed', { type: 'run.completed', sequence: 3, run_id: runId, result }) }
  ], false, 700)
  await mockEditorApi(page)

  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  await page.getByPlaceholder('说明你希望 Agent 提供什么写作提案……').fill('测试最终清理锁')
  const runButton = page.getByRole('button', { name: '运行 Agent' })
  await runButton.click()
  await expect(page.getByTestId('agent-stream-content')).toHaveText('等待清理完成')
  await expect(page.getByTestId('agent-stream-card')).toBeVisible()
  await expect(page.getByTestId('agent-stream-card').getByRole('status')).toHaveText('正在整理提案')
  await expect(page.getByRole('button', { name: '取消生成' })).toHaveCount(0)
  await expect(runButton).toBeDisabled()
  await runButton.evaluate((button: HTMLButtonElement) => button.click())
  expect(await page.evaluate(() => (window as Window & { __agentStreamFetchCount?: number }).__agentStreamFetchCount)).toBe(1)
  await expect(page.getByRole('button', { name: '覆盖当前正文' })).toBeVisible()
  await expect(page.getByTestId('agent-stream-card')).toHaveCount(0)
  await expect(runButton).toBeEnabled()
  expect(await page.evaluate(() => (window as Window & { __agentStreamFetchCount?: number }).__agentStreamFetchCount)).toBe(1)
})

test('run.failed 保留部分文本但 Promise 失败不会留下可应用提案', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  const runId = 'run-failed'
  await installChunkedAgentStream(page, [
    { text: streamEvent('run.started', { type: 'run.started', sequence: 1, run_id: runId, run: { id: runId, agent_id: 'agent-1', status: 'running' } }) },
    { text: streamEvent('content.delta', { type: 'content.delta', sequence: 2, run_id: runId, delta: '失败前部分文本' }) },
    { text: streamEvent('run.failed', { type: 'run.failed', sequence: 3, run_id: runId, error: '模型流失败' }) }
  ])
  await mockEditorApi(page)

  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  await page.getByPlaceholder('说明你希望 Agent 提供什么写作提案……').fill('触发失败')
  await page.getByRole('button', { name: '运行 Agent' }).click()

  await expect(page.getByText('失败前部分文本')).toBeVisible()
  await expect(page.getByText('生成失败', { exact: true })).toBeVisible()
  await expect(page.getByText('模型流失败')).toBeVisible()
  await expect(page.getByRole('button', { name: '覆盖当前正文' })).toHaveCount(0)
})

test('取消后保留部分文本但不可应用，章节切换隔离迟到旧流事件', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  const chapters = [
    { id: 'chapter-1', project_id: 'project-1', number: 1, title: '第一章', status: 'drafting', summary: '从雨夜开始。', metadata: {} },
    { id: 'chapter-2', project_id: 'project-1', number: 2, title: '第二章', status: 'drafting', summary: '转入清晨。', metadata: {} }
  ]
  const runId = 'run-stale'
  await installChunkedAgentStream(page, [
    { text: streamEvent('run.started', { type: 'run.started', sequence: 1, run_id: runId, run: { id: runId, agent_id: 'agent-1', status: 'running' } }) },
    { text: streamEvent('content.delta', { type: 'content.delta', sequence: 2, run_id: runId, delta: '旧章部分文本' }) },
    { text: streamEvent('content.delta', { type: 'content.delta', sequence: 3, run_id: runId, delta: '不应污染新章' }), delay: 700 },
    { text: streamEvent('run.completed', { type: 'run.completed', sequence: 4, run_id: runId, result: agentResult('旧章部分文本不应污染新章', runId) }) }
  ], true)
  await mockEditorApi(page, { chapters })

  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  await page.getByPlaceholder('说明你希望 Agent 提供什么写作提案……').fill('继续这一幕')
  await page.getByRole('button', { name: '运行 Agent' }).click()
  await expect(page.getByTestId('agent-stream-content')).toHaveText('旧章部分文本')
  await page.getByRole('button', { name: '取消生成' }).click()
  await expect(page.getByText('生成已取消', { exact: true })).toBeVisible()
  await expect(page.getByText('旧章部分文本')).toBeVisible()
  await expect(page.getByRole('button', { name: '覆盖当前正文' })).toHaveCount(0)

  await page.getByRole('button', { name: '当前章节' }).click()
  await page.getByRole('option', { name: /第二章/ }).click()
  await expect(page).toHaveURL(/chapter=chapter-2/)
  await expect(page.locator('input[placeholder="第二章"]')).toBeVisible()
  await page.waitForTimeout(900)

  await expect(page.getByText('旧章部分文本')).toHaveCount(0)
  await expect(page.getByText('旧章部分文本不应污染新章')).toHaveCount(0)
  await expect(page.getByTestId('chapter-content')).toHaveValue('')
})

test('移动 Sheet 与桌面 resize 始终只有一份 AssistantPanel，覆盖确认关闭后焦点稳定回正文', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile-chromium')
  const mobileViewport = page.viewportSize()!
  await mockEditorApi(page)
  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  await expect(page.getByTestId('assistant-panel')).toHaveCount(1)

  await page.getByRole('button', { name: 'AI 助手' }).first().click()
  const sheet = page.getByRole('dialog', { name: 'AI 助手' })
  await expect(sheet).toBeVisible()
  await expect(page.getByTestId('assistant-panel')).toHaveCount(1)

  await page.setViewportSize({ width: 1280, height: 900 })
  await expect(sheet).toBeVisible()
  await expect(page.getByTestId('assistant-panel')).toHaveCount(1)
  await page.setViewportSize(mobileViewport)
  await expect(page.getByTestId('assistant-panel')).toHaveCount(1)

  await page.getByPlaceholder('说明你希望 Agent 提供什么写作提案……').fill('覆盖正文')
  await page.getByRole('button', { name: '运行 Agent' }).click()
  await expect(page.getByText('AI 提案正文')).toBeVisible()
  await page.getByRole('button', { name: '覆盖当前正文' }).click()

  const confirm = page.getByRole('dialog', { name: '覆盖当前正文？' })
  await expect(confirm).toBeVisible()
  await expect(page.getByTestId('assistant-panel')).toHaveCount(1)
  await confirm.getByRole('button', { name: '确认覆盖正文' }).click()
  await expect(confirm).toHaveCount(0)
  await expect(page.getByTestId('chapter-content')).toHaveValue('AI 提案正文')
  await expect.poll(() => page.evaluate(() => (document.activeElement as HTMLElement | null)?.dataset.testid || '')).toBe('chapter-content')
})

test('移动端保持安全文字边距、正文高度、全屏与 AI Sheet', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile-chromium')
  await mockEditorApi(page)
  await page.goto('/projects/project-1/editor?chapter=chapter-1')
  await expect(page.getByTestId('assistant-panel')).toHaveCount(1)

  const workspace = page.getByTestId('writing-workspace')
  const writingSurface = page.getByTestId('writing-surface')
  const titleSurface = page.getByTestId('chapter-title-surface')
  const contentSurface = page.getByTestId('chapter-content-surface')
  const title = page.getByRole('textbox', { name: '章节标题' })
  const editor = page.getByRole('textbox', { name: '正文' })
  const viewport = page.viewportSize()!
  expect(await writingSurface.evaluate(element => parseFloat(getComputedStyle(element).paddingLeft))).toBe(12)
  expect(await titleSurface.evaluate(element => parseFloat(getComputedStyle(element).paddingLeft))).toBe(16)
  expect(await contentSurface.evaluate(element => parseFloat(getComputedStyle(element).paddingLeft))).toBe(16)

  const titleBox = await title.boundingBox()
  const editorBox = await editor.boundingBox()
  expect(titleBox).not.toBeNull()
  expect(editorBox).not.toBeNull()
  expect(titleBox!.x).toBeGreaterThanOrEqual(28)
  expect(viewport.width - titleBox!.x - titleBox!.width).toBeGreaterThanOrEqual(28)
  expect(editorBox!.x).toBeGreaterThanOrEqual(28)
  expect(viewport.width - editorBox!.x - editorBox!.width).toBeGreaterThanOrEqual(28)

  const editorMinHeight = await editor.evaluate(element => parseFloat(getComputedStyle(element).minHeight))
  expect(editorMinHeight).toBeCloseTo(Math.min(576, Math.max(288, viewport.height * 0.48)), 0)
  const colors = await Promise.all([titleSurface, contentSurface].map(locator => locator.evaluate(element => getComputedStyle(element).backgroundColor)))
  expect(colors[0]).not.toBe('rgba(0, 0, 0, 0)')
  expect(colors[1]).not.toBe('rgba(0, 0, 0, 0)')
  expect(colors[0]).not.toBe(colors[1])

  await page.getByRole('button', { name: 'AI 助手' }).first().click()
  await expect(page.getByRole('dialog', { name: 'AI 助手' })).toBeVisible()
  await expect(page.getByTestId('assistant-panel')).toHaveCount(1)
  await page.keyboard.press('Escape')
  await page.getByRole('button', { name: '正文全屏' }).click()
  await expect(workspace).toHaveClass(/fixed/)

  const fullscreenTitleBox = await title.boundingBox()
  expect(fullscreenTitleBox).not.toBeNull()
  expect(fullscreenTitleBox!.x).toBeGreaterThanOrEqual(28)
  expect(viewport.width - fullscreenTitleBox!.x - fullscreenTitleBox!.width).toBeGreaterThanOrEqual(28)
})
