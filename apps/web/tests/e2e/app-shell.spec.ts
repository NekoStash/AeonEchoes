import { expect, test } from '@playwright/test'

async function gotoApp(page: import('@playwright/test').Page) {
  await page.goto('/', { waitUntil: 'domcontentloaded' })
  await expect(page.locator('#main-content')).toBeAttached()
}

async function mockShellApi(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url())
    const pathname = url.pathname
    const now = '2026-01-01T00:00:00Z'
    let data: unknown = []

    if (pathname.endsWith('/health')) {
      data = { status: 'ok', time: now, postgres_configured: true, qdrant_configured: true }
    } else if (pathname.endsWith('/system/status')) {
      data = { status: 'ok', postgres_configured: true, qdrant_configured: true, provider_count: 0, model_count: 0, pending_jobs_count: 0, checked_at: now }
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data, meta: { request_id: 'e2e' } })
    })
  })
}

test.beforeEach(async ({ page }) => {
  await mockShellApi(page)
})

test('桌面壳层分离创作与设置导航', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await gotoApp(page)

  const navigation = page.getByRole('navigation', { name: '主导航' })
  await expect(navigation.getByRole('heading', { name: '创作' })).toBeVisible()
  await expect(navigation.getByRole('heading', { name: '设置' })).toBeVisible()
  await expect(navigation.getByRole('link', { name: '新建项目' })).toBeVisible()
  await expect(page.getByRole('main')).toBeVisible()
})

test('移动端使用独立底部导航与设置面板', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile-chromium')
  await gotoApp(page)

  const mobileNavigation = page.getByRole('navigation', { name: '移动端导航' })
  await expect(mobileNavigation).toBeVisible()
  await mobileNavigation.getByRole('button', { name: '打开设置导航' }).click()
  await expect(page.getByRole('dialog', { name: '导航' })).toBeVisible()
  await expect(page.getByRole('link', { name: '提供商' })).toBeVisible()
  await expect(page.getByRole('link', { name: '模型' })).toBeVisible()
  await expect(page.getByRole('link', { name: '索引维护' })).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog', { name: '导航' })).toBeHidden()
})

test('语言切换会同步页面元数据与术语', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await gotoApp(page)

  const language = page.getByRole('button', { name: '语言' })
  await language.click()
  await page.getByRole('option', { name: 'English' }).click()

  await expect(page.locator('html')).toHaveAttribute('lang', 'en-US')
  await expect(page).toHaveTitle('Aeon Echoes')
  await expect(page.locator('meta[name="description"]')).toHaveAttribute('content', 'A focused workspace for projects, Story Bibles, and long-form fiction writing.')
  await expect(page.getByRole('navigation', { name: 'Primary navigation' })).toContainText('Settings')
})

test('可跳到主要内容', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await gotoApp(page)
  await page.keyboard.press('Tab')
  const skipLink = page.getByRole('link', { name: '跳到主要内容' })
  await expect(skipLink).toBeFocused()
  await skipLink.press('Enter')
  await expect(page.locator('#main-content')).toBeFocused()
})

test('项目库使用纵向目录且未知章节数不显示为 0', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.route('**/api/v1/projects', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [
          {
            id: 'project-unknown',
            title: '未知章节作品',
            slug: 'unknown-chapters',
            status: 'active',
            seed: { title: '未知章节作品', premise: '章节数不应被猜测。', themes: ['验证'] },
            active_story_bible_id: 'bible-unknown',
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-02T00:00:00Z'
          }
        ],
        meta: { request_id: 'e2e-projects' }
      })
    })
  })

  await page.goto('/projects', { waitUntil: 'domcontentloaded' })
  await expect(page.getByRole('heading', { name: '项目库', exact: true })).toBeVisible()
  await expect(page.getByText('未知章节作品')).toBeVisible()
  await expect(page.getByText('0 章')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '高级筛选' })).toHaveAttribute('aria-expanded', 'false')
  await page.getByRole('button', { name: '高级筛选' }).click()
  await expect(page.getByText('故事设定集状态', { exact: true })).toBeVisible()
})

test('项目库加载失败时显示错误并写入日志', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  const consoleErrors: string[] = []
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors.push(message.text())
  })
  await page.route('**/api/v1/projects', async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({ error: { code: 'projects_unavailable', message: '项目服务暂不可用' }, meta: { request_id: 'e2e-project-error' } })
    })
  })

  await page.goto('/projects', { waitUntil: 'domcontentloaded' })
  await expect(page.getByRole('alert')).toContainText('项目加载失败')
  await expect(page.getByRole('alert')).toContainText('项目服务暂不可用')
  await expect.poll(() => consoleErrors.some((message) => message.includes('[AeonEchoes Store] projects.list failed'))).toBe(true)
})

test('创建项目后只给出完善故事设定集或新建章节', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.route('**/api/v1/projects', async (route) => {
    if (route.request().method() !== 'POST') return route.continue()
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: {
          project: {
            id: 'project-created',
            title: '记忆档案',
            slug: 'memory-files',
            status: 'active',
            seed: { title: '记忆档案', premise: '档案与记忆冲突。', themes: ['悬疑'] },
            active_story_bible_id: 'bible-created',
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-01T00:00:00Z'
          },
          story_bible: {
            id: 'bible-created',
            project_id: 'project-created',
            approved: false,
            premise: '档案与记忆冲突。',
            themes: ['悬疑'],
            world_rules: [],
            characters: [],
            foreshadows: [],
            chapter_plan: [{ id: 'plan-1', title: '计划第一章', status: 'planned', summary: '仅是计划。' }]
          },
          workflow: { id: 'workflow-created', project_id: 'project-created', intent: 'optimize_seed', status: 'completed', steps: [] }
        },
        meta: { request_id: 'e2e-create' }
      })
    })
  })

  await page.goto('/projects/new', { waitUntil: 'domcontentloaded' })
  await page.getByRole('button', { name: '下一步' }).click()
  await page.getByRole('button', { name: '下一步' }).click()
  await page.getByRole('button', { name: '下一步' }).click()
  await page.getByRole('button', { name: '创建项目' }).click()

  const success = page.getByText('项目与故事设定集已创建，当前真实章节为 0。请选择一个明确的下一步。')
  await expect(success).toBeVisible()
  await expect(page.getByRole('link', { name: /完善故事设定集/ })).toHaveAttribute('href', '/projects/project-created?section=story')
  await expect(page.getByRole('link', { name: /新建章节/ })).toHaveAttribute('href', '/projects/project-created?createChapter=1')
  await expect(page.getByRole('link', { name: '打开工作区' })).toHaveCount(0)
})
