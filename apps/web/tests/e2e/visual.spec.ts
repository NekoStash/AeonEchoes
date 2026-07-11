import { expect, test, type Page, type Route } from '@playwright/test'

const now = '2026-01-01T00:00:00Z'
const project = {
  id: 'project-1',
  title: '墨色档案',
  slug: 'ink-archive',
  status: 'active',
  logline: '记录员发现公共档案与私人记忆冲突。',
  tags: ['记忆', '真相'],
  active_story_bible_id: 'bible-1',
  created_at: now,
  updated_at: now,
  bible_status: 'ready',
  chapter_count: 1
}
const bible = {
  id: 'bible-1',
  project_id: 'project-1',
  title: '墨色档案',
  premise: '记录员发现公共档案与私人记忆冲突。',
  themes: ['记忆', '真相'],
  world_rules: ['档案只记录可被证实的事件。'],
  characters: [],
  foreshadows: [],
  chapter_plan: []
}
const chapter = {
  id: 'chapter-1',
  project_id: 'project-1',
  number: 1,
  title: '雨夜档案',
  status: 'drafting',
  summary: '记录员在雨夜发现第一份冲突档案。',
  metadata: {},
  created_at: now,
  updated_at: now
}
const version = {
  id: 'version-1',
  project_id: 'project-1',
  chapter_id: 'chapter-1',
  version: 1,
  title: '雨夜档案',
  content: '雨水沿着档案馆的窗格向下滑落。\n\n林澈翻开那份不该存在的记录。',
  author_role: 'editor',
  index_status: 'completed',
  metadata: { change_note: '初始版本' },
  created_at: now
}
const visualProviders = [
  { id: 'provider-1', name: 'Primary Ink', provider_type: 'openai-responses', type: 'openai-responses', base_url: 'https://example.test/v1', enabled: true, streaming: true, status: 'online', api_key_hint: 'sk-…1234' },
  { id: 'provider-2', name: 'Secondary Archive', provider_type: 'anthropic', type: 'anthropic', base_url: 'https://secondary.example.test', enabled: true, streaming: true, status: 'online', api_key_hint: 'sk-…5678' }
]
const visualModels = [
  ...Array.from({ length: 14 }, (_, index) => ({
    id: `provider-${index % 2 + 1}:visual-${index + 1}`,
    provider_id: `provider-${index % 2 + 1}`,
    name: `visual-${index + 1}`,
    display_name: `叙事模型 ${String(index + 1).padStart(2, '0')}`,
    kind: 'text',
    enabled: index !== 11,
    context_window: 32000 + index * 16000,
    max_output_tokens: 2048 + index * 512,
    supports_tools: index % 3 !== 0,
    supports_streaming: index % 4 !== 0,
    default_for_kind: index === 0,
    cost_input_per_mtok: index + 0.25,
    cost_output_per_mtok: index + 1.25,
    routing_weight: 100 - index,
    allowed_agent_roles: index === 2 ? ['editor', 'continuity-auditor'] : []
  })),
  { id: 'provider-1:visual-embedding', provider_id: 'provider-1', name: 'visual-embedding', display_name: '档案向量模型', kind: 'embedding', enabled: true, dimension: 1536, default_for_kind: true, routing_weight: 100, cost_input_per_mtok: 0.1, cost_output_per_mtok: 0, allowed_agent_roles: [] },
  { id: 'provider-2:old-embedding', provider_id: 'provider-2', name: 'old-embedding', display_name: '停用向量模型', kind: 'embedding', enabled: false, dimension: 3072, routing_weight: 5, allowed_agent_roles: [] }
]

function envelope(data: unknown) {
  return JSON.stringify({ data, meta: { request_id: 'visual' }, page: { count: Array.isArray(data) ? data.length : 1 } })
}

async function fulfill(route: Route, data: unknown, status = 200) {
  await route.fulfill({ status, contentType: 'application/json', body: envelope(data) })
}

