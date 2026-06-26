package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
)

func (s *Store) CreateModel(cfg domain.ModelConfig) (domain.ModelConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.ModelConfig{}, err
	}
	if strings.TrimSpace(cfg.ProviderID) == "" {
		return domain.ModelConfig{}, fmt.Errorf("model provider_id must not be empty")
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return domain.ModelConfig{}, fmt.Errorf("model name must not be empty")
	}
	if !cfg.Kind.Valid() {
		return domain.ModelConfig{}, fmt.Errorf("model kind %q is invalid", cfg.Kind)
	}
	providerCfg, err := s.GetProvider(cfg.ProviderID)
	if err != nil {
		return domain.ModelConfig{}, fmt.Errorf("provider %q not found for model: %w", cfg.ProviderID, err)
	}
	if strings.TrimSpace(cfg.ID) == "" {
		id, err := s.NewID("model")
		if err != nil {
			return domain.ModelConfig{}, fmt.Errorf("generate model id: %w", err)
		}
		cfg.ID = id
	}
	cfg.ProviderType = providerCfg.Type
	if cfg.RoutingWeight == 0 {
		cfg.RoutingWeight = 100
	}
	n := now()
	cfg.CreatedAt = n
	cfg.UpdatedAt = n
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return domain.ModelConfig{}, fmt.Errorf("begin create model %q: %w", cfg.ID, err)
	}
	defer tx.Rollback(context.Background())
	if cfg.DefaultForKind {
		if _, err := tx.Exec(context.Background(), `UPDATE model_configs SET default_for_kind=FALSE, updated_at=$2 WHERE kind=$1 AND id <> $3`, string(cfg.Kind), n, cfg.ID); err != nil {
			return domain.ModelConfig{}, fmt.Errorf("clear default models for kind %q: %w", cfg.Kind, err)
		}
	}
	roles, metadata, err := modelJSON(cfg)
	if err != nil {
		return domain.ModelConfig{}, err
	}
	_, err = tx.Exec(context.Background(), `
INSERT INTO model_configs(id, provider_id, provider_type, name, display_name, kind, context_window, max_output_tokens, dimension,
    supports_tools, supports_streaming, default_for_kind, enabled, cost_input_per_mtok, cost_output_per_mtok, routing_weight,
    allowed_agent_roles, metadata, last_seen_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`, cfg.ID, cfg.ProviderID, string(cfg.ProviderType), cfg.Name, cfg.DisplayName, string(cfg.Kind), cfg.ContextWindow, cfg.MaxOutputTokens, cfg.Dimension, cfg.SupportsTools, cfg.SupportsStreaming, cfg.DefaultForKind, cfg.Enabled, cfg.CostInputPerMTok, cfg.CostOutputPerMTok, cfg.RoutingWeight, roles, metadata, cfg.LastSeenAt, cfg.CreatedAt, cfg.UpdatedAt)
	if err != nil {
		return domain.ModelConfig{}, fmt.Errorf("insert model %q: %w", cfg.ID, err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return domain.ModelConfig{}, fmt.Errorf("commit create model %q: %w", cfg.ID, err)
	}
	return cfg, nil
}

func (s *Store) UpdateModel(id string, cfg domain.ModelConfig) (domain.ModelConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.ModelConfig{}, err
	}
	if strings.TrimSpace(id) == "" {
		return domain.ModelConfig{}, fmt.Errorf("model id must not be empty")
	}
	if !cfg.Kind.Valid() {
		return domain.ModelConfig{}, fmt.Errorf("model kind %q is invalid", cfg.Kind)
	}
	existing, err := s.GetModel(id)
	if err != nil {
		return domain.ModelConfig{}, err
	}
	providerCfg, err := s.GetProvider(cfg.ProviderID)
	if err != nil {
		return domain.ModelConfig{}, fmt.Errorf("provider %q not found for model: %w", cfg.ProviderID, err)
	}
	cfg.ID = id
	cfg.ProviderType = providerCfg.Type
	cfg.CreatedAt = existing.CreatedAt
	cfg.UpdatedAt = now()
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return domain.ModelConfig{}, fmt.Errorf("begin update model %q: %w", id, err)
	}
	defer tx.Rollback(context.Background())
	if cfg.DefaultForKind {
		if _, err := tx.Exec(context.Background(), `UPDATE model_configs SET default_for_kind=FALSE, updated_at=$2 WHERE kind=$1 AND id <> $3`, string(cfg.Kind), cfg.UpdatedAt, id); err != nil {
			return domain.ModelConfig{}, fmt.Errorf("clear default models for kind %q: %w", cfg.Kind, err)
		}
	}
	roles, metadata, err := modelJSON(cfg)
	if err != nil {
		return domain.ModelConfig{}, err
	}
	result, err := tx.Exec(context.Background(), `
UPDATE model_configs
SET provider_id=$2, provider_type=$3, name=$4, display_name=$5, kind=$6, context_window=$7, max_output_tokens=$8,
    dimension=$9, supports_tools=$10, supports_streaming=$11, default_for_kind=$12, enabled=$13,
    cost_input_per_mtok=$14, cost_output_per_mtok=$15, routing_weight=$16, allowed_agent_roles=$17,
    metadata=$18, last_seen_at=$19, updated_at=$20
WHERE id=$1`, cfg.ID, cfg.ProviderID, string(cfg.ProviderType), cfg.Name, cfg.DisplayName, string(cfg.Kind), cfg.ContextWindow, cfg.MaxOutputTokens, cfg.Dimension, cfg.SupportsTools, cfg.SupportsStreaming, cfg.DefaultForKind, cfg.Enabled, cfg.CostInputPerMTok, cfg.CostOutputPerMTok, cfg.RoutingWeight, roles, metadata, cfg.LastSeenAt, cfg.UpdatedAt)
	if err != nil {
		return domain.ModelConfig{}, fmt.Errorf("update model %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return domain.ModelConfig{}, fmt.Errorf("model %q not found", id)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return domain.ModelConfig{}, fmt.Errorf("commit update model %q: %w", id, err)
	}
	return cfg, nil
}

