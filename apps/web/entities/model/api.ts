import type { ApiResult } from '~/shared/api'
import type { ModelConfig, ModelUsageSettings } from './types'

export interface ModelApi {
  listModels(kind?: string): Promise<ApiResult<ModelConfig[]>>
  saveModel(model: ModelConfig): Promise<ApiResult<ModelConfig>>
  deleteModel(id: string): Promise<ApiResult<{ status: string }>>
  refreshModels(providerId: string): Promise<ApiResult<ModelConfig[]>>
  getModelUsageSettings(): Promise<ApiResult<ModelUsageSettings>>
  saveModelUsageSettings(settings: ModelUsageSettings): Promise<ApiResult<ModelUsageSettings>>
}