async function mockApi(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('aeon-echoes:opened-projects', JSON.stringify([{
      id: 'project-1', title: '墨色档案', slug: 'ink-archive', status: 'active', logline: '记录员发现公共档案与私人记忆冲突。', tags: ['记忆', '真相'], active_story_bible_id: 'bible-1', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z', bible_status: 'ready', chapter_count: 1
    }]))
  })
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname
    if (path.endsWith('/health')) return fulfill(route, { status: 'ok', time: now, postgres_configured: true, qdrant_configured: true })
    if (path.endsWith('/system/status')) return fulfill(route, { status: 'ok', postgres_configured: true, qdrant_configured: true, provider_count: 1, model_count: 1, pending_jobs_count: 0, checked_at: now })
    if (path.endsWith('/projects')) return fulfill(route, [project])
    if (path.endsWith('/projects/project-1/story-bibles/current')) return fulfill(route, bible)
    if (path.endsWith('/projects/project-1/chapters')) return fulfill(route, [chapter])
    if (path.endsWith('/projects/project-1/chapters/chapter-1/versions')) return fulfill(route, [version])
    if (path.endsWith('/providers')) return fulfill(route, visualProviders)
    if (path.endsWith('/models')) return fulfill(route, visualModels)
    if (path.endsWith('/model-routing')) return fulfill(route, { routes: { writer: 'provider-1:visual-1', editor: '', 'genesis-optimizer': '', 'plot-architect': '', 'world-builder': '', 'character-keeper': '', 'continuity-auditor': '', 'fact-extractor': '', 'graph-curator': '', embedding: 'provider-1:visual-embedding' } })
    if (path.endsWith('/agents') && request.method() === 'GET') return fulfill(route, [{ id: 'agent-1', project_id: 'project-1', name: '写作 Agent', role: 'writer', enabled: true }])
    if (path.endsWith('/skills') || path.endsWith('/mcp-servers') || path.endsWith('/tools') || path.endsWith('/index-jobs')) return fulfill(route, [])
    if (path.endsWith('/projects/project-1/graph/expansions')) return fulfill(route, {
      project_id: 'project-1', depth: 2,
      entities: [
        { id: 'entity-a', project_id: 'project-1', name: '林澈', label: '林澈', type: 'character', summary: '档案记录员', importance: 1, status: 'stable', metadata: { timeline: '1', depth: '1' }, created_at: now, updated_at: now },
        { id: 'entity-b', project_id: 'project-1', name: '第七档案馆', label: '第七档案馆', type: 'location', summary: '保存冲突档案的地点', importance: 0.8, status: 'stable', created_at: now, updated_at: now }
      ],
      edges: [{ id: 'edge-1', project_id: 'project-1', source_entity_id: 'entity-a', target_entity_id: 'entity-b', type: 'located_at', label: '任职于', weight: 1, created_at: now, updated_at: now }],
      facts: [],
      generated_at: now
    })
    return fulfill(route, [])
  })
}

async function capture(page: Page, name: string) {
  await expect(page.locator('#main-content')).toBeAttached()
  await expect(page).toHaveScreenshot(name, { fullPage: true, animations: 'disabled', caret: 'hide' })
}

async function captureWorkspace(page: Page, name: string) {
  const workspace = page.getByTestId('writing-workspace')
  await expect(workspace).toBeVisible()
  const clip = await workspace.boundingBox()
  if (!clip) throw new Error('Writing workspace has no visual bounds.')
  await expect(page).toHaveScreenshot(name, { clip, animations: 'disabled', caret: 'hide' })
}

async function readyEditor(page: Page) {
  const workspace = page.getByTestId('writing-workspace')
  const title = page.getByRole('textbox', { name: '章节标题' })
  const content = page.getByRole('textbox', { name: '正文' })
  await expect(workspace).toBeVisible()
  await expect(title).toBeVisible()
  await expect(content).toBeVisible()
  await expect(page.getByTestId('editor-assistant').getByText('写作 Agent')).toBeAttached()

  const surfaceColors = await Promise.all([
    page.getByTestId('chapter-title-surface'),
    page.getByTestId('chapter-content-surface')
  ].map(locator => locator.evaluate(element => getComputedStyle(element).backgroundColor)))
  for (const color of surfaceColors) expect(color).not.toBe('rgba(0, 0, 0, 0)')
}

