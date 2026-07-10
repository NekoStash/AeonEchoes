import type { ApiResult } from '~/shared/api'
import type { Chapter } from './types'
import type { ChapterApi } from './api'
import type { CreateChapterRequest, UpdateChapterRequest } from './types'
import { applyCreatedChapter, applyUpdatedChapter } from './state'

export async function createChapterOperation(
  api: ChapterApi,
  current: Chapter[],
  projectId: string,
  request: CreateChapterRequest
): Promise<{ result: ApiResult<Chapter>; chapters: Chapter[] }> {
  const result = await api.createChapter(projectId, request)
  return { result, chapters: applyCreatedChapter(current, result.data) }
}

export async function updateChapterOperation(
  api: ChapterApi,
  current: Chapter[],
  projectId: string,
  request: UpdateChapterRequest
): Promise<{ result: ApiResult<Chapter>; chapters: Chapter[] }> {
  const result = await api.updateChapter(projectId, request)
  return { result, chapters: applyUpdatedChapter(current, result.data) }
}
