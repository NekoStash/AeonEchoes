package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config contains process-level server and infrastructure settings.
type Config struct {
	Host                                string
	Port                                int
	DataDir                             string
	ProviderTraceEnabled                bool
	TraceRetentionDays                  int
	DefaultProviderTimeout              time.Duration
	PostgresDSN                         string
	QdrantURL                           string
	QdrantAPIKey                        string
	CORSAllowedOrigins                  []string
	IndexWorkerEnabled                  bool
	IndexWorkerIntervalSeconds          int
	IndexWorkerBatchSize                int
	IndexWorkerWakeDebounceMilliseconds int
	SkillsDir                           string
	SkillsAutoScan                      bool
	SkillsScanOnStart                   bool
	MCPDefaultTimeout                   time.Duration
}

// Load reads configuration from environment variables and validates it immediately.
func Load() (Config, error) {
	port, err := getEnvInt("AE_SERVER_PORT", 8080)
	if err != nil {
		return Config{}, err
	}
	traceEnabled, err := getEnvBool("AE_PROVIDER_TRACE_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	traceRetentionDays, err := getEnvInt("AE_PROVIDER_TRACE_RETENTION_DAYS", 14)
	if err != nil {
		return Config{}, err
	}
	timeoutSeconds, err := getEnvInt("AE_PROVIDER_TIMEOUT_SECONDS", 60)
	if err != nil {
		return Config{}, err
	}
	indexWorkerEnabled, err := getEnvBool("AE_INDEX_WORKER_ENABLED", true)
	if err != nil {
		return Config{}, err
	}
	indexWorkerIntervalSeconds, err := getEnvInt("AE_INDEX_WORKER_INTERVAL_SECONDS", 15)
	if err != nil {
		return Config{}, err
	}
	indexWorkerBatchSize, err := getEnvInt("AE_INDEX_WORKER_BATCH_SIZE", 10)
	if err != nil {
		return Config{}, err
	}
	indexWorkerWakeDebounceMilliseconds, err := getEnvInt("AE_INDEX_WORKER_WAKE_DEBOUNCE_MILLISECONDS", 250)
	if err != nil {
		return Config{}, err
	}
	skillsAutoScan, err := getEnvBool("AE_SKILLS_AUTO_SCAN", true)
	if err != nil {
		return Config{}, err
	}
	skillsScanOnStart, err := getEnvBool("AE_SKILLS_SCAN_ON_START", true)
	if err != nil {
		return Config{}, err
	}
	mcpDefaultTimeoutSeconds, err := getEnvInt("AE_MCP_DEFAULT_TIMEOUT_SECONDS", 60)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Host:                                getEnv("AE_SERVER_HOST", "127.0.0.1"),
		Port:                                port,
		DataDir:                             getEnv("AE_DATA_DIR", filepath.Join(".", "data")),
		ProviderTraceEnabled:                traceEnabled,
		TraceRetentionDays:                  traceRetentionDays,
		DefaultProviderTimeout:              time.Duration(timeoutSeconds) * time.Second,
		PostgresDSN:                         strings.TrimSpace(os.Getenv("AE_POSTGRES_DSN")),
		QdrantURL:                           strings.TrimSpace(os.Getenv("AE_QDRANT_URL")),
		QdrantAPIKey:                        strings.TrimSpace(os.Getenv("AE_QDRANT_API_KEY")),
		CORSAllowedOrigins:                  splitCSV(getEnv("AE_CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000,http://localhost:13000,http://127.0.0.1:13000")),
		IndexWorkerEnabled:                  indexWorkerEnabled,
		IndexWorkerIntervalSeconds:          indexWorkerIntervalSeconds,
		IndexWorkerBatchSize:                indexWorkerBatchSize,
		IndexWorkerWakeDebounceMilliseconds: indexWorkerWakeDebounceMilliseconds,
		SkillsDir:                           getEnv("AE_SKILLS_DIR", filepath.Join(".", "skills")),
		SkillsAutoScan:                      skillsAutoScan,
		SkillsScanOnStart:                   skillsScanOnStart,
		MCPDefaultTimeout:                   time.Duration(mcpDefaultTimeoutSeconds) * time.Second,
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("AE_SERVER_HOST must not be empty")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("AE_SERVER_PORT must be between 1 and 65535, got %d", c.Port)
	}
	if strings.TrimSpace(c.DataDir) == "" {
		return fmt.Errorf("AE_DATA_DIR must not be empty")
	}
	if c.TraceRetentionDays < 0 {
		return fmt.Errorf("AE_PROVIDER_TRACE_RETENTION_DAYS must not be negative")
	}
	if c.DefaultProviderTimeout <= 0 {
		return fmt.Errorf("AE_PROVIDER_TIMEOUT_SECONDS must be positive")
	}
	if c.IndexWorkerIntervalSeconds <= 0 {
		return fmt.Errorf("AE_INDEX_WORKER_INTERVAL_SECONDS must be positive")
	}
	if c.IndexWorkerBatchSize <= 0 {
		return fmt.Errorf("AE_INDEX_WORKER_BATCH_SIZE must be positive")
	}
	if c.IndexWorkerWakeDebounceMilliseconds < 0 {
		return fmt.Errorf("AE_INDEX_WORKER_WAKE_DEBOUNCE_MILLISECONDS must not be negative")
	}
	if strings.TrimSpace(c.SkillsDir) == "" {
		return fmt.Errorf("AE_SKILLS_DIR must not be empty")
	}
	if c.MCPDefaultTimeout <= 0 {
		return fmt.Errorf("AE_MCP_DEFAULT_TIMEOUT_SECONDS must be positive")
	}
	return nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func getEnvInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}

func getEnvBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}
	return parsed, nil
}
