package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/config"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/extractor"
	"aeonechoes/server/internal/indexing"
	httpapi "aeonechoes/server/internal/infra/http"
	"aeonechoes/server/internal/memory"
	"aeonechoes/server/internal/postgres"
	"aeonechoes/server/internal/providerregistry"
	"aeonechoes/server/internal/repository"
	"aeonechoes/server/internal/retrieval"
	"aeonechoes/server/internal/skills"
	"aeonechoes/server/internal/tooling"
	"aeonechoes/server/internal/vector"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, closeStore, err := initStore(ctx, cfg)
	if err != nil {
		logger.Error("initialize store failed", "error", err)
		os.Exit(1)
	}
	defer closeStore()

	providerRegistry := providerregistry.New(nil, cfg.DefaultProviderTimeout)
	roleRegistry := agent.NewAgentRoleRegistry()
	modelRouter := agent.NewModelRouter(store, roleRegistry)
	toolRuntime := agent.NewToolRuntime(store)
	contextBuilder := agent.NewContextPackBuilder(store, toolRuntime, store)
	workflowRunner := agent.NewWorkflowRunner(store, modelRouter, contextBuilder, providerRegistry)
	toolRegistry := tooling.NewRegistry(store, store)
	if err := toolRegistry.SeedBuiltinTools(ctx); err != nil {
		logger.Error("seed builtin tools failed", "error", err)
		os.Exit(1)
	}
	if err := seedDefaultAgents(store); err != nil {
		logger.Error("seed default agents failed", "error", err)
		os.Exit(1)
	}
	skillService := skills.NewService(store, cfg.SkillsDir)
	if cfg.SkillsAutoScan && cfg.SkillsScanOnStart {
		if err := os.MkdirAll(cfg.SkillsDir, 0o755); err != nil {
			logger.Error("create skills directory failed", "path", cfg.SkillsDir, "error", err)
			os.Exit(1)
		}
		if _, err := skillService.ScanDefault(ctx); err != nil {
			logger.Error("scan default skills failed", "path", cfg.SkillsDir, "error", err)
			os.Exit(1)
		}
	} else if _, err := skillService.EnsureDefaultSource(ctx); err != nil {
		logger.Error("ensure default skills source failed", "error", err)
		os.Exit(1)
	}
	agentRuntime := agent.NewRuntime(store, modelRouter, contextBuilder, providerRegistry, toolRegistry)
	indexingService := initIndexingService(cfg, store, modelRouter, providerRegistry, logger)
	retrievalService := initRetrievalService(cfg, modelRouter, providerRegistry, logger)

	var indexWorker *indexing.Worker
	var indexWake indexing.WakeNotifier
	if indexingService != nil && cfg.IndexWorkerEnabled {
		indexWorker, err = indexing.NewWorker(indexingService, logger, time.Duration(cfg.IndexWorkerIntervalSeconds)*time.Second, cfg.IndexWorkerBatchSize, time.Duration(cfg.IndexWorkerWakeDebounceMilliseconds)*time.Millisecond)
		if err != nil {
			logger.Error("initialize index worker failed", "error", err)
			os.Exit(1)
		}
		indexWake = indexWorker
	}

	api := httpapi.NewServer(cfg, store, providerRegistry, workflowRunner, indexingService, retrievalService, indexWake, logger)
	api.ConfigureAgents(agentRuntime, skillService, toolRegistry, cfg.MCPDefaultTimeout)
	if indexWorker != nil {
		go func() {
			if err := indexWorker.Run(ctx); err != nil {
				logger.Error("index worker stopped with error", "error", err)
				stop()
			}
		}()
	}
	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("novelai server starting", "addr", cfg.Addr(), "data_dir", cfg.DataDir, "postgres_configured", cfg.PostgresDSN != "", "qdrant_configured", cfg.QdrantURL != "")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("novelai server stopped")
}

func seedDefaultAgents(store repository.AppStore) error {
	defaults := []domain.AgentConfig{
		{ID: "plot-architect", Name: "Plot Architect", Role: domain.AgentRolePlotArchitect, Enabled: true, SystemPrompt: "你负责规划章节、冲突推进和叙事承诺，输出可执行的小说策划结果。"},
		{ID: "character-keeper", Name: "Character Keeper", Role: domain.AgentRoleCharacterKeeper, Enabled: true, SystemPrompt: "你负责维护角色连续性、动机、秘密和人物弧，避免破坏既有设定。"},
		{ID: "writer", Name: "Writer", Role: domain.AgentRoleWriter, Enabled: true, SystemPrompt: "你负责根据上下文包写作正文，保持风格一致并遵守连续性。"},
		{ID: "editor", Name: "Editor", Role: domain.AgentRoleEditor, Enabled: true, SystemPrompt: "你负责润色、修订和压实文本，不改变核心事实。"},
		{ID: "fact-extractor", Name: "Fact Extractor", Role: domain.AgentRoleFactExtractor, Enabled: true, SystemPrompt: "你负责从文本中抽取可验证事实，保持原子化和可追溯。"},
		{ID: "graph-curator", Name: "Graph Curator", Role: domain.AgentRoleGraphCurator, Enabled: true, SystemPrompt: "你负责维护叙事图谱、关系和时间线结构。"},
	}
	for _, cfg := range defaults {
		if _, err := store.GetAgentConfig(cfg.ID); err == nil {
			continue
		}
		if _, err := store.CreateAgentConfig(cfg); err != nil {
			return err
		}
	}
	return nil
}

func initStore(ctx context.Context, cfg config.Config) (repository.AppStore, func(), error) {
	if strings.TrimSpace(cfg.PostgresDSN) == "" {
		return memory.NewStore(), func() {}, nil
	}
	pool, err := postgres.Connect(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, nil, err
	}
	if err := postgres.RunMigrations(ctx, pool, ""); err != nil {
		postgres.Close(pool)
		return nil, nil, err
	}
	store, err := postgres.NewStore(pool)
	if err != nil {
		postgres.Close(pool)
		return nil, nil, err
	}
	return store, func() { postgres.Close(pool) }, nil
}

func initIndexingService(cfg config.Config, store repository.AppStore, router *agent.ModelRouter, providers *providerregistry.Registry, logger *slog.Logger) *indexing.Service {
	knowledgeExtractor := extractor.NewDeterministicExtractor()
	if strings.TrimSpace(cfg.QdrantURL) == "" {
		return indexing.NewService(store, router, providers, nil, knowledgeExtractor)
	}
	client, err := vector.NewQdrantClient(cfg.QdrantURL, cfg.QdrantAPIKey, "aeonechoes_context", nil)
	if err != nil {
		logger.Error("initialize qdrant client failed", "error", err)
		return indexing.NewService(store, router, providers, nil, knowledgeExtractor)
	}
	return indexing.NewService(store, router, providers, client, knowledgeExtractor)
}

func initRetrievalService(cfg config.Config, router *agent.ModelRouter, providers *providerregistry.Registry, logger *slog.Logger) *retrieval.Service {
	if strings.TrimSpace(cfg.QdrantURL) == "" {
		return nil
	}
	client, err := vector.NewQdrantClient(cfg.QdrantURL, cfg.QdrantAPIKey, "aeonechoes_context", nil)
	if err != nil {
		logger.Error("initialize retrieval qdrant client failed", "error", err)
		return nil
	}
	return retrieval.NewService(router, providers, client)
}
