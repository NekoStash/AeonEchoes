package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"
)

type auditFakeTextClient struct {
	requests  []provider.TextRequest
	responses []provider.ModelResponse
}

func (c *auditFakeTextClient) Generate(_ context.Context, req provider.TextRequest) (provider.ModelResponse, error) {
	c.requests = append(c.requests, req)
	if len(c.responses) == 0 {
		return provider.ModelResponse{}, context.Canceled
	}
	resp := c.responses[0]
	c.responses = c.responses[1:]
	return resp, nil
}

func (c *auditFakeTextClient) Stream(context.Context, provider.TextRequest) (<-chan provider.StreamEvent, error) {
	return nil, context.Canceled
}

type auditFakeClientFactory struct {
	client provider.TextModelClient
}

func (f auditFakeClientFactory) NewTextClient(_ domain.ProviderConfig) (provider.TextModelClient, error) {
	return f.client, nil
}

type auditFakeCatalog struct {
	models []domain.ModelConfig
}

func (c auditFakeCatalog) ListModelsByKind(kind domain.ModelKind) ([]domain.ModelConfig, error) {
	items := make([]domain.ModelConfig, 0)
	for _, model := range c.models {
		if model.Kind == kind {
			items = append(items, model)
		}
	}
	return items, nil
}

func (c auditFakeCatalog) GetProvider(id string) (domain.ProviderConfig, error) {
	return domain.ProviderConfig{ID: id, Name: id, Enabled: true}, nil
}

func (c auditFakeCatalog) ListSettings(string) ([]domain.AppSetting, error) {
	return nil, nil
}

type auditFakeToolStore struct {
	versions []domain.ChapterVersion
}

func (s auditFakeToolStore) NewID(prefix string) (string, error) { return prefix + "-1", nil }
func (s auditFakeToolStore) SaveEntity(item domain.Entity) (domain.Entity, error) {
	return item, nil
}
func (s auditFakeToolStore) SaveGraphEdge(item domain.GraphEdge) (domain.GraphEdge, error) {
	return item, nil
}
func (s auditFakeToolStore) SavePlotThread(item domain.PlotThread) (domain.PlotThread, error) {
	return item, nil
}
func (s auditFakeToolStore) ListEntities(string) ([]domain.Entity, error) { return nil, nil }
func (s auditFakeToolStore) ListPlotThreads(string) ([]domain.PlotThread, error) {
	return nil, nil
}
func (s auditFakeToolStore) ExpandGraph(string, []string, int) (domain.GraphExpansion, error) {
	return domain.GraphExpansion{}, nil
}
func (s auditFakeToolStore) ListChapters(string) ([]domain.Chapter, error) { return nil, nil }
func (s auditFakeToolStore) ListChapterVersions(projectID, chapterID string) ([]domain.ChapterVersion, error) {
	items := make([]domain.ChapterVersion, 0)
	for _, version := range s.versions {
		if projectID != "" && version.ProjectID != projectID {
			continue
		}
		if chapterID != "" && version.ChapterID != chapterID {
			continue
		}
		items = append(items, version)
	}
	return items, nil
}

