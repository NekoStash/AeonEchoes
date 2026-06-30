package repository

import "aeonechoes/server/internal/domain"

// AppStore defines the full persistence surface used by HTTP handlers, agents,
// routing, tools, context construction and indexing orchestration.
type AppStore interface {
	NewID(prefix string) (string, error)

	CreateProvider(cfg domain.ProviderConfig) (domain.ProviderConfig, error)
	UpdateProvider(id string, cfg domain.ProviderConfig) (domain.ProviderConfig, error)
	GetProvider(id string) (domain.ProviderConfig, error)
	DeleteProvider(id string) error
	ListProviders() ([]domain.ProviderConfig, error)
	TouchProviderModelRefresh(id string) error

	CreateModel(cfg domain.ModelConfig) (domain.ModelConfig, error)
	UpdateModel(id string, cfg domain.ModelConfig) (domain.ModelConfig, error)
	UpsertModel(cfg domain.ModelConfig) (domain.ModelConfig, error)
	GetModel(id string) (domain.ModelConfig, error)
	DeleteModel(id string) error
	ListModels() ([]domain.ModelConfig, error)
	ListModelsByKind(kind domain.ModelKind) ([]domain.ModelConfig, error)

	CreateProject(project domain.Project, bible domain.StoryBible) (domain.Project, domain.StoryBible, error)
	GetProject(id string) (domain.Project, error)
	ListProjects() ([]domain.Project, error)
	GetStoryBible(projectID string) (domain.StoryBible, error)
	UpdateStoryBible(projectID string, bible domain.StoryBible) (domain.StoryBible, error)

	GetSetting(scope, key string) (domain.AppSetting, error)
	UpsertSetting(setting domain.AppSetting) (domain.AppSetting, error)
	ListSettings(scope string) ([]domain.AppSetting, error)

	SaveWorldline(item domain.Worldline) (domain.Worldline, error)
	SaveEntity(item domain.Entity) (domain.Entity, error)
	SaveFact(item domain.Fact) (domain.Fact, error)
	SaveGraphEdge(item domain.GraphEdge) (domain.GraphEdge, error)
	SavePlotThread(item domain.PlotThread) (domain.PlotThread, error)
	ListEntities(projectID string) ([]domain.Entity, error)
	ListFacts(projectID string) ([]domain.Fact, error)
	ListPlotThreads(projectID string) ([]domain.PlotThread, error)
	ExpandGraph(projectID string, entityIDs []string, depth int) (domain.GraphExpansion, error)

	EnsureChapter(req domain.ChapterEnsureRequest) (domain.Chapter, error)
	GetChapter(id string) (domain.Chapter, error)
	ListChapters(projectID string) ([]domain.Chapter, error)
	SaveChapterVersion(version domain.ChapterVersion) (domain.ChapterVersion, domain.IndexJob, error)
	GetChapterVersion(id string) (domain.ChapterVersion, error)
	UpdateChapterVersionIndexStatus(id, status string) (domain.ChapterVersion, error)
	ListChapterVersions(projectID, chapterID string) ([]domain.ChapterVersion, error)
	CreateIndexJob(job domain.IndexJob) (domain.IndexJob, error)

	GetIndexJob(id string) (domain.IndexJob, error)
	UpdateIndexJobStatus(id, status, errorMessage string) (domain.IndexJob, error)
	ListIndexJobs(projectID string) ([]domain.IndexJob, error)
	ListPendingIndexJobs(projectID string, limit int) ([]domain.IndexJob, error)

	SaveWorkflow(workflow domain.AIWorkflow) (domain.AIWorkflow, error)
	ListWorkflows(projectID string) ([]domain.AIWorkflow, error)
	GetWorkflow(id string) (domain.AIWorkflow, error)
}
