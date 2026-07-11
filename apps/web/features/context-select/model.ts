import type { Chapter } from '~/entities/chapter'
import type { ContextSelection, StoryBible } from '~/lib/types'
import { assertRealChapter } from '~/features/chapter-write'

export interface ContextSelectState {
  previousChapterCount: number
  includeCurrentChapter: boolean
  includeWorldRules: boolean
  characterIds: string[]
}

/** Agent run input assembled from the editor + context selection switches. */
export interface AgentRunInputBuildArgs {
  chapterId: string
  title: string
  instruction: string
  content: string
  selectedText: string
  state: ContextSelectState
}

export function createContextSelectState(): ContextSelectState {
  return {
    previousChapterCount: 0,
    includeCurrentChapter: true,
    includeWorldRules: true,
    characterIds: []
  }
}

/**
 * Expand UI context switches into the API ContextSelection.
 *
 * - includeCurrentChapter=false means rewrite mode for the current chapter:
 *   the current chapter id is omitted from chapter_ids so the model does not
 *   receive this chapter's prior text via ContextPack summaries.
 * - previousChapterCount / character multi-select only contribute real ids.
 * - include_world_rules is always an explicit boolean (false must not be dropped).
 */
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
    include_world_rules: state.includeWorldRules,
    character_ids: characterIds.length > 0 ? characterIds : undefined
  }
}

/**
 * Build the AgentRun input payload.
 *
 * When includeCurrentChapter is false, this is rewrite-the-chapter mode:
 * current body and selection must not be sent, otherwise the model still
 * "sees" the chapter and cannot rewrite from other context alone.
 */
export function buildAgentRunInput(args: AgentRunInputBuildArgs): Record<string, unknown> {
  const instruction = args.instruction.trim()
  if (!instruction) {
    throw new Error('Agent run instruction must not be empty.')
  }
  const chapterId = args.chapterId.trim()
  if (!chapterId) {
    throw new Error('Agent run chapter_id must not be empty.')
  }

  const input: Record<string, unknown> = {
    chapter_id: chapterId,
    instruction,
    title: args.title
  }

  if (args.state.includeCurrentChapter) {
    input.content = args.content
    const selected = args.selectedText.trim()
    if (selected) {
      input.selected_text = selected
    }
  }

  return input
}
