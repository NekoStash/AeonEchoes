package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/repository"

	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) CreateAgentConfig(cfg domain.AgentConfig) (domain.AgentConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.AgentConfig{}, err
	}
	if err := cfg.Valid(); err != nil {
		return domain.AgentConfig{}, err
	}
	if strings.TrimSpace(cfg.ID) == "" {
		id, err := s.NewID("agent")
		if err != nil {
			return domain.AgentConfig{}, fmt.Errorf("generate agent config id: %w", err)
		}
		cfg.ID = id
	}
	n := now()
	cfg.CreatedAt = n
	cfg.UpdatedAt = n
	skillIDs, toolIDs, mcpServerIDs, memoryPolicy, runtimeOptions, metadata, err := agentConfigJSON(cfg)
	if err != nil {
		return domain.AgentConfig{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO agent_configs(id, project_id, name, description, role, model_id, enabled, system_prompt, skill_ids, tool_ids, mcp_server_ids, memory_policy, runtime_options, metadata, created_at, updated_at)
VALUES ($1,NULLIF($2,''),$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, cfg.ID, cfg.ProjectID, cfg.Name, cfg.Description, string(cfg.Role), cfg.ModelID, cfg.Enabled, cfg.SystemPrompt, skillIDs, toolIDs, mcpServerIDs, memoryPolicy, runtimeOptions, metadata, cfg.CreatedAt, cfg.UpdatedAt)
	if err != nil {
		return domain.AgentConfig{}, fmt.Errorf("insert agent config %q: %w", cfg.ID, err)
	}
	return cfg, nil
}

func (s *Store) UpdateAgentConfig(id string, cfg domain.AgentConfig) (domain.AgentConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.AgentConfig{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.AgentConfig{}, fmt.Errorf("agent config id must not be empty")
	}
	if err := cfg.Valid(); err != nil {
		return domain.AgentConfig{}, err
	}
	existing, err := s.GetAgentConfig(id)
	if err != nil {
		return domain.AgentConfig{}, err
	}
	cfg.ID = id
	cfg.CreatedAt = existing.CreatedAt
	cfg.UpdatedAt = now()
	skillIDs, toolIDs, mcpServerIDs, memoryPolicy, runtimeOptions, metadata, err := agentConfigJSON(cfg)
	if err != nil {
		return domain.AgentConfig{}, err
	}
	result, err := s.pool.Exec(context.Background(), `
UPDATE agent_configs
SET project_id=NULLIF($2,''), name=$3, description=$4, role=$5, model_id=NULLIF($6,''), enabled=$7, system_prompt=$8,
    skill_ids=$9, tool_ids=$10, mcp_server_ids=$11, memory_policy=$12, runtime_options=$13, metadata=$14, updated_at=$15
WHERE id=$1`, cfg.ID, cfg.ProjectID, cfg.Name, cfg.Description, string(cfg.Role), cfg.ModelID, cfg.Enabled, cfg.SystemPrompt, skillIDs, toolIDs, mcpServerIDs, memoryPolicy, runtimeOptions, metadata, cfg.UpdatedAt)
	if err != nil {
		return domain.AgentConfig{}, fmt.Errorf("update agent config %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return domain.AgentConfig{}, fmt.Errorf("agent config %q not found", id)
	}
	return cfg, nil
}

func (s *Store) GetAgentConfig(id string) (domain.AgentConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.AgentConfig{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.AgentConfig{}, fmt.Errorf("agent config id must not be empty")
	}
	row := s.pool.QueryRow(context.Background(), agentConfigSelectSQL()+` WHERE id=$1`, id)
	item, err := scanAgentConfig(row)
	if err != nil {
		if isNoRows(err) {
			return domain.AgentConfig{}, fmt.Errorf("agent config %q not found", id)
		}
		return domain.AgentConfig{}, fmt.Errorf("get agent config %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) DeleteAgentConfig(id string) error {
	if err := requireStore(s); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("agent config id must not be empty")
	}
	result, err := s.pool.Exec(context.Background(), `DELETE FROM agent_configs WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete agent config %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("agent config %q not found", id)
	}
	return nil
}

func (s *Store) ListAgentConfigs(filter repository.AgentConfigFilter) ([]domain.AgentConfig, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	query := agentConfigSelectSQL()
	args := []any{}
	conditions := []string{}
	if filter.Enabled != nil {
		args = append(args, *filter.Enabled)
		conditions = append(conditions, fmt.Sprintf("enabled=$%d", len(args)))
	}
	if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" {
		args = append(args, projectID)
		conditions = append(conditions, fmt.Sprintf("project_id=$%d", len(args)))
	}
	return s.queryAgentConfigs(applyAgentListFilter(query, conditions, filter.Limit, &args), args...)
}

func (s *Store) CreateAgentRun(run domain.AgentRun) (domain.AgentRun, error) {
	if err := requireStore(s); err != nil {
		return domain.AgentRun{}, err
	}
	if run.Status == "" {
		run.Status = domain.AgentRunStatusRunning
	}
	if err := run.Valid(); err != nil {
		return domain.AgentRun{}, err
	}
	if strings.TrimSpace(run.ID) == "" {
		id, err := s.NewID("agent_run")
		if err != nil {
			return domain.AgentRun{}, fmt.Errorf("generate agent run id: %w", err)
		}
		run.ID = id
	}
	n := now()
	run.CreatedAt = n
	run.UpdatedAt = n
	if run.StartedAt == nil && run.Status == domain.AgentRunStatusRunning {
		run.StartedAt = agentV2TimePtr(n)
	}
	input, output, toolInvocationIDs, err := agentRunJSON(run)
	if err != nil {
		return domain.AgentRun{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO agent_runs(id, agent_id, project_id, status, input, output, error, tool_invocation_ids, started_at, completed_at, created_at, updated_at)
VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,$12)`, run.ID, run.AgentID, run.ProjectID, string(run.Status), input, output, run.Error, toolInvocationIDs, run.StartedAt, run.CompletedAt, run.CreatedAt, run.UpdatedAt)
	if err != nil {
		return domain.AgentRun{}, fmt.Errorf("insert agent run %q: %w", run.ID, err)
	}
	return run, nil
}

func (s *Store) UpdateAgentRun(id string, run domain.AgentRun) (domain.AgentRun, error) {
	if err := requireStore(s); err != nil {
		return domain.AgentRun{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.AgentRun{}, fmt.Errorf("agent run id must not be empty")
	}
	if run.Status == "" {
		return domain.AgentRun{}, fmt.Errorf("agent run status must not be empty")
	}
	if err := run.Valid(); err != nil {
		return domain.AgentRun{}, err
	}
	existing, err := s.GetAgentRun(id)
	if err != nil {
		return domain.AgentRun{}, err
	}
	run.ID = id
	run.CreatedAt = existing.CreatedAt
	run.UpdatedAt = now()
	if run.StartedAt == nil {
		run.StartedAt = existing.StartedAt
	}
	if run.CompletedAt == nil && (run.Status == domain.AgentRunStatusCompleted || run.Status == domain.AgentRunStatusFailed) {
		run.CompletedAt = agentV2TimePtr(run.UpdatedAt)
	}
	input, output, toolInvocationIDs, err := agentRunJSON(run)
	if err != nil {
		return domain.AgentRun{}, err
	}
	result, err := s.pool.Exec(context.Background(), `
UPDATE agent_runs
SET agent_id=$2, project_id=NULLIF($3,''), status=$4, input=$5, output=$6, error=$7, tool_invocation_ids=$8,
    started_at=$9, completed_at=$10, updated_at=$11
WHERE id=$1`, run.ID, run.AgentID, run.ProjectID, string(run.Status), input, output, run.Error, toolInvocationIDs, run.StartedAt, run.CompletedAt, run.UpdatedAt)
	if err != nil {
		return domain.AgentRun{}, fmt.Errorf("update agent run %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return domain.AgentRun{}, fmt.Errorf("agent run %q not found", id)
	}
	return run, nil
}

func (s *Store) GetAgentRun(id string) (domain.AgentRun, error) {
	if err := requireStore(s); err != nil {
		return domain.AgentRun{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.AgentRun{}, fmt.Errorf("agent run id must not be empty")
	}
	row := s.pool.QueryRow(context.Background(), agentRunSelectSQL()+` WHERE id=$1`, id)
	item, err := scanAgentRun(row)
	if err != nil {
		if isNoRows(err) {
			return domain.AgentRun{}, fmt.Errorf("agent run %q not found", id)
		}
		return domain.AgentRun{}, fmt.Errorf("get agent run %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) ListAgentRuns(filter repository.AgentRunFilter) ([]domain.AgentRun, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, fmt.Errorf("agent run status %q is invalid", filter.Status)
	}
	query := agentRunSelectSQL()
	args := []any{}
	conditions := []string{}
	if agentID := strings.TrimSpace(filter.AgentID); agentID != "" {
		args = append(args, agentID)
		conditions = append(conditions, fmt.Sprintf("agent_id=$%d", len(args)))
	}
	if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" {
		args = append(args, projectID)
		conditions = append(conditions, fmt.Sprintf("project_id=$%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, string(filter.Status))
		conditions = append(conditions, fmt.Sprintf("status=$%d", len(args)))
	}
	return s.queryAgentRuns(applyAgentListFilter(query, conditions, filter.Limit, &args), args...)
}

func (s *Store) CreateSkillSource(source domain.SkillSource) (domain.SkillSource, error) {
	if err := requireStore(s); err != nil {
		return domain.SkillSource{}, err
	}
	if err := source.Valid(); err != nil {
		return domain.SkillSource{}, err
	}
	if strings.TrimSpace(source.ID) == "" {
		id, err := s.NewID("skill_source")
		if err != nil {
			return domain.SkillSource{}, fmt.Errorf("generate skill source id: %w", err)
		}
		source.ID = id
	}
	n := now()
	source.CreatedAt = n
	source.UpdatedAt = n
	metadata, err := jsonbOrEmptyObject(source.Metadata)
	if err != nil {
		return domain.SkillSource{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO skill_sources(id, project_id, name, type, path, inline_text, enabled, metadata, created_at, updated_at)
VALUES ($1,NULLIF($2,''),$3,$4,$5,$6,$7,$8,$9,$10)`, source.ID, source.ProjectID, source.Name, string(source.Type), source.Path, source.InlineText, source.Enabled, metadata, source.CreatedAt, source.UpdatedAt)
	if err != nil {
		return domain.SkillSource{}, fmt.Errorf("insert skill source %q: %w", source.ID, err)
	}
	return source, nil
}

func (s *Store) UpdateSkillSource(id string, source domain.SkillSource) (domain.SkillSource, error) {
	if err := requireStore(s); err != nil {
		return domain.SkillSource{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.SkillSource{}, fmt.Errorf("skill source id must not be empty")
	}
	if err := source.Valid(); err != nil {
		return domain.SkillSource{}, err
	}
	existing, err := s.GetSkillSource(id)
	if err != nil {
		return domain.SkillSource{}, err
	}
	source.ID = id
	source.CreatedAt = existing.CreatedAt
	source.UpdatedAt = now()
	metadata, err := jsonbOrEmptyObject(source.Metadata)
	if err != nil {
		return domain.SkillSource{}, err
	}
	result, err := s.pool.Exec(context.Background(), `
UPDATE skill_sources
SET project_id=NULLIF($2,''), name=$3, type=$4, path=$5, inline_text=$6, enabled=$7, metadata=$8, updated_at=$9
WHERE id=$1`, source.ID, source.ProjectID, source.Name, string(source.Type), source.Path, source.InlineText, source.Enabled, metadata, source.UpdatedAt)
	if err != nil {
		return domain.SkillSource{}, fmt.Errorf("update skill source %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return domain.SkillSource{}, fmt.Errorf("skill source %q not found", id)
	}
	return source, nil
}

func (s *Store) GetSkillSource(id string) (domain.SkillSource, error) {
	if err := requireStore(s); err != nil {
		return domain.SkillSource{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.SkillSource{}, fmt.Errorf("skill source id must not be empty")
	}
	row := s.pool.QueryRow(context.Background(), skillSourceSelectSQL()+` WHERE id=$1`, id)
	item, err := scanSkillSource(row)
	if err != nil {
		if isNoRows(err) {
			return domain.SkillSource{}, fmt.Errorf("skill source %q not found", id)
		}
		return domain.SkillSource{}, fmt.Errorf("get skill source %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) DeleteSkillSource(id string) error {
	if err := requireStore(s); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("skill source id must not be empty")
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin delete skill source %q: %w", id, err)
	}
	defer tx.Rollback(context.Background())
	rows, err := tx.Query(context.Background(), `SELECT id FROM skills WHERE source_id=$1 ORDER BY created_at ASC, id ASC`, id)
	if err != nil {
		return fmt.Errorf("list skills for source %q: %w", id, err)
	}
	skillIDs := make([]string, 0)
	for rows.Next() {
		var skillID string
		if err := rows.Scan(&skillID); err != nil {
			rows.Close()
			return fmt.Errorf("scan skill for source %q: %w", id, err)
		}
		skillIDs = append(skillIDs, skillID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate skills for source %q: %w", id, err)
	}
	rows.Close()
	for _, skillID := range skillIDs {
		if err := unlinkSkillFromAgentConfigs(context.Background(), tx, skillID, now()); err != nil {
			return err
		}
	}
	result, err := tx.Exec(context.Background(), `DELETE FROM skill_sources WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete skill source %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("skill source %q not found", id)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit delete skill source %q: %w", id, err)
	}
	return nil
}

func (s *Store) ListSkillSources(filter repository.SkillSourceFilter) ([]domain.SkillSource, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	query := skillSourceSelectSQL()
	args := []any{}
	conditions := []string{}
	if filter.Enabled != nil {
		args = append(args, *filter.Enabled)
		conditions = append(conditions, fmt.Sprintf("enabled=$%d", len(args)))
	}
	if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" {
		args = append(args, projectID)
		conditions = append(conditions, fmt.Sprintf("project_id=$%d", len(args)))
	}
	return s.querySkillSources(applyAgentListFilter(query, conditions, filter.Limit, &args), args...)
}

func (s *Store) CreateSkill(skill domain.Skill) (domain.Skill, error) {
	if err := requireStore(s); err != nil {
		return domain.Skill{}, err
	}
	if err := skill.Valid(); err != nil {
		return domain.Skill{}, err
	}
	if strings.TrimSpace(skill.ID) == "" {
		id, err := s.NewID("skill")
		if err != nil {
			return domain.Skill{}, fmt.Errorf("generate skill id: %w", err)
		}
		skill.ID = id
	}
	n := now()
	skill.CreatedAt = n
	skill.UpdatedAt = n
	metadata, err := jsonbOrEmptyObject(skill.Metadata)
	if err != nil {
		return domain.Skill{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO skills(id, project_id, source_id, name, description, content, path, enabled, metadata, created_at, updated_at)
VALUES ($1,NULLIF($2,''),$3,$4,$5,$6,$7,$8,$9,$10,$11)`, skill.ID, skill.ProjectID, skill.SourceID, skill.Name, skill.Description, skill.Content, skill.Path, skill.Enabled, metadata, skill.CreatedAt, skill.UpdatedAt)
	if err != nil {
		return domain.Skill{}, fmt.Errorf("insert skill %q: %w", skill.ID, err)
	}
	return skill, nil
}

func (s *Store) UpdateSkill(id string, skill domain.Skill) (domain.Skill, error) {
	if err := requireStore(s); err != nil {
		return domain.Skill{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Skill{}, fmt.Errorf("skill id must not be empty")
	}
	if err := skill.Valid(); err != nil {
		return domain.Skill{}, err
	}
	existing, err := s.GetSkill(id)
	if err != nil {
		return domain.Skill{}, err
	}
	skill.ID = id
	skill.CreatedAt = existing.CreatedAt
	skill.UpdatedAt = now()
	metadata, err := jsonbOrEmptyObject(skill.Metadata)
	if err != nil {
		return domain.Skill{}, err
	}
	result, err := s.pool.Exec(context.Background(), `
UPDATE skills
SET project_id=NULLIF($2,''), source_id=$3, name=$4, description=$5, content=$6, path=$7, enabled=$8, metadata=$9, updated_at=$10
WHERE id=$1`, skill.ID, skill.ProjectID, skill.SourceID, skill.Name, skill.Description, skill.Content, skill.Path, skill.Enabled, metadata, skill.UpdatedAt)
	if err != nil {
		return domain.Skill{}, fmt.Errorf("update skill %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return domain.Skill{}, fmt.Errorf("skill %q not found", id)
	}
	return skill, nil
}

func (s *Store) GetSkill(id string) (domain.Skill, error) {
	if err := requireStore(s); err != nil {
		return domain.Skill{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Skill{}, fmt.Errorf("skill id must not be empty")
	}
	row := s.pool.QueryRow(context.Background(), skillSelectSQL()+` WHERE id=$1`, id)
	item, err := scanSkill(row)
	if err != nil {
		if isNoRows(err) {
			return domain.Skill{}, fmt.Errorf("skill %q not found", id)
		}
		return domain.Skill{}, fmt.Errorf("get skill %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) DeleteSkill(id string) error {
	if err := requireStore(s); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("skill id must not be empty")
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin delete skill %q: %w", id, err)
	}
	defer tx.Rollback(context.Background())
	if err := unlinkSkillFromAgentConfigs(context.Background(), tx, id, now()); err != nil {
		return err
	}
	result, err := tx.Exec(context.Background(), `DELETE FROM skills WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete skill %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("skill %q not found", id)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit delete skill %q: %w", id, err)
	}
	return nil
}

func (s *Store) ListSkills(filter repository.SkillFilter) ([]domain.Skill, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	query := skillSelectSQL()
	args := []any{}
	conditions := []string{}
	if sourceID := strings.TrimSpace(filter.SourceID); sourceID != "" {
		args = append(args, sourceID)
		conditions = append(conditions, fmt.Sprintf("source_id=$%d", len(args)))
	}
	if filter.Enabled != nil {
		args = append(args, *filter.Enabled)
		conditions = append(conditions, fmt.Sprintf("enabled=$%d", len(args)))
	}
	if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" {
		args = append(args, projectID)
		conditions = append(conditions, fmt.Sprintf("project_id=$%d", len(args)))
	}
	return s.querySkills(applyAgentListFilter(query, conditions, filter.Limit, &args), args...)
}

func (s *Store) CreateMCPServerConfig(cfg domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.MCPServerConfig{}, err
	}
	if cfg.Status == "" {
		cfg.Status = domain.MCPServerStatusUnknown
	}
	if err := cfg.Valid(); err != nil {
		return domain.MCPServerConfig{}, err
	}
	if strings.TrimSpace(cfg.ID) == "" {
		id, err := s.NewID("mcp_server")
		if err != nil {
			return domain.MCPServerConfig{}, fmt.Errorf("generate mcp server config id: %w", err)
		}
		cfg.ID = id
	}
	n := now()
	cfg.CreatedAt = n
	cfg.UpdatedAt = n
	argsJSON, headers, secretHeaders, env, secretEnv, metadata, err := mcpServerConfigJSON(cfg)
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO mcp_server_configs(id, project_id, name, transport, status, enabled, command, args, url, headers, secret_headers, env, secret_env, timeout_sec, metadata, last_seen_at, created_at, updated_at)
VALUES ($1,NULLIF($2,''),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`, cfg.ID, cfg.ProjectID, cfg.Name, string(cfg.Transport), string(cfg.Status), cfg.Enabled, cfg.Command, argsJSON, cfg.URL, headers, secretHeaders, env, secretEnv, cfg.TimeoutSec, metadata, cfg.LastSeenAt, cfg.CreatedAt, cfg.UpdatedAt)
	if err != nil {
		return domain.MCPServerConfig{}, fmt.Errorf("insert mcp server config %q: %w", cfg.ID, err)
	}
	return cfg, nil
}

func (s *Store) UpdateMCPServerConfig(id string, cfg domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.MCPServerConfig{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.MCPServerConfig{}, fmt.Errorf("mcp server config id must not be empty")
	}
	if cfg.Status == "" {
		return domain.MCPServerConfig{}, fmt.Errorf("mcp server status must not be empty")
	}
	if err := cfg.Valid(); err != nil {
		return domain.MCPServerConfig{}, err
	}
	existing, err := s.GetMCPServerConfig(id)
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	cfg.ID = id
	cfg.CreatedAt = existing.CreatedAt
	cfg.UpdatedAt = now()
	argsJSON, headers, secretHeaders, env, secretEnv, metadata, err := mcpServerConfigJSON(cfg)
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	result, err := s.pool.Exec(context.Background(), `
UPDATE mcp_server_configs
SET project_id=NULLIF($2,''), name=$3, transport=$4, status=$5, enabled=$6, command=$7, args=$8, url=$9,
    headers=$10, secret_headers=$11, env=$12, secret_env=$13, timeout_sec=$14, metadata=$15, last_seen_at=$16, updated_at=$17
WHERE id=$1`, cfg.ID, cfg.ProjectID, cfg.Name, string(cfg.Transport), string(cfg.Status), cfg.Enabled, cfg.Command, argsJSON, cfg.URL, headers, secretHeaders, env, secretEnv, cfg.TimeoutSec, metadata, cfg.LastSeenAt, cfg.UpdatedAt)
	if err != nil {
		return domain.MCPServerConfig{}, fmt.Errorf("update mcp server config %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return domain.MCPServerConfig{}, fmt.Errorf("mcp server config %q not found", id)
	}
	return cfg, nil
}

func (s *Store) GetMCPServerConfig(id string) (domain.MCPServerConfig, error) {
	if err := requireStore(s); err != nil {
		return domain.MCPServerConfig{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.MCPServerConfig{}, fmt.Errorf("mcp server config id must not be empty")
	}
	row := s.pool.QueryRow(context.Background(), mcpServerConfigSelectSQL()+` WHERE id=$1`, id)
	item, err := scanMCPServerConfig(row)
	if err != nil {
		if isNoRows(err) {
			return domain.MCPServerConfig{}, fmt.Errorf("mcp server config %q not found", id)
		}
		return domain.MCPServerConfig{}, fmt.Errorf("get mcp server config %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) DeleteMCPServerConfig(id string) error {
	if err := requireStore(s); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("mcp server config id must not be empty")
	}
	result, err := s.pool.Exec(context.Background(), `DELETE FROM mcp_server_configs WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete mcp server config %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("mcp server config %q not found", id)
	}
	return nil
}

func (s *Store) ListMCPServerConfigs(filter repository.MCPServerConfigFilter) ([]domain.MCPServerConfig, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, fmt.Errorf("mcp server status %q is invalid", filter.Status)
	}
	query := mcpServerConfigSelectSQL()
	args := []any{}
	conditions := []string{}
	if filter.Enabled != nil {
		args = append(args, *filter.Enabled)
		conditions = append(conditions, fmt.Sprintf("enabled=$%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, string(filter.Status))
		conditions = append(conditions, fmt.Sprintf("status=$%d", len(args)))
	}
	if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" {
		args = append(args, projectID)
		conditions = append(conditions, fmt.Sprintf("project_id=$%d", len(args)))
	}
	return s.queryMCPServerConfigs(applyAgentListFilter(query, conditions, filter.Limit, &args), args...)
}

func (s *Store) UpsertToolDefinition(tool domain.ToolDefinition) (domain.ToolDefinition, error) {
	if err := requireStore(s); err != nil {
		return domain.ToolDefinition{}, err
	}
	if tool.Status == "" {
		tool.Status = domain.ToolStatusActive
	}
	if err := tool.Valid(); err != nil {
		return domain.ToolDefinition{}, err
	}
	if strings.TrimSpace(tool.ID) == "" {
		id, err := s.NewID("tool")
		if err != nil {
			return domain.ToolDefinition{}, fmt.Errorf("generate tool definition id: %w", err)
		}
		tool.ID = id
	}
	n := now()
	tool.UpdatedAt = n
	existing, err := s.GetToolDefinition(tool.ID)
	if err == nil {
		tool.CreatedAt = existing.CreatedAt
	} else if strings.Contains(err.Error(), "not found") {
		tool.CreatedAt = n
	} else {
		return domain.ToolDefinition{}, err
	}
	inputSchema, metadata, err := toolDefinitionJSON(tool)
	if err != nil {
		return domain.ToolDefinition{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO tool_definitions(id, project_id, name, display_name, description, kind, status, mcp_server_id, source_id, skill_id, input_schema, metadata, created_at, updated_at)
VALUES ($1,NULLIF($2,''),$3,$4,$5,$6,$7,NULLIF($8,''),NULLIF($9,''),NULLIF($10,''),$11,$12,$13,$14)
ON CONFLICT (id) DO UPDATE SET
    project_id=EXCLUDED.project_id, name=EXCLUDED.name, display_name=EXCLUDED.display_name, description=EXCLUDED.description,
    kind=EXCLUDED.kind, status=EXCLUDED.status, mcp_server_id=EXCLUDED.mcp_server_id, source_id=EXCLUDED.source_id,
    skill_id=EXCLUDED.skill_id, input_schema=EXCLUDED.input_schema, metadata=EXCLUDED.metadata, updated_at=EXCLUDED.updated_at`, tool.ID, tool.ProjectID, tool.Name, tool.DisplayName, tool.Description, string(tool.Kind), string(tool.Status), tool.MCPServerID, tool.SourceID, tool.SkillID, inputSchema, metadata, tool.CreatedAt, tool.UpdatedAt)
	if err != nil {
		return domain.ToolDefinition{}, fmt.Errorf("upsert tool definition %q: %w", tool.ID, err)
	}
	return tool, nil
}

func (s *Store) GetToolDefinition(id string) (domain.ToolDefinition, error) {
	if err := requireStore(s); err != nil {
		return domain.ToolDefinition{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.ToolDefinition{}, fmt.Errorf("tool definition id must not be empty")
	}
	row := s.pool.QueryRow(context.Background(), toolDefinitionSelectSQL()+` WHERE id=$1`, id)
	item, err := scanToolDefinition(row)
	if err != nil {
		if isNoRows(err) {
			return domain.ToolDefinition{}, fmt.Errorf("tool definition %q not found", id)
		}
		return domain.ToolDefinition{}, fmt.Errorf("get tool definition %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) DeleteToolDefinition(id string) error {
	if err := requireStore(s); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("tool definition id must not be empty")
	}
	result, err := s.pool.Exec(context.Background(), `DELETE FROM tool_definitions WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete tool definition %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("tool definition %q not found", id)
	}
	return nil
}

func (s *Store) ListToolDefinitions(filter repository.ToolDefinitionFilter) ([]domain.ToolDefinition, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	if filter.Kind != "" && !filter.Kind.Valid() {
		return nil, fmt.Errorf("tool definition kind %q is invalid", filter.Kind)
	}
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, fmt.Errorf("tool status %q is invalid", filter.Status)
	}
	query := toolDefinitionSelectSQL()
	args := []any{}
	conditions := []string{}
	if filter.Kind != "" {
		args = append(args, string(filter.Kind))
		conditions = append(conditions, fmt.Sprintf("kind=$%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, string(filter.Status))
		conditions = append(conditions, fmt.Sprintf("status=$%d", len(args)))
	}
	if mcpServerID := strings.TrimSpace(filter.MCPServerID); mcpServerID != "" {
		args = append(args, mcpServerID)
		conditions = append(conditions, fmt.Sprintf("mcp_server_id=$%d", len(args)))
	}
	if sourceID := strings.TrimSpace(filter.SourceID); sourceID != "" {
		args = append(args, sourceID)
		conditions = append(conditions, fmt.Sprintf("source_id=$%d", len(args)))
	}
	if skillID := strings.TrimSpace(filter.SkillID); skillID != "" {
		args = append(args, skillID)
		conditions = append(conditions, fmt.Sprintf("skill_id=$%d", len(args)))
	}
	if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" {
		args = append(args, projectID)
		conditions = append(conditions, fmt.Sprintf("project_id=$%d", len(args)))
	}
	return s.queryToolDefinitions(applyAgentListFilter(query, conditions, filter.Limit, &args), args...)
}

func (s *Store) SetToolDefinitionEnabled(id string, enabled bool) (domain.ToolDefinition, error) {
	if err := requireStore(s); err != nil {
		return domain.ToolDefinition{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.ToolDefinition{}, fmt.Errorf("tool definition id must not be empty")
	}
	status := domain.ToolStatusDisabled
	if enabled {
		status = domain.ToolStatusActive
	}
	result, err := s.pool.Exec(context.Background(), `UPDATE tool_definitions SET status=$2, updated_at=$3 WHERE id=$1`, id, string(status), now())
	if err != nil {
		return domain.ToolDefinition{}, fmt.Errorf("set tool definition enabled %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return domain.ToolDefinition{}, fmt.Errorf("tool definition %q not found", id)
	}
	return s.GetToolDefinition(id)
}

func (s *Store) CreateToolInvocation(invocation domain.ToolInvocation) (domain.ToolInvocation, error) {
	if err := requireStore(s); err != nil {
		return domain.ToolInvocation{}, err
	}
	if invocation.Status == "" {
		invocation.Status = domain.ToolInvocationStatusRunning
	}
	if err := invocation.Valid(); err != nil {
		return domain.ToolInvocation{}, err
	}
	if strings.TrimSpace(invocation.ID) == "" {
		id, err := s.NewID("tool_invocation")
		if err != nil {
			return domain.ToolInvocation{}, fmt.Errorf("generate tool invocation id: %w", err)
		}
		invocation.ID = id
	}
	n := now()
	invocation.CreatedAt = n
	invocation.UpdatedAt = n
	if invocation.StartedAt == nil && invocation.Status == domain.ToolInvocationStatusRunning {
		invocation.StartedAt = agentV2TimePtr(n)
	}
	if invocation.CompletedAt == nil && (invocation.Status == domain.ToolInvocationStatusSucceeded || invocation.Status == domain.ToolInvocationStatusFailed) {
		invocation.CompletedAt = agentV2TimePtr(n)
	}
	arguments, resultJSON, err := toolInvocationJSON(invocation)
	if err != nil {
		return domain.ToolInvocation{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO tool_invocations(id, agent_run_id, agent_id, project_id, tool_id, tool_name, status, arguments, result, error, started_at, completed_at, created_at, updated_at)
VALUES ($1,NULLIF($2,''),NULLIF($3,''),NULLIF($4,''),NULLIF($5,''),$6,$7,$8,$9,$10,$11,$12,$13,$14)`, invocation.ID, invocation.AgentRunID, invocation.AgentID, invocation.ProjectID, invocation.ToolID, invocation.ToolName, string(invocation.Status), arguments, resultJSON, invocation.Error, invocation.StartedAt, invocation.CompletedAt, invocation.CreatedAt, invocation.UpdatedAt)
	if err != nil {
		return domain.ToolInvocation{}, fmt.Errorf("insert tool invocation %q: %w", invocation.ID, err)
	}
	return invocation, nil
}

func (s *Store) UpdateToolInvocation(id string, invocation domain.ToolInvocation) (domain.ToolInvocation, error) {
	if err := requireStore(s); err != nil {
		return domain.ToolInvocation{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.ToolInvocation{}, fmt.Errorf("tool invocation id must not be empty")
	}
	if invocation.Status == "" {
		return domain.ToolInvocation{}, fmt.Errorf("tool invocation status must not be empty")
	}
	if err := invocation.Valid(); err != nil {
		return domain.ToolInvocation{}, err
	}
	existing, err := s.GetToolInvocation(id)
	if err != nil {
		return domain.ToolInvocation{}, err
	}
	invocation.ID = id
	invocation.CreatedAt = existing.CreatedAt
	invocation.UpdatedAt = now()
	if invocation.StartedAt == nil {
		invocation.StartedAt = existing.StartedAt
	}
	if invocation.CompletedAt == nil && (invocation.Status == domain.ToolInvocationStatusSucceeded || invocation.Status == domain.ToolInvocationStatusFailed) {
		invocation.CompletedAt = agentV2TimePtr(invocation.UpdatedAt)
	}
	arguments, resultJSON, err := toolInvocationJSON(invocation)
	if err != nil {
		return domain.ToolInvocation{}, err
	}
	result, err := s.pool.Exec(context.Background(), `
UPDATE tool_invocations
SET agent_run_id=NULLIF($2,''), agent_id=NULLIF($3,''), project_id=NULLIF($4,''), tool_id=NULLIF($5,''), tool_name=$6,
    status=$7, arguments=$8, result=$9, error=$10, started_at=$11, completed_at=$12, updated_at=$13
WHERE id=$1`, invocation.ID, invocation.AgentRunID, invocation.AgentID, invocation.ProjectID, invocation.ToolID, invocation.ToolName, string(invocation.Status), arguments, resultJSON, invocation.Error, invocation.StartedAt, invocation.CompletedAt, invocation.UpdatedAt)
	if err != nil {
		return domain.ToolInvocation{}, fmt.Errorf("update tool invocation %q: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return domain.ToolInvocation{}, fmt.Errorf("tool invocation %q not found", id)
	}
	return invocation, nil
}

func (s *Store) GetToolInvocation(id string) (domain.ToolInvocation, error) {
	if err := requireStore(s); err != nil {
		return domain.ToolInvocation{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.ToolInvocation{}, fmt.Errorf("tool invocation id must not be empty")
	}
	row := s.pool.QueryRow(context.Background(), toolInvocationSelectSQL()+` WHERE id=$1`, id)
	item, err := scanToolInvocation(row)
	if err != nil {
		if isNoRows(err) {
			return domain.ToolInvocation{}, fmt.Errorf("tool invocation %q not found", id)
		}
		return domain.ToolInvocation{}, fmt.Errorf("get tool invocation %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) ListToolInvocations(filter repository.ToolInvocationFilter) ([]domain.ToolInvocation, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, fmt.Errorf("tool invocation status %q is invalid", filter.Status)
	}
	query := toolInvocationSelectSQL()
	args := []any{}
	conditions := []string{}
	if agentRunID := strings.TrimSpace(filter.AgentRunID); agentRunID != "" {
		args = append(args, agentRunID)
		conditions = append(conditions, fmt.Sprintf("agent_run_id=$%d", len(args)))
	}
	if agentID := strings.TrimSpace(filter.AgentID); agentID != "" {
		args = append(args, agentID)
		conditions = append(conditions, fmt.Sprintf("agent_id=$%d", len(args)))
	}
	if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" {
		args = append(args, projectID)
		conditions = append(conditions, fmt.Sprintf("project_id=$%d", len(args)))
	}
	if toolID := strings.TrimSpace(filter.ToolID); toolID != "" {
		args = append(args, toolID)
		conditions = append(conditions, fmt.Sprintf("tool_id=$%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, string(filter.Status))
		conditions = append(conditions, fmt.Sprintf("status=$%d", len(args)))
	}
	return s.queryToolInvocations(applyAgentListFilter(query, conditions, filter.Limit, &args), args...)
}

func agentConfigJSON(cfg domain.AgentConfig) ([]byte, []byte, []byte, []byte, []byte, []byte, error) {
	skillIDs, err := jsonbOrEmptyArray(cfg.SkillIDs)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	toolIDs, err := jsonbOrEmptyArray(cfg.ToolIDs)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	mcpServerIDs, err := jsonbOrEmptyArray(cfg.MCPServerIDs)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	memoryPolicy, err := jsonbOrEmptyObject(cfg.MemoryPolicy)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	runtimeOptions, err := jsonbOrEmptyObject(cfg.RuntimeOptions)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	metadata, err := jsonbOrEmptyObject(cfg.Metadata)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	return skillIDs, toolIDs, mcpServerIDs, memoryPolicy, runtimeOptions, metadata, nil
}

func agentRunJSON(run domain.AgentRun) ([]byte, []byte, []byte, error) {
	input, err := jsonbOrEmptyObject(run.Input)
	if err != nil {
		return nil, nil, nil, err
	}
	output, err := jsonbOrEmptyObject(run.Output)
	if err != nil {
		return nil, nil, nil, err
	}
	toolInvocationIDs, err := jsonbOrEmptyArray(run.ToolInvocationIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	return input, output, toolInvocationIDs, nil
}

func mcpServerConfigJSON(cfg domain.MCPServerConfig) ([]byte, []byte, []byte, []byte, []byte, []byte, error) {
	argsJSON, err := jsonbOrEmptyArray(cfg.Args)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	headers, err := jsonbOrEmptyObject(cfg.Headers)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	secretHeaders, err := jsonbOrEmptyObject(cfg.SecretHeaders)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	env, err := jsonbOrEmptyObject(cfg.Env)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	secretEnv, err := jsonbOrEmptyObject(cfg.SecretEnv)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	metadata, err := jsonbOrEmptyObject(cfg.Metadata)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	return argsJSON, headers, secretHeaders, env, secretEnv, metadata, nil
}

func toolDefinitionJSON(tool domain.ToolDefinition) ([]byte, []byte, error) {
	inputSchema, err := jsonbOrEmptyObject(tool.InputSchema)
	if err != nil {
		return nil, nil, err
	}
	metadata, err := jsonbOrEmptyObject(tool.Metadata)
	if err != nil {
		return nil, nil, err
	}
	return inputSchema, metadata, nil
}

func toolInvocationJSON(invocation domain.ToolInvocation) ([]byte, []byte, error) {
	arguments, err := jsonbOrEmptyObject(invocation.Arguments)
	if err != nil {
		return nil, nil, err
	}
	resultJSON, err := jsonbOrEmptyObject(invocation.Result)
	if err != nil {
		return nil, nil, err
	}
	return arguments, resultJSON, nil
}

func applyAgentListFilter(query string, conditions []string, limit int, args *[]any) string {
	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, ` AND `)
	}
	query += ` ORDER BY created_at ASC, id ASC`
	if limit > 0 {
		*args = append(*args, limit)
		query += fmt.Sprintf(" LIMIT $%d", len(*args))
	}
	return query
}

type agentV2Scanner interface{ Scan(dest ...any) error }

func agentConfigSelectSQL() string {
	return `SELECT id, COALESCE(project_id, ''), name, description, role, COALESCE(model_id, ''), enabled, system_prompt, skill_ids, tool_ids, mcp_server_ids, memory_policy, runtime_options, metadata, created_at, updated_at FROM agent_configs`
}

func scanAgentConfig(scanner agentV2Scanner) (domain.AgentConfig, error) {
	var item domain.AgentConfig
	var role string
	var skillIDs, toolIDs, mcpServerIDs, memoryPolicy, runtimeOptions, metadata []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.Name, &item.Description, &role, &item.ModelID, &item.Enabled, &item.SystemPrompt, &skillIDs, &toolIDs, &mcpServerIDs, &memoryPolicy, &runtimeOptions, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.AgentConfig{}, err
	}
	parsedSkillIDs, err := unmarshalJSONB[[]string](skillIDs)
	if err != nil {
		return domain.AgentConfig{}, err
	}
	parsedToolIDs, err := unmarshalJSONB[[]string](toolIDs)
	if err != nil {
		return domain.AgentConfig{}, err
	}
	parsedMCPServerIDs, err := unmarshalJSONB[[]string](mcpServerIDs)
	if err != nil {
		return domain.AgentConfig{}, err
	}
	parsedMemoryPolicy, err := unmarshalJSONB[map[string]any](memoryPolicy)
	if err != nil {
		return domain.AgentConfig{}, err
	}
	parsedRuntimeOptions, err := unmarshalJSONB[map[string]any](runtimeOptions)
	if err != nil {
		return domain.AgentConfig{}, err
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.AgentConfig{}, err
	}
	item.Role = domain.AgentRole(role)
	item.SkillIDs = parsedSkillIDs
	item.ToolIDs = parsedToolIDs
	item.MCPServerIDs = parsedMCPServerIDs
	item.MemoryPolicy = parsedMemoryPolicy
	item.RuntimeOptions = parsedRuntimeOptions
	item.Metadata = parsedMetadata
	return item, nil
}

func (s *Store) queryAgentConfigs(sql string, args ...any) ([]domain.AgentConfig, error) {
	rows, err := s.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query agent configs: %w", err)
	}
	defer rows.Close()
	items := make([]domain.AgentConfig, 0)
	for rows.Next() {
		item, err := scanAgentConfig(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent configs: %w", err)
	}
	return items, nil
}

func agentRunSelectSQL() string {
	return `SELECT id, agent_id, COALESCE(project_id, ''), status, input, output, error, tool_invocation_ids, started_at, completed_at, created_at, updated_at FROM agent_runs`
}

func scanAgentRun(scanner agentV2Scanner) (domain.AgentRun, error) {
	var item domain.AgentRun
	var status string
	var input, output, toolInvocationIDs []byte
	var startedAt, completedAt *time.Time
	if err := scanner.Scan(&item.ID, &item.AgentID, &item.ProjectID, &status, &input, &output, &item.Error, &toolInvocationIDs, &startedAt, &completedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.AgentRun{}, err
	}
	parsedInput, err := unmarshalJSONB[map[string]any](input)
	if err != nil {
		return domain.AgentRun{}, err
	}
	parsedOutput, err := unmarshalJSONB[map[string]any](output)
	if err != nil {
		return domain.AgentRun{}, err
	}
	parsedToolInvocationIDs, err := unmarshalJSONB[[]string](toolInvocationIDs)
	if err != nil {
		return domain.AgentRun{}, err
	}
	item.Status = domain.AgentRunStatus(status)
	item.Input = parsedInput
	item.Output = parsedOutput
	item.ToolInvocationIDs = parsedToolInvocationIDs
	item.StartedAt = startedAt
	item.CompletedAt = completedAt
	return item, nil
}

func (s *Store) queryAgentRuns(sql string, args ...any) ([]domain.AgentRun, error) {
	rows, err := s.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query agent runs: %w", err)
	}
	defer rows.Close()
	items := make([]domain.AgentRun, 0)
	for rows.Next() {
		item, err := scanAgentRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent runs: %w", err)
	}
	return items, nil
}

func skillSourceSelectSQL() string {
	return `SELECT id, COALESCE(project_id, ''), name, type, path, inline_text, enabled, metadata, created_at, updated_at FROM skill_sources`
}

func scanSkillSource(scanner agentV2Scanner) (domain.SkillSource, error) {
	var item domain.SkillSource
	var typ string
	var metadata []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.Name, &typ, &item.Path, &item.InlineText, &item.Enabled, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.SkillSource{}, err
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.SkillSource{}, err
	}
	item.Type = domain.SkillSourceType(typ)
	item.Metadata = parsedMetadata
	return item, nil
}

func (s *Store) querySkillSources(sql string, args ...any) ([]domain.SkillSource, error) {
	rows, err := s.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query skill sources: %w", err)
	}
	defer rows.Close()
	items := make([]domain.SkillSource, 0)
	for rows.Next() {
		item, err := scanSkillSource(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skill sources: %w", err)
	}
	return items, nil
}

func skillSelectSQL() string {
	return `SELECT id, COALESCE(project_id, ''), source_id, name, description, content, path, enabled, metadata, created_at, updated_at FROM skills`
}

func scanSkill(scanner agentV2Scanner) (domain.Skill, error) {
	var item domain.Skill
	var metadata []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.SourceID, &item.Name, &item.Description, &item.Content, &item.Path, &item.Enabled, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.Skill{}, err
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.Skill{}, err
	}
	item.Metadata = parsedMetadata
	return item, nil
}

func (s *Store) querySkills(sql string, args ...any) ([]domain.Skill, error) {
	rows, err := s.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query skills: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Skill, 0)
	for rows.Next() {
		item, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skills: %w", err)
	}
	return items, nil
}

func mcpServerConfigSelectSQL() string {
	return `SELECT id, COALESCE(project_id, ''), name, transport, status, enabled, command, args, url, headers, secret_headers, env, secret_env, timeout_sec, metadata, last_seen_at, created_at, updated_at FROM mcp_server_configs`
}

func scanMCPServerConfig(scanner agentV2Scanner) (domain.MCPServerConfig, error) {
	var item domain.MCPServerConfig
	var transport, status string
	var argsJSON, headers, secretHeaders, env, secretEnv, metadata []byte
	var lastSeenAt *time.Time
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.Name, &transport, &status, &item.Enabled, &item.Command, &argsJSON, &item.URL, &headers, &secretHeaders, &env, &secretEnv, &item.TimeoutSec, &metadata, &lastSeenAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.MCPServerConfig{}, err
	}
	parsedArgs, err := unmarshalJSONB[[]string](argsJSON)
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	parsedHeaders, err := unmarshalJSONB[map[string]string](headers)
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	parsedSecretHeaders, err := unmarshalJSONB[map[string]string](secretHeaders)
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	parsedEnv, err := unmarshalJSONB[map[string]string](env)
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	parsedSecretEnv, err := unmarshalJSONB[map[string]string](secretEnv)
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.MCPServerConfig{}, err
	}
	item.Transport = domain.MCPTransport(transport)
	item.Status = domain.MCPServerStatus(status)
	item.Args = parsedArgs
	item.Headers = parsedHeaders
	item.SecretHeaders = parsedSecretHeaders
	item.Env = parsedEnv
	item.SecretEnv = parsedSecretEnv
	item.Metadata = parsedMetadata
	item.LastSeenAt = lastSeenAt
	return item, nil
}

func (s *Store) queryMCPServerConfigs(sql string, args ...any) ([]domain.MCPServerConfig, error) {
	rows, err := s.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query mcp server configs: %w", err)
	}
	defer rows.Close()
	items := make([]domain.MCPServerConfig, 0)
	for rows.Next() {
		item, err := scanMCPServerConfig(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mcp server configs: %w", err)
	}
	return items, nil
}

func toolDefinitionSelectSQL() string {
	return `SELECT id, COALESCE(project_id, ''), name, display_name, description, kind, status, COALESCE(mcp_server_id, ''), COALESCE(source_id, ''), COALESCE(skill_id, ''), input_schema, metadata, created_at, updated_at FROM tool_definitions`
}

func scanToolDefinition(scanner agentV2Scanner) (domain.ToolDefinition, error) {
	var item domain.ToolDefinition
	var kind, status string
	var inputSchema, metadata []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.Name, &item.DisplayName, &item.Description, &kind, &status, &item.MCPServerID, &item.SourceID, &item.SkillID, &inputSchema, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.ToolDefinition{}, err
	}
	parsedInputSchema, err := unmarshalJSONB[map[string]any](inputSchema)
	if err != nil {
		return domain.ToolDefinition{}, err
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.ToolDefinition{}, err
	}
	item.Kind = domain.ToolDefinitionKind(kind)
	item.Status = domain.ToolStatus(status)
	item.InputSchema = parsedInputSchema
	item.Metadata = parsedMetadata
	return item, nil
}

func (s *Store) queryToolDefinitions(sql string, args ...any) ([]domain.ToolDefinition, error) {
	rows, err := s.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query tool definitions: %w", err)
	}
	defer rows.Close()
	items := make([]domain.ToolDefinition, 0)
	for rows.Next() {
		item, err := scanToolDefinition(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tool definitions: %w", err)
	}
	return items, nil
}

func toolInvocationSelectSQL() string {
	return `SELECT id, COALESCE(agent_run_id, ''), COALESCE(agent_id, ''), COALESCE(project_id, ''), COALESCE(tool_id, ''), tool_name, status, arguments, result, error, started_at, completed_at, created_at, updated_at FROM tool_invocations`
}

func scanToolInvocation(scanner agentV2Scanner) (domain.ToolInvocation, error) {
	var item domain.ToolInvocation
	var status string
	var arguments, resultJSON []byte
	var startedAt, completedAt *time.Time
	if err := scanner.Scan(&item.ID, &item.AgentRunID, &item.AgentID, &item.ProjectID, &item.ToolID, &item.ToolName, &status, &arguments, &resultJSON, &item.Error, &startedAt, &completedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.ToolInvocation{}, err
	}
	parsedArguments, err := unmarshalJSONB[map[string]any](arguments)
	if err != nil {
		return domain.ToolInvocation{}, err
	}
	parsedResult, err := unmarshalJSONB[map[string]any](resultJSON)
	if err != nil {
		return domain.ToolInvocation{}, err
	}
	item.Status = domain.ToolInvocationStatus(status)
	item.Arguments = parsedArguments
	item.Result = parsedResult
	item.StartedAt = startedAt
	item.CompletedAt = completedAt
	return item, nil
}

func (s *Store) queryToolInvocations(sql string, args ...any) ([]domain.ToolInvocation, error) {
	rows, err := s.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query tool invocations: %w", err)
	}
	defer rows.Close()
	items := make([]domain.ToolInvocation, 0)
	for rows.Next() {
		item, err := scanToolInvocation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tool invocations: %w", err)
	}
	return items, nil
}

type agentV2Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func unlinkSkillFromAgentConfigs(ctx context.Context, exec agentV2Executor, skillID string, updatedAt time.Time) error {
	_, err := exec.Exec(ctx, `
UPDATE agent_configs
SET skill_ids = COALESCE((SELECT jsonb_agg(value) FROM jsonb_array_elements_text(skill_ids) AS value WHERE value <> $1), '[]'::jsonb), updated_at=$2
WHERE skill_ids ? $1`, skillID, updatedAt)
	if err != nil {
		return fmt.Errorf("unlink skill %q from agent configs: %w", skillID, err)
	}
	return nil
}

func agentV2TimePtr(value time.Time) *time.Time {
	copied := value
	return &copied
}
