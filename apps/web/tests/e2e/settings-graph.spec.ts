import { expect, test } from '@playwright/test'

const now = '2026-01-01T00:00:00Z'

async function mockApi(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url())
    const path = url.pathname
    let data: unknown = []

    if (path.endsWith('/health')) data = { status: 'ok', time: now, postgres_configured: true, qdrant_configured: true }
    else if (path.endsWith('/providers')) data = [{ id: 'provider-1', name: 'Primary', type: 'openai-responses', base_url: 'https://example.test/v1', enabled: true, streaming: true, status: 'online', api_key_hint: 'sk-…1234' }]
    else if (path.endsWith('/models')) data = [{ id: 'provider-1:model-1', provider_id: 'provider-1', name: 'model-1', display_name: 'Model One', kind: 'text', enabled: true, context_window: 1000, max_output_tokens: 200, supports_tools: true, supports_streaming: true, routing_weight: 100, allowed_agent_roles: [] }]
    else if (path.endsWith('/model-routing')) data = { routes: {} }
    else if (path.endsWith('/agents') || path.endsWith('/skills') || path.endsWith('/mcp-servers') || path.endsWith('/tools') || path.endsWith('/index-jobs')) data = []
    else if (path.includes('/graph/expansions')) data = {
      project_id: 'project-1', depth: 2,
      entities: [{ id: 'entity-a', project_id: 'project-1', name: 'Alpha', type: 'character', summary: 'Lead', importance: 1, status: 'stable', metadata: { timeline: '1', depth: '1' }, created_at: now, updated_at: now }],
      edges: [], facts: [], generated_at: now
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data, meta: { request_id: 'e2e' }, page: { count: Array.isArray(data) ? data.length : 1 } }) })
  })
}

test.beforeEach(async ({ page }) => {
  await mockApi(page)
})

test('设置区使用独立任务导航且旧管理链接重定向', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/admin/models')
  await expect(page).toHaveURL(/\/settings\/models$/)
  await expect(page.getByRole('navigation', { name: '设置导航' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '模型与用途路由' })).toBeVisible()
  await page.getByRole('navigation', { name: '设置导航' }).getByRole('link', { name: /提供商/ }).click()
  await expect(page).toHaveURL(/\/settings\/providers$/)
  await expect(page.getByText('Primary')).toBeVisible()
})

test('设置加载失败保留页内错误并可重试', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.route('**/api/v1/providers', async (route) => {
    await route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: { code: 'providers_failed', message: '提供商服务暂不可用' }, meta: { request_id: 'settings-error' } }) })
  })
  await page.goto('/settings/providers')
  await expect(page.getByRole('alert').first()).toContainText('提供商服务暂不可用')
  await expect(page.getByRole('button', { name: '重试' })).toBeVisible()
})

test('索引维护失败后重试成功会清除旧错误', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  let attempts = 0
  let releaseRetry: (() => void) | undefined
  const retryResponse = new Promise<void>((resolve) => { releaseRetry = resolve })
  await page.route('**/api/v1/index-jobs**', async (route) => {
    attempts += 1
    if (attempts === 1) {
      return route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: { code: 'index_jobs_failed', message: '索引任务暂不可用' }, meta: { request_id: 'index-jobs-error' } })
      })
    }
    await retryResponse
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [], meta: { request_id: 'index-jobs-retry' } }) })
  })

  await page.goto('/settings/index-maintenance')
  const retryButton = page.getByRole('button', { name: '重试' })
  const errorNotice = page.getByRole('alert').filter({ has: retryButton })
  await expect(errorNotice).toContainText('索引任务暂不可用')
  await retryButton.click()
  await expect.poll(() => attempts).toBe(2)
  await expect(errorNotice).toHaveCount(0)
  releaseRetry?.()
  await expect(page.getByText('没有符合条件的索引任务。')).toBeVisible()
  expect(attempts).toBe(2)
})

test('设置页不重复展示壳层设置导航', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/settings/models')
  await expect(page.getByRole('navigation', { name: '主导航' }).getByRole('heading', { name: '设置' })).toHaveCount(0)
  await expect(page.getByRole('navigation', { name: '设置导航' })).toBeVisible()
})

test('图谱移动端默认列表并只发送 entity_ids/depth', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile-chromium')
  let graphBody: unknown = null
  page.on('request', (request) => {
    if (request.url().includes('/graph/expansions')) graphBody = request.postDataJSON()
  })
  await page.goto('/projects/project-1/graph')
  await expect(page.getByRole('button', { name: '列表', exact: true })).toBeVisible()
  await expect(page.getByText('Alpha')).toBeVisible()
  expect(graphBody).toEqual({ depth: 2 })
  await expect(page.getByText('只有 entity_ids 与 depth 会发送到图谱扩展接口。')).not.toBeVisible()
  await page.getByRole('button', { name: '本地视图筛选' }).click()
  await expect(page.getByRole('dialog', { name: '本地视图筛选' })).toBeVisible()
  await expect(page.getByRole('dialog', { name: '本地视图筛选' })).toContainText('entity_ids')
})
