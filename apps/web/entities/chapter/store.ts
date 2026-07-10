import { defineStore } from 'pinia'
import type { Chapter, ChapterVersion, ChapterVersionWriteRequest, SaveChapterVersionResponse } from '~/lib/types'
import type { CreateChapterRequest, UpdateChapterRequest } from './types'
import { createChapterOperation, updateChapterOperation } from './operations'
import { createApiRequestState, withApiRequestState } from '~/shared/store'

function mergeChapterVersions(current: ChapterVersion[], response: SaveChapterVersionResponse) {
  const created = response.chapter_version
  return [created, ...current.filter((version) => version.id !== created.id)]
}

export const useChapterStore = defineStore('chapter-domain', {
  state: () => ({
    byProjectId: {} as Record<string, Chapter[]>,
    versionsByChapterId: {} as Record<string, ChapterVersion[]>,
    listRequest: createApiRequestState(),
    createRequest: createApiRequestState(),
    updateRequest: createApiRequestState(),
    versionListRequest: createApiRequestState(),
    versionSaveRequest: createApiRequestState()
  }),
  actions: {
    async load(projectId: string) {
      return withApiRequestState(this.listRequest, 'chapters.list', async () => {
        const result = await useApi().chapter.listChapters(projectId)
        this.byProjectId[projectId] = result.data
        return result
      })
    },
    async create(projectId: string, request: CreateChapterRequest) {
      return withApiRequestState(this.createRequest, 'chapters.create', async () => {
        const operation = await createChapterOperation(useApi().chapter, this.byProjectId[projectId] || [], projectId, request)
        this.byProjectId[projectId] = operation.chapters
        return operation.result
      })
    },
    async update(projectId: string, request: UpdateChapterRequest) {
      return withApiRequestState(this.updateRequest, 'chapters.update', async () => {
        const operation = await updateChapterOperation(useApi().chapter, this.byProjectId[projectId] || [], projectId, request)
        this.byProjectId[projectId] = operation.chapters
        return operation.result
      })
    },
    async loadVersions(projectId: string, chapterId: string) {
      return withApiRequestState(this.versionListRequest, 'chapter-versions.list', async () => {
        const result = await useApi().chapter.listChapterVersions(projectId, chapterId)
        this.versionsByChapterId[chapterId] = result.data
        return result
      })
    },
    async saveVersion(projectId: string, version: ChapterVersionWriteRequest) {
      return withApiRequestState(this.versionSaveRequest, 'chapter-versions.save', async () => {
        const result = await useApi().chapter.saveChapterVersion(projectId, version)
        this.versionsByChapterId[result.data.chapter_version.chapter_id] = mergeChapterVersions(
          this.versionsByChapterId[result.data.chapter_version.chapter_id] || [],
          result.data
        )
        return result
      })
    },
    setProjectChapters(projectId: string, chapters: Chapter[]) {
      this.byProjectId[projectId] = chapters
    }
  }
})
