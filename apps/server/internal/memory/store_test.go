package memory

import (
	"testing"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/repository"
)

func TestProviderModelAndProjectLifecycle(t *testing.T) {
	store := NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{Name: "OpenAI", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	modelCfg, err := store.CreateModel(domain.ModelConfig{ProviderID: providerCfg.ID, Name: "gpt-test", Kind: domain.ModelKindText, Enabled: true, DefaultForKind: true})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	if modelCfg.ProviderType != domain.ProviderOpenAI {
		t.Fatalf("model provider type was not derived: %+v", modelCfg)
	}

	project, bible, err := store.CreateProject(domain.Project{Title: "星海回声", Seed: domain.ProjectSeed{Title: "星海回声", Premise: "远航者寻找失落文明"}}, domain.StoryBible{Title: "星海回声", Logline: "远航者寻找失落文明"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}
	if project.ActiveStoryBibleID != bible.ID {
		t.Fatalf("project active bible mismatch")
	}
	chapter, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: project.ID, Title: "第一章"})
	if err != nil {
		t.Fatalf("CreateChapter() error: %v", err)
	}
	if chapter.Status != domain.ChapterStatusDrafting {
		t.Fatalf("default chapter status = %q, want %q", chapter.Status, domain.ChapterStatusDrafting)
	}
	version, job, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: project.ID, ChapterID: chapter.ID, Title: "第一章", Content: "群星在船窗外燃烧。", AuthorRole: domain.AgentRoleWriter})
	if err != nil {
		t.Fatalf("SaveChapterVersion() error: %v", err)
	}
	if version.Version != 1 || job.Status != "pending" || job.ChapterVersionID != version.ID {
		t.Fatalf("unexpected version/job: %+v %+v", version, job)
	}
}

func TestSaveChapterVersionSupersedesOlderPendingJobs(t *testing.T) {
	store := NewStore()
	project, _, err := store.CreateProject(domain.Project{Title: "Supersede", Seed: domain.ProjectSeed{Title: "Supersede", Premise: "测试 supersede"}}, domain.StoryBible{Title: "Supersede", Logline: "测试 supersede"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}
	chapter, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: project.ID, Title: "第一章"})
	if err != nil {
		t.Fatalf("CreateChapter() error: %v", err)
	}
	firstVersion, firstJob, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: project.ID, ChapterID: chapter.ID, Title: "第一版", Content: "内容 1", AuthorRole: domain.AgentRoleWriter})
	if err != nil {
		t.Fatalf("SaveChapterVersion(first) error: %v", err)
	}
	if _, err := store.UpdateIndexJobStatus(firstJob.ID, "running", ""); err != nil {
		t.Fatalf("UpdateIndexJobStatus(running) error: %v", err)
	}
	secondVersion, secondJob, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: project.ID, ChapterID: chapter.ID, Title: "第二版", Content: "内容 2", AuthorRole: domain.AgentRoleWriter})
	if err != nil {
		t.Fatalf("SaveChapterVersion(second) error: %v", err)
	}
	if secondVersion.ID == firstVersion.ID {
		t.Fatalf("expected a new chapter version, got same id %q", secondVersion.ID)
	}
	if secondJob.Status != "pending" {
		t.Fatalf("second job status = %q, want pending", secondJob.Status)
	}
	allJobs, err := store.ListIndexJobs(repository.IndexJobFilter{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("ListIndexJobs() error: %v", err)
	}
	statusByID := map[string]string{}
	for _, job := range allJobs {
		statusByID[job.ID] = job.Status
	}
	if statusByID[firstJob.ID] != "running" {
		t.Fatalf("running job should not be superseded, got %q", statusByID[firstJob.ID])
	}
	if statusByID[secondJob.ID] != "pending" {
		t.Fatalf("latest job status = %q, want pending", statusByID[secondJob.ID])
	}
	thirdVersion, thirdJob, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: project.ID, ChapterID: chapter.ID, Title: "第三版", Content: "内容 3", AuthorRole: domain.AgentRoleWriter})
	if err != nil {
		t.Fatalf("SaveChapterVersion(third) error: %v", err)
	}
	if thirdVersion.ID == secondVersion.ID {
		t.Fatalf("expected third chapter version to be new")
	}
	allJobs, err = store.ListIndexJobs(repository.IndexJobFilter{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("ListIndexJobs() error: %v", err)
	}
	statusByID = map[string]string{}
	for _, job := range allJobs {
		statusByID[job.ID] = job.Status
	}
	if statusByID[secondJob.ID] != "superseded" {
		t.Fatalf("older pending job status = %q, want superseded", statusByID[secondJob.ID])
	}
	if statusByID[thirdJob.ID] != "pending" {
		t.Fatalf("newest job status = %q, want pending", statusByID[thirdJob.ID])
	}
	pendingJobs, err := store.ListPendingIndexJobs(project.ID, 0)
	if err != nil {
		t.Fatalf("ListPendingIndexJobs() error: %v", err)
	}
	if len(pendingJobs) != 1 || pendingJobs[0].ID != thirdJob.ID {
		t.Fatalf("pending jobs = %+v, want only newest pending job", pendingJobs)
	}
}

