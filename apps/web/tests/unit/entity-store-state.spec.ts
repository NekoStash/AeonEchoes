import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { preferredAgent } from '../../entities/agent/selection'
import { useAgentStore } from '../../entities/agent/store'
import { useChapterStore } from '../../entities/chapter/store'
import { useStoryBibleStore } from '../../entities/story-bible/store'
import { useWorkspaceStore } from '../../stores/workspace'
import type { AgentConfig, ChapterVersion, ChapterVersionWriteRequest, StoryBible } from '../../lib/types'

const bible: StoryBible = {
  id: 'bible-1',
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

const version: ChapterVersion = {
  id: 'version-2',
  project_id: 'project-1',
  chapter_id: 'chapter-1',
  version: 2,
  title: '第二版',
  content: '第二版正文',
  author_role: 'editor',
  index_status: 'pending',
  parent_version_id: 'version-1',
  metadata: {},
  created_at: '2026-01-02T00:00:00Z'
}

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('Agent 查询作用域', () => {
  const agent = (id: string, projectId: string | undefined, role: AgentConfig['role'] = 'writer', enabled = true): AgentConfig => ({
    id, project_id: projectId, name: id, role, enabled
  })

  it('按 projectId/enabled 缓存，设置页加载不会覆盖编辑器集合', async () => {
    const listAgents = vi.fn()
      .mockResolvedValueOnce({ data: [agent('project-writer', 'project-1'), agent('global-writer', undefined)] })
      .mockResolvedValueOnce({ data: [agent('disabled', 'project-1', 'editor', false), agent('global-writer', undefined)] })
    vi.stubGlobal('useApi', () => ({ agent: { listAgents } }))
    const store = useAgentStore()
    const editorOptions = { projectId: 'project-1', enabled: true }
    const settingsOptions = { limit: 100 }

    await store.load(editorOptions)
    await store.load(settingsOptions)

    expect(listAgents).toHaveBeenNthCalledWith(1, editorOptions)
    expect(listAgents).toHaveBeenNthCalledWith(2, settingsOptions)
    expect(store.itemsFor(editorOptions).map((item) => item.id)).toEqual(['project-writer', 'global-writer'])
    expect(store.itemsFor(settingsOptions).map((item) => item.id)).toEqual(['disabled', 'global-writer'])
  })

  it('保存和删除会同步所有已加载 scope，而不会引入禁用或其他项目 Agent', async () => {
    const projectWriter = agent('project-writer', 'project-1')
    const listAgents = vi.fn().mockResolvedValue({ data: [projectWriter] })
    const saveAgent = vi.fn().mockResolvedValue({ data: { ...projectWriter, enabled: false } })
    const deleteAgent = vi.fn().mockResolvedValue({ data: { status: 'deleted' } })
    vi.stubGlobal('useApi', () => ({ agent: { listAgents, saveAgent, deleteAgent } }))
    const store = useAgentStore()
    const options = { projectId: 'project-1', enabled: true }
    await store.load(options)

    await store.save({ ...projectWriter, enabled: false }, 'edit')
    expect(store.itemsFor(options)).toEqual([])

    await store.remove(projectWriter.id)
    expect(store.itemsFor(options)).toEqual([])
  })

  it('自动选择优先项目 writer，再全局 writer，再项目任意，再全局任意', () => {
    const globalWriter = agent('global-writer', undefined)
    const projectWriter = agent('project-writer', 'project-1')
    const projectEditor = agent('project-editor', 'project-1', 'editor')
    const globalEditor = agent('global-editor', undefined, 'editor')

    expect(preferredAgent([globalWriter], 'project-1')?.id).toBe('global-writer')
    expect(preferredAgent([globalWriter, projectWriter], 'project-1')?.id).toBe('project-writer')
    expect(preferredAgent([globalEditor, projectEditor], 'project-1')?.id).toBe('project-editor')
    expect(preferredAgent([globalEditor], 'project-1')?.id).toBe('global-editor')
  })
})

describe('实体 Store 状态收敛', () => {
  it('workspace 只维护最近项目壳层状态', () => {
    const workspace = useWorkspaceStore()

    expect(Object.keys(workspace.$state)).toEqual(['openedProjects'])
    expect('chapters' in workspace.$state).toBe(false)
    expect('activeBible' in workspace.$state).toBe(false)
    expect('models' in workspace.$state).toBe(false)
  })

  it('章节版本保存成功后只更新 Chapter Store 缓存', async () => {
    const saveChapterVersion = vi.fn().mockResolvedValue({
      data: {
        chapter_version: version,
        index_job: {
          id: 'job-1',
          project_id: 'project-1',
          chapter_id: 'chapter-1',
          chapter_version_id: version.id,
          kind: 'chapter-version',
          status: 'pending',
          attempts: 0,
          created_at: '2026-01-02T00:00:00Z',
          updated_at: '2026-01-02T00:00:00Z'
        }
      }
    })
    vi.stubGlobal('useApi', () => ({ chapter: { saveChapterVersion } }))
    const store = useChapterStore()
    const request: ChapterVersionWriteRequest = {
      chapter_id: 'chapter-1',
      title: version.title,
      content: version.content,
      author_role: 'editor',
      parent_version_id: 'version-1'
    }

    await store.saveVersion('project-1', request)

    expect(saveChapterVersion).toHaveBeenCalledWith('project-1', request)
    expect(store.versionsByChapterId['chapter-1']).toEqual([version])
    expect(store.versionSaveRequest.error).toBeNull()
  })

  it('章节版本加载失败由 Chapter Store 暴露且不写入空缓存', async () => {
    const failure = new Error('版本列表不可用')
    vi.stubGlobal('useApi', () => ({ chapter: { listChapterVersions: vi.fn().mockRejectedValue(failure) } }))
    const store = useChapterStore()

    await expect(store.loadVersions('project-1', 'chapter-1')).rejects.toThrow('版本列表不可用')

    expect(store.versionsByChapterId['chapter-1']).toBeUndefined()
    expect(store.versionListRequest.error?.message).toBe('版本列表不可用')
  })

  it('角色同步只返回合并后的草稿，显式保存才更新故事设定集缓存', async () => {
    const syncCharacters = vi.fn().mockResolvedValue({
      data: {
        project_id: 'project-1',
        story_bible_id: 'bible-1',
        characters: [{
          id: 'entity-1',
          project_id: 'project-1',
          name: '林澈',
          type: 'character',
          aliases: [],
          summary: '同步摘要',
          traits: {},
          importance: 1,
          status: 'active',
          metadata: {},
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-02T00:00:00Z'
        }],
        mappings: [{ name: '林澈', entity_id: 'entity-1', action: 'synced' }]
      }
    })
    const updateStoryBible = vi.fn().mockImplementation(async (_projectId: string, nextBible: StoryBible) => ({ data: nextBible }))
    vi.stubGlobal('useApi', () => ({ storyBible: { syncCharacters, updateStoryBible } }))
    const store = useStoryBibleStore()
    store.set('project-1', bible)

    const syncedBible = await store.syncCharacters('project-1', bible)

    expect(syncedBible.characters[0]).toMatchObject({ entity_id: 'entity-1', sync_status: 'synced' })
    expect(store.byProjectId['project-1']?.characters[0]?.entity_id).toBeUndefined()
    expect(updateStoryBible).not.toHaveBeenCalled()

    await store.save('project-1', syncedBible)

    expect(updateStoryBible).toHaveBeenCalledOnce()
    expect(store.byProjectId['project-1']?.characters[0]).toMatchObject({ entity_id: 'entity-1', sync_status: 'synced' })
  })
})
