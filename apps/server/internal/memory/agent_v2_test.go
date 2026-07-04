package memory

import (
	"testing"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/repository"
)

func TestAgentInlineSkillLifecycle(t *testing.T) {
	store := NewStore()

	source, err := store.CreateSkillSource(domain.SkillSource{
		ProjectID:  "project-inline",
		Name:       "Inline source",
		Type:       domain.SkillSourceInlineText,
		InlineText: "Draft with a clear promise.",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("CreateSkillSource() error: %v", err)
	}

	skill, err := store.CreateSkill(domain.Skill{
		ProjectID:   source.ProjectID,
		SourceID:    source.ID,
		Name:        "Promise Keeper",
		Description: "Tracks narrative promises",
		Content:     source.InlineText,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("CreateSkill() error: %v", err)
	}
	if skill.ID == "" || skill.SourceID != source.ID || !skill.Enabled {
		t.Fatalf("created skill mismatch: %+v", skill)
	}

	updated, err := store.UpdateSkill(skill.ID, domain.Skill{
		ProjectID:   source.ProjectID,
		SourceID:    source.ID,
		Name:        "Promise Keeper Updated",
		Description: "Tracks and resolves narrative promises",
		Content:     "Resolve promises before payoff.",
		Enabled:     false,
	})
	if err != nil {
		t.Fatalf("UpdateSkill() error: %v", err)
	}
	if updated.ID != skill.ID || updated.Name != "Promise Keeper Updated" || updated.Enabled {
		t.Fatalf("updated skill mismatch: %+v", updated)
	}
	if !updated.CreatedAt.Equal(skill.CreatedAt) || updated.UpdatedAt.Before(skill.UpdatedAt) {
		t.Fatalf("skill timestamps were not preserved/non-decreasing: before=%+v after=%+v", skill, updated)
	}

	disabled := false
	listed, err := store.ListSkills(repository.SkillFilter{ProjectID: source.ProjectID, SourceID: source.ID, Enabled: &disabled})
	if err != nil {
		t.Fatalf("ListSkills() error: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != skill.ID || listed[0].Content != "Resolve promises before payoff." {
		t.Fatalf("filtered skills = %+v, want updated skill", listed)
	}

	if err := store.DeleteSkill(skill.ID); err != nil {
		t.Fatalf("DeleteSkill() error: %v", err)
	}
	listed, err = store.ListSkills(repository.SkillFilter{ProjectID: source.ProjectID})
	if err != nil {
		t.Fatalf("ListSkills() after delete error: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("skills remain after DeleteSkill(): %+v", listed)
	}
	if _, err := store.GetSkill(skill.ID); err == nil {
		t.Fatalf("GetSkill() after delete succeeded, want error")
	}
}

func TestAgentDeleteDirectorySkillSourceUnlinksAgentConfigSkills(t *testing.T) {
	store := NewStore()

	directorySource, err := store.CreateSkillSource(domain.SkillSource{
		ProjectID: "project-directory",
		Name:      "Directory source",
		Type:      domain.SkillSourceDirectory,
		Path:      "skills/continuity",
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("CreateSkillSource(directory) error: %v", err)
	}
	inlineSource, err := store.CreateSkillSource(domain.SkillSource{
		ProjectID:  "project-directory",
		Name:       "Inline source",
		Type:       domain.SkillSourceInlineText,
		InlineText: "Keep this skill.",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("CreateSkillSource(inline) error: %v", err)
	}
	directorySkill, err := store.CreateSkill(domain.Skill{ProjectID: directorySource.ProjectID, SourceID: directorySource.ID, Name: "Directory skill", Path: "skills/continuity/rules.md", Enabled: true})
	if err != nil {
		t.Fatalf("CreateSkill(directory) error: %v", err)
	}
	keptSkill, err := store.CreateSkill(domain.Skill{ProjectID: inlineSource.ProjectID, SourceID: inlineSource.ID, Name: "Kept skill", Content: "Keep", Enabled: true})
	if err != nil {
		t.Fatalf("CreateSkill(kept) error: %v", err)
	}
	agent, err := store.CreateAgentConfig(domain.AgentConfig{
		ProjectID: "project-directory",
		Name:      "Writer",
		Enabled:   true,
		SkillIDs:  []string{directorySkill.ID, keptSkill.ID, directorySkill.ID},
	})
	if err != nil {
		t.Fatalf("CreateAgentConfig() error: %v", err)
	}

	if err := store.DeleteSkillSource(directorySource.ID); err != nil {
		t.Fatalf("DeleteSkillSource() error: %v", err)
	}

	loadedAgent, err := store.GetAgentConfig(agent.ID)
	if err != nil {
		t.Fatalf("GetAgentConfig() error: %v", err)
	}
	if len(loadedAgent.SkillIDs) != 1 || loadedAgent.SkillIDs[0] != keptSkill.ID {
		t.Fatalf("agent SkillIDs = %+v, want only kept skill %q", loadedAgent.SkillIDs, keptSkill.ID)
	}
	if _, err := store.GetSkill(directorySkill.ID); err == nil {
		t.Fatalf("directory skill still exists after deleting directory source")
	}
	if _, err := store.GetSkill(keptSkill.ID); err != nil {
		t.Fatalf("kept skill missing after deleting directory source: %v", err)
	}
}

func TestAgentMCPServerCreateAndListByEnabledAndStatus(t *testing.T) {
	store := NewStore()

	online, err := store.CreateMCPServerConfig(domain.MCPServerConfig{
		ProjectID: "project-mcp",
		Name:      "filesystem",
		Transport: domain.MCPTransportStdio,
		Status:    domain.MCPServerStatusOnline,
		Enabled:   true,
		Command:   "mcp-filesystem",
		Args:      []string{"--root", "."},
	})
	if err != nil {
		t.Fatalf("CreateMCPServerConfig(online) error: %v", err)
	}
	offline, err := store.CreateMCPServerConfig(domain.MCPServerConfig{
		ProjectID: "project-mcp",
		Name:      "browser",
		Transport: domain.MCPTransportStreamableHTTP,
		Status:    domain.MCPServerStatusOffline,
		Enabled:   true,
		URL:       "http://localhost:9123/mcp",
	})
	if err != nil {
		t.Fatalf("CreateMCPServerConfig(offline) error: %v", err)
	}
	_, err = store.CreateMCPServerConfig(domain.MCPServerConfig{
		ProjectID: "project-mcp",
		Name:      "disabled",
		Transport: domain.MCPTransportSSE,
		Status:    domain.MCPServerStatusDisabled,
		Enabled:   false,
		URL:       "http://localhost:9124/sse",
	})
	if err != nil {
		t.Fatalf("CreateMCPServerConfig(disabled) error: %v", err)
	}

	enabled := true
	enabledServers, err := store.ListMCPServerConfigs(repository.MCPServerConfigFilter{ProjectID: "project-mcp", Enabled: &enabled})
	if err != nil {
		t.Fatalf("ListMCPServerConfigs(enabled) error: %v", err)
	}
	if len(enabledServers) != 2 || enabledServers[0].ID != online.ID || enabledServers[1].ID != offline.ID {
		t.Fatalf("enabled MCP servers = %+v, want online and offline", enabledServers)
	}
	offlineServers, err := store.ListMCPServerConfigs(repository.MCPServerConfigFilter{ProjectID: "project-mcp", Status: domain.MCPServerStatusOffline})
	if err != nil {
		t.Fatalf("ListMCPServerConfigs(status) error: %v", err)
	}
	if len(offlineServers) != 1 || offlineServers[0].ID != offline.ID || offlineServers[0].Status != domain.MCPServerStatusOffline {
		t.Fatalf("offline MCP servers = %+v, want %+v", offlineServers, offline)
	}
}

func TestAgentToolDefinitionUpsertSetEnabledAndListFilters(t *testing.T) {
	store := NewStore()

	mcp, err := store.CreateMCPServerConfig(domain.MCPServerConfig{ProjectID: "project-tools", Name: "fs", Transport: domain.MCPTransportStdio, Status: domain.MCPServerStatusOnline, Enabled: true, Command: "mcp-fs"})
	if err != nil {
		t.Fatalf("CreateMCPServerConfig() error: %v", err)
	}
	source, err := store.CreateSkillSource(domain.SkillSource{ProjectID: "project-tools", Name: "inline", Type: domain.SkillSourceInlineText, InlineText: "Skill content", Enabled: true})
	if err != nil {
		t.Fatalf("CreateSkillSource() error: %v", err)
	}
	skill, err := store.CreateSkill(domain.Skill{ProjectID: "project-tools", SourceID: source.ID, Name: "Skill tool", Content: "Use skill", Enabled: true})
	if err != nil {
		t.Fatalf("CreateSkill() error: %v", err)
	}

	builtin, err := store.UpsertToolDefinition(domain.ToolDefinition{ID: "tool_builtin", ProjectID: "project-tools", Name: "semantic_search", Kind: domain.ToolDefinitionBuiltin})
	if err != nil {
		t.Fatalf("UpsertToolDefinition(builtin create) error: %v", err)
	}
	if builtin.Status != domain.ToolStatusActive {
		t.Fatalf("default tool status = %q, want active", builtin.Status)
	}
	updatedBuiltin, err := store.UpsertToolDefinition(domain.ToolDefinition{ID: builtin.ID, ProjectID: "project-tools", Name: "semantic_search", DisplayName: "Semantic Search", Kind: domain.ToolDefinitionBuiltin, Status: domain.ToolStatusUnavailable})
	if err != nil {
		t.Fatalf("UpsertToolDefinition(builtin update) error: %v", err)
	}
	if updatedBuiltin.ID != builtin.ID || !updatedBuiltin.CreatedAt.Equal(builtin.CreatedAt) || updatedBuiltin.Status != domain.ToolStatusUnavailable {
		t.Fatalf("upserted builtin mismatch: before=%+v after=%+v", builtin, updatedBuiltin)
	}
	mcpTool, err := store.UpsertToolDefinition(domain.ToolDefinition{ID: "tool_mcp", ProjectID: "project-tools", Name: "read_file", Kind: domain.ToolDefinitionMCP, Status: domain.ToolStatusActive, MCPServerID: mcp.ID})
	if err != nil {
		t.Fatalf("UpsertToolDefinition(mcp) error: %v", err)
	}
	skillTool, err := store.UpsertToolDefinition(domain.ToolDefinition{ID: "tool_skill", ProjectID: "project-tools", Name: "apply_skill", Kind: domain.ToolDefinitionSkill, Status: domain.ToolStatusActive, SourceID: source.ID, SkillID: skill.ID})
	if err != nil {
		t.Fatalf("UpsertToolDefinition(skill) error: %v", err)
	}

	disabled, err := store.SetToolDefinitionEnabled(mcpTool.ID, false)
	if err != nil {
		t.Fatalf("SetToolDefinitionEnabled(false) error: %v", err)
	}
	if disabled.Status != domain.ToolStatusDisabled {
		t.Fatalf("disabled tool status = %q, want disabled", disabled.Status)
	}
	reenabled, err := store.SetToolDefinitionEnabled(mcpTool.ID, true)
	if err != nil {
		t.Fatalf("SetToolDefinitionEnabled(true) error: %v", err)
	}
	if reenabled.Status != domain.ToolStatusActive {
		t.Fatalf("reenabled tool status = %q, want active", reenabled.Status)
	}

	mcpTools, err := store.ListToolDefinitions(repository.ToolDefinitionFilter{ProjectID: "project-tools", Kind: domain.ToolDefinitionMCP, MCPServerID: mcp.ID, Status: domain.ToolStatusActive})
	if err != nil {
		t.Fatalf("ListToolDefinitions(mcp) error: %v", err)
	}
	if len(mcpTools) != 1 || mcpTools[0].ID != mcpTool.ID {
		t.Fatalf("mcp tools = %+v, want %+v", mcpTools, mcpTool)
	}
	skillTools, err := store.ListToolDefinitions(repository.ToolDefinitionFilter{ProjectID: "project-tools", Kind: domain.ToolDefinitionSkill, SourceID: source.ID, SkillID: skill.ID})
	if err != nil {
		t.Fatalf("ListToolDefinitions(skill) error: %v", err)
	}
	if len(skillTools) != 1 || skillTools[0].ID != skillTool.ID {
		t.Fatalf("skill tools = %+v, want %+v", skillTools, skillTool)
	}
	unavailableTools, err := store.ListToolDefinitions(repository.ToolDefinitionFilter{ProjectID: "project-tools", Status: domain.ToolStatusUnavailable, Limit: 1})
	if err != nil {
		t.Fatalf("ListToolDefinitions(unavailable) error: %v", err)
	}
	if len(unavailableTools) != 1 || unavailableTools[0].ID != builtin.ID {
		t.Fatalf("unavailable tools = %+v, want updated builtin", unavailableTools)
	}
}

func TestAgentToolInvocationCreateUpdateAndListFilters(t *testing.T) {
	store := NewStore()

	first, err := store.CreateToolInvocation(domain.ToolInvocation{
		AgentRunID: "run-1",
		AgentID:    "agent-1",
		ProjectID:  "project-invocations",
		ToolID:     "tool-search",
		ToolName:   "semantic_search",
		Arguments:  map[string]any{"query": "alice"},
	})
	if err != nil {
		t.Fatalf("CreateToolInvocation(first) error: %v", err)
	}
	if first.Status != domain.ToolInvocationStatusRunning || first.StartedAt == nil || first.CompletedAt != nil {
		t.Fatalf("created running invocation mismatch: %+v", first)
	}
	second, err := store.CreateToolInvocation(domain.ToolInvocation{
		AgentRunID: "run-1",
		AgentID:    "agent-1",
		ProjectID:  "project-invocations",
		ToolID:     "tool-write",
		ToolName:   "write_chapter",
		Status:     domain.ToolInvocationStatusFailed,
		Error:      "draft rejected",
	})
	if err != nil {
		t.Fatalf("CreateToolInvocation(second) error: %v", err)
	}
	if second.CompletedAt == nil {
		t.Fatalf("failed invocation did not receive CompletedAt: %+v", second)
	}
	third, err := store.CreateToolInvocation(domain.ToolInvocation{
		AgentRunID: "run-2",
		AgentID:    "agent-2",
		ProjectID:  "project-other",
		ToolID:     "tool-search",
		ToolName:   "semantic_search",
	})
	if err != nil {
		t.Fatalf("CreateToolInvocation(third) error: %v", err)
	}

	updatedFirst, err := store.UpdateToolInvocation(first.ID, domain.ToolInvocation{
		AgentRunID: first.AgentRunID,
		AgentID:    first.AgentID,
		ProjectID:  first.ProjectID,
		ToolID:     first.ToolID,
		ToolName:   first.ToolName,
		Status:     domain.ToolInvocationStatusSucceeded,
		Arguments:  map[string]any{"query": "alice"},
		Result:     map[string]any{"count": 2},
	})
	if err != nil {
		t.Fatalf("UpdateToolInvocation() error: %v", err)
	}
	if updatedFirst.ID != first.ID || updatedFirst.Status != domain.ToolInvocationStatusSucceeded || updatedFirst.CompletedAt == nil || updatedFirst.StartedAt == nil {
		t.Fatalf("updated invocation mismatch: %+v", updatedFirst)
	}
	if !updatedFirst.CreatedAt.Equal(first.CreatedAt) || updatedFirst.UpdatedAt.Before(first.UpdatedAt) {
		t.Fatalf("invocation timestamps were not preserved/non-decreasing: before=%+v after=%+v", first, updatedFirst)
	}

	succeeded, err := store.ListToolInvocations(repository.ToolInvocationFilter{ProjectID: "project-invocations", AgentRunID: "run-1", AgentID: "agent-1", ToolID: "tool-search", Status: domain.ToolInvocationStatusSucceeded})
	if err != nil {
		t.Fatalf("ListToolInvocations(succeeded) error: %v", err)
	}
	if len(succeeded) != 1 || succeeded[0].ID != first.ID {
		t.Fatalf("succeeded invocations = %+v, want updated first", succeeded)
	}
	failed, err := store.ListToolInvocations(repository.ToolInvocationFilter{ProjectID: "project-invocations", Status: domain.ToolInvocationStatusFailed})
	if err != nil {
		t.Fatalf("ListToolInvocations(failed) error: %v", err)
	}
	if len(failed) != 1 || failed[0].ID != second.ID {
		t.Fatalf("failed invocations = %+v, want second", failed)
	}
	limited, err := store.ListToolInvocations(repository.ToolInvocationFilter{ToolID: "tool-search", Limit: 1})
	if err != nil {
		t.Fatalf("ListToolInvocations(limit) error: %v", err)
	}
	if len(limited) != 1 || limited[0].ID != first.ID {
		t.Fatalf("limited invocations = %+v, want first created search invocation before %+v", limited, third)
	}
}
