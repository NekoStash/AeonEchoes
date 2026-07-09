package routes

import (
	"net/http"
	"time"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/config"
	"aeonechoes/server/internal/indexing"
	"aeonechoes/server/internal/provider"
	"aeonechoes/server/internal/repository"
	"aeonechoes/server/internal/retrieval"
	"aeonechoes/server/internal/skills"
	"aeonechoes/server/internal/tooling"
)

// Dependencies contains the infrastructure required by the versioned HTTP API.
type Dependencies struct {
	Config            config.Config
	Store             repository.AppStore
	Providers         provider.ProviderFactory
	Workflow          *agent.WorkflowRunner
	AgentRuntime      *agent.Runtime
	SkillService      *skills.Service
	ToolRegistry      *tooling.Registry
	MCPDefaultTimeout time.Duration
	Indexing          *indexing.Service
	Retrieval         *retrieval.Service
	NotifyIndexWorker func()
}

// Router owns /api/v1 route registration and handlers.
type Router struct {
	cfg               config.Config
	store             repository.AppStore
	providers         provider.ProviderFactory
	workflow          *agent.WorkflowRunner
	agentRuntime      *agent.Runtime
	skillService      *skills.Service
	toolRegistry      *tooling.Registry
	mcpDefaultTimeout time.Duration
	indexing          *indexing.Service
	retrieval         *retrieval.Service
	notifyIndexWorker func()
}

func NewRouter(deps Dependencies) *Router {
	return &Router{
		cfg:               deps.Config,
		store:             deps.Store,
		providers:         deps.Providers,
		workflow:          deps.Workflow,
		agentRuntime:      deps.AgentRuntime,
		skillService:      deps.SkillService,
		toolRegistry:      deps.ToolRegistry,
		mcpDefaultTimeout: deps.MCPDefaultTimeout,
		indexing:          deps.Indexing,
		retrieval:         deps.Retrieval,
		notifyIndexWorker: deps.NotifyIndexWorker,
	}
}

