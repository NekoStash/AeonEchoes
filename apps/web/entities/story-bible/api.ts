import type { ApiResult } from '~/shared/api'
import type { CharacterSyncResponse, StoryBible } from './types'

export interface StoryBibleApi {
  getStoryBible(projectId: string): Promise<ApiResult<StoryBible>>
  updateStoryBible(projectId: string, bible: StoryBible): Promise<ApiResult<StoryBible>>
  syncCharacters(projectId: string, bible: StoryBible): Promise<ApiResult<CharacterSyncResponse>>
}
