import { expect, test } from '@playwright/test'

const now = '2026-01-01T00:00:00Z'
const projectId = 'project-1'

function envelope(data: unknown) {
  return { data, meta: { request_id: 'project-e2e' } }
}

async function mockProjectApi(page: import('@playwright/test').Page) {
  let chapters: Array<Record<string, unknown>> = []
  let createRequests = 0

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const pathname = url.pathname
    let data: unknown = []

    if (pathname.endsWith('/health')) {
      data = { status: 'ok', time: now, postgres_configured: true, qdrant_configured: true }
    } else if (pathname.endsWith('/system/status')) {
      data = { status: 'ok', postgres_configured: true, qdrant_configured: true, provider_count: 0, model_count: 0, pending_jobs_count: 0, checked_at: now }
    } else if (pathname.endsWith(`/projects/${projectId}/story-bibles/current`)) {
      data = {
        id: 'bible-1',
        project_id: projectId,
        title: '墨色档案',
        premise: '记录员发现公共档案与私人记忆冲突。',
        themes: ['记忆', '真相'],
        world_rules: [],
        characters: [],
        foreshadows: [],
        chapter_plan: []
      }
    } else if (pathname.endsWith(`/projects/${projectId}/chapters`) && request.method() === 'GET') {
      data = chapters
    } else if (pathname.endsWith(`/projects/${projectId}/chapters`) && request.method() === 'POST') {
      createRequests += 1
      const body = request.postDataJSON()
      const chapter = { id: `chapter-${createRequests}`, project_id: projectId, number: body.number, title: body.title, status: body.status, summary: body.summary || '', metadata: body.metadata || {}, created_at: now, updated_at: now }
      chapters = [...chapters, chapter]
      data = chapter
    } else if (pathname.endsWith('/projects')) {
      data = [{
        id: projectId,
        title: '墨色档案',
        status: 'draft',
        logline: '档案与记忆冲突。',
        tags: [],
        active_story_bible_id: 'bible-1',
        updated_at: now,
        bible_status: 'ready',
        chapter_count: chapters.length
      }]
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(envelope(data)) })
  })

  return { getCreateRequests: () => createRequests }
}

test('章节规划与真实章节创建是两个独立操作', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  const api = await mockProjectApi(page)
  await page.goto(`/projects/${projectId}?createChapter=1`, { waitUntil: 'domcontentloaded' })

  await expect(page.getByRole('dialog', { name: '新建真实章节' })).toBeVisible()
  expect(api.getCreateRequests()).toBe(0)
  await page.getByRole('button', { name: '取消' }).click()
  await expect(page.getByRole('heading', { name: '墨色档案' })).toBeVisible()
  await expect(page.getByText('真实章节为 0', { exact: true })).toBeVisible()
  expect(api.getCreateRequests()).toBe(0)

  await page.getByRole('link', { name: '新建章节' }).first().click()
  const dialog = page.getByRole('dialog', { name: '新建真实章节' })
  await expect(dialog).toBeVisible()
  await dialog.getByRole('textbox', { name: '章节标题' }).fill('第一章')
  await dialog.getByRole('button', { name: '确认新建章节' }).click()

  await expect(page.getByText('第一章', { exact: true }).last()).toBeVisible()
  expect(api.getCreateRequests()).toBe(1)
})

test('工作区任一初始请求失败时保持错误态且重试后恢复', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  let chapterAttempts = 0
  const pageErrors: string[] = []
  page.on('pageerror', (error) => pageErrors.push(error.message))
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const pathname = new URL(request.url()).pathname
    let data: unknown = []

    if (pathname.endsWith(`/projects/${projectId}/story-bibles/current`)) {
      data = {
        id: 'bible-1', project_id: projectId, title: '墨色档案', premise: '记录员发现公共档案与私人记忆冲突。',
        themes: ['记忆'], world_rules: [], characters: [], foreshadows: [], chapter_plan: []
      }
    } else if (pathname.endsWith(`/projects/${projectId}/chapters`) && request.method() === 'GET') {
      chapterAttempts += 1
      if (chapterAttempts === 1) {
        return route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: { code: 'chapter_list_failed', message: '章节列表暂不可用' }, meta: { request_id: 'workspace-load-error' } })
        })
      }
      data = []
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(envelope(data)) })
  })

  await page.goto(`/projects/${projectId}`, { waitUntil: 'domcontentloaded' })
  const errorNotice = page.getByTestId('project-workspace-load-error')
  await expect(errorNotice).toContainText('章节列表暂不可用')
  await expect(page.getByRole('heading', { name: '墨色档案' })).toHaveCount(0)
  await expect(errorNotice).toBeVisible()
  expect(pageErrors).toEqual([])

  await errorNotice.getByRole('button', { name: '重试' }).click()
  await expect(page.getByRole('heading', { name: '墨色档案' })).toBeVisible()
  await expect(errorNotice).toHaveCount(0)
  expect(chapterAttempts).toBe(2)
  expect(pageErrors).toEqual([])
})

test('故事设定集 CTA 会滚动并聚焦真实编辑区', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await mockProjectApi(page)
  await page.goto(`/projects/${projectId}?section=story`, { waitUntil: 'domcontentloaded' })
  await expect(page.getByRole('region', { name: '故事设定集' })).toBeVisible()
  await expect(page.getByRole('region', { name: '故事设定集' }).locator('..')).toBeFocused()
})
