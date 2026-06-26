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
	"aeonechoes/server/internal/extractor"
	"aeonechoes/server/internal/indexing"
	httpapi "aeonechoes/server/internal/infra/http"
	"aeonechoes/server/internal/memory"
	"aeonechoes/server/internal/postgres"
	"aeonechoes/server/internal/providerregistry"
	"aeonechoes/server/internal/repository"
	"aeonechoes/server/internal/retrieval"
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
