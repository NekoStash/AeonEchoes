import { expect, test, type Locator } from '@playwright/test'

const now = '2026-01-01T00:00:00Z'
const providers = [
  { id: 'provider-1', name: 'Primary', type: 'openai-responses', provider_type: 'openai-responses', base_url: 'https://example.test/v1', enabled: true, streaming: true, status: 'online', api_key_hint: 'sk-…1234' },
  { id: 'provider-2', name: 'Secondary', type: 'anthropic', provider_type: 'anthropic', base_url: 'https://secondary.example.test', enabled: true, streaming: true, status: 'online', api_key_hint: 'sk-…5678' }
]
const models = [
  ...Array.from({ length: 18 }, (_, index) => ({
    id: `provider-${index % 2 + 1}:model-${index + 1}`,
    provider_id: `provider-${index % 2 + 1}`,
    name: `model-${index + 1}`,
    display_name: `Text Model ${String(index + 1).padStart(2, '0')}`,
    kind: 'text',
    enabled: true,
    context_window: 32000 + index * 8000,
    max_output_tokens: 2048 + index * 256,
    supports_tools: index % 3 !== 0,
    supports_streaming: true,
    routing_weight: 100 - index,
    cost_input_per_mtok: index + 0.5,
    cost_output_per_mtok: index + 1.5,
    default_for_kind: index === 0,
    allowed_agent_roles: index === 2 ? ['editor'] : []
  })),
  { id: 'provider-1:disabled-text', provider_id: 'provider-1', name: 'disabled-text', display_name: 'Disabled Text', kind: 'text', enabled: false, context_window: 64000, max_output_tokens: 4096, supports_tools: true, supports_streaming: true, routing_weight: 1, allowed_agent_roles: [] },
  { id: 'provider-1:embedding-main', provider_id: 'provider-1', name: 'embedding-main', display_name: 'Embedding Main', kind: 'embedding', enabled: true, dimension: 1536, routing_weight: 100, default_for_kind: true, allowed_agent_roles: [] },
  { id: 'provider-2:embedding-disabled', provider_id: 'provider-2', name: 'embedding-disabled', display_name: 'Embedding Disabled', kind: 'embedding', enabled: false, dimension: 3072, routing_weight: 10, allowed_agent_roles: [] }
]
const emptyRoutes = {
  writer: '', editor: '', 'genesis-optimizer': '', 'plot-architect': '', 'world-builder': '', 'character-keeper': '', 'continuity-auditor': '', 'fact-extractor': '', 'graph-curator': '', embedding: ''
}
const initialRoutes = { ...emptyRoutes, writer: 'provider-1:model-1', embedding: 'provider-1:embedding-main' }

async function mockApi(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url())
    const path = url.pathname
    let data: unknown = []

    if (path.endsWith('/health')) data = { status: 'ok', time: now, postgres_configured: true, qdrant_configured: true }
    else if (path.endsWith('/providers')) data = providers
    else if (path.endsWith('/models')) data = models
    else if (path.endsWith('/model-routing')) {
      const requestRoutes = route.request().method() === 'PUT' ? route.request().postDataJSON()?.routes : undefined
      data = { routes: requestRoutes ? { ...emptyRoutes, ...requestRoutes } : initialRoutes }
    }
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

function routeSelect(page: import('@playwright/test').Page, role: string) {
  return page.getByRole('button', { name: new RegExp(`${role}.*模型`) })
}

async function chooseRoute(page: import('@playwright/test').Page, role: string, option: string | RegExp) {
  const trigger = routeSelect(page, role)
  await trigger.click()
  await page.getByRole('option', { name: option }).click()
  return trigger
}

async function expectSquare(locator: Locator) {
  await expect(locator).toHaveCSS('border-radius', '0px')
}

test('旧管理链接重定向到通用主导航中的设置页面', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/admin/models')
  await expect(page).toHaveURL(/\/settings\/models$/)
  const navigation = page.getByRole('navigation', { name: '主导航' })
  await expect(navigation.getByRole('heading', { name: '设置' })).toBeVisible()
  await expect(navigation.getByRole('link', { name: '模型' })).toHaveAttribute('aria-current', 'page')
  await expect(page.getByRole('heading', { name: '模型与用途路由' })).toBeVisible()
  await navigation.getByRole('link', { name: '提供商' }).click()
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

test('设置子页持续使用唯一的通用主导航', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/settings/models')

  const navigation = page.getByRole('navigation', { name: '主导航' })
  await expect(navigation.getByRole('heading', { name: '设置' })).toBeVisible()
  await expect(navigation.getByRole('link', { name: '模型' })).toHaveAttribute('aria-current', 'page')
  await expect(page.getByRole('navigation', { name: '设置导航' })).toHaveCount(0)
  await expect(page.getByRole('link', { name: '模型' })).toHaveCount(1)

  await navigation.getByRole('link', { name: '提供商' }).click()
  await expect(page).toHaveURL(/\/settings\/providers$/)
  await expect(navigation.getByRole('heading', { name: '设置' })).toBeVisible()
  await expect(navigation.getByRole('link', { name: '提供商' })).toHaveAttribute('aria-current', 'page')
  await expect(page.getByRole('navigation', { name: '设置导航' })).toHaveCount(0)
})

