package postgres

import (
	"context"
	"os"
	"testing"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func openPostgresTestStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("AE_POSTGRES_TEST_DSN")
	if dsn == "" {
		dsn = os.Getenv("AE_POSTGRES_DSN")
	}
	if dsn == "" {
		t.Skip("set AE_POSTGRES_TEST_DSN or AE_POSTGRES_DSN to run postgres tests")
	}
	ctx := context.Background()
	pool, err := Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	t.Cleanup(func() { Close(pool) })
	if err := RunMigrations(ctx, pool, ""); err != nil {
		t.Fatalf("RunMigrations() error: %v", err)
	}
	store, err := NewStore(pool)
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}
	return store, pool
}

func TestListAgentConfigsUsesEffectiveProjectScope(t *testing.T) {
	store, pool := openPostgresTestStore(t)
	ids := []string{"test-effective-project-a", "test-effective-project-b", "test-effective-global", "test-effective-other", "test-effective-disabled"}
	_, _ = pool.Exec(context.Background(), `DELETE FROM agent_configs WHERE id = ANY($1)`, ids)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM agent_configs WHERE id = ANY($1)`, ids); err != nil {
			t.Logf("cleanup effective Agent configs: %v", err)
		}
	})
	configs := []domain.AgentConfig{
		{ID: ids[0], ProjectID: "project-effective", Name: "Alpha", Role: domain.AgentRoleWriter, Enabled: true},
		{ID: ids[1], ProjectID: "project-effective", Name: "Beta", Role: domain.AgentRoleEditor, Enabled: true},
		{ID: ids[2], Name: "Alpha", Role: domain.AgentRoleWriter, Enabled: true},
		{ID: ids[3], ProjectID: "project-other", Name: "Other", Role: domain.AgentRoleWriter, Enabled: true},
		{ID: ids[4], ProjectID: "project-effective", Name: "Disabled", Role: domain.AgentRoleWriter, Enabled: false},
	}
	for _, cfg := range configs {
		if _, err := store.CreateAgentConfig(cfg); err != nil {
			t.Fatalf("CreateAgentConfig(%q) error: %v", cfg.ID, err)
		}
	}
	enabled := true
	items, err := store.ListAgentConfigs(repository.AgentConfigFilter{ProjectID: "project-effective", Enabled: &enabled})
	if err != nil {
		t.Fatalf("ListAgentConfigs(enabled) error: %v", err)
	}
	positions := map[string]int{}
	for index, item := range items {
		positions[item.ID] = index
		if item.ID == ids[3] || item.ProjectID == "project-other" {
			t.Fatalf("other-project Agent leaked into effective scope: %+v", item)
		}
	}
	for _, id := range []string{ids[0], ids[1], ids[2]} {
		if _, ok := positions[id]; !ok {
			t.Fatalf("effective Agent %q missing: items=%+v", id, items)
		}
	}
	if positions[ids[0]] >= positions[ids[1]] || positions[ids[1]] >= positions[ids[2]] {
		t.Fatalf("effective Agent ordering is not project name/id then global: positions=%+v items=%+v", positions, items)
	}
	disabled := false
	items, err = store.ListAgentConfigs(repository.AgentConfigFilter{ProjectID: "project-effective", Enabled: &disabled})
	if err != nil {
		t.Fatalf("ListAgentConfigs(disabled) error: %v", err)
	}
	foundDisabled := false
	for _, item := range items {
		if item.ID == ids[3] || item.ProjectID == "project-other" {
			t.Fatalf("other-project disabled Agent leaked into effective scope: %+v", item)
		}
		if item.ID == ids[4] {
			foundDisabled = !item.Enabled
		}
	}
	if !foundDisabled {
		t.Fatalf("disabled effective agents = %+v, want %q", items, ids[4])
	}
}

func TestChapterAndGraphContractsMatchMemoryStore(t *testing.T) {
	store, pool := openPostgresTestStore(t)
	projectA, _, err := store.CreateProject(domain.Project{Title: "Postgres 严格契约 A"}, domain.StoryBible{Title: "Postgres 严格契约 A", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject(A) error: %v", err)
	}
	projectB, _, err := store.CreateProject(domain.Project{Title: "Postgres 严格契约 B"}, domain.StoryBible{Title: "Postgres 严格契约 B", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject(B) error: %v", err)
	}
	t.Cleanup(func() {
		for _, projectID := range []string{projectA.ID, projectB.ID} {
			if _, err := pool.Exec(context.Background(), `DELETE FROM projects WHERE id=$1`, projectID); err != nil {
				t.Logf("cleanup project %q: %v", projectID, err)
			}
		}
	})
	if _, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectA.ID, Title: "  "}); err == nil {
		t.Fatalf("CreateChapter(blank title) error = nil")
	}
	if _, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectA.ID, Title: "非法状态", Status: "draft"}); err == nil {
		t.Fatalf("CreateChapter(invalid status) error = nil")
	}
	chapter, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectA.ID, Title: "第一章"})
	if err != nil {
		t.Fatalf("CreateChapter() error: %v", err)
	}
	if chapter.Status != domain.ChapterStatusDrafting {
		t.Fatalf("default chapter status = %q, want %q", chapter.Status, domain.ChapterStatusDrafting)
	}
	invalidStatus := domain.ChapterStatus("done")
	if _, err := store.UpdateChapter(domain.UpdateChapterRequest{ProjectID: projectA.ID, ChapterID: chapter.ID, Status: &invalidStatus}); err == nil {
		t.Fatalf("UpdateChapter(invalid status) error = nil")
	}
	if _, err := store.UpdateChapter(domain.UpdateChapterRequest{ProjectID: projectA.ID, ChapterID: chapter.ID}); err == nil {
		t.Fatalf("UpdateChapter(empty) error = nil")
	}
	if _, err := store.ListChapters("missing"); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("ListChapters(missing) error = %v, want not found", err)
	}
	if _, err := store.ListChapterVersions(projectB.ID, chapter.ID); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("ListChapterVersions(wrong project) error = %v, want not found", err)
	}
	if _, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapter.ID, Title: "版本", Content: "正文", AuthorRole: "reader"}); err == nil {
		t.Fatalf("SaveChapterVersion(invalid author role) error = nil")
	}
	if _, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapter.ID, ParentVersionID: "missing", Title: "缺失父版本", Content: "正文", AuthorRole: domain.AgentRoleWriter}); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("SaveChapterVersion(missing parent) error = %v, want not found", err)
	}
	firstVersion, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapter.ID, Title: "第一版", Content: "正文一", AuthorRole: domain.AgentRoleWriter})
	if err != nil {
		t.Fatalf("SaveChapterVersion(first) error: %v", err)
	}
	secondVersion, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapter.ID, ParentVersionID: firstVersion.ID, Title: "第二版", Content: "正文二", AuthorRole: domain.AgentRoleEditor})
	if err != nil {
		t.Fatalf("SaveChapterVersion(second) error: %v", err)
	}
	if secondVersion.ParentVersionID != firstVersion.ID {
		t.Fatalf("second parent = %q, want %q", secondVersion.ParentVersionID, firstVersion.ID)
	}
	otherChapter, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectA.ID, Title: "第二章"})
	if err != nil {
		t.Fatalf("CreateChapter(other) error: %v", err)
	}
	if _, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: otherChapter.ID, ParentVersionID: firstVersion.ID, Title: "跨章节", Content: "正文", AuthorRole: domain.AgentRoleEditor}); err == nil {
		t.Fatalf("SaveChapterVersion(cross chapter parent) error = nil")
	}
	entity, err := store.SaveEntity(domain.Entity{ProjectID: projectA.ID, Name: "林烬", Type: "character"})
	if err != nil {
		t.Fatalf("SaveEntity() error: %v", err)
	}
	for _, depth := range []int{0, 5} {
		if _, err := store.ExpandGraph(projectA.ID, []string{entity.ID}, depth); err == nil {
			t.Fatalf("ExpandGraph(depth=%d) error = nil", depth)
		}
	}
	if _, err := store.ExpandGraph(projectA.ID, []string{"missing"}, 1); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("ExpandGraph(missing entity) error = %v, want not found", err)
	}
	expansion, err := store.ExpandGraph(projectA.ID, []string{entity.ID}, 1)
	if err != nil {
		t.Fatalf("ExpandGraph(valid) error: %v", err)
	}
	if expansion.GeneratedAt.IsZero() || len(expansion.Entities) != 1 {
		t.Fatalf("ExpandGraph(valid) result = %+v", expansion)
	}
}

func TestUpdateStoryBibleCreatesNewActiveVersionWhenExistingBibleIsSavedAgain(t *testing.T) {
	store, pool := openPostgresTestStore(t)

	project, initialBible, err := store.CreateProject(domain.Project{Title: "Postgres Story Bible 版本测试", Seed: domain.ProjectSeed{Title: "Postgres Story Bible 版本测试", Premise: "测试重复保存设定集"}}, domain.StoryBible{Title: "Postgres Story Bible 版本测试", Logline: "初始设定"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM projects WHERE id=$1`, project.ID); err != nil {
			t.Logf("cleanup project %q: %v", project.ID, err)
		}
	})

	loadedBible, err := store.GetStoryBible(project.ID)
	if err != nil {
		t.Fatalf("GetStoryBible() error: %v", err)
	}
	loadedBible.Logline = "第一次更新"
	firstUpdate, err := store.UpdateStoryBible(project.ID, loadedBible)
	if err != nil {
		t.Fatalf("UpdateStoryBible() first error: %v", err)
	}
	loadedBible.Logline = "第二次更新"
	secondUpdate, err := store.UpdateStoryBible(project.ID, loadedBible)
	if err != nil {
		t.Fatalf("UpdateStoryBible() second error: %v", err)
	}

	if firstUpdate.ID == initialBible.ID || secondUpdate.ID == initialBible.ID || secondUpdate.ID == firstUpdate.ID {
		t.Fatalf("updates reused story bible version IDs: initial=%q first=%q second=%q", initialBible.ID, firstUpdate.ID, secondUpdate.ID)
	}
	if firstUpdate.Version != initialBible.Version+1 || secondUpdate.Version != firstUpdate.Version+1 {
		t.Fatalf("versions did not increment: initial=%d first=%d second=%d", initialBible.Version, firstUpdate.Version, secondUpdate.Version)
	}
	activeProject, err := store.GetProject(project.ID)
	if err != nil {
		t.Fatalf("GetProject() error: %v", err)
	}
	if activeProject.ActiveStoryBibleID != secondUpdate.ID {
		t.Fatalf("active story bible = %q, want %q", activeProject.ActiveStoryBibleID, secondUpdate.ID)
	}
	activeBible, err := store.GetStoryBible(project.ID)
	if err != nil {
		t.Fatalf("GetStoryBible() active error: %v", err)
	}
	if activeBible.ID != secondUpdate.ID || activeBible.Logline != "第二次更新" {
		t.Fatalf("active story bible mismatch: %+v", activeBible)
	}
}
