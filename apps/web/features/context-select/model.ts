import type { Chapter } from '~/entities/chapter'
import type { ContextSelection, StoryBible } from '~/lib/types'
import { assertRealChapter } from '~/features/chapter-write'

export interface ContextSelectState {
  previousChapterCount: number
  includeCurrentChapter: boolean
  includeWorldRules: boolean
  characterIds: string[]
}

export function createContextSelectState(): ContextSelectState {
  return {
    previousChapterCount: 0,
    includeCurrentChapter: true,
    includeWorldRules: true,
    characterIds: []
  }
}

export function buildContextSelection(
  chapters: Chapter[],
  bible: StoryBible | null,
  projectId: string,
  chapterId: string,
  state: ContextSelectState
): ContextSelection {
  const currentChapter = assertRealChapter(chapters, projectId, chapterId)
  const currentIndex = chapters.findIndex((chapter) => chapter.id === currentChapter.id)
  const previousCount = Math.min(Math.max(0, Math.trunc(state.previousChapterCount)), Math.max(0, currentIndex))
  const previousChapterIds = previousCount > 0
    ? chapters.slice(currentIndex - previousCount, currentIndex).map((chapter) => chapter.id)
    : []
  const validCharacters = new Map((bible?.characters || [])
    .filter((character) => character.entity_id?.trim())
    .map((character) => [character.id, character.entity_id!.trim()]))
  const characterIds = Array.from(new Set(state.characterIds.map((id) => validCharacters.get(id)).filter((id): id is string => Boolean(id))))
  const chapterIds = [
    ...previousChapterIds,
    ...(state.includeCurrentChapter ? [currentChapter.id] : [])
  ]

  return {
    chapter_ids: chapterIds.length > 0 ? Array.from(new Set(chapterIds)) : undefined,
    previous_chapter_count: previousCount || undefined,
    include_current_chapter: state.includeCurrentChapter || undefined,
    include_world_rules: state.includeWorldRules || undefined,
    character_ids: characterIds.length > 0 ? characterIds : undefined
  }
}