test('设置子页移动端仍可打开完整通用菜单', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile-chromium')
  await page.goto('/settings/models')

  const mobileNavigation = page.getByRole('navigation', { name: '移动端导航' })
  await mobileNavigation.getByRole('button', { name: '打开设置导航' }).click()
  const dialog = page.getByRole('dialog', { name: '导航' })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('heading', { name: '设置' })).toBeVisible()
  await expect(dialog.getByRole('link', { name: '模型' })).toHaveAttribute('aria-current', 'page')
  await expect(dialog.getByRole('link', { name: '提供商' })).toBeVisible()
  await expect(dialog.getByRole('link', { name: '智能体' })).toBeVisible()
  await expect(dialog.getByRole('link', { name: '索引维护' })).toBeVisible()
})

test('模型页桌面列表独立滚动且右侧路由面板保持位置', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/settings/models')
  await expect(page.getByText('Text Model 18')).toBeAttached()

  const list = page.getByTestId('model-list-scroll')
  const routingPanel = page.getByTestId('model-routing-panel')
  await expect.poll(() => list.evaluate((element) => element.scrollHeight > element.clientHeight)).toBe(true)
  const before = await routingPanel.boundingBox()
  await list.evaluate((element) => { element.scrollTop = 700 })
  await expect.poll(() => list.evaluate((element) => element.scrollTop)).toBeGreaterThan(0)
  const after = await routingPanel.boundingBox()

  expect(before).not.toBeNull()
  expect(after).not.toBeNull()
  expect(Math.abs((before?.y || 0) - (after?.y || 0))).toBeLessThanOrEqual(1)
  expect(after?.y).toBeGreaterThanOrEqual(0)
  expect(await page.evaluate(() => document.documentElement.scrollHeight <= window.innerHeight + 4)).toBe(true)
})

test('模型页移动端路由面板位于模型目录之前', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile-chromium')
  await page.goto('/settings/models')
  await expect(routeSelect(page, '写作 Agent')).toBeVisible()
  const routing = await page.getByTestId('model-routing-panel').boundingBox()
  const catalog = await page.getByTestId('model-catalog').boundingBox()
  expect(routing).not.toBeNull()
  expect(catalog).not.toBeNull()
  expect(routing!.y).toBeLessThan(catalog!.y)
  await expect(page.getByTestId('model-routing-actions')).toBeVisible()
})

test('核心按钮、输入、弹窗、下拉与状态标签运行时均为直角', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/settings/models')
  await expect(page.getByText('Text Model 18')).toBeAttached()

  const addButton = page.getByRole('button', { name: '新增模型' })
  const searchInput = page.getByRole('searchbox', { name: '搜索模型目录' })
  const routeTrigger = routeSelect(page, '写作 Agent')
  const statusBadge = page.getByTestId('routing-save-state')
  await expectSquare(addButton)
  await expectSquare(searchInput)
  await expectSquare(routeTrigger)
  await expectSquare(statusBadge)

  await routeTrigger.click()
  await expectSquare(page.getByRole('listbox', { name: /写作 Agent.*模型/ }))
  await page.keyboard.press('Escape')

  await addButton.click()
  const dialog = page.getByRole('dialog', { name: '新增模型' })
  await expectSquare(dialog)
  await expectSquare(dialog.getByLabel('上游模型 ID'))
  await expectSquare(dialog.getByRole('button', { name: '关闭' }))
})

test('用途路由支持 dirty、重置、成功保存与继承空值提交', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/settings/models')
  const writer = routeSelect(page, '写作 Agent')
  await expect(writer).toContainText('Text Model 01')
  const saveButton = page.getByRole('button', { name: '保存路由' })
  await expect(saveButton).toBeDisabled()

  await chooseRoute(page, '写作 Agent', /Text Model 02/)
  await expect(page.getByTestId('routing-save-state')).toContainText('有未保存更改')
  await expect(saveButton).toBeEnabled()
  await page.getByRole('button', { name: '重置' }).click()
  await expect(writer).toContainText('Text Model 01')
  await expect(saveButton).toBeDisabled()

  await chooseRoute(page, '写作 Agent', /Text Model 02/)
  const explicitRequest = page.waitForRequest((request) => request.url().endsWith('/api/v1/model-routing') && request.method() === 'PUT')
  await saveButton.click()
  expect((await explicitRequest).postDataJSON().routes.writer).toBe('provider-2:model-2')
  await expect(page.getByTestId('routing-save-state')).toContainText('已保存')
  await expect(saveButton).toBeDisabled()

  await chooseRoute(page, '写作 Agent', '继承后端路由')
  const inheritedRequest = page.waitForRequest((request) => request.url().endsWith('/api/v1/model-routing') && request.method() === 'PUT')
  await saveButton.click()
  expect((await inheritedRequest).postDataJSON().routes.writer).toBe('')
  await expect(page.getByTestId('routing-save-state')).toContainText('已保存')
})

