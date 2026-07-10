import { describe, expect, it, vi } from 'vitest'
import type { ChapterApi } from '../../entities/chapter/api'
import { updateChapterOperation } from '../../entities/chapter/operations'

const updatedChapter = {
  id: 'chapter-1',
  project_id: 'project-1',
  number: 1,
  title: 'Updated',
  status: 'drafting',
  summary: '',
  metadata: {}
}

describe('Chapter update operation', () => {
  it('缓存为空时仍只调用 update，不会回退到 create', async () => {
    const createChapter = vi.fn()
    const updateChapter = vi.fn().mockResolvedValue({ data: updatedChapter })
    const api = {
      createChapter,
      updateChapter,
      listChapters: vi.fn(),
      listChapterVersions: vi.fn(),
      saveChapterVersion: vi.fn()
    } as unknown as ChapterApi

    const result = await updateChapterOperation(api, [], 'project-1', {
      chapter_id: 'chapter-1',
      title: 'Updated'
    })

    expect(updateChapter).toHaveBeenCalledOnce()
    expect(updateChapter).toHaveBeenCalledWith('project-1', { chapter_id: 'chapter-1', title: 'Updated' })
    expect(createChapter).not.toHaveBeenCalled()
    expect(result.chapters).toEqual([updatedChapter])
  })
})
