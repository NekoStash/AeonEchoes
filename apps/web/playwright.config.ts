import { existsSync } from 'node:fs'
import { defineConfig, devices } from '@playwright/test'

const port = Number(process.env.PLAYWRIGHT_PORT || 4173)
const localChromePath = 'C:/Program Files/Google/Chrome/Application/chrome.exe'
const chromiumExecutable = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE || (existsSync(localChromePath) ? localChromePath : undefined)

export default defineConfig({
  testDir: './tests/e2e',
  outputDir: './test-results',
  snapshotPathTemplate: '{testDir}/__screenshots__/{testFilePath}/{arg}{ext}',
  fullyParallel: false,
  workers: 1,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    colorScheme: 'light',
    locale: 'zh-CN',
    launchOptions: chromiumExecutable ? { executablePath: chromiumExecutable } : undefined
  },
  webServer: {
    command: `yarn build && node .output/server/index.mjs`,
    env: { NITRO_HOST: '127.0.0.1', NITRO_PORT: String(port) },
    url: `http://127.0.0.1:${port}`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'mobile-chromium', use: { ...devices['Pixel 7'] } }
  ]
})