test('用途路由保存失败保留草稿并展示持久错误', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.route('**/api/v1/model-routing', async (route) => {
    if (route.request().method() !== 'PUT') return route.fallback()
    await route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: { code: 'routing_failed', message: '路由服务暂不可用' }, meta: { request_id: 'routing-failed' } }) })
  })
  await page.goto('/settings/models')
  const editor = await chooseRoute(page, '编辑 Agent', /Text Model 02/)
  await page.getByRole('button', { name: '保存路由' }).click()

  await expect(page.getByTestId('routing-save-state')).toContainText('保存失败')
  await expect(page.getByTestId('routing-error-notice')).toContainText('路由服务暂不可用')
  await expect(editor).toContainText('Text Model 02')
  await expect(page.getByRole('button', { name: '保存路由' })).toBeEnabled()
})

test('路由选项禁止无资格模型并显示已存无效路由', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/settings/models')
  const writer = routeSelect(page, '写作 Agent')
  await writer.click()
  await expect(page.getByRole('option', { name: /Disabled Text/ })).toBeDisabled()
  await expect(page.getByRole('option', { name: /Embedding Main/ })).toBeDisabled()
  await expect(page.getByRole('option', { name: /Text Model 03/ })).toBeDisabled()

  await page.route('**/api/v1/model-routing', async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: { routes: { ...emptyRoutes, writer: 'provider-1:disabled-text' } }, meta: { request_id: 'invalid-route' } }) })
  })
  await page.reload()
  await expect(routeSelect(page, '写作 Agent')).toContainText('Disabled Text')
  await expect(page.getByText(/已停用，不能用于显式路由/).first()).toBeVisible()
})

test('已保存与草稿引用的模型都禁止删除并列出角色', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/settings/models')
  const savedModel = page.getByTestId('model-item-provider-1:model-1')
  await expect(savedModel.getByRole('button', { name: '删除' })).toBeDisabled()
  await expect(savedModel).toContainText('已保存路由引用：写作 Agent')

  await chooseRoute(page, '写作 Agent', /Text Model 02/)
  const draftModel = page.getByTestId('model-item-provider-2:model-2')
  await expect(draftModel.getByRole('button', { name: '删除' })).toBeDisabled()
  await expect(draftModel).toContainText('未保存草稿引用：写作 Agent')
})

test('刷新与离开都会确认未保存路由草稿', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/settings/models')
  await chooseRoute(page, '写作 Agent', /Text Model 02/)

  await page.getByRole('button', { name: '刷新' }).click()
  const refreshDialog = page.getByRole('dialog', { name: '放弃路由草稿并刷新？' })
  await expect(refreshDialog).toBeVisible()
  await refreshDialog.getByRole('button', { name: '取消' }).click()
  await expect(routeSelect(page, '写作 Agent')).toContainText('Text Model 02')

  await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '提供商' }).click()
  const leaveDialog = page.getByRole('dialog', { name: '放弃未保存路由？' })
  await expect(leaveDialog).toBeVisible()
  await expect(page).toHaveURL(/\/settings\/models$/)
  await leaveDialog.getByRole('button', { name: '取消' }).click()
  await expect(page).toHaveURL(/\/settings\/models$/)

  await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '提供商' }).click()
  await page.getByRole('dialog', { name: '放弃未保存路由？' }).getByRole('button', { name: '放弃并离开' }).click()
  await expect(page).toHaveURL(/\/settings\/providers$/)
})

test('模型配置弹窗关闭时确认未保存更改', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium')
  await page.goto('/settings/models')
  await page.getByRole('button', { name: '新增模型' }).click()
  const modelDialog = page.getByRole('dialog', { name: '新增模型' })
  await modelDialog.getByLabel('上游模型 ID').fill('draft-model')
  await page.keyboard.press('Escape')
  const discardDialog = page.getByRole('dialog', { name: '放弃模型表单更改？' })
  await expect(discardDialog).toBeVisible()
  await discardDialog.getByRole('button', { name: '取消' }).click()
  await expect(modelDialog).toBeVisible()

  await modelDialog.getByRole('button', { name: '关闭' }).click()
  await page.getByRole('dialog', { name: '放弃模型表单更改？' }).getByRole('button', { name: '放弃更改' }).click()
  await expect(modelDialog).toHaveCount(0)
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
  const range = page.locator('input[type="range"]').first()
  await expectSquare(range)
  await expect.poll(() => range.evaluate((element) => getComputedStyle(element, '::-webkit-slider-thumb').borderRadius)).toBe('0px')
  await expect(page.getByText('只有 entity_ids 与 depth 会发送到图谱扩展接口。')).not.toBeVisible()
  await page.getByRole('button', { name: '本地视图筛选' }).click()
  const filterDialog = page.getByRole('dialog', { name: '本地视图筛选' })
  await expect(filterDialog).toBeVisible()
  await expectSquare(filterDialog)
  await expect(filterDialog).toContainText('entity_ids')
})
