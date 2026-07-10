import type { ChapterWriteRequest } from '~/lib/types'

export { CHAPTER_STATUS_VALUES } from '~/lib/types'
export type { Chapter, ChapterStatus, ChapterVersion, ChapterVersionWriteRequest, ChapterWriteRequest, SaveChapterVersionResponse } from '~/lib/types'

export type CreateChapterRequest = Omit<ChapterWriteRequest, 'title'> & { title: string }
export type UpdateChapterRequest = ChapterWriteRequest & { chapter_id: string }
