package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("AE_SERVER_HOST", "0.0.0.0")
	t.Setenv("AE_SERVER_PORT", "18080")
	t.Setenv("AE_DATA_DIR", "./tmp-data")
	t.Setenv("AE_PROVIDER_TRACE_ENABLED", "true")
	t.Setenv("AE_PROVIDER_TRACE_RETENTION_DAYS", "30")
	t.Setenv("AE_PROVIDER_TIMEOUT_SECONDS", "12")
	t.Setenv("AE_POSTGRES_DSN", "postgres://example")
	t.Setenv("AE_QDRANT_URL", "http://qdrant:6333")
	t.Setenv("AE_INDEX_WORKER_ENABLED", "false")
	t.Setenv("AE_INDEX_WORKER_INTERVAL_SECONDS", "21")
	t.Setenv("AE_INDEX_WORKER_BATCH_SIZE", "7")
	t.Setenv("AE_INDEX_WORKER_WAKE_DEBOUNCE_MILLISECONDS", "345")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Host != "0.0.0.0" || cfg.Port != 18080 {
		t.Fatalf("unexpected address config: %+v", cfg)
	}
	if !cfg.ProviderTraceEnabled || cfg.TraceRetentionDays != 30 {
		t.Fatalf("unexpected trace config: %+v", cfg)
	}
	if cfg.DefaultProviderTimeout != 12*time.Second {
		t.Fatalf("unexpected timeout: %s", cfg.DefaultProviderTimeout)
	}
	if cfg.PostgresDSN == "" || cfg.QdrantURL == "" {
		t.Fatalf("expected postgres and qdrant config to be loaded")
	}
	if cfg.IndexWorkerEnabled {
		t.Fatalf("expected index worker enabled false, got true")
	}
	if cfg.IndexWorkerIntervalSeconds != 21 || cfg.IndexWorkerBatchSize != 7 || cfg.IndexWorkerWakeDebounceMilliseconds != 345 {
		t.Fatalf("unexpected index worker config: %+v", cfg)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("AE_SERVER_PORT", "not-a-number")
	if _, err := Load(); err == nil {
		t.Fatalf("expected invalid port error")
	}
}

func TestLoadRejectsNegativeWakeDebounce(t *testing.T) {
	t.Setenv("AE_INDEX_WORKER_WAKE_DEBOUNCE_MILLISECONDS", "-1")
	_, err := Load()
	if err == nil {
		t.Fatalf("expected negative wake debounce error")
	}
	if !strings.Contains(err.Error(), "AE_INDEX_WORKER_WAKE_DEBOUNCE_MILLISECONDS") {
		t.Fatalf("unexpected error: %v", err)
	}
}
