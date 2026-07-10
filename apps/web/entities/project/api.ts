import type { ApiResult } from '~/shared/api'
import type { InitializeProjectResponse, ProjectSeed, ProjectSummary, StoryBible } from '~/lib/types'

export interface ProjectApi {
  listProjects(): Promise<ApiResult<ProjectSummary[]>>
  initializeProject(seed: ProjectSeed): Promise<ApiResult<StoryBible>>
  initializeProjectFull(seed: ProjectSeed): Promise<ApiResult<InitializeProjectResponse>>
  optimizeProjectSeed(seed: ProjectSeed): Promise<ApiResult<ProjectSeed>>
}