func (s *Router) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", s.v1Health)
	mux.HandleFunc("GET /api/v1/system/status", s.v1SystemStatus)

	mux.HandleFunc("GET /api/v1/projects", s.v1ListProjects)
	mux.HandleFunc("POST /api/v1/projects", s.v1CreateProject)
	mux.HandleFunc("GET /api/v1/projects/{projectID}", s.v1GetProject)
	mux.HandleFunc("POST /api/v1/project-seed-optimizations", s.v1OptimizeProjectSeed)

	mux.HandleFunc("GET /api/v1/projects/{projectID}/story-bibles/current", s.v1GetCurrentStoryBible)
	mux.HandleFunc("PUT /api/v1/projects/{projectID}/story-bibles/{storyBibleID}", s.v1UpdateStoryBible)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/story-bibles/{storyBibleID}/character-syncs", s.v1SyncCharacters)

	mux.HandleFunc("GET /api/v1/projects/{projectID}/chapters", s.v1ListChapters)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/chapters", s.v1CreateChapter)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/chapters/{chapterID}", s.v1GetChapter)
	mux.HandleFunc("PUT /api/v1/projects/{projectID}/chapters/{chapterID}", s.v1UpdateChapter)
	mux.HandleFunc("PATCH /api/v1/projects/{projectID}/chapters/{chapterID}", s.v1UpdateChapter)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/chapters/{chapterID}/versions", s.v1ListChapterVersions)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/chapters/{chapterID}/versions", s.v1CreateChapterVersion)

	mux.HandleFunc("GET /api/v1/projects/{projectID}/workflows", s.v1ListProjectWorkflows)
	mux.HandleFunc("GET /api/v1/workflows/{id}", s.v1GetWorkflow)

	mux.HandleFunc("POST /api/v1/projects/{projectID}/context-previews", s.v1PreviewContextSelection)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/chapters/{chapterID}/ideas", s.v1GenerateChapterIdea)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/chapters/{chapterID}/drafts", s.v1DraftChapter)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/character-profiles", s.v1GenerateCharacterProfiles)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/graph/expansions", s.v1ExpandGraph)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/retrieval/semantic-searches", s.v1SemanticSearch)

	mux.HandleFunc("GET /api/v1/providers", s.v1ListProviders)
	mux.HandleFunc("POST /api/v1/providers", s.v1CreateProvider)
	mux.HandleFunc("GET /api/v1/providers/{id}", s.v1GetProvider)
	mux.HandleFunc("PUT /api/v1/providers/{id}", s.v1UpdateProvider)
	mux.HandleFunc("DELETE /api/v1/providers/{id}", s.v1DeleteProvider)
	mux.HandleFunc("POST /api/v1/providers/{id}/model-refreshes", s.v1RefreshProviderModels)

	mux.HandleFunc("GET /api/v1/models", s.v1ListModels)
	mux.HandleFunc("POST /api/v1/models", s.v1CreateModel)
	mux.HandleFunc("GET /api/v1/models/{id}", s.v1GetModel)
	mux.HandleFunc("PUT /api/v1/models/{id}", s.v1UpdateModel)
	mux.HandleFunc("DELETE /api/v1/models/{id}", s.v1DeleteModel)
	mux.HandleFunc("GET /api/v1/model-routing", s.v1GetModelRouting)
	mux.HandleFunc("PUT /api/v1/model-routing", s.v1PutModelRouting)

	mux.HandleFunc("GET /api/v1/agents", s.v1ListAgents)
	mux.HandleFunc("POST /api/v1/agents", s.v1CreateAgent)
	mux.HandleFunc("GET /api/v1/agents/{id}", s.v1GetAgent)
	mux.HandleFunc("PUT /api/v1/agents/{id}", s.v1UpdateAgent)
	mux.HandleFunc("DELETE /api/v1/agents/{id}", s.v1DeleteAgent)
	mux.HandleFunc("POST /api/v1/agents/{id}/runs", s.v1RunAgent)
	mux.HandleFunc("GET /api/v1/agent-runs", s.v1ListAgentRuns)
	mux.HandleFunc("GET /api/v1/agent-runs/{id}", s.v1GetAgentRun)

	mux.HandleFunc("GET /api/v1/skill-sources", s.v1ListSkillSources)
	mux.HandleFunc("POST /api/v1/skill-sources", s.v1CreateSkillSource)
	mux.HandleFunc("POST /api/v1/skill-sources/default/scans", s.v1ScanDefaultSkillSource)
	mux.HandleFunc("POST /api/v1/skill-sources/{id}/scans", s.v1ScanSkillSource)
	mux.HandleFunc("GET /api/v1/skills", s.v1ListSkills)
	mux.HandleFunc("POST /api/v1/skills", s.v1CreateSkill)
	mux.HandleFunc("GET /api/v1/skills/{id}", s.v1GetSkill)
	mux.HandleFunc("PUT /api/v1/skills/{id}", s.v1UpdateSkill)
	mux.HandleFunc("PATCH /api/v1/skills/{id}", s.v1SetSkillEnabled)
	mux.HandleFunc("DELETE /api/v1/skills/{id}", s.v1DeleteSkill)

	mux.HandleFunc("GET /api/v1/mcp-servers", s.v1ListMCPServers)
	mux.HandleFunc("POST /api/v1/mcp-servers", s.v1CreateMCPServer)
	mux.HandleFunc("GET /api/v1/mcp-servers/{id}", s.v1GetMCPServer)
	mux.HandleFunc("PUT /api/v1/mcp-servers/{id}", s.v1UpdateMCPServer)
	mux.HandleFunc("PATCH /api/v1/mcp-servers/{id}", s.v1SetMCPServerEnabled)
	mux.HandleFunc("DELETE /api/v1/mcp-servers/{id}", s.v1DeleteMCPServer)
	mux.HandleFunc("POST /api/v1/mcp-servers/{id}/connection-tests", s.v1TestMCPServer)
	mux.HandleFunc("POST /api/v1/mcp-servers/{id}/tool-refreshes", s.v1RefreshMCPTools)
	mux.HandleFunc("GET /api/v1/mcp-servers/{id}/tools", s.v1ListMCPServerTools)

	mux.HandleFunc("GET /api/v1/tools", s.v1ListToolCatalog)
	mux.HandleFunc("PATCH /api/v1/tools/{id}", s.v1SetToolEnabled)
	mux.HandleFunc("GET /api/v1/tool-invocations", s.v1ListToolInvocations)

	mux.HandleFunc("GET /api/v1/index-jobs", s.v1ListIndexJobs)
	mux.HandleFunc("POST /api/v1/index-jobs/{id}/runs", s.v1RunIndexJob)
	mux.HandleFunc("POST /api/v1/index-runs", s.v1RunPendingIndexJobs)
	mux.HandleFunc("POST /api/v1/vector-index-rebuilds", s.v1RebuildVectors)

	mux.HandleFunc("GET /api/v1/settings", s.v1ListSettings)
	mux.HandleFunc("PUT /api/v1/settings/{scope}/{key}", s.v1UpsertSetting)
}
