import { describe, expect, it } from 'vitest'
import { CHAPTER_STATUS_VALUES } from '../../lib/types'
import enUS from '../../i18n/locales/en-US.json'
import zhCN from '../../i18n/locales/zh-CN.json'

function visibleStrings(value: unknown): string[] {
  if (typeof value === 'string') return [value]
  if (Array.isArray(value)) return value.flatMap(visibleStrings)
  if (value && typeof value === 'object') return Object.values(value).flatMap(visibleStrings)
  return []
}

describe('locale terminology', () => {
  it('中文词表使用故事设定集和提供商的自然文案', () => {
    const values = visibleStrings(zhCN)
    expect(zhCN.projects.activeStoryBible).toBe('当前故事设定集')
    expect(zhCN.models.providerConfig).toBe('提供商配置')
    expect(zhCN.settings.nav.providers).toBe('提供商')
    expect(values.some((value) => value.includes('Story Bible') || value.includes('Provider') || value.includes('模型提供商'))).toBe(false)
    expect(values.some((value) => /\s提供商|提供商\s/.test(value))).toBe(false)
    expect(values.some((value) => /\s故事设定集|故事设定集\s(?=[，。；：、])/u.test(value))).toBe(false)
  })

  it('统一章节状态在中文词表中都有非空翻译', () => {
    const translations = zhCN.status.chapter as Record<string, string>
    expect(CHAPTER_STATUS_VALUES.every((status) => Boolean(translations[status]?.trim()))).toBe(true)
  })

  it('英文词表自然使用 Story Bible 术语', () => {
    const values = visibleStrings(enUS)
    expect(enUS.projects.activeStoryBible).toBe('Active Story Bible')
    expect(enUS.projects.stats.readyBibleHint).toContain('Story Bibles')
    expect(values.some((value) => value.includes('故事设定集'))).toBe(false)
  })
})
