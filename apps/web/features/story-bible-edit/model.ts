import type { StoryBible } from '~/entities/story-bible'

export type StoryBibleSaveState = 'dirty' | 'saving' | 'saved' | 'failed' | 'conflict'

export function cloneStoryBible(bible: StoryBible): StoryBible {
  return JSON.parse(JSON.stringify(bible)) as StoryBible
}

export function storyBibleSignature(bible: StoryBible): string {
  return JSON.stringify(bible)
}

export function isStoryBibleDirty(draft: StoryBible, persisted: StoryBible): boolean {
  return storyBibleSignature(draft) !== storyBibleSignature(persisted)
}

export function createStoryBibleItemId(prefix: string, existingIds: string[]): string {
  const existing = new Set(existingIds)
  let index = existing.size + 1
  while (existing.has(`${prefix}-${index}`)) index += 1
  return `${prefix}-${index}`
}

export function isConflictError(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false
  const candidate = error as { status?: number; statusCode?: number; response?: { status?: number }; state?: { code?: string; kind?: string } }
  return candidate.status === 409
    || candidate.statusCode === 409
    || candidate.response?.status === 409
    || candidate.state?.code === 'conflict'
    || candidate.state?.kind === 'conflict'
}
