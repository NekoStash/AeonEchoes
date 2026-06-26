package postgres

import (
	"context"
	"os"
	"testing"

	"aeonechoes/server/internal/domain"
)

func TestUpdateStoryBibleCreatesNewActiveVersionWhenExistingBibleIsSavedAgain(t *testing.T) {
	dsn := os.Getenv("AE_POSTGRES_TEST_DSN")
	if dsn == "" {
		dsn = os.Getenv("AE_POSTGRES_DSN")
	}
	if dsn == "" {
		t.Skip("set AE_POSTGRES_TEST_DSN or AE_POSTGRES_DSN to run postgres story bible versioning test")
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
