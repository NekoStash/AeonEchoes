import '@testing-library/jest-dom/vitest'
import { config } from '@vue/test-utils'
import { vi } from 'vitest'

vi.stubGlobal('confirm', vi.fn(() => true))

vi.stubGlobal('useI18n', () => ({
  t: (key: string) => ({
    'actions.close': '关闭',
    'actions.cancel': '取消',
    'actions.dismiss': '关闭通知',
    'ui.field.required': '必填',
    'ui.select.noResults': '没有匹配结果',
    'ui.select.search': '搜索选项',
    'ui.dialog.label': '对话框',
    'ui.sheet.label': '侧边面板',
    'projectOverview.chapterCreate.title': '新建真实章节',
    'projectOverview.chapterCreate.description': '确认后才会创建真实章节。',
    'projectOverview.chapterCreate.fromPlan': '引用章节规划',
    'projectOverview.chapterCreate.noPlan': '不引用规划',
    'projectOverview.chapterCreate.confirm': '确认新建章节',
    'projectOverview.chapterCreate.creating': '正在新建章节',
    'projectOverview.chapterCreate.titleRequired': '章节标题不能为空。',
    'projectOverview.fields.chapterTitle': '章节标题',
    'projectOverview.fields.chapterStatus': '状态',
    'projectOverview.fields.chapterSummary': '章节摘要',
    'projectOverview.chapterPlan.title': '章节规划',
    'projectOverview.chapterPlan.eyebrow': '故事设定集字段 · chapter_plan',
    'projectOverview.chapterPlan.description': '新增规划只修改故事设定集，不会创建真实章节。',
    'projectOverview.chapterPlan.add': '新增规划',
    'projectOverview.chapterPlan.empty': '暂无章节规划。',
    'projectOverview.actions.removeChapterPlanNamed': '删除章节规划',
    'status.chapter.planned': '计划中',
    'status.chapter.drafting': '写作中',
    'status.chapter.reviewing': '审阅中',
    'status.chapter.locked': '已锁定'
  })[key] || key
}))

config.global.stubs = {
  NuxtLink: {
    props: ['to'],
    template: '<a :href="String(to)"><slot /></a>'
  }
}
