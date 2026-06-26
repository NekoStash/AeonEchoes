package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
)

func (s *Store) CreateProvider(cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.ProviderConfig{}, err
	}
	if !cfg.Type.Valid() {
		return domain.ProviderConfig{}, fmt.Errorf("provider type %q is not supported", cfg.Type)
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return domain.ProviderConfig{}, fmt.Errorf("provider name must not be empty")
	}
	if strings.TrimSpace(cfg.ID) == "" {
		id, err := s.NewID("provider")
		if err != nil {
			return domain.ProviderConfig{}, fmt.Errorf("generate provider id: %w", err)
		}
		cfg.ID = id
	}
	if cfg.DefaultRequestTimeoutSec <= 0 {
		cfg.DefaultRequestTimeoutSec = 60
	}
	n := now()
	cfg.CreatedAt = n
	cfg.UpdatedAt = n
	metadata, err := jsonbOrEmptyObject(cfg.Metadata)
	if err != nil {
		return domain.ProviderConfig{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO provider_configs(id, name, type, base_url, api_key_ciphertext, api_key_env, enabled, trace_enabled, trace_retention_days, default_request_timeout_sec, metadata, last_model_refresh_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, cfg.ID, cfg.Name, string(cfg.Type), cfg.BaseURL, cfg.APIKey, cfg.APIKeyEnv, cfg.Enabled, cfg.TraceEnabled, cfg.TraceRetentionDays, cfg.DefaultRequestTimeoutSec, metadata, cfg.LastModelRefreshAt, cfg.CreatedAt, cfg.UpdatedAt)
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("insert provider %q: %w", cfg.ID, err)
	}
	return cfg, nil
}

func (s *Store) UpdateProvider(id string, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.ProviderConfig{}, err
	}
	if strings.TrimSpace(id) == "" {
		return domain.ProviderConfig{}, fmt.Errorf("provider id must not be empty")
	}
	if !cfg.Type.Valid() {
		return domain.ProviderConfig{}, fmt.Errorf("provider type %q is not supported", cfg.Type)
	}
	existing, err := s.GetProvider(id)
	if err != nil {
		return domain.ProviderConfig{}, err
	}
	cfg.ID = id
	cfg.CreatedAt = existing.CreatedAt
	cfg.UpdatedAt = now()
	metadata, err := jsonbOrEmptyObject(cfg.Metadata)
	if err != nil {
		return domain.ProviderConfig{}, err
	}
	result, err := s.pool.Exec(context.Background(), `
UPDATE provider_configs
SET name=$2, type=$3, base_url=$4, api_key_ciphertext=$5, api_key_env=$6, enabled=$7, trace_enabled=$8,
    trace_retention_days=$9, default_request_timeout_sec=$10, metadata=$11, last_model_refresh_at=$12, updated_at=$13
WHERE id=$1`, cfg.ID, cfg.Name, string(cfg.Type), cfg.BaseURL, cfg.APIKey, cfg.APIKeyEnv, cfg.Enabled, cfg.TraceEnabled, cfg.TraceRetentionDays, cfg.DefaultRequestTimeoutSec, metadata, cfg.LastModelRefreshAt, cfg.UpdatedAt)
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("update provider %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return domain.ProviderConfig{}, fmt.Errorf("provider %q not found", id)
	}
	return cfg, nil
}

func (s *Store) GetProvider(id string) (domain.ProviderConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.ProviderConfig{}, err
	}
	row := s.pool.QueryRow(context.Background(), `
SELECT id, name, type, base_url, api_key_ciphertext, api_key_env, enabled, trace_enabled, trace_retention_days,
       default_request_timeout_sec, metadata, created_at, updated_at, last_model_refresh_at
FROM provider_configs WHERE id=$1`, id)
	item, err := scanProvider(row)
	if err != nil {
		if isNoRows(err) {
			return domain.ProviderConfig{}, fmt.Errorf("provider %q not found", id)
		}
		return domain.ProviderConfig{}, fmt.Errorf("get provider %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) DeleteProvider(id string) error {
	if err := requireStore(s); err != nil {
		return err
	}
	result, err := s.pool.Exec(context.Background(), `DELETE FROM provider_configs WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete provider %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("provider %q not found", id)
	}
	return nil
}

func (s *Store) ListProviders() ([]domain.ProviderConfig, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(context.Background(), `
SELECT id, name, type, base_url, api_key_ciphertext, api_key_env, enabled, trace_enabled, trace_retention_days,
       default_request_timeout_sec, metadata, created_at, updated_at, last_model_refresh_at
FROM provider_configs ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer rows.Close()
	items := make([]domain.ProviderConfig, 0)
	for rows.Next() {
		item, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate providers: %w", err)
	}
	return items, nil
}

func (s *Store) TouchProviderModelRefresh(id string) error {
	if err := requireStore(s); err != nil {
		return err
	}
	n := now()
	result, err := s.pool.Exec(context.Background(), `UPDATE provider_configs SET last_model_refresh_at=$2, updated_at=$2 WHERE id=$1`, id, n)
	if err != nil {
		return fmt.Errorf("touch provider model refresh %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("provider %q not found", id)
	}
	return nil
}

type providerScanner interface {
	Scan(dest ...any) error
}

func scanProvider(scanner providerScanner) (domain.ProviderConfig, error) {
	var item domain.ProviderConfig
	var typ string
	var metadata []byte
	var refreshedAt *time.Time
	if err := scanner.Scan(&item.ID, &item.Name, &typ, &item.BaseURL, &item.APIKey, &item.APIKeyEnv, &item.Enabled, &item.TraceEnabled, &item.TraceRetentionDays, &item.DefaultRequestTimeoutSec, &metadata, &item.CreatedAt, &item.UpdatedAt, &refreshedAt); err != nil {
		return domain.ProviderConfig{}, err
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.ProviderConfig{}, err
	}
	item.Type = domain.ProviderType(typ)
	item.Metadata = parsedMetadata
	item.LastModelRefreshAt = refreshedAt
	return item, nil
}
