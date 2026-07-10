import type { Chapter, ChapterStatus, CreateChapterRequest } from '~/entities/chapter'
import type { StoryBibleChapter } from '~/entities/story-bible'

export interface ChapterCreateDraft {
  title: string
  status: ChapterStatus
  summary: string
  planId: string
}

export function nextChapterNumber(chapters: Chapter[]): number {
  return chapters.reduce((largest, chapter) => Math.max(largest, chapter.number), 0) + 1
}

export function createChapterDraft(plan?: StoryBibleChapter): ChapterCreateDraft {
  return {
    title: plan?.title || '',
    status: plan?.status || 'drafting',
    summary: plan?.summary || '',
    planId: plan?.id || ''
  }
}

export function toCreateChapterRequest(draft: ChapterCreateDraft, chapters: Chapter[]): CreateChapterRequest {
  const title = draft.title.trim()
  if (!title) throw new Error('chapter_title_required')

  return {
    number: nextChapterNumber(chapters),
    title,
    status: draft.status || 'drafting',
    summary: draft.summary.trim() || undefined,
    metadata: draft.planId ? { story_bible_chapter_plan_id: draft.planId } : undefined
  }
}
