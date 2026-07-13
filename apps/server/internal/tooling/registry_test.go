package tooling

import (
	"context"
	"strings"
	"testing"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/memory"
)

func TestSeedBuiltinToolsDeletesObsoleteBuiltinsAndScrubsAgentToolIDs(t *testing.T) {
	store := memory.NewStore()
	registry := NewRegistry(store, store)

	// Pre-upgrade catalog still contains tools removed from NarrativeToolSpecs.
	for _, name := range []string{"chapter.ensure", "chapter.create", "orphan.builtin"} {
		if _, err := store.UpsertToolDefinition(domain.ToolDefinition{
			ID:          builtinToolID(name),
			Name:        name,
			DisplayName: name,
			Kind:        domain.ToolDefinitionBuiltin,
			Status:      domain.ToolStatusActive,
			Metadata:    map[string]string{"source": "upgrade-residue"},
		}); err != nil {
			t.Fatalf("UpsertToolDefinition(%s) error: %v", name, err)
		}
	}
	// Non-builtin tools must survive catalog cleanup.
	if _, err := store.UpsertToolDefinition(domain.ToolDefinition{
		ID:         "tool_skill_keep",
		Name:       "style.guard",
		Kind:       domain.ToolDefinitionSkill,
		Status:     domain.ToolStatusActive,
		SourceID:   "source_keep",
		SkillID:    "skill_keep",
	}); err != nil {
		t.Fatalf("UpsertToolDefinition(skill) error: %v", err)
	}

	agentCfg, err := store.CreateAgentConfig(domain.AgentConfig{
		Name:    "writer",
		Role:    domain.AgentRoleWriter,
		Enabled: true,
		ToolIDs: []string{
			builtinToolID("character.search"),
			builtinToolID("chapter.ensure"),
			"builtin.chapter.create",
			"tool_skill_keep",
			"  ",
		},
	})
	if err != nil {
		t.Fatalf("CreateAgentConfig() error: %v", err)
	}

	if err := registry.SeedBuiltinTools(context.Background()); err != nil {
		t.Fatalf("SeedBuiltinTools() error: %v", err)
	}

	for _, name := range []string{"chapter.ensure", "chapter.create", "orphan.builtin"} {
		if _, err := store.GetToolDefinition(builtinToolID(name)); err == nil {
			t.Fatalf("obsolete builtin %q still exists after seed", name)
		} else if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("GetToolDefinition(%s) error = %v, want not found", name, err)
		}
	}

	current, err := store.GetToolDefinition(builtinToolID("chapter.list"))
	if err != nil {
		t.Fatalf("GetToolDefinition(chapter.list) error: %v", err)
	}
	if current.Status != domain.ToolStatusActive {
		t.Fatalf("chapter.list status = %q, want active", current.Status)
	}
	if _, err := store.GetToolDefinition("tool_skill_keep"); err != nil {
		t.Fatalf("non-builtin tool was deleted: %v", err)
	}

	updatedAgent, err := store.GetAgentConfig(agentCfg.ID)
	if err != nil {
		t.Fatalf("GetAgentConfig() error: %v", err)
	}
	wantToolIDs := []string{builtinToolID("character.search"), "tool_skill_keep"}
	if len(updatedAgent.ToolIDs) != len(wantToolIDs) {
		t.Fatalf("agent tool_ids = %#v, want %#v", updatedAgent.ToolIDs, wantToolIDs)
	}
	for i, id := range wantToolIDs {
		if updatedAgent.ToolIDs[i] != id {
			t.Fatalf("agent tool_ids = %#v, want %#v", updatedAgent.ToolIDs, wantToolIDs)
		}
	}
}

func TestListProviderToolsOmitsObsoleteBuiltinEvenIfStillActive(t *testing.T) {
	store := memory.NewStore()
	registry := NewRegistry(store, store)

	if err := registry.SeedBuiltinTools(context.Background()); err != nil {
		t.Fatalf("SeedBuiltinTools() error: %v", err)
	}
	// Force a stale active row after seed so the list path cannot rely only on delete cleanup.
	if _, err := store.UpsertToolDefinition(domain.ToolDefinition{
		ID:          builtinToolID("chapter.ensure"),
		Name:        "chapter.ensure",
		DisplayName: "chapter.ensure",
		Kind:        domain.ToolDefinitionBuiltin,
		Status:      domain.ToolStatusActive,
	}); err != nil {
		t.Fatalf("UpsertToolDefinition(chapter.ensure) error: %v", err)
	}

	tools, err := registry.ListProviderTools(context.Background(), domain.AgentConfig{})
	if err != nil {
		t.Fatalf("ListProviderTools() error: %v", err)
	}
	for _, tool := range tools {
		if tool.Name == "chapter.ensure" || tool.Name == "chapter.create" {
			t.Fatalf("ListProviderTools() exposed removed tool %q", tool.Name)
		}
	}

	want := map[string]bool{}
	for _, spec := range agent.NarrativeToolSpecs() {
		if agent.IsOptInBuiltinTool(spec.Name) {
			continue
		}
		want[spec.Name] = true
	}
	if len(tools) != len(want) {
		t.Fatalf("ListProviderTools() count = %d, want %d (default excludes opt-in tools)", len(tools), len(want))
	}
	for _, tool := range tools {
		if !want[tool.Name] {
			t.Fatalf("ListProviderTools() unexpected tool %q", tool.Name)
		}
		if agent.IsOptInBuiltinTool(tool.Name) {
			t.Fatalf("ListProviderTools() exposed opt-in tool %q without tool_ids", tool.Name)
		}
	}

	// Explicit tool_ids may opt into nested LLM audit tools.
	optInTools, err := registry.ListProviderTools(context.Background(), domain.AgentConfig{
		ToolIDs: []string{agent.BuiltinChapterAuditToolID},
	})
	if err != nil {
		t.Fatalf("ListProviderTools(opt-in) error: %v", err)
	}
	if len(optInTools) != 1 || optInTools[0].Name != agent.ChapterAuditToolName {
		t.Fatalf("ListProviderTools(opt-in) = %+v, want only %s", optInTools, agent.ChapterAuditToolName)
	}
}

func TestScrubObsoleteBuiltinToolIDs(t *testing.T) {
	current := map[string]bool{"character.search": true, "chapter.list": true}
	cleaned, changed := scrubObsoleteBuiltinToolIDs([]string{
		"builtin:character.search",
		"builtin:chapter.ensure",
		"builtin.chapter.create",
		"tool_mcp_keep",
		"",
	}, current)
	if !changed {
		t.Fatalf("scrubObsoleteBuiltinToolIDs() changed = false, want true")
	}
	want := []string{"builtin:character.search", "tool_mcp_keep"}
	if len(cleaned) != len(want) {
		t.Fatalf("cleaned = %#v, want %#v", cleaned, want)
	}
	for i := range want {
		if cleaned[i] != want[i] {
			t.Fatalf("cleaned = %#v, want %#v", cleaned, want)
		}
	}
}
