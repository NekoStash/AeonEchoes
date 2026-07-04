package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/memory"
	"aeonechoes/server/internal/repository"
)

func TestScanDefaultSynchronizesDirectorySkills(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeSkillFile(t, root, "outline", "SKILL.md", "# Outline Coach\n\nKeep chapters focused.\nMore detail.")
	writeSkillFile(t, root, "voice", "skill.json", `{"name":"Voice Keeper","description":"Preserves voice","content":"Keep POV stable.","version":"1.0.0","metadata":{"tone":"steady"}}`)

	store := memory.NewStore()
	service := NewService(store, root)

	created, err := service.ScanDefault(ctx)
	if err != nil {
		t.Fatalf("ScanDefault() error: %v", err)
	}
	if created.SourceID != defaultSourceID || created.Path != root || created.Created != 2 || created.Updated != 0 || created.Deleted != 0 || created.Unchanged != 0 {
		t.Fatalf("first ScanDefault() result = %+v, want two created", created)
	}

	source, err := store.GetSkillSource(defaultSourceID)
	if err != nil {
		t.Fatalf("GetSkillSource(default) error: %v", err)
	}
	if source.Type != domain.SkillSourceDirectory || source.Path != root || !source.Enabled {
		t.Fatalf("default source mismatch: %+v", source)
	}

	skills, err := store.ListSkills(repository.SkillFilter{SourceID: defaultSourceID})
	if err != nil {
		t.Fatalf("ListSkills() error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("skills length = %d, want 2: %+v", len(skills), skills)
	}
	outline := skillByRelativePath(t, skills, "outline")
	if outline.Name != "Outline Coach" || outline.Description != "Keep chapters focused." || outline.Content == "" || outline.Metadata[metadataChecksum] == "" || outline.Metadata[metadataSourceTyp] != "directory" || !outline.Enabled {
		t.Fatalf("outline skill mismatch: %+v", outline)
	}
	voice := skillByRelativePath(t, skills, "voice")
	if voice.Name != "Voice Keeper" || voice.Metadata["version"] != "1.0.0" || voice.Metadata["tone"] != "steady" {
		t.Fatalf("voice skill mismatch: %+v", voice)
	}

	if _, err := store.UpdateSkill(outline.ID, domain.Skill{
		SourceID:    outline.SourceID,
		Name:        outline.Name,
		Description: outline.Description,
		Content:     outline.Content,
		Path:        outline.Path,
		Enabled:     false,
		Metadata:    outline.Metadata,
	}); err != nil {
		t.Fatalf("disable outline skill: %v", err)
	}
	writeSkillFile(t, root, "outline", "SKILL.md", "# Outline Coach v2\n\nKeep chapters sharper.\nMore detail.")
	if err := os.RemoveAll(filepath.Join(root, "voice")); err != nil {
		t.Fatalf("remove voice skill dir: %v", err)
	}

	updated, err := service.ScanDefault(ctx)
	if err != nil {
		t.Fatalf("second ScanDefault() error: %v", err)
	}
	if updated.Created != 0 || updated.Updated != 1 || updated.Deleted != 1 || updated.Unchanged != 0 {
		t.Fatalf("second ScanDefault() result = %+v, want one updated and one deleted", updated)
	}

	skills, err = store.ListSkills(repository.SkillFilter{SourceID: defaultSourceID})
	if err != nil {
		t.Fatalf("ListSkills() after update error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("skills after update length = %d, want 1: %+v", len(skills), skills)
	}
	updatedOutline := skills[0]
	if updatedOutline.ID != outline.ID || updatedOutline.Name != "Outline Coach v2" || updatedOutline.Description != "Keep chapters sharper." || updatedOutline.Enabled {
		t.Fatalf("updated outline mismatch: %+v", updatedOutline)
	}
	if _, err := store.GetSkill(voice.ID); err == nil {
		t.Fatalf("deleted skill %q can still be loaded", voice.ID)
	}

	unchanged, err := service.ScanDefault(ctx)
	if err != nil {
		t.Fatalf("third ScanDefault() error: %v", err)
	}
	if unchanged.Created != 0 || unchanged.Updated != 0 || unchanged.Deleted != 0 || unchanged.Unchanged != 1 {
		t.Fatalf("third ScanDefault() result = %+v, want one unchanged", unchanged)
	}
}

func TestScanSourceRequiresDirectorySource(t *testing.T) {
	store := memory.NewStore()
	source, err := store.CreateSkillSource(domain.SkillSource{Name: "Inline", Type: domain.SkillSourceInlineText, InlineText: "Use inline content.", Enabled: true})
	if err != nil {
		t.Fatalf("CreateSkillSource() error: %v", err)
	}

	if _, err := NewService(store, t.TempDir()).ScanSource(context.Background(), source.ID); err == nil {
		t.Fatalf("ScanSource() succeeded for inline source, want error")
	}
}

func TestCreateInlineCreatesSourceAndSkill(t *testing.T) {
	store := memory.NewStore()
	service := NewService(store, t.TempDir())

	skill, err := service.CreateInline(context.Background(), "Promise Keeper", "Tracks promises", "Resolve every setup.", false, map[string]string{"scope": "draft"})
	if err != nil {
		t.Fatalf("CreateInline() error: %v", err)
	}
	if skill.ID == "" || skill.SourceID == "" || skill.Name != "Promise Keeper" || skill.Description != "Tracks promises" || skill.Content != "Resolve every setup." || skill.Enabled || skill.Metadata["scope"] != "draft" {
		t.Fatalf("inline skill mismatch: %+v", skill)
	}
	source, err := store.GetSkillSource(skill.SourceID)
	if err != nil {
		t.Fatalf("GetSkillSource(inline) error: %v", err)
	}
	if source.Type != domain.SkillSourceInlineText || source.InlineText != skill.Content || source.Enabled {
		t.Fatalf("inline source mismatch: %+v", source)
	}
}

func writeSkillFile(t *testing.T, root, dir, name, content string) {
	t.Helper()
	path := filepath.Join(root, dir)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error: %v", path, err)
	}
	if err := os.WriteFile(filepath.Join(path, name), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q/%q) error: %v", path, name, err)
	}
}

func skillByRelativePath(t *testing.T, skills []domain.Skill, relativePath string) domain.Skill {
	t.Helper()
	for _, skill := range skills {
		if skill.Metadata[metadataRelPath] == relativePath {
			return skill
		}
	}
	t.Fatalf("skill with relative path %q not found in %+v", relativePath, skills)
	return domain.Skill{}
}