func TestParseChapterAuditResponseAcceptsStrictJSON(t *testing.T) {
	result, err := parseChapterAuditResponse(`{"status":"warning","summary":"节奏偏慢","issues":[{"type":"pacing","severity":"warning","message":"后半缺少推进","draft_excerpt":"他坐着","suggestion":"加入外部冲突"}],"rewrite_hints":["加快收束"]}`)
	if err != nil {
		t.Fatalf("parseChapterAuditResponse() error: %v", err)
	}
	if result.Status != "warning" || result.Summary != "节奏偏慢" || len(result.Issues) != 1 || result.Issues[0].Type != "pacing" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestParseChapterAuditResponseRejectsInvalidStatus(t *testing.T) {
	_, err := parseChapterAuditResponse(`{"status":"ok","summary":"x","issues":[]}`)
	if err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestParseAuditMaxRounds(t *testing.T) {
	value, err := ParseAuditMaxRounds(map[string]any{"audit_max_rounds": 3})
	if err != nil || value != 3 {
		t.Fatalf("ParseAuditMaxRounds() = %d, %v", value, err)
	}
	if _, err := ParseAuditMaxRounds(map[string]any{"audit_max_rounds": 0}); err == nil {
		t.Fatal("expected invalid zero rounds")
	}
	if _, err := ParseAuditMaxRounds(map[string]any{"audit_max_rounds": 99}); err == nil {
		t.Fatal("expected hard max rejection")
	}
	value, err = ParseAuditMaxRounds(nil)
	if err != nil || value != defaultAuditMaxRounds {
		t.Fatalf("default rounds = %d, %v", value, err)
	}
}

func TestAuditCallLimiterBoundsRounds(t *testing.T) {
	limiter := NewAuditCallLimiter(2)
	if _, _, err := limiter.TryConsume(); err != nil {
		t.Fatalf("first consume error: %v", err)
	}
	if _, _, err := limiter.TryConsume(); err != nil {
		t.Fatalf("second consume error: %v", err)
	}
	if _, _, err := limiter.TryConsume(); err == nil || !strings.Contains(err.Error(), "exceeded max rounds") {
		t.Fatalf("expected exceeded error, got %v", err)
	}
}

func TestLLMChapterAuditorAuditUsesContinuityAuditorWithoutTools(t *testing.T) {
	client := &auditFakeTextClient{responses: []provider.ModelResponse{{
		Content: `{"status":"warning","summary":"存在软漂移","issues":[{"type":"soft_drift","severity":"warning","message":"目标未推进","suggestion":"回到线索"}],"rewrite_hints":["收束到主线"]}`,
	}}}
	catalog := auditFakeCatalog{models: []domain.ModelConfig{{
		ID: "provider-1:auditor", ProviderID: "provider-1", Name: "auditor", Kind: domain.ModelKindText, Enabled: true, MaxOutputTokens: 800, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleContinuityAudit},
	}}}
	router := NewModelRouter(catalog, NewAgentRoleRegistry())
	auditor := NewLLMChapterAuditor(router, auditFakeClientFactory{client: client}, nil, NewRuleBasedContinuityAuditor(), nil)

	result, err := auditor.Audit(context.Background(), ChapterAuditRequest{
		ProjectID: "project-1",
		Title:     "第一章",
		Brief:     "推进主线",
		Draft:     "林烬在巷口坐下，只是看着远处灯火。",
	})
	if err != nil {
		t.Fatalf("Audit() error: %v", err)
	}
	if result.Status != "warning" || result.Summary == "" || len(client.requests) != 1 {
		t.Fatalf("unexpected result=%+v requests=%d", result, len(client.requests))
	}
	if len(client.requests[0].Tools) != 0 {
		t.Fatalf("audit generate must not attach tools: %+v", client.requests[0].Tools)
	}
	if !strings.Contains(client.requests[0].SystemPrompt, "Continuity Auditor") {
		t.Fatalf("system prompt missing continuity auditor contract: %q", client.requests[0].SystemPrompt)
	}
	if result.ModelResolution == nil || result.ModelResolution.ModelID == "" {
		t.Fatalf("expected model resolution, got %+v", result.ModelResolution)
	}
}

func TestLLMChapterAuditorLoadsDraftFromChapterID(t *testing.T) {
	client := &auditFakeTextClient{responses: []provider.ModelResponse{{
		Content: `{"status":"passed","summary":"无明显问题","issues":[]}`,
	}}}
	catalog := auditFakeCatalog{models: []domain.ModelConfig{{
		ID: "provider-1:auditor", ProviderID: "provider-1", Name: "auditor", Kind: domain.ModelKindText, Enabled: true, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleContinuityAudit},
	}}}
	store := auditFakeToolStore{versions: []domain.ChapterVersion{{
		ProjectID: "project-1", ChapterID: "chapter-1", Content: "林烬推开木门，继续追查失踪的罗盘。",
	}}}
	auditor := NewLLMChapterAuditor(NewModelRouter(catalog, NewAgentRoleRegistry()), auditFakeClientFactory{client: client}, nil, nil, store)
	result, err := auditor.Audit(context.Background(), ChapterAuditRequest{ProjectID: "project-1", ChapterID: "chapter-1"})
	if err != nil {
		t.Fatalf("Audit() error: %v", err)
	}
	if result.Status != "passed" {
		t.Fatalf("status = %q", result.Status)
	}
	if !strings.Contains(client.requests[0].UserPrompt, "罗盘") {
		t.Fatalf("user prompt missing loaded draft: %q", client.requests[0].UserPrompt)
	}
}

func TestToolExecutorChapterAuditRespectsLimiter(t *testing.T) {
	client := &auditFakeTextClient{responses: []provider.ModelResponse{
		{Content: `{"status":"passed","summary":"ok","issues":[]}`},
		{Content: `{"status":"passed","summary":"ok","issues":[]}`},
	}}
	catalog := auditFakeCatalog{models: []domain.ModelConfig{{
		ID: "provider-1:auditor", ProviderID: "provider-1", Name: "auditor", Kind: domain.ModelKindText, Enabled: true, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleContinuityAudit},
	}}}
	auditor := NewLLMChapterAuditor(NewModelRouter(catalog, NewAgentRoleRegistry()), auditFakeClientFactory{client: client}, nil, nil, nil)
	executor := NewToolExecutor(auditFakeToolStore{}, ToolExecutorOptions{
		ChapterAuditor: auditor,
		AuditLimiter:   NewAuditCallLimiter(1),
	})
	args, _ := json.Marshal(map[string]any{"project_id": "project-1", "draft": "正文"})
	if _, err := executor.Execute(context.Background(), provider.ToolCall{Name: ChapterAuditToolName, Arguments: args}); err != nil {
		t.Fatalf("first chapter.audit error: %v", err)
	}
	if _, err := executor.Execute(context.Background(), provider.ToolCall{Name: ChapterAuditToolName, Arguments: args}); err == nil || !strings.Contains(err.Error(), "exceeded max rounds") {
		t.Fatalf("expected limiter error, got %v", err)
	}
}

func TestNarrativeToolSpecsForWorkflowExcludesChapterAudit(t *testing.T) {
	for _, spec := range NarrativeToolSpecsForWorkflow() {
		if spec.Name == ChapterAuditToolName {
			t.Fatalf("workflow specs must exclude %s", ChapterAuditToolName)
		}
	}
	found := false
	for _, spec := range NarrativeToolSpecs() {
		if spec.Name == ChapterAuditToolName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("NarrativeToolSpecs must include %s", ChapterAuditToolName)
	}
}

func TestIsOptInBuiltinTool(t *testing.T) {
	if !IsOptInBuiltinTool(ChapterAuditToolName) {
		t.Fatal("chapter.audit should be opt-in")
	}
	if IsOptInBuiltinTool("chapter.list") {
		t.Fatal("chapter.list should not be opt-in")
	}
}