func (s *Store) UpsertModel(cfg domain.ModelConfig) (domain.ModelConfig, error) {
	if strings.TrimSpace(cfg.ID) == "" {
		return s.CreateModel(cfg)
	}
	_, err := s.GetModel(cfg.ID)
	if err == nil {
		return s.UpdateModel(cfg.ID, cfg)
	}
	if strings.Contains(err.Error(), "not found") {
		return s.CreateModel(cfg)
	}
	return domain.ModelConfig{}, err
}

func (s *Store) GetModel(id string) (domain.ModelConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.ModelConfig{}, err
	}
	row := s.pool.QueryRow(context.Background(), modelSelectSQL()+` WHERE id=$1`, id)
	item, err := scanModel(row)
	if err != nil {
		if isNoRows(err) {
			return domain.ModelConfig{}, fmt.Errorf("model %q not found", id)
		}
		return domain.ModelConfig{}, fmt.Errorf("get model %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) DeleteModel(id string) error {
	if err := requireStore(s); err != nil {
		return err
	}
	result, err := s.pool.Exec(context.Background(), `DELETE FROM model_configs WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete model %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("model %q not found", id)
	}
	return nil
}

func (s *Store) ListModels() ([]domain.ModelConfig, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	return s.queryModels(modelSelectSQL() + ` ORDER BY created_at ASC, id ASC`)
}

func (s *Store) ListModelsByKind(kind domain.ModelKind) ([]domain.ModelConfig, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	return s.queryModels(modelSelectSQL()+` WHERE kind=$1 AND enabled=TRUE ORDER BY default_for_kind DESC, routing_weight DESC, created_at ASC, id ASC`, string(kind))
}

func (s *Store) queryModels(sql string, args ...any) ([]domain.ModelConfig, error) {
	rows, err := s.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query models: %w", err)
	}
	defer rows.Close()
	items := make([]domain.ModelConfig, 0)
	for rows.Next() {
		item, err := scanModel(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate models: %w", err)
	}
	return items, nil
}

func modelJSON(cfg domain.ModelConfig) ([]byte, []byte, error) {
	roles, err := jsonbOrEmptyArray(cfg.AllowedAgentRoles)
	if err != nil {
		return nil, nil, err
	}
	metadata, err := jsonbOrEmptyObject(cfg.Metadata)
	if err != nil {
		return nil, nil, err
	}
	return roles, metadata, nil
}

func modelSelectSQL() string {
	return `SELECT id, provider_id, provider_type, name, display_name, kind, context_window, max_output_tokens, dimension,
       supports_tools, supports_streaming, default_for_kind, enabled, cost_input_per_mtok, cost_output_per_mtok,
       routing_weight, allowed_agent_roles, metadata, created_at, updated_at, last_seen_at FROM model_configs`
}

type modelScanner interface {
	Scan(dest ...any) error
}

func scanModel(scanner modelScanner) (domain.ModelConfig, error) {
	var item domain.ModelConfig
	var providerType string
	var kind string
	var roles []byte
	var metadata []byte
	var lastSeenAt *time.Time
	if err := scanner.Scan(&item.ID, &item.ProviderID, &providerType, &item.Name, &item.DisplayName, &kind, &item.ContextWindow, &item.MaxOutputTokens, &item.Dimension, &item.SupportsTools, &item.SupportsStreaming, &item.DefaultForKind, &item.Enabled, &item.CostInputPerMTok, &item.CostOutputPerMTok, &item.RoutingWeight, &roles, &metadata, &item.CreatedAt, &item.UpdatedAt, &lastSeenAt); err != nil {
		return domain.ModelConfig{}, err
	}
	parsedRoles, err := unmarshalJSONB[[]domain.AgentRole](roles)
	if err != nil {
		return domain.ModelConfig{}, err
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.ModelConfig{}, err
	}
	item.ProviderType = domain.ProviderType(providerType)
	item.Kind = domain.ModelKind(kind)
	item.AllowedAgentRoles = parsedRoles
	item.Metadata = parsedMetadata
	item.LastSeenAt = lastSeenAt
	return item, nil
}
