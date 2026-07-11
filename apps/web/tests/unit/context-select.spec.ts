import { describe, expect, it } from 'vitest'
import {
  buildAgentRunInput,
  buildContextSelection,
  createContextSelectState
} from '../../features/context-select'
import type { Chapter } from '../../entities/chapter'
import type { StoryBible } from '../../lib/types'

function chapter(partial: Partial<Chapter> & Pick<Chapter, 'id' | 'number'>): Chapter {
  return {
    project_id: 'project-1',
    title: `第${partial.number}章`,
    summary: `摘要 ${partial.number}`,
    status: 'draft',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...partial
  }
}

const chapters: Chapter[] = [
  chapter({ id: 'ch-1', number: 1 }),
  chapter({ id: 'ch-2', number: 2 }),
  chapter({ id: 'ch-3', number: 3 })
]

const bible: StoryBible = {
  id: 'bible-1',
  project_id: 'project-1',
  version: 1,
  premise: '测试设定',
  themes: [],
  world_rules: [],
  characters: [
    {
      id: 'card-1',
      name: '林烬',
      role: 'protagonist',
      desire: '找回钥匙',
      wound: '失去故乡',
      entity_id: 'entity-lin'
    },
    {
      id: 'card-2',
      name: '未同步角色',
      role: 'support',
      desire: '未知',
      wound: '未知'
    }
  ],
  foreshadows: [],
  chapter_plan: [],
  updated_at: '2026-01-01T00:00:00Z'
}

describe('buildContextSelection', () => {
  it('包含本章时把当前章节 id 写入 chapter_ids', () => {
    const state = createContextSelectState()
    state.includeCurrentChapter = true
    state.previousChapterCount = 1

    const selection = buildContextSelection(chapters, bible, 'project-1', 'ch-3', state)

    expect(selection.chapter_ids).toEqual(['ch-2', 'ch-3'])
    expect(selection.include_world_rules).toBe(true)
  })

  it('不包含本章时只保留前序章节，表示重写本章', () => {
    const state = createContextSelectState()
    state.includeCurrentChapter = false
    state.previousChapterCount = 2

    const selection = buildContextSelection(chapters, bible, 'project-1', 'ch-3', state)

    expect(selection.chapter_ids).toEqual(['ch-1', 'ch-2'])
    expect(selection.chapter_ids).not.toContain('ch-3')
  })

  it('不包含本章且无前序章节时 chapter_ids 为空，但仍显式保留世界规则布尔值', () => {
    const state = createContextSelectState()
    state.includeCurrentChapter = false
    state.previousChapterCount = 0
    state.includeWorldRules = false

    const selection = buildContextSelection(chapters, bible, 'project-1', 'ch-1', state)

    expect(selection.chapter_ids).toBeUndefined()
    expect(selection.include_world_rules).toBe(false)
  })

  it('角色选择只映射已同步 entity_id', () => {
    const state = createContextSelectState()
    state.characterIds = ['card-1', 'card-2', 'missing']

    const selection = buildContextSelection(chapters, bible, 'project-1', 'ch-2', state)

    expect(selection.character_ids).toEqual(['entity-lin'])
  })
})

describe('buildAgentRunInput', () => {
  it('包含本章时发送正文与选区', () => {
    const input = buildAgentRunInput({
      chapterId: 'ch-3',
      title: '第三章',
      instruction: '续写冲突',
      content: '已有正文',
      selectedText: '冲突',
      state: { ...createContextSelectState(), includeCurrentChapter: true }
    })

    expect(input).toEqual({
      chapter_id: 'ch-3',
      instruction: '续写冲突',
      title: '第三章',
      content: '已有正文',
      selected_text: '冲突'
    })
  })

  it('不包含本章时不发送正文与选区，语义为重写本章', () => {
    const input = buildAgentRunInput({
      chapterId: 'ch-3',
      title: '第三章',
      instruction: '按前文重写本章',
      content: '旧正文不应出现',
      selectedText: '旧选区',
      state: { ...createContextSelectState(), includeCurrentChapter: false }
    })

    expect(input).toEqual({
      chapter_id: 'ch-3',
      instruction: '按前文重写本章',
      title: '第三章'
    })
    expect(input).not.toHaveProperty('content')
    expect(input).not.toHaveProperty('selected_text')
  })

  it('instruction 为空时 Fail Fast', () => {
    expect(() => buildAgentRunInput({
      chapterId: 'ch-1',
      title: '第一章',
      instruction: '   ',
      content: 'x',
      selectedText: '',
      state: createContextSelectState()
    })).toThrow(/instruction/)
  })
})
