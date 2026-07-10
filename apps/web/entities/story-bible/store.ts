import { defineStore } from 'pinia'
import type { StoryBible } from './types'
import { syncStoryBibleCharacters } from '~/features/character-sync/operations'
import { createApiRequestState, withApiRequestState } from '~/shared/store'

export const useStoryBibleStore = defineStore('story-bible-domain', {
  state: () => ({
    byProjectId: {} as Record<string, StoryBible>,
    activeProjectId: '',
    loadRequest: createApiRequestState(),
    saveRequest: createApiRequestState(),
    syncRequest: createApiRequestState()
  }),
  getters: {
    active: (state) => state.byProjectId[state.activeProjectId] || null
  },
  actions: {
    async load(projectId: string) {
      return withApiRequestState(this.loadRequest, 'story-bibles.get', async () => {
        const result = await useApi().storyBible.getStoryBible(projectId)
        this.byProjectId[projectId] = result.data
        this.activeProjectId = projectId
        return result
      })
    },
    async save(projectId: string, bible: StoryBible) {
      return withApiRequestState(this.saveRequest, 'story-bibles.update', async () => {
        const result = await useApi().storyBible.updateStoryBible(projectId, bible)
        this.byProjectId[projectId] = result.data
        this.activeProjectId = projectId
        return result
      })
    },
    async syncCharacters(projectId: string, bible: StoryBible) {
      return withApiRequestState(this.syncRequest, 'story-bibles.character-sync', async () => {
        return syncStoryBibleCharacters(useApi().storyBible, projectId, bible)
      })
    },
    set(projectId: string, bible: StoryBible) {
      this.byProjectId[projectId] = bible
      this.activeProjectId = projectId
    }
  }
})
