import type { ApiResult } from '~/shared/api'
import type { Chapter, ChapterVersion, ChapterVersionWriteRequest, SaveChapterVersionResponse } from '~/lib/types'
import type { CreateChapterRequest, UpdateChapterRequest } from './types'

export interface ChapterApi {
  listChapters(projectId: string): Promise<ApiResult<Chapter[]>>
  createChapter(projectId: string, request: CreateChapterRequest): Promise<ApiResult<Chapter>>
  updateChapter(projectId: string, request: UpdateChapterRequest): Promise<ApiResult<Chapter>>
  listChapterVersions(projectId: string, chapterId: string): Promise<ApiResult<ChapterVersion[]>>
  saveChapterVersion(projectId: string, version: ChapterVersionWriteRequest): Promise<ApiResult<SaveChapterVersionResponse>>
}