test.beforeEach(async ({ page }) => {
  await mockApi(page)
})

const cases = [
  { name: 'home', path: '/', ready: (page: Page) => expect(page.getByRole('button', { name: /墨色档案/ })).toBeVisible() },
  { name: 'projects', path: '/projects', ready: (page: Page) => expect(page.getByRole('heading', { name: '项目库', exact: true })).toBeVisible() },
  { name: 'project-workspace', path: '/projects/project-1', ready: (page: Page) => expect(page.getByRole('heading', { name: '墨色档案' })).toBeVisible() },
  { name: 'editor', path: '/projects/project-1/editor?chapter=chapter-1', ready: readyEditor },
  { name: 'settings', path: '/settings/providers', ready: (page: Page) => expect(page.getByRole('heading', { name: '提供商连接' })).toBeVisible() },
  { name: 'models-settings', path: '/settings/models', ready: (page: Page) => expect(page.getByText('叙事模型 14')).toBeAttached() },
  { name: 'graph', path: '/projects/project-1/graph', ready: async (page: Page) => {
    await expect(page.getByText('林澈', { exact: true })).toBeVisible()
    await expect(page.getByRole('button').filter({ hasText: '第七档案馆' })).toContainText('— · —')
    await expect(page.getByText(/invalid_api_response/)).toHaveCount(0)
  } }
]

for (const item of cases) {
  test(`@visual ${item.name} desktop`, async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'chromium')
    await page.goto(item.path, { waitUntil: 'domcontentloaded' })
    await item.ready(page)
    await capture(page, `${item.name}-desktop.png`)
  })

  test(`@visual ${item.name} mobile`, async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'mobile-chromium')
    await page.goto(item.path, { waitUntil: 'domcontentloaded' })
    await item.ready(page)
    await capture(page, `${item.name}-mobile.png`)
  })
}

test('@visual editor dark desktop workspace', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.emulateMedia({ colorScheme: 'dark' })
  await page.goto('/projects/project-1/editor?chapter=chapter-1', { waitUntil: 'domcontentloaded' })
  await readyEditor(page)
  await expect(page.locator('html')).toHaveClass(/dark/)
  await captureWorkspace(page, 'editor-dark-desktop.png')
})

test('@visual editor single content focus desktop', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/projects/project-1/editor?chapter=chapter-1', { waitUntil: 'domcontentloaded' })
  await readyEditor(page)

  const paper = page.getByTestId('writing-paper')
  const title = page.getByRole('textbox', { name: '章节标题' })
  const content = page.getByRole('textbox', { name: '正文' })
  const paperShadow = await paper.evaluate(element => getComputedStyle(element).boxShadow)

  await title.focus()
  await expect(title).toBeFocused()
  expect(await title.evaluate(element => getComputedStyle(element).outlineStyle)).not.toBe('none')
  expect(await paper.evaluate(element => getComputedStyle(element).boxShadow)).toBe(paperShadow)

  await content.focus()
  await expect(content).toBeFocused()
  expect(await content.evaluate(element => getComputedStyle(element).outlineStyle)).not.toBe('none')
  expect(await paper.evaluate(element => getComputedStyle(element).boxShadow)).toBe(paperShadow)
  await captureWorkspace(page, 'editor-content-focus-desktop.png')
})

test('@visual editor fullscreen mobile landscape', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile-chromium')
  await page.setViewportSize({ width: 915, height: 412 })
  await page.goto('/projects/project-1/editor?chapter=chapter-1', { waitUntil: 'domcontentloaded' })
  await readyEditor(page)

  await page.getByRole('button', { name: '正文全屏' }).click()
  const workspace = page.getByTestId('writing-workspace')
  await expect(workspace).toHaveClass(/fixed/)
  await expect(page.getByRole('button', { name: '退出全屏' })).toBeVisible()
  await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur())
  await captureWorkspace(page, 'editor-fullscreen-landscape.png')
})