func TestSaveChapterVersionParentContracts(t *testing.T) {
	store := NewStore()
	projectA, _, err := store.CreateProject(domain.Project{Title: "父版本 A"}, domain.StoryBible{Title: "父版本 A", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject(A) error: %v", err)
	}
	projectB, _, err := store.CreateProject(domain.Project{Title: "父版本 B"}, domain.StoryBible{Title: "父版本 B", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject(B) error: %v", err)
	}
	chapterA1, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectA.ID, Title: "A 第一章"})
	if err != nil {
		t.Fatalf("CreateChapter(A1) error: %v", err)
	}
	chapterA2, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectA.ID, Title: "A 第二章"})
	if err != nil {
		t.Fatalf("CreateChapter(A2) error: %v", err)
	}
	chapterB, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectB.ID, Title: "B 第一章"})
	if err != nil {
		t.Fatalf("CreateChapter(B) error: %v", err)
	}
	first, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapterA1.ID, Title: "第一版", Content: "正文一", AuthorRole: domain.AgentRoleWriter})
	if err != nil {
		t.Fatalf("SaveChapterVersion(first) error: %v", err)
	}
	second, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapterA1.ID, ParentVersionID: first.ID, Title: "第二版", Content: "正文二", AuthorRole: domain.AgentRoleEditor})
	if err != nil {
		t.Fatalf("SaveChapterVersion(second) error: %v", err)
	}
	third, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapterA1.ID, ParentVersionID: second.ID, Title: "第三版", Content: "正文三", AuthorRole: domain.AgentRoleEditor})
	if err != nil {
		t.Fatalf("SaveChapterVersion(third) error: %v", err)
	}
	if second.ParentVersionID != first.ID || third.ParentVersionID != second.ID {
		t.Fatalf("valid parent chain not preserved: first=%+v second=%+v third=%+v", first, second, third)
	}
	if _, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapterA1.ID, ParentVersionID: "missing", Title: "缺失父版本", Content: "正文", AuthorRole: domain.AgentRoleEditor}); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("missing parent error = %v, want not found", err)
	}
	crossChapter, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapterA2.ID, Title: "另一章版本", Content: "正文", AuthorRole: domain.AgentRoleWriter})
	if err != nil {
		t.Fatalf("SaveChapterVersion(cross chapter seed) error: %v", err)
	}
	if _, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapterA1.ID, ParentVersionID: crossChapter.ID, Title: "跨章节", Content: "正文", AuthorRole: domain.AgentRoleEditor}); err == nil {
		t.Fatalf("SaveChapterVersion(cross chapter parent) error = nil")
	}
	crossProject, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectB.ID, ChapterID: chapterB.ID, Title: "跨项目版本", Content: "正文", AuthorRole: domain.AgentRoleWriter})
	if err != nil {
		t.Fatalf("SaveChapterVersion(cross project seed) error: %v", err)
	}
	if _, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapterA1.ID, ParentVersionID: crossProject.ID, Title: "跨项目", Content: "正文", AuthorRole: domain.AgentRoleEditor}); err == nil {
		t.Fatalf("SaveChapterVersion(cross project parent) error = nil")
	}
	if _, _, err := store.SaveChapterVersion(domain.ChapterVersion{ID: "self", ProjectID: projectA.ID, ChapterID: chapterA1.ID, ParentVersionID: "self", Title: "自指", Content: "正文", AuthorRole: domain.AgentRoleEditor}); err == nil {
		t.Fatalf("SaveChapterVersion(self parent) error = nil")
	}
	corruptedFirst := first
	corruptedFirst.ParentVersionID = second.ID
	store.chapterVersions[first.ID] = corruptedFirst
	if _, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapterA1.ID, ParentVersionID: second.ID, Title: "循环链", Content: "正文", AuthorRole: domain.AgentRoleEditor}); err == nil {
		t.Fatalf("SaveChapterVersion(cyclic parent chain) error = nil")
	}
}

