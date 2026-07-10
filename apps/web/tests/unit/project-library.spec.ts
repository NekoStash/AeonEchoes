import { describe, expect, it } from 'vitest'
import type { ProjectSummary } from '../../lib/types'
import { createProjectLibraryFilters, filterProjects, projectChapterCount } from '../../features/project-library/project-library'

function project(overrides: Partial<ProjectSummary> = {}): ProjectSummary {
  return {
    id: 'project-1',
    title: '记忆档案',
    logline: '记录员发现公共档案与私人记忆冲突。',
    tags: ['悬疑'],
    updated_at: '2026-01-02T00:00:00Z',
    bible_status: 'draft',
    ...overrides
  }
}

describe('project library behavior', () => {
  it('章节数未知时保持未知，不伪装为 0', () => {
    expect(projectChapterCount(project({ chapter_count: undefined }))).toBeNull()
    expect(projectChapterCount(project({ chapter_count: null }))).toBeNull()
    expect(projectChapterCount(project({ chapter_count: 0 }))).toBe(0)
  })

  it('搜索始终可用，高级条件按真实字段筛选并排序', () => {
    const filters = createProjectLibraryFilters()
    filters.query = '悬疑'
    filters.storyBible = 'ready'
    filters.recent = 'recent'
    filters.sort = 'title'

    const projects = [
      project({ id: 'b', title: 'B 项目', bible_status: 'ready' }),
      project({ id: 'a', title: 'A 项目', bible_status: 'ready' }),
      project({ id: 'c', title: 'C 项目', bible_status: 'draft' })
    ]

    expect(filterProjects(projects, filters, new Set(['a', 'b'])).map((item) => item.id)).toEqual(['a', 'b'])
  })
})
