import type { Chapter, ChapterStatus, CreateChapterRequest } from '~/entities/chapter'

export interface ChapterCreateDraft {
  title: string
  status: ChapterStatus
  summary: string
}

export function nextChapterNumber(chapters: Chapter[]): number {
  return chapters.reduce((largest, chapter) => Math.max(largest, chapter.number), 0) + 1
}

export function createChapterDraft(): ChapterCreateDraft {
  return {
    title: '',
    status: 'drafting',
    summary: ''
  }
}

export function toCreateChapterRequest(draft: ChapterCreateDraft, chapters: Chapter[]): CreateChapterRequest {
  const title = draft.title.trim()
  if (!title) throw new Error('chapter_title_required')

  return {
    number: nextChapterNumber(chapters),
    title,
    status: draft.status || 'drafting',
    summary: draft.summary.trim() || undefined
  }
}
