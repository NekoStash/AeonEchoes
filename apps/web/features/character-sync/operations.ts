import type { StoryBibleApi } from '~/entities/story-bible'
import type { StoryBible } from '~/entities/story-bible'
import { applyCharacterSyncResult } from './model'

export async function syncStoryBibleCharacters(
  api: StoryBibleApi,
  projectId: string,
  bible: StoryBible
): Promise<StoryBible> {
  const syncResult = await api.syncCharacters(projectId, bible)
  return applyCharacterSyncResult(bible, syncResult.data)
}
