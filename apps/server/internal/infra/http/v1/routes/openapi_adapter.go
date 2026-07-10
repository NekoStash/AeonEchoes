package routes

import (
	"net/http"

	v1openapi "aeonechoes/server/internal/infra/http/v1/openapi"
)

var _ v1openapi.ServerInterface = (*Router)(nil)

func (s *Router) ListAgentRuns(w http.ResponseWriter, r *http.Request, params v1openapi.ListAgentRunsParams) {
	s.v1ListAgentRuns(w, r)
}

func (s *Router) GetAgentRun(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1GetAgentRun(w, r)
}

func (s *Router) ListAgents(w http.ResponseWriter, r *http.Request, params v1openapi.ListAgentsParams) {
	s.v1ListAgents(w, r)
}

func (s *Router) CreateAgent(w http.ResponseWriter, r *http.Request) {
	s.v1CreateAgent(w, r)
}

func (s *Router) DeleteAgent(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1DeleteAgent(w, r)
}

func (s *Router) GetAgent(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1GetAgent(w, r)
}

func (s *Router) UpdateAgent(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1UpdateAgent(w, r)
}

func (s *Router) RunAgent(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1RunAgent(w, r)
}

func (s *Router) GetHealth(w http.ResponseWriter, r *http.Request) {
	s.v1Health(w, r)
}

func (s *Router) ListIndexJobs(w http.ResponseWriter, r *http.Request, params v1openapi.ListIndexJobsParams) {
	s.v1ListIndexJobs(w, r)
}

func (s *Router) RunIndexJob(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1RunIndexJob(w, r)
}

func (s *Router) RunPendingIndexJobs(w http.ResponseWriter, r *http.Request, params v1openapi.RunPendingIndexJobsParams) {
	s.v1RunPendingIndexJobs(w, r)
}

func (s *Router) ListMCPServers(w http.ResponseWriter, r *http.Request, params v1openapi.ListMCPServersParams) {
	s.v1ListMCPServers(w, r)
}

func (s *Router) CreateMCPServer(w http.ResponseWriter, r *http.Request) {
	s.v1CreateMCPServer(w, r)
}

func (s *Router) DeleteMCPServer(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1DeleteMCPServer(w, r)
}

func (s *Router) GetMCPServer(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1GetMCPServer(w, r)
}

func (s *Router) PatchMCPServer(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1SetMCPServerEnabled(w, r)
}

func (s *Router) UpdateMCPServer(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1UpdateMCPServer(w, r)
}

func (s *Router) TestMCPServerConnection(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1TestMCPServer(w, r)
}

func (s *Router) RefreshMCPTools(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1RefreshMCPTools(w, r)
}

func (s *Router) ListMCPServerTools(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1ListMCPServerTools(w, r)
}

func (s *Router) GetModelRouting(w http.ResponseWriter, r *http.Request) {
	s.v1GetModelRouting(w, r)
}

func (s *Router) PutModelRouting(w http.ResponseWriter, r *http.Request) {
	s.v1PutModelRouting(w, r)
}

func (s *Router) ListModels(w http.ResponseWriter, r *http.Request, params v1openapi.ListModelsParams) {
	s.v1ListModels(w, r)
}

func (s *Router) CreateModel(w http.ResponseWriter, r *http.Request) {
	s.v1CreateModel(w, r)
}

func (s *Router) DeleteModel(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1DeleteModel(w, r)
}

func (s *Router) GetModel(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1GetModel(w, r)
}

func (s *Router) UpdateModel(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1UpdateModel(w, r)
}

func (s *Router) OptimizeProjectSeed(w http.ResponseWriter, r *http.Request) {
	s.v1OptimizeProjectSeed(w, r)
}

func (s *Router) ListProjects(w http.ResponseWriter, r *http.Request) {
	s.v1ListProjects(w, r)
}

func (s *Router) CreateProject(w http.ResponseWriter, r *http.Request) {
	s.v1CreateProject(w, r)
}

func (s *Router) GetProject(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID) {
	s.v1GetProject(w, r)
}

func (s *Router) ListChapters(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID) {
	s.v1ListChapters(w, r)
}

func (s *Router) CreateChapter(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID) {
	s.v1CreateChapter(w, r)
}

func (s *Router) GetChapter(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID, chapterID v1openapi.ChapterID) {
	s.v1GetChapter(w, r)
}

func (s *Router) PatchChapter(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID, chapterID v1openapi.ChapterID) {
	s.v1UpdateChapter(w, r)
}

func (s *Router) UpdateChapter(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID, chapterID v1openapi.ChapterID) {
	s.v1UpdateChapter(w, r)
}

