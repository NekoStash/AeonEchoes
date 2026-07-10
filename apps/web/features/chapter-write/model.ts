import type { Chapter } from '~/entities/chapter'

export interface TextSelection {
  start: number
  end: number
}

export interface ChapterDocument {
  title: string
  content: string
}

export type StrictChapterResolution =
  | { state: 'ready'; chapter: Chapter }
  | { state: 'empty'; chapter: null }
  | { state: 'invalid'; chapter: null; requestedChapterId: string }

export function resolveStrictChapter(chapters: Chapter[], requestedChapterId: string): StrictChapterResolution {
  if (chapters.length === 0) return { state: 'empty', chapter: null }

  const normalizedId = requestedChapterId.trim()
  if (!normalizedId) return { state: 'ready', chapter: chapters[0]! }

  const chapter = chapters.find((item) => item.id === normalizedId)
  if (!chapter) return { state: 'invalid', chapter: null, requestedChapterId: normalizedId }
  return { state: 'ready', chapter }
}

export function assertRealChapter(chapters: Chapter[], projectId: string, chapterId: string): Chapter {
  const normalizedProjectId = projectId.trim()
  const normalizedChapterId = chapterId.trim()
  if (!normalizedProjectId || !normalizedChapterId) {
    throw new Error('A real project ID and chapter ID are required for writing operations.')
  }

  const chapter = chapters.find((item) => item.id === normalizedChapterId && item.project_id === normalizedProjectId)
  if (!chapter) {
    throw new Error(`Chapter ${normalizedChapterId} does not exist in project ${normalizedProjectId}.`)
  }
  return chapter
}

export function normalizeTextSelection(selection: TextSelection, contentLength: number): TextSelection {
  const start = Math.max(0, Math.min(contentLength, Math.trunc(selection.start)))
  const end = Math.max(start, Math.min(contentLength, Math.trunc(selection.end)))
  return { start, end }
}

export function countWritingMetrics(content: string) {
  return {
    characters: content.replace(/\s/g, '').length,
    paragraphs: content.split(/\n\s*\n/).map((item) => item.trim()).filter(Boolean).length
  }
}