func TestSaveChapterVersionRequiresExistingProjectChapter(t *testing.T) {
	store := NewStore()
	project, _, err := store.CreateProject(domain.Project{Title: "严格版本", Seed: domain.ProjectSeed{Title: "严格版本", Premise: "测试"}}, domain.StoryBible{Title: "严格版本", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}

	if _, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: project.ID, Title: "缺失章节", Content: "缺失章节 ID", AuthorRole: domain.AgentRoleWriter}); err == nil {
		t.Fatalf("SaveChapterVersion() missing chapter_id error = nil")
	}
	if _, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: project.ID, ChapterID: "missing", Title: "不存在章节", Content: "不存在章节", AuthorRole: domain.AgentRoleWriter}); err == nil {
		t.Fatalf("SaveChapterVersion() missing chapter error = nil")
	}
	chapters, err := store.ListChapters(project.ID)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}
	if len(chapters) != 0 {
		t.Fatalf("failed version saves created chapters: %+v", chapters)
	}
}

func TestChapterStorageStrictContracts(t *testing.T) {
	store := NewStore()
	projectA, _, err := store.CreateProject(domain.Project{Title: "严格章节 A"}, domain.StoryBible{Title: "严格章节 A", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject(A) error: %v", err)
	}
	projectB, _, err := store.CreateProject(domain.Project{Title: "严格章节 B"}, domain.StoryBible{Title: "严格章节 B", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject(B) error: %v", err)
	}
	if _, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectA.ID, Title: "   "}); err == nil {
		t.Fatalf("CreateChapter(blank title) error = nil")
	}
	if _, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectA.ID, Title: "非法状态", Status: "draft"}); err == nil {
		t.Fatalf("CreateChapter(invalid status) error = nil")
	}
	chapter, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectA.ID, Title: "第一章"})
	if err != nil {
		t.Fatalf("CreateChapter() error: %v", err)
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
}

func TestExpandGraphRejectsInvalidDepthAndEntityIDs(t *testing.T) {
	store := NewStore()
	project, _, err := store.CreateProject(domain.Project{Title: "严格图谱"}, domain.StoryBible{Title: "严格图谱", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}
	entity, err := store.SaveEntity(domain.Entity{ProjectID: project.ID, Name: "林烬", Type: "character"})
	if err != nil {
		t.Fatalf("SaveEntity() error: %v", err)
	}
	for _, depth := range []int{0, 5} {
		if _, err := store.ExpandGraph(project.ID, []string{entity.ID}, depth); err == nil {
			t.Fatalf("ExpandGraph(depth=%d) error = nil", depth)
		}
	}
	if _, err := store.ExpandGraph(project.ID, []string{"missing"}, 1); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("ExpandGraph(missing entity) error = %v, want not found", err)
	}
}

func TestDeleteProviderRemovesAssociatedModels(t *testing.T) {
	store := NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_delete", Name: "Delete Me", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	otherProviderCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_keep", Name: "Keep Me", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() other error: %v", err)
	}
	deletedModel, err := store.CreateModel(domain.ModelConfig{ProviderID: providerCfg.ID, Name: "gpt-delete", Kind: domain.ModelKindText, Enabled: true})
	if err != nil {
		t.Fatalf("CreateModel() delete error: %v", err)
	}
	keptModel, err := store.CreateModel(domain.ModelConfig{ProviderID: otherProviderCfg.ID, Name: "gpt-keep", Kind: domain.ModelKindText, Enabled: true})
	if err != nil {
		t.Fatalf("CreateModel() keep error: %v", err)
	}

	if err := store.DeleteProvider(providerCfg.ID); err != nil {
		t.Fatalf("DeleteProvider() error: %v", err)
	}
	models, err := store.ListModels()
	if err != nil {
		t.Fatalf("ListModels() error: %v", err)
	}
	for _, model := range models {
		if model.ID == deletedModel.ID || model.ProviderID == providerCfg.ID {
			t.Fatalf("deleted provider model remains after DeleteProvider(): %+v", model)
		}
	}
	if len(models) != 1 || models[0].ID != keptModel.ID {
		t.Fatalf("remaining models = %+v, want only %+v", models, keptModel)
	}
}

