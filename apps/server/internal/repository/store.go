package repository

import "aeonechoes/server/internal/domain"

// IndexJobFilter describes optional filters for browsing index jobs without
// changing the legacy "no parameters returns all jobs" behavior.
type IndexJobFilter struct {
	ProjectID string
	Status    string
	Limit     int
}

// AgentConfigFilter describes optional filters for browsing agent configurations.
type AgentConfigFilter struct {
	Enabled   *bool
	ProjectID string
	Limit     int
}

// AgentRunFilter describes optional filters for browsing agent runs.
type AgentRunFilter struct {
	AgentID   string
	ProjectID string
	Status    domain.AgentRunStatus
	Limit     int
}

// SkillSourceFilter describes optional filters for browsing skill sources.
type SkillSourceFilter struct {
	Enabled   *bool
	ProjectID string
	Limit     int
}

// SkillFilter describes optional filters for browsing skills.
type SkillFilter struct {
	SourceID  string
	Enabled   *bool
	ProjectID string
	Limit     int
}

// MCPServerConfigFilter describes optional filters for browsing MCP servers.
type MCPServerConfigFilter struct {
	Enabled   *bool
	Status    domain.MCPServerStatus
	ProjectID string
	Limit     int
}

// ToolDefinitionFilter describes optional filters for browsing tool catalog entries.
type ToolDefinitionFilter struct {
	Kind        domain.ToolDefinitionKind
	Status      domain.ToolStatus
	MCPServerID string
	SourceID    string
	SkillID     string
	ProjectID   string
	Limit       int
}

// ToolInvocationFilter describes optional filters for browsing tool invocation history.
type ToolInvocationFilter struct {
	AgentRunID string
	AgentID    string
	ProjectID  string
	ToolID     string
	Status     domain.ToolInvocationStatus
	Limit      int
}

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

	CreateChapter(req domain.CreateChapterRequest) (domain.Chapter, error)
	UpdateChapter(req domain.UpdateChapterRequest) (domain.Chapter, error)
	GetChapter(id string) (domain.Chapter, error)
	ListChapters(projectID string) ([]domain.Chapter, error)
	SaveChapterVersion(version domain.ChapterVersion) (domain.ChapterVersion, domain.IndexJob, error)
	GetChapterVersion(id string) (domain.ChapterVersion, error)
	UpdateChapterVersionIndexStatus(id, status string) (domain.ChapterVersion, error)
	ListChapterVersions(projectID, chapterID string) ([]domain.ChapterVersion, error)
	CreateIndexJob(job domain.IndexJob) (domain.IndexJob, error)

	GetIndexJob(id string) (domain.IndexJob, error)
	UpdateIndexJobStatus(id, status, errorMessage string) (domain.IndexJob, error)
	ListIndexJobs(filter IndexJobFilter) ([]domain.IndexJob, error)
	ListPendingIndexJobs(projectID string, limit int) ([]domain.IndexJob, error)

	SaveWorkflow(workflow domain.AIWorkflow) (domain.AIWorkflow, error)
	ListWorkflows(projectID string) ([]domain.AIWorkflow, error)
	GetWorkflow(id string) (domain.AIWorkflow, error)

	CreateAgentConfig(cfg domain.AgentConfig) (domain.AgentConfig, error)
	UpdateAgentConfig(id string, cfg domain.AgentConfig) (domain.AgentConfig, error)
	GetAgentConfig(id string) (domain.AgentConfig, error)
	DeleteAgentConfig(id string) error
	ListAgentConfigs(filter AgentConfigFilter) ([]domain.AgentConfig, error)

	CreateAgentRun(run domain.AgentRun) (domain.AgentRun, error)
	UpdateAgentRun(id string, run domain.AgentRun) (domain.AgentRun, error)
	GetAgentRun(id string) (domain.AgentRun, error)
	ListAgentRuns(filter AgentRunFilter) ([]domain.AgentRun, error)

	CreateSkillSource(source domain.SkillSource) (domain.SkillSource, error)
	UpdateSkillSource(id string, source domain.SkillSource) (domain.SkillSource, error)
	GetSkillSource(id string) (domain.SkillSource, error)
	DeleteSkillSource(id string) error
	ListSkillSources(filter SkillSourceFilter) ([]domain.SkillSource, error)

	CreateSkill(skill domain.Skill) (domain.Skill, error)
	UpdateSkill(id string, skill domain.Skill) (domain.Skill, error)
	GetSkill(id string) (domain.Skill, error)
	DeleteSkill(id string) error
	ListSkills(filter SkillFilter) ([]domain.Skill, error)

	CreateMCPServerConfig(cfg domain.MCPServerConfig) (domain.MCPServerConfig, error)
	UpdateMCPServerConfig(id string, cfg domain.MCPServerConfig) (domain.MCPServerConfig, error)
	GetMCPServerConfig(id string) (domain.MCPServerConfig, error)
	DeleteMCPServerConfig(id string) error
	ListMCPServerConfigs(filter MCPServerConfigFilter) ([]domain.MCPServerConfig, error)

	UpsertToolDefinition(tool domain.ToolDefinition) (domain.ToolDefinition, error)
	GetToolDefinition(id string) (domain.ToolDefinition, error)
	DeleteToolDefinition(id string) error
	ListToolDefinitions(filter ToolDefinitionFilter) ([]domain.ToolDefinition, error)
	SetToolDefinitionEnabled(id string, enabled bool) (domain.ToolDefinition, error)

	CreateToolInvocation(invocation domain.ToolInvocation) (domain.ToolInvocation, error)
	UpdateToolInvocation(id string, invocation domain.ToolInvocation) (domain.ToolInvocation, error)
	GetToolInvocation(id string) (domain.ToolInvocation, error)
	ListToolInvocations(filter ToolInvocationFilter) ([]domain.ToolInvocation, error)
}