func (s *Router) DraftChapter(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID, chapterID v1openapi.ChapterID) {
	s.v1DraftChapter(w, r)
}

func (s *Router) GenerateChapterIdea(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID, chapterID v1openapi.ChapterID) {
	s.v1GenerateChapterIdea(w, r)
}

func (s *Router) ListChapterVersions(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID, chapterID v1openapi.ChapterID) {
	s.v1ListChapterVersions(w, r)
}

func (s *Router) CreateChapterVersion(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID, chapterID v1openapi.ChapterID) {
	s.v1CreateChapterVersion(w, r)
}

func (s *Router) GenerateCharacterProfiles(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID) {
	s.v1GenerateCharacterProfiles(w, r)
}

func (s *Router) PreviewContextSelection(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID) {
	s.v1PreviewContextSelection(w, r)
}

func (s *Router) ExpandGraph(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID) {
	s.v1ExpandGraph(w, r)
}

func (s *Router) SemanticSearch(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID) {
	s.v1SemanticSearch(w, r)
}

func (s *Router) GetCurrentStoryBible(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID) {
	s.v1GetCurrentStoryBible(w, r)
}

func (s *Router) UpdateStoryBible(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID, storyBibleID v1openapi.StoryBibleID) {
	s.v1UpdateStoryBible(w, r)
}

func (s *Router) SyncStoryBibleCharacters(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID, storyBibleID v1openapi.StoryBibleID) {
	s.v1SyncCharacters(w, r)
}

func (s *Router) ListProjectWorkflows(w http.ResponseWriter, r *http.Request, projectID v1openapi.ProjectID) {
	s.v1ListProjectWorkflows(w, r)
}

func (s *Router) ListProviders(w http.ResponseWriter, r *http.Request) {
	s.v1ListProviders(w, r)
}

func (s *Router) CreateProvider(w http.ResponseWriter, r *http.Request) {
	s.v1CreateProvider(w, r)
}

func (s *Router) DeleteProvider(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1DeleteProvider(w, r)
}

func (s *Router) GetProvider(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1GetProvider(w, r)
}

func (s *Router) UpdateProvider(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1UpdateProvider(w, r)
}

func (s *Router) RefreshProviderModels(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1RefreshProviderModels(w, r)
}

func (s *Router) ListSettings(w http.ResponseWriter, r *http.Request, params v1openapi.ListSettingsParams) {
	s.v1ListSettings(w, r)
}

func (s *Router) UpsertSetting(w http.ResponseWriter, r *http.Request, scope string, key string) {
	s.v1UpsertSetting(w, r)
}

func (s *Router) ListSkillSources(w http.ResponseWriter, r *http.Request, params v1openapi.ListSkillSourcesParams) {
	s.v1ListSkillSources(w, r)
}

func (s *Router) CreateSkillSource(w http.ResponseWriter, r *http.Request) {
	s.v1CreateSkillSource(w, r)
}

func (s *Router) ScanDefaultSkillSource(w http.ResponseWriter, r *http.Request) {
	s.v1ScanDefaultSkillSource(w, r)
}

func (s *Router) ScanSkillSource(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1ScanSkillSource(w, r)
}

func (s *Router) ListSkills(w http.ResponseWriter, r *http.Request, params v1openapi.ListSkillsParams) {
	s.v1ListSkills(w, r)
}

func (s *Router) CreateSkill(w http.ResponseWriter, r *http.Request) {
	s.v1CreateSkill(w, r)
}

func (s *Router) DeleteSkill(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1DeleteSkill(w, r)
}

func (s *Router) GetSkill(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1GetSkill(w, r)
}

func (s *Router) PatchSkill(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1SetSkillEnabled(w, r)
}

func (s *Router) UpdateSkill(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1UpdateSkill(w, r)
}

func (s *Router) GetSystemStatus(w http.ResponseWriter, r *http.Request) {
	s.v1SystemStatus(w, r)
}

func (s *Router) ListToolInvocations(w http.ResponseWriter, r *http.Request, params v1openapi.ListToolInvocationsParams) {
	s.v1ListToolInvocations(w, r)
}

func (s *Router) ListTools(w http.ResponseWriter, r *http.Request, params v1openapi.ListToolsParams) {
	s.v1ListToolCatalog(w, r)
}

func (s *Router) PatchTool(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1SetToolEnabled(w, r)
}

func (s *Router) RebuildVectorIndex(w http.ResponseWriter, r *http.Request) {
	s.v1RebuildVectors(w, r)
}

func (s *Router) GetWorkflow(w http.ResponseWriter, r *http.Request, id v1openapi.ID) {
	s.v1GetWorkflow(w, r)
}
