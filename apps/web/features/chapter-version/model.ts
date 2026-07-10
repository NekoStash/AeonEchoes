import type { Chapter, ChapterVersion, ChapterVersionWriteRequest } from '~/entities/chapter'
import { assertRealChapter } from '~/features/chapter-write'

export interface ChapterVersionDraft {
  title: string
  content: string
  changeNote?: string
  parentVersionId?: string
}

export interface LoadedChapterVersion {
  title: string
  content: string
  parentVersionId: string
}

export function sortChapterVersions(versions: ChapterVersion[]) {
  return [...versions].sort((left, right) => {
    if (left.version !== right.version) return right.version - left.version
    return right.created_at.localeCompare(left.created_at)
  })
}

export function latestChapterVersion(versions: ChapterVersion[]) {
  return sortChapterVersions(versions)[0] || null
}

export function loadChapterVersion(version: ChapterVersion): LoadedChapterVersion {
  return {
    title: version.title,
    content: version.content,
    parentVersionId: version.id
  }
}

export function buildChapterVersionPayload(
  chapters: Chapter[],
  projectId: string,
  chapterId: string,
  draft: ChapterVersionDraft
): ChapterVersionWriteRequest {
  const chapter = assertRealChapter(chapters, projectId, chapterId)
  return {
    chapter_id: chapter.id,
    title: draft.title.trim() || chapter.title,
    content: draft.content,
    summary: draft.content.trim().slice(0, 180),
    author_role: 'editor',
    change_note: draft.changeNote?.trim() || 'manual-save',
    parent_version_id: draft.parentVersionId
  }
}
