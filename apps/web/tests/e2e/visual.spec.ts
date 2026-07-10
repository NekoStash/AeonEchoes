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
    if (path.endsWith('/providers')) return fulfill(route, [{ id: 'provider-1', name: 'Primary', provider_type: 'openai-responses', type: 'openai-responses', base_url: 'https://example.test/v1', enabled: true, streaming: true, status: 'online', api_key_hint: 'sk-…1234' }])
    if (path.endsWith('/models')) return fulfill(route, [{ id: 'provider-1:model-1', provider_id: 'provider-1', name: 'model-1', display_name: 'Model One', kind: 'text', enabled: true, context_window: 128000, max_output_tokens: 4096, supports_tools: true, supports_streaming: true, routing_weight: 100, allowed_agent_roles: [] }])
    if (path.endsWith('/model-routing')) return fulfill(route, { routes: {} })
    if (path.endsWith('/agents') && request.method() === 'GET') return fulfill(route, [{ id: 'agent-1', project_id: 'project-1', name: '写作 Agent', role: 'writer', enabled: true }])
    if (path.endsWith('/skills') || path.endsWith('/mcp-servers') || path.endsWith('/tools') || path.endsWith('/index-jobs')) return fulfill(route, [])
    if (path.endsWith('/projects/project-1/graph/expansions')) return fulfill(route, {
      project_id: 'project-1', depth: 2,
      entities: [
        { id: 'entity-a', project_id: 'project-1', name: '林澈', label: '林澈', type: 'character', summary: '档案记录员', importance: 1, status: 'stable', metadata: { timeline: '1', depth: '1' }, created_at: now, updated_at: now },
        { id: 'entity-b', project_id: 'project-1', name: '第七档案馆', label: '第七档案馆', type: 'location', summary: '保存冲突档案的地点', importance: 0.8, status: 'stable', metadata: { timeline: '1', depth: '2' }, created_at: now, updated_at: now }
      ],
      edges: [{ id: 'edge-1', project_id: 'project-1', source_entity_id: 'entity-a', target_entity_id: 'entity-b', type: 'located_at', label: '任职于', weight: 1, metadata: { timeline: '1' }, created_at: now, updated_at: now }],
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

test.beforeEach(async ({ page }) => {
  await mockApi(page)
})

const cases = [
  { name: 'home', path: '/', ready: (page: Page) => expect(page.getByRole('button', { name: /墨色档案/ })).toBeVisible() },
  { name: 'projects', path: '/projects', ready: (page: Page) => expect(page.getByRole('heading', { name: '项目库', exact: true })).toBeVisible() },
  { name: 'project-workspace', path: '/projects/project-1', ready: (page: Page) => expect(page.getByRole('heading', { name: '墨色档案' })).toBeVisible() },
  { name: 'editor', path: '/projects/project-1/editor?chapter=chapter-1', ready: (page: Page) => expect(page.getByTestId('writing-workspace')).toBeVisible() },
  { name: 'settings', path: '/settings/providers', ready: (page: Page) => expect(page.getByRole('heading', { name: '提供商连接' })).toBeVisible() },
  { name: 'graph', path: '/projects/project-1/graph', ready: (page: Page) => expect(page.getByText('林澈', { exact: true })).toBeVisible() }
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
