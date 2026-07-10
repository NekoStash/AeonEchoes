import { defineStore } from 'pinia'
import type { ModelConfig, ModelUsageSettings } from './types'
import { createApiRequestState, withApiRequestState } from '~/shared/store'

function mergeModels(current: ModelConfig[], updates: ModelConfig[]) {
  const updatedIds = new Set(updates.map((model) => model.id))
  return [...current.filter((model) => !updatedIds.has(model.id)), ...updates]
    .sort((left, right) => (left.display_name || left.name).localeCompare(right.display_name || right.name))
}

export const useModelStore = defineStore('model-domain', {
  state: () => ({
    items: [] as ModelConfig[],
    usageSettings: null as ModelUsageSettings | null,
    listRequest: createApiRequestState(),
    saveRequest: createApiRequestState(),
    deleteRequest: createApiRequestState(),
    refreshRequest: createApiRequestState(),
    usageLoadRequest: createApiRequestState(),
    usageSaveRequest: createApiRequestState()
  }),
  getters: {
    enabled: (state) => state.items.filter((model) => model.enabled)
  },
  actions: {
    async load(kind?: string) {
      return withApiRequestState(this.listRequest, 'models.list', async () => {
        const result = await useApi().model.listModels(kind)
        this.items = result.data
        return result
      })
    },
    async save(model: ModelConfig) {
      return withApiRequestState(this.saveRequest, 'models.save', async () => {
        const result = await useApi().model.saveModel(model)
        this.items = mergeModels(this.items, [result.data])
        return result
      })
    },
    async remove(id: string) {
      return withApiRequestState(this.deleteRequest, 'models.delete', async () => {
        const result = await useApi().model.deleteModel(id)
        this.items = this.items.filter((item) => item.id !== id)
        return result
      })
    },
    async refresh(providerId: string) {
      return withApiRequestState(this.refreshRequest, 'models.refresh', async () => {
        const result = await useApi().model.refreshModels(providerId)
        const refreshedIds = new Set(result.data.map((model) => model.id))
        this.items = mergeModels(this.items.filter((model) => !refreshedIds.has(model.id)), result.data)
        return result
      })
    },
    async loadUsageSettings() {
      return withApiRequestState(this.usageLoadRequest, 'models.usage-settings.get', async () => {
        const result = await useApi().model.getModelUsageSettings()
        this.usageSettings = result.data
        return result
      })
    },
    async saveUsageSettings(settings: ModelUsageSettings) {
      return withApiRequestState(this.usageSaveRequest, 'models.usage-settings.save', async () => {
        const result = await useApi().model.saveModelUsageSettings(settings)
        this.usageSettings = result.data
        return result
      })
    }
  }
})
