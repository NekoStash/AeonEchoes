import type { ApiResult } from '~/shared/api'
import type { GraphExpandRequest, GraphExpandResponse, SemanticSearchRequest, SemanticSearchResponse } from './types'

export interface GraphApi {
  expandGraph(request: GraphExpandRequest): Promise<ApiResult<GraphExpandResponse>>
  semanticSearch(projectId: string, request: SemanticSearchRequest): Promise<ApiResult<SemanticSearchResponse>>
}
