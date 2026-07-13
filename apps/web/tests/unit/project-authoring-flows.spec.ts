import { describe, expect, it, vi } from 'vitest'
import type { ChapterApi } from '../../entities/chapter/api'
import type { Chapter } from '../../entities/chapter/types'
import { createChapterOperation } from '../../entities/chapter/operations'
import type { StoryBibleApi } from '../../entities/story-bible/api'
import { toCreateChapterRequest } from '../../features/chapter-create/model'
import { applyCharacterSyncResult } from '../../features/character-sync/model'
import { cloneStoryBible, isStoryBibleDirty } from '../../features/story-bible-edit/model'
import type { StoryBible } from '../../lib/types'

const bible: StoryBible = {
  id: 'bible-old',
  project_id: 'project-1',
  title: '墨色档案',
  premise: '原始前提',
  themes: [],
  world_rules: [],
  characters: [{
    id: 'local-character',
    name: '林澈',
    role: '记录员',
    desire: '找到真相',
    wound: '不再信任档案',
    secret: '',
    summary: ''
  }],
  foreshadows: [],
  chapter_plan: []
}

describe('项目创作流程', () => {
  it('章节创建请求只在显式执行创建操作时发送，失败不污染真实章节缓存', async () => {
    const request = toCreateChapterRequest({ title: '第一章', status: 'planned', summary: '' }, [])
    const createChapter = vi.fn().mockRejectedValue(new Error('network failed'))
    const api = {
      createChapter,
      updateChapter: vi.fn(),
      listChapters: vi.fn(),
      listChapterVersions: vi.fn(),
      saveChapterVersion: vi.fn()
    } as unknown as ChapterApi
    const current: Chapter[] = []

    expect(createChapter).not.toHaveBeenCalled()
    await expect(createChapterOperation(api, current, 'project-1', request)).rejects.toThrow('network failed')
    expect(createChapter).toHaveBeenCalledOnce()
    expect(current).toEqual([])
  })

  it('真实章节创建请求按既有章节序号递增且不附带规划元数据', () => {
    const request = toCreateChapterRequest({
      title: '雨夜档案',
      status: 'drafting',
      summary: '调查第一份冲突记录。'
    }, [{ id: 'chapter-4', project_id: 'project-1', number: 4, title: '旧章', status: 'locked', summary: '' }])

    expect(request).toEqual({
      number: 5,
      title: '雨夜档案',
      status: 'drafting',
      summary: '调查第一份冲突记录。'
    })
  })

  it('故事设定集保存状态以响应新 ID 为基准', async () => {
    const save = vi.fn().mockResolvedValue({ data: { ...bible, id: 'bible-new', premise: '新前提' } })
    const api = { updateStoryBible: save } as unknown as StoryBibleApi
    const draft = cloneStoryBible(bible)
    draft.premise = '新前提'

    expect(isStoryBibleDirty(draft, bible)).toBe(true)
    const result = await api.updateStoryBible('project-1', draft)
    const persisted = cloneStoryBible(result.data)

    expect(persisted.id).toBe('bible-new')
    expect(isStoryBibleDirty(persisted, result.data)).toBe(false)
  })

  it('角色同步映射作为显式副作用合并，不会生成不存在的角色', () => {
    const synced = applyCharacterSyncResult(bible, {
      project_id: 'project-1',
      story_bible_id: 'bible-old',
      characters: [{
        id: 'entity-1',
        project_id: 'project-1',
        name: '林澈',
        type: 'character',
        aliases: [],
        summary: '真实角色摘要',
        traits: {},
        importance: 1,
        status: 'active',
        metadata: {},
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-02T00:00:00Z'
      }],
      mappings: [{ name: '林澈', entity_id: 'entity-1', action: 'synced' }]
    })

    expect(synced.characters).toHaveLength(1)
    expect(synced.characters[0]).toMatchObject({ entity_id: 'entity-1', sync_status: 'synced', summary: '真实角色摘要' })
  })
})
