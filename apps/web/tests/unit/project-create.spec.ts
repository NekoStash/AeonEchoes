import { describe, expect, it } from 'vitest'
import type { InitializeProjectResponse, ProjectSeed } from '../../lib/types'
import { createdProjectDestinations, projectSummaryFromInitialization, splitProjectTags } from '../../features/project-create/project-create'

const seed: ProjectSeed = {
  title: '记忆档案',
  one_sentence_core: '档案与记忆冲突。',
  tags: ['悬疑'],
  world_background: '城市档案保存所有事件。',
  protagonist: '林澈，档案记录员。',
  central_conflict: '找出可验证真相。',
  style: '克制。',
  taboos: '避免机械降神。'
}

const initialized: InitializeProjectResponse = {
  project: {
    id: 'project-1',
    title: '记忆档案',
    slug: 'memory-files',
    status: 'active',
    seed,
    active_story_bible_id: 'bible-1',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z'
  },
  story_bible: {
    id: 'bible-1',
    project_id: 'project-1',
    approved: false,
    premise: '档案与记忆冲突。',
    themes: ['真相'],
    world_rules: [],
    characters: [],
    foreshadows: [],
    chapter_plan: [
      { id: 'plan-1', title: '计划中的第一章', status: 'planned', summary: '这不是章节记录。' },
      { id: 'plan-2', title: '计划中的第二章', status: 'planned', summary: '仍不是章节记录。' }
    ]
  },
  workflow: {
    id: 'workflow-1',
    project_id: 'project-1',
    intent: 'optimize_seed',
    status: 'completed',
    steps: []
  }
}

describe('project creation behavior', () => {
  it('创建后真实章节固定为 0，不把故事设定集章节计划计入章节数', () => {
    const summary = projectSummaryFromInitialization(initialized)
    expect(initialized.story_bible.chapter_plan).toHaveLength(2)
    expect(summary.chapter_count).toBe(0)
  })

  it('创建后只提供完善故事设定集或新建章节两个目的地', () => {
    expect(createdProjectDestinations('project-1')).toEqual({
      storyBible: '/projects/project-1?section=story',
      newChapter: '/projects/project-1?createChapter=1'
    })
  })

  it('标签输入去重并清理空值', () => {
    expect(splitProjectTags('悬疑, 时间线，悬疑, ')).toEqual(['悬疑', '时间线'])
  })
})
