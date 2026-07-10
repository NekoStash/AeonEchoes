import type { ApiResult } from '~/shared/api'
import type { IndexJob, IndexJobListOptions, RebuildVectorsResponse, RunPendingIndexResponse } from './types'

export interface IndexJobApi {
  listIndexJobs(options?: string | IndexJobListOptions): Promise<ApiResult<IndexJob[]>>
  runIndexJob(id: string): Promise<ApiResult<IndexJob>>
  runPendingIndexJobs(projectId?: string, limit?: number): Promise<ApiResult<RunPendingIndexResponse>>
  rebuildVectors(): Promise<ApiResult<RebuildVectorsResponse>>
}