func TestUpdateStoryBibleCreatesNewActiveVersionWhenExistingBibleIsSavedAgain(t *testing.T) {
	store := NewStore()
	project, initialBible, err := store.CreateProject(domain.Project{Title: "版本测试", Seed: domain.ProjectSeed{Title: "版本测试", Premise: "测试重复保存设定集"}}, domain.StoryBible{Title: "版本测试", Logline: "初始设定"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}

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

func TestExpandGraphIncludesNeighborsAndFacts(t *testing.T) {
	store := NewStore()
	project, _, err := store.CreateProject(domain.Project{Title: "图谱测试", Seed: domain.ProjectSeed{Title: "图谱测试", Premise: "测试"}}, domain.StoryBible{Title: "图谱测试", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}
	alice, err := store.SaveEntity(domain.Entity{ProjectID: project.ID, Name: "艾莉丝", Type: "character"})
	if err != nil {
		t.Fatalf("SaveEntity alice: %v", err)
	}
	city, err := store.SaveEntity(domain.Entity{ProjectID: project.ID, Name: "钟城", Type: "place"})
	if err != nil {
		t.Fatalf("SaveEntity city: %v", err)
	}
	fact, err := store.SaveFact(domain.Fact{ProjectID: project.ID, EntityID: alice.ID, Claim: "艾莉丝来自钟城", Confidence: 0.9, Status: "active"})
	if err != nil {
		t.Fatalf("SaveFact: %v", err)
	}
	if _, err := store.SaveGraphEdge(domain.GraphEdge{ProjectID: project.ID, SourceEntityID: alice.ID, TargetEntityID: city.ID, Type: "origin", Label: "来自", EvidenceFactIDs: []string{fact.ID}}); err != nil {
		t.Fatalf("SaveGraphEdge: %v", err)
	}
	expansion, err := store.ExpandGraph(project.ID, []string{alice.ID}, 2)
	if err != nil {
		t.Fatalf("ExpandGraph() error: %v", err)
	}
	if len(expansion.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d: %+v", len(expansion.Entities), expansion.Entities)
	}
	if len(expansion.Edges) != 1 || len(expansion.Facts) != 1 {
		t.Fatalf("expected edge and fact, got %+v", expansion)
	}
}

func TestSettingsAndWorkflowsLifecycle(t *testing.T) {
	store := NewStore()
	project, _, err := store.CreateProject(domain.Project{Title: "设置测试", Seed: domain.ProjectSeed{Title: "设置测试", Premise: "测试设置与工作流"}}, domain.StoryBible{Title: "设置测试", Logline: "测试设置与工作流"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}
	setting, err := store.UpsertSetting(domain.AppSetting{Scope: "project", Key: project.ID, Value: map[string]any{"theme": "sci-fi", "drafting": true}})
	if err != nil {
		t.Fatalf("UpsertSetting() error: %v", err)
	}
	if setting.Scope != "project" || setting.Key != project.ID {
		t.Fatalf("unexpected setting: %+v", setting)
	}
	loaded, err := store.GetSetting("project", project.ID)
	if err != nil {
		t.Fatalf("GetSetting() error: %v", err)
	}
	if loaded.Value["theme"] != "sci-fi" {
		t.Fatalf("unexpected setting value: %+v", loaded)
	}
	settings, err := store.ListSettings("project")
	if err != nil {
		t.Fatalf("ListSettings() error: %v", err)
	}
	if len(settings) != 1 {
		t.Fatalf("expected 1 setting, got %d", len(settings))
	}
	wf1, err := store.SaveWorkflow(domain.AIWorkflow{ProjectID: project.ID, Kind: "draft", Status: "running"})
	if err != nil {
		t.Fatalf("SaveWorkflow wf1 error: %v", err)
	}
	wf2, err := store.SaveWorkflow(domain.AIWorkflow{ProjectID: project.ID, Kind: "review", Status: "completed"})
	if err != nil {
		t.Fatalf("SaveWorkflow wf2 error: %v", err)
	}
	loadedWorkflow, err := store.GetWorkflow(wf1.ID)
	if err != nil {
		t.Fatalf("GetWorkflow() error: %v", err)
	}
	if loadedWorkflow.ID != wf1.ID {
		t.Fatalf("unexpected workflow: %+v", loadedWorkflow)
	}
	workflows, err := store.ListWorkflows(project.ID)
	if err != nil {
		t.Fatalf("ListWorkflows() error: %v", err)
	}
	if len(workflows) != 2 {
		t.Fatalf("expected 2 workflows, got %d", len(workflows))
	}
	if workflows[0].ProjectID != project.ID || workflows[1].ID != wf2.ID {
		t.Fatalf("unexpected workflows ordering/content: %+v", workflows)
	}
}
