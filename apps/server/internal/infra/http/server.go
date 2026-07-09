package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/config"
	"aeonechoes/server/internal/indexing"
	"aeonechoes/server/internal/infra/http/v1/respond"
	v1routes "aeonechoes/server/internal/infra/http/v1/routes"
	"aeonechoes/server/internal/provider"
	"aeonechoes/server/internal/repository"
	"aeonechoes/server/internal/retrieval"
	"aeonechoes/server/internal/skills"
	"aeonechoes/server/internal/tooling"
)

// Server owns HTTP routing and request middleware.
type Server struct {
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
	indexWake         indexing.WakeNotifier
	logger            *slog.Logger
}

func NewServer(cfg config.Config, store repository.AppStore, providers provider.ProviderFactory, workflow *agent.WorkflowRunner, indexingService *indexing.Service, retrievalService *retrieval.Service, indexWake indexing.WakeNotifier, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{cfg: cfg, store: store, providers: providers, workflow: workflow, mcpDefaultTimeout: cfg.MCPDefaultTimeout, indexing: indexingService, retrieval: retrievalService, indexWake: indexWake, logger: logger}
}

func (s *Server) ConfigureAgents(runtime *agent.Runtime, skillService *skills.Service, toolRegistry *tooling.Registry, mcpDefaultTimeout time.Duration) {
	s.agentRuntime = runtime
	s.skillService = skillService
	s.toolRegistry = toolRegistry
	if mcpDefaultTimeout > 0 {
		s.mcpDefaultTimeout = mcpDefaultTimeout
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	v1routes.NewRouter(v1routes.Dependencies{
		Config:            s.cfg,
		Store:             s.store,
		Providers:         s.providers,
		Workflow:          s.workflow,
		AgentRuntime:      s.agentRuntime,
		SkillService:      s.skillService,
		ToolRegistry:      s.toolRegistry,
		MCPDefaultTimeout: s.mcpDefaultTimeout,
		Indexing:          s.indexing,
		Retrieval:         s.retrieval,
		NotifyIndexWorker: s.notifyIndexWorker,
	}).Register(mux)
	return loggingMiddleware(s.logger, corsMiddleware(s.cfg.CORSAllowedOrigins, jsonMiddleware(respond.RequestIDMiddleware(mux))))
}

func (s *Server) notifyIndexWorker() {
	if s == nil || s.indexWake == nil {
		return
	}
	s.indexWake.Notify()
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, origin := range allowedOrigins {
		if strings.TrimSpace(origin) != "" {
			allowed[strings.TrimSpace(origin)] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (allowed[origin] || allowed["*"]) {
			if allowed["*"] {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept,Authorization,Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	})
}
