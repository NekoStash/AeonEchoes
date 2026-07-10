package routes

import (
	"net/http"
	"time"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/config"
	"aeonechoes/server/internal/indexing"
	v1openapi "aeonechoes/server/internal/infra/http/v1/openapi"
	"aeonechoes/server/internal/infra/http/v1/respond"
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
	v1openapi.HandlerWithOptions(s, v1openapi.StdHTTPServerOptions{
		BaseURL:          "/api/v1",
		BaseRouter:       mux,
		ErrorHandlerFunc: s.handleOpenAPIError,
	})
}

func (s *Router) handleOpenAPIError(w http.ResponseWriter, r *http.Request, err error) {
	details := map[string]any{"cause": err.Error()}
	respond.Error(w, r, http.StatusBadRequest, "bad_request", "invalid request parameters", details)
}
