package memory

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/repository"
)

func (s *Store) CreateAgentConfig(cfg domain.AgentConfig) (domain.AgentConfig, error) {
	if err := cfg.Valid(); err != nil {
		return domain.AgentConfig{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(cfg.ID) == "" {
		cfg.ID = s.nextIDLocked("agent")
	}
	if _, exists := s.agentConfigs[cfg.ID]; exists {
		return domain.AgentConfig{}, fmt.Errorf("agent config %q already exists", cfg.ID)
	}
	n := now()
	cfg.CreatedAt = n
	cfg.UpdatedAt = n
	cfg = cloneAgentConfig(cfg)
	s.agentConfigs[cfg.ID] = cfg
	return cloneAgentConfig(cfg), nil
}

func (s *Store) UpdateAgentConfig(id string, cfg domain.AgentConfig) (domain.AgentConfig, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.AgentConfig{}, fmt.Errorf("agent config id must not be empty")
	}
	if err := cfg.Valid(); err != nil {
		return domain.AgentConfig{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.agentConfigs[id]
	if !ok {
		return domain.AgentConfig{}, fmt.Errorf("agent config %q not found", id)
	}
	cfg.ID = id
	cfg.CreatedAt = existing.CreatedAt
	cfg.UpdatedAt = now()
	cfg = cloneAgentConfig(cfg)
	s.agentConfigs[id] = cfg
	return cloneAgentConfig(cfg), nil
}

func (s *Store) GetAgentConfig(id string) (domain.AgentConfig, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.AgentConfig{}, fmt.Errorf("agent config id must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.agentConfigs[id]
	if !ok {
		return domain.AgentConfig{}, fmt.Errorf("agent config %q not found", id)
	}
	return cloneAgentConfig(cfg), nil
}

func (s *Store) DeleteAgentConfig(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("agent config id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.agentConfigs[id]; !ok {
		return fmt.Errorf("agent config %q not found", id)
	}
	delete(s.agentConfigs, id)
	return nil
}

func (s *Store) ListAgentConfigs(filter repository.AgentConfigFilter) ([]domain.AgentConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	projectID := strings.TrimSpace(filter.ProjectID)
	items := make([]domain.AgentConfig, 0)
	for _, item := range s.agentConfigs {
		itemProjectID := strings.TrimSpace(item.ProjectID)
		if projectID != "" && itemProjectID != "" && itemProjectID != projectID {
			continue
		}
		if filter.Enabled != nil && item.Enabled != *filter.Enabled {
			continue
		}
		items = append(items, cloneAgentConfig(item))
	}
	sortAgentConfigs(items, projectID)
	return limitSlice(items, filter.Limit), nil
}

func (s *Store) CreateAgentRun(run domain.AgentRun) (domain.AgentRun, error) {
	if run.Status == "" {
		run.Status = domain.AgentRunStatusRunning
	}
	if err := run.Valid(); err != nil {
		return domain.AgentRun{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(run.ID) == "" {
		run.ID = s.nextIDLocked("agent_run")
	}
	if _, exists := s.agentRuns[run.ID]; exists {
		return domain.AgentRun{}, fmt.Errorf("agent run %q already exists", run.ID)
	}
	n := now()
	run.CreatedAt = n
	run.UpdatedAt = n
	if run.StartedAt == nil && run.Status == domain.AgentRunStatusRunning {
		run.StartedAt = timePtr(n)
	}
	run = cloneAgentRun(run)
	s.agentRuns[run.ID] = run
	return cloneAgentRun(run), nil
}

func (s *Store) UpdateAgentRun(id string, run domain.AgentRun) (domain.AgentRun, error) {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.agentRuns[id]
	if !ok {
		return domain.AgentRun{}, fmt.Errorf("agent run %q not found", id)
	}
	run.ID = id
	run.CreatedAt = existing.CreatedAt
	run.UpdatedAt = now()
	if run.StartedAt == nil {
		run.StartedAt = cloneTimePtr(existing.StartedAt)
	}
	if run.CompletedAt == nil && (run.Status == domain.AgentRunStatusCompleted || run.Status == domain.AgentRunStatusFailed) {
		run.CompletedAt = timePtr(run.UpdatedAt)
	}
	run = cloneAgentRun(run)
	s.agentRuns[id] = run
	return cloneAgentRun(run), nil
}

func (s *Store) GetAgentRun(id string) (domain.AgentRun, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.AgentRun{}, fmt.Errorf("agent run id must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.agentRuns[id]
	if !ok {
		return domain.AgentRun{}, fmt.Errorf("agent run %q not found", id)
	}
	return cloneAgentRun(run), nil
}

func (s *Store) ListAgentRuns(filter repository.AgentRunFilter) ([]domain.AgentRun, error) {
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, fmt.Errorf("agent run status %q is invalid", filter.Status)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	agentID := strings.TrimSpace(filter.AgentID)
	projectID := strings.TrimSpace(filter.ProjectID)
	items := make([]domain.AgentRun, 0)
	for _, item := range s.agentRuns {
		if agentID != "" && item.AgentID != agentID {
			continue
		}
		if projectID != "" && item.ProjectID != projectID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, cloneAgentRun(item))
	}
	sortAgentRuns(items)
	return limitSlice(items, filter.Limit), nil
}

func (s *Store) CreateSkillSource(source domain.SkillSource) (domain.SkillSource, error) {
	if err := source.Valid(); err != nil {
		return domain.SkillSource{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(source.ID) == "" {
		source.ID = s.nextIDLocked("skill_source")
	}
	if _, exists := s.skillSources[source.ID]; exists {
		return domain.SkillSource{}, fmt.Errorf("skill source %q already exists", source.ID)
	}
	n := now()
	source.CreatedAt = n
	source.UpdatedAt = n
	source = cloneSkillSource(source)
	s.skillSources[source.ID] = source
	return cloneSkillSource(source), nil
}

func (s *Store) UpdateSkillSource(id string, source domain.SkillSource) (domain.SkillSource, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.SkillSource{}, fmt.Errorf("skill source id must not be empty")
	}
	if err := source.Valid(); err != nil {
		return domain.SkillSource{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.skillSources[id]
	if !ok {
		return domain.SkillSource{}, fmt.Errorf("skill source %q not found", id)
	}
	source.ID = id
	source.CreatedAt = existing.CreatedAt
	source.UpdatedAt = now()
	source = cloneSkillSource(source)
	s.skillSources[id] = source
	return cloneSkillSource(source), nil
}

func (s *Store) GetSkillSource(id string) (domain.SkillSource, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.SkillSource{}, fmt.Errorf("skill source id must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	source, ok := s.skillSources[id]
	if !ok {
		return domain.SkillSource{}, fmt.Errorf("skill source %q not found", id)
	}
	return cloneSkillSource(source), nil
}

func (s *Store) DeleteSkillSource(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("skill source id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.skillSources[id]; !ok {
		return fmt.Errorf("skill source %q not found", id)
	}
	delete(s.skillSources, id)
	for skillID, skill := range s.skills {
		if skill.SourceID == id {
			delete(s.skills, skillID)
			s.unlinkSkillFromAgentsLocked(skillID)
		}
	}
	return nil
}

func (s *Store) ListSkillSources(filter repository.SkillSourceFilter) ([]domain.SkillSource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	projectID := strings.TrimSpace(filter.ProjectID)
	items := make([]domain.SkillSource, 0)
	for _, item := range s.skillSources {
		if projectID != "" && item.ProjectID != projectID {
			continue
		}
		if filter.Enabled != nil && item.Enabled != *filter.Enabled {
			continue
		}
		items = append(items, cloneSkillSource(item))
	}
	sortSkillSources(items)
	return limitSlice(items, filter.Limit), nil
}

func (s *Store) CreateSkill(skill domain.Skill) (domain.Skill, error) {
	if err := skill.Valid(); err != nil {
		return domain.Skill{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.skillSources[skill.SourceID]; !ok {
		return domain.Skill{}, fmt.Errorf("skill source %q not found", skill.SourceID)
	}
	if strings.TrimSpace(skill.ID) == "" {
		skill.ID = s.nextIDLocked("skill")
	}
	if _, exists := s.skills[skill.ID]; exists {
		return domain.Skill{}, fmt.Errorf("skill %q already exists", skill.ID)
	}
	n := now()
	skill.CreatedAt = n
	skill.UpdatedAt = n
	skill = cloneSkill(skill)
	s.skills[skill.ID] = skill
	return cloneSkill(skill), nil
}

func (s *Store) UpdateSkill(id string, skill domain.Skill) (domain.Skill, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Skill{}, fmt.Errorf("skill id must not be empty")
	}
	if err := skill.Valid(); err != nil {
		return domain.Skill{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.skills[id]
	if !ok {
		return domain.Skill{}, fmt.Errorf("skill %q not found", id)
	}
	if _, ok := s.skillSources[skill.SourceID]; !ok {
		return domain.Skill{}, fmt.Errorf("skill source %q not found", skill.SourceID)
	}
	skill.ID = id
	skill.CreatedAt = existing.CreatedAt
	skill.UpdatedAt = now()
	skill = cloneSkill(skill)
	s.skills[id] = skill
	return cloneSkill(skill), nil
}

func (s *Store) GetSkill(id string) (domain.Skill, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Skill{}, fmt.Errorf("skill id must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	skill, ok := s.skills[id]
	if !ok {
		return domain.Skill{}, fmt.Errorf("skill %q not found", id)
	}
	return cloneSkill(skill), nil
}

func (s *Store) DeleteSkill(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("skill id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.skills[id]; !ok {
		return fmt.Errorf("skill %q not found", id)
	}
	delete(s.skills, id)
	s.unlinkSkillFromAgentsLocked(id)
	return nil
}

func (s *Store) ListSkills(filter repository.SkillFilter) ([]domain.Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sourceID := strings.TrimSpace(filter.SourceID)
	projectID := strings.TrimSpace(filter.ProjectID)
	items := make([]domain.Skill, 0)
	for _, item := range s.skills {
		if sourceID != "" && item.SourceID != sourceID {
			continue
		}
		if projectID != "" && item.ProjectID != projectID {
			continue
		}
		if filter.Enabled != nil && item.Enabled != *filter.Enabled {
			continue
		}
		items = append(items, cloneSkill(item))
	}
	sortSkills(items)
	return limitSlice(items, filter.Limit), nil
}

func (s *Store) CreateMCPServerConfig(cfg domain.MCPServerConfig) (domain.MCPServerConfig, error) {
	if cfg.Status == "" {
		cfg.Status = domain.MCPServerStatusUnknown
	}
	if err := cfg.Valid(); err != nil {
		return domain.MCPServerConfig{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(cfg.ID) == "" {
		cfg.ID = s.nextIDLocked("mcp_server")
	}
	if _, exists := s.mcpServers[cfg.ID]; exists {
		return domain.MCPServerConfig{}, fmt.Errorf("mcp server config %q already exists", cfg.ID)
	}
	n := now()
	cfg.CreatedAt = n
	cfg.UpdatedAt = n
	cfg = cloneMCPServerConfig(cfg)
	s.mcpServers[cfg.ID] = cfg
	return cloneMCPServerConfig(cfg), nil
}

func (s *Store) UpdateMCPServerConfig(id string, cfg domain.MCPServerConfig) (domain.MCPServerConfig, error) {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.mcpServers[id]
	if !ok {
		return domain.MCPServerConfig{}, fmt.Errorf("mcp server config %q not found", id)
	}
	cfg.ID = id
	cfg.CreatedAt = existing.CreatedAt
	cfg.UpdatedAt = now()
	cfg = cloneMCPServerConfig(cfg)
	s.mcpServers[id] = cfg
	return cloneMCPServerConfig(cfg), nil
}

func (s *Store) GetMCPServerConfig(id string) (domain.MCPServerConfig, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.MCPServerConfig{}, fmt.Errorf("mcp server config id must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.mcpServers[id]
	if !ok {
		return domain.MCPServerConfig{}, fmt.Errorf("mcp server config %q not found", id)
	}
	return cloneMCPServerConfig(cfg), nil
}

func (s *Store) DeleteMCPServerConfig(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("mcp server config id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.mcpServers[id]; !ok {
		return fmt.Errorf("mcp server config %q not found", id)
	}
	delete(s.mcpServers, id)
	for toolID, tool := range s.toolDefinitions {
		if tool.MCPServerID == id {
			tool.MCPServerID = ""
			tool.Status = domain.ToolStatusUnavailable
			tool.UpdatedAt = now()
			s.toolDefinitions[toolID] = tool
		}
	}
	return nil
}

func (s *Store) ListMCPServerConfigs(filter repository.MCPServerConfigFilter) ([]domain.MCPServerConfig, error) {
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, fmt.Errorf("mcp server status %q is invalid", filter.Status)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	projectID := strings.TrimSpace(filter.ProjectID)
	items := make([]domain.MCPServerConfig, 0)
	for _, item := range s.mcpServers {
		if projectID != "" && item.ProjectID != projectID {
			continue
		}
		if filter.Enabled != nil && item.Enabled != *filter.Enabled {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, cloneMCPServerConfig(item))
	}
	sortMCPServerConfigs(items)
	return limitSlice(items, filter.Limit), nil
}

func (s *Store) UpsertToolDefinition(tool domain.ToolDefinition) (domain.ToolDefinition, error) {
	if tool.Status == "" {
		tool.Status = domain.ToolStatusActive
	}
	if err := tool.Valid(); err != nil {
		return domain.ToolDefinition{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(tool.ID) == "" {
		tool.ID = s.nextIDLocked("tool")
	}
	n := now()
	if existing, ok := s.toolDefinitions[tool.ID]; ok {
		tool.CreatedAt = existing.CreatedAt
	} else {
		tool.CreatedAt = n
	}
	tool.UpdatedAt = n
	tool = cloneToolDefinition(tool)
	s.toolDefinitions[tool.ID] = tool
	return cloneToolDefinition(tool), nil
}

func (s *Store) GetToolDefinition(id string) (domain.ToolDefinition, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.ToolDefinition{}, fmt.Errorf("tool definition id must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	tool, ok := s.toolDefinitions[id]
	if !ok {
		return domain.ToolDefinition{}, fmt.Errorf("tool definition %q not found", id)
	}
	return cloneToolDefinition(tool), nil
}

func (s *Store) DeleteToolDefinition(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("tool definition id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.toolDefinitions[id]; !ok {
		return fmt.Errorf("tool definition %q not found", id)
	}
	delete(s.toolDefinitions, id)
	return nil
}

func (s *Store) ListToolDefinitions(filter repository.ToolDefinitionFilter) ([]domain.ToolDefinition, error) {
	if filter.Kind != "" && !filter.Kind.Valid() {
		return nil, fmt.Errorf("tool definition kind %q is invalid", filter.Kind)
	}
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, fmt.Errorf("tool status %q is invalid", filter.Status)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	projectID := strings.TrimSpace(filter.ProjectID)
	mcpServerID := strings.TrimSpace(filter.MCPServerID)
	sourceID := strings.TrimSpace(filter.SourceID)
	skillID := strings.TrimSpace(filter.SkillID)
	items := make([]domain.ToolDefinition, 0)
	for _, item := range s.toolDefinitions {
		if projectID != "" && item.ProjectID != projectID {
			continue
		}
		if filter.Kind != "" && item.Kind != filter.Kind {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if mcpServerID != "" && item.MCPServerID != mcpServerID {
			continue
		}
		if sourceID != "" && item.SourceID != sourceID {
			continue
		}
		if skillID != "" && item.SkillID != skillID {
			continue
		}
		items = append(items, cloneToolDefinition(item))
	}
	sortToolDefinitions(items)
	return limitSlice(items, filter.Limit), nil
}

func (s *Store) SetToolDefinitionEnabled(id string, enabled bool) (domain.ToolDefinition, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.ToolDefinition{}, fmt.Errorf("tool definition id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tool, ok := s.toolDefinitions[id]
	if !ok {
		return domain.ToolDefinition{}, fmt.Errorf("tool definition %q not found", id)
	}
	if enabled {
		tool.Status = domain.ToolStatusActive
	} else {
		tool.Status = domain.ToolStatusDisabled
	}
	tool.UpdatedAt = now()
	s.toolDefinitions[id] = tool
	return cloneToolDefinition(tool), nil
}

func (s *Store) CreateToolInvocation(invocation domain.ToolInvocation) (domain.ToolInvocation, error) {
	if invocation.Status == "" {
		invocation.Status = domain.ToolInvocationStatusRunning
	}
	if err := invocation.Valid(); err != nil {
		return domain.ToolInvocation{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(invocation.ID) == "" {
		invocation.ID = s.nextIDLocked("tool_invocation")
	}
	if _, exists := s.toolInvocations[invocation.ID]; exists {
		return domain.ToolInvocation{}, fmt.Errorf("tool invocation %q already exists", invocation.ID)
	}
	n := now()
	invocation.CreatedAt = n
	invocation.UpdatedAt = n
	if invocation.StartedAt == nil && invocation.Status == domain.ToolInvocationStatusRunning {
		invocation.StartedAt = timePtr(n)
	}
	if invocation.CompletedAt == nil && (invocation.Status == domain.ToolInvocationStatusSucceeded || invocation.Status == domain.ToolInvocationStatusFailed) {
		invocation.CompletedAt = timePtr(n)
	}
	invocation = cloneToolInvocation(invocation)
	s.toolInvocations[invocation.ID] = invocation
	return cloneToolInvocation(invocation), nil
}

func (s *Store) UpdateToolInvocation(id string, invocation domain.ToolInvocation) (domain.ToolInvocation, error) {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.toolInvocations[id]
	if !ok {
		return domain.ToolInvocation{}, fmt.Errorf("tool invocation %q not found", id)
	}
	invocation.ID = id
	invocation.CreatedAt = existing.CreatedAt
	invocation.UpdatedAt = now()
	if invocation.StartedAt == nil {
		invocation.StartedAt = cloneTimePtr(existing.StartedAt)
	}
	if invocation.CompletedAt == nil && (invocation.Status == domain.ToolInvocationStatusSucceeded || invocation.Status == domain.ToolInvocationStatusFailed) {
		invocation.CompletedAt = timePtr(invocation.UpdatedAt)
	}
	invocation = cloneToolInvocation(invocation)
	s.toolInvocations[id] = invocation
	return cloneToolInvocation(invocation), nil
}

func (s *Store) GetToolInvocation(id string) (domain.ToolInvocation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.ToolInvocation{}, fmt.Errorf("tool invocation id must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	invocation, ok := s.toolInvocations[id]
	if !ok {
		return domain.ToolInvocation{}, fmt.Errorf("tool invocation %q not found", id)
	}
	return cloneToolInvocation(invocation), nil
}

func (s *Store) ListToolInvocations(filter repository.ToolInvocationFilter) ([]domain.ToolInvocation, error) {
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, fmt.Errorf("tool invocation status %q is invalid", filter.Status)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	agentRunID := strings.TrimSpace(filter.AgentRunID)
	agentID := strings.TrimSpace(filter.AgentID)
	projectID := strings.TrimSpace(filter.ProjectID)
	toolID := strings.TrimSpace(filter.ToolID)
	items := make([]domain.ToolInvocation, 0)
	for _, item := range s.toolInvocations {
		if agentRunID != "" && item.AgentRunID != agentRunID {
			continue
		}
		if agentID != "" && item.AgentID != agentID {
			continue
		}
		if projectID != "" && item.ProjectID != projectID {
			continue
		}
		if toolID != "" && item.ToolID != toolID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, cloneToolInvocation(item))
	}
	sortToolInvocations(items)
	return limitSlice(items, filter.Limit), nil
}

func (s *Store) unlinkSkillFromAgentsLocked(skillID string) {
	for agentID, cfg := range s.agentConfigs {
		filtered := removeStringValue(cfg.SkillIDs, skillID)
		if len(filtered) == len(cfg.SkillIDs) {
			continue
		}
		cfg.SkillIDs = filtered
		cfg.UpdatedAt = now()
		s.agentConfigs[agentID] = cfg
	}
}

func removeStringValue(values []string, target string) []string {
	if len(values) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func sortAgentConfigs(items []domain.AgentConfig, projectID string) {
	sort.Slice(items, func(i, j int) bool {
		leftRank := agentScopeRank(items[i].ProjectID, projectID)
		rightRank := agentScopeRank(items[j].ProjectID, projectID)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		leftName := strings.ToLower(strings.TrimSpace(items[i].Name))
		rightName := strings.ToLower(strings.TrimSpace(items[j].Name))
		if leftName != rightName {
			return leftName < rightName
		}
		return items[i].ID < items[j].ID
	})
}

func agentScopeRank(itemProjectID string, requestedProjectID string) int {
	itemProjectID = strings.TrimSpace(itemProjectID)
	if requestedProjectID != "" && itemProjectID == requestedProjectID {
		return 0
	}
	if itemProjectID == "" {
		return 1
	}
	return 0
}

func sortAgentRuns(items []domain.AgentRun) {
	sort.Slice(items, func(i, j int) bool {
		return lessCreatedID(items[i].CreatedAt, items[i].ID, items[j].CreatedAt, items[j].ID)
	})
}

func sortSkillSources(items []domain.SkillSource) {
	sort.Slice(items, func(i, j int) bool {
		return lessCreatedID(items[i].CreatedAt, items[i].ID, items[j].CreatedAt, items[j].ID)
	})
}

func sortSkills(items []domain.Skill) {
	sort.Slice(items, func(i, j int) bool {
		return lessCreatedID(items[i].CreatedAt, items[i].ID, items[j].CreatedAt, items[j].ID)
	})
}

func sortMCPServerConfigs(items []domain.MCPServerConfig) {
	sort.Slice(items, func(i, j int) bool {
		return lessCreatedID(items[i].CreatedAt, items[i].ID, items[j].CreatedAt, items[j].ID)
	})
}

func sortToolDefinitions(items []domain.ToolDefinition) {
	sort.Slice(items, func(i, j int) bool {
		return lessCreatedID(items[i].CreatedAt, items[i].ID, items[j].CreatedAt, items[j].ID)
	})
}

func sortToolInvocations(items []domain.ToolInvocation) {
	sort.Slice(items, func(i, j int) bool {
		return lessCreatedID(items[i].CreatedAt, items[i].ID, items[j].CreatedAt, items[j].ID)
	})
}

func lessCreatedID(leftTime time.Time, leftID string, rightTime time.Time, rightID string) bool {
	if leftTime.Equal(rightTime) {
		return leftID < rightID
	}
	return leftTime.Before(rightTime)
}

func limitSlice[T any](items []T, limit int) []T {
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	return items
}

func cloneAgentConfig(item domain.AgentConfig) domain.AgentConfig {
	item.SkillIDs = copyStringSlice(item.SkillIDs)
	item.ToolIDs = copyStringSlice(item.ToolIDs)
	item.MCPServerIDs = copyStringSlice(item.MCPServerIDs)
	item.MemoryPolicy = copyAnyMap(item.MemoryPolicy)
	item.RuntimeOptions = copyAnyMap(item.RuntimeOptions)
	item.Metadata = copyStringMap(item.Metadata)
	return item
}

func cloneAgentRun(item domain.AgentRun) domain.AgentRun {
	item.Input = copyAnyMap(item.Input)
	item.Output = copyAnyMap(item.Output)
	item.ToolInvocationIDs = copyStringSlice(item.ToolInvocationIDs)
	item.StartedAt = cloneTimePtr(item.StartedAt)
	item.CompletedAt = cloneTimePtr(item.CompletedAt)
	return item
}

func cloneSkillSource(item domain.SkillSource) domain.SkillSource {
	item.Metadata = copyStringMap(item.Metadata)
	return item
}

func cloneSkill(item domain.Skill) domain.Skill {
	item.Metadata = copyStringMap(item.Metadata)
	return item
}

func cloneMCPServerConfig(item domain.MCPServerConfig) domain.MCPServerConfig {
	item.Args = copyStringSlice(item.Args)
	item.Headers = copyStringMap(item.Headers)
	item.SecretHeaders = copyStringMap(item.SecretHeaders)
	item.Env = copyStringMap(item.Env)
	item.SecretEnv = copyStringMap(item.SecretEnv)
	item.Metadata = copyStringMap(item.Metadata)
	item.LastSeenAt = cloneTimePtr(item.LastSeenAt)
	return item
}

func cloneToolDefinition(item domain.ToolDefinition) domain.ToolDefinition {
	item.InputSchema = copyAnyMap(item.InputSchema)
	item.Metadata = copyStringMap(item.Metadata)
	return item
}

func cloneToolInvocation(item domain.ToolInvocation) domain.ToolInvocation {
	item.Arguments = copyAnyMap(item.Arguments)
	item.Result = copyAnyMap(item.Result)
	item.StartedAt = cloneTimePtr(item.StartedAt)
	item.CompletedAt = cloneTimePtr(item.CompletedAt)
	return item
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	copied := make([]string, len(values))
	copy(copied, values)
	return copied
}

func copyAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]any, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func timePtr(value time.Time) *time.Time {
	copied := value
	return &copied
}
