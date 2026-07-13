package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"
)

const (
	// ChapterAuditToolName is the opt-in builtin used by writer/editor agents to request LLM reflection.
	ChapterAuditToolName = "chapter.audit"
	// BuiltinChapterAuditToolID is the stable catalog id for chapter.audit.
	BuiltinChapterAuditToolID = "builtin:chapter.audit"

	defaultAuditMaxRounds     = 2
	hardAuditMaxRounds        = 6
	defaultAuditMaxDraftRunes = 12000
	defaultAuditMaxTokens     = 1200
	auditRuntimeOptionKey     = "audit_max_rounds"
)

// ChapterAuditRequest is the structured input for one audit/reflect invocation.
type ChapterAuditRequest struct {
	ProjectID       string
	ChapterID       string
	Draft           string
	Title           string
	Brief           string
	ChapterIdea     string
	Focus           []string
	MaxOutputTokens int
}

// ChapterAuditIssue is one finding returned by the continuity-auditor model.
type ChapterAuditIssue struct {
	Type         string                         `json:"type"`
	Severity     string                         `json:"severity"`
	Message      string                         `json:"message"`
	DraftExcerpt string                         `json:"draft_excerpt,omitempty"`
	Suggestion   string                         `json:"suggestion,omitempty"`
	Evidence     []domain.ContinuityEvidenceRef `json:"evidence,omitempty"`
}

// ChapterAuditResult is the stable tool result shape returned to the outer agent.
type ChapterAuditResult struct {
	Status          string                 `json:"status"`
	Summary         string                 `json:"summary"`
	Issues          []ChapterAuditIssue    `json:"issues"`
	RewriteHints    []string               `json:"rewrite_hints,omitempty"`
	RulesAudit      *domain.ContinuityAudit `json:"rules_audit,omitempty"`
	ModelResolution *domain.ModelResolution `json:"model_resolution,omitempty"`
	Metadata        map[string]string      `json:"metadata,omitempty"`
}

// ChapterAuditor performs LLM-backed draft reflection for chapter.audit.
type ChapterAuditor interface {
	Audit(ctx context.Context, req ChapterAuditRequest) (ChapterAuditResult, error)
}

// LLMChapterAuditor routes one tool-free Generate call through the continuity-auditor role.
type LLMChapterAuditor struct {
	router   *ModelRouter
	clients  TextClientFactory
	builder  *ContextPackBuilder
	rules    ContinuityAuditor
	store    ToolStore
}

func NewLLMChapterAuditor(router *ModelRouter, clients TextClientFactory, builder *ContextPackBuilder, rules ContinuityAuditor, store ToolStore) *LLMChapterAuditor {
	return &LLMChapterAuditor{router: router, clients: clients, builder: builder, rules: rules, store: store}
}

func (a *LLMChapterAuditor) Audit(ctx context.Context, req ChapterAuditRequest) (ChapterAuditResult, error) {
	if a == nil || a.router == nil || a.clients == nil {
		return ChapterAuditResult{}, fmt.Errorf("chapter auditor is not configured")
	}
	select {
	case <-ctx.Done():
		return ChapterAuditResult{}, ctx.Err()
	default:
	}
	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		return ChapterAuditResult{}, fmt.Errorf("chapter.audit project_id must not be empty")
	}
	draft, truncated, err := a.resolveDraft(req)
	if err != nil {
		return ChapterAuditResult{}, err
	}
	if strings.TrimSpace(draft) == "" {
		return ChapterAuditResult{}, fmt.Errorf("chapter.audit requires draft or a chapter_id with latest content")
	}

	var pack domain.ContextPack
	if a.builder != nil {
		pack, err = a.builder.Build(projectID, strings.TrimSpace(req.ChapterID), domain.AgentRoleContinuityAudit, req.Title+" "+req.Brief, 3000)
		if err != nil {
			return ChapterAuditResult{}, fmt.Errorf("build chapter.audit context pack: %w", err)
		}
	}

	var rulesAudit *domain.ContinuityAudit
	if a.rules != nil {
		audit, auditErr := a.rules.Audit(ContinuityAuditInput{
			Title:       req.Title,
			Brief:       req.Brief,
			ChapterIdea: req.ChapterIdea,
			Draft:       draft,
			ContextPack: pack,
		})
		if auditErr != nil {
			return ChapterAuditResult{}, fmt.Errorf("rules audit for chapter.audit: %w", auditErr)
		}
		rulesAudit = &audit
	}

	selection, err := a.router.SelectTextModel(domain.AgentRoleContinuityAudit)
	if err != nil {
		return ChapterAuditResult{}, fmt.Errorf("route continuity-auditor model for chapter.audit: %w", err)
	}
	client, err := a.clients.NewTextClient(selection.Provider)
	if err != nil {
		return ChapterAuditResult{}, fmt.Errorf("create continuity-auditor client: %w", err)
	}
	resolution := buildModelResolution(selection)
	maxTokens := firstPositive(req.MaxOutputTokens, selection.Model.MaxOutputTokens, defaultAuditMaxTokens)
	userPrompt, err := chapterAuditUserPrompt(req, draft, pack, rulesAudit)
	if err != nil {
		return ChapterAuditResult{}, err
	}
	resp, err := client.Generate(ctx, provider.TextRequest{
		Model:           selection.Model.Name,
		SystemPrompt:    chapterAuditSystemPrompt(),
		UserPrompt:      userPrompt,
		MaxOutputTokens: maxTokens,
		Temperature:     0.2,
		// Intentionally no Tools: nested tool loops would recurse into chapter.audit.
	})
	if err != nil {
		return ChapterAuditResult{}, fmt.Errorf("continuity-auditor generate: %w", err)
	}
	result, err := parseChapterAuditResponse(resp.Content)
	if err != nil {
		return ChapterAuditResult{}, err
	}
	result.RulesAudit = rulesAudit
	result.ModelResolution = &resolution
	if result.Metadata == nil {
		result.Metadata = map[string]string{}
	}
	result.Metadata["draft_truncated"] = fmt.Sprintf("%t", truncated)
	result.Metadata["draft_runes"] = fmt.Sprintf("%d", utf8.RuneCountInString(draft))
	if rulesAudit != nil {
		result.Status = mergeAuditStatus(result.Status, rulesAudit.Status)
	}
	return result, nil
}

func (a *LLMChapterAuditor) resolveDraft(req ChapterAuditRequest) (string, bool, error) {
	draft := strings.TrimSpace(req.Draft)
	if draft == "" && strings.TrimSpace(req.ChapterID) != "" {
		if a.store == nil {
			return "", false, fmt.Errorf("chapter.audit chapter_id requires tool store to load content")
		}
		versions, err := a.store.ListChapterVersions(req.ProjectID, strings.TrimSpace(req.ChapterID))
		if err != nil {
			return "", false, err
		}
		if len(versions) == 0 {
			return "", false, fmt.Errorf("chapter.audit chapter_id %q has no versions", strings.TrimSpace(req.ChapterID))
		}
		draft = strings.TrimSpace(versions[0].Content)
	}
	return truncateRunes(draft, defaultAuditMaxDraftRunes)
}

// AuditCallLimiter bounds chapter.audit invocations inside one agent run.
type AuditCallLimiter struct {
	mu      sync.Mutex
	max     int
	used    int
}

func NewAuditCallLimiter(maxRounds int) *AuditCallLimiter {
	if maxRounds <= 0 {
		maxRounds = defaultAuditMaxRounds
	}
	if maxRounds > hardAuditMaxRounds {
		maxRounds = hardAuditMaxRounds
	}
	return &AuditCallLimiter{max: maxRounds}
}

func (l *AuditCallLimiter) TryConsume() (used, max int, err error) {
	if l == nil {
		return 0, 0, fmt.Errorf("chapter.audit limiter is not configured")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.used >= l.max {
		return l.used, l.max, fmt.Errorf("chapter.audit exceeded max rounds %d for this agent run", l.max)
	}
	l.used++
	return l.used, l.max, nil
}

func ParseAuditMaxRounds(runtimeOptions map[string]any) (int, error) {
	if len(runtimeOptions) == 0 {
		return defaultAuditMaxRounds, nil
	}
	raw, ok := runtimeOptions[auditRuntimeOptionKey]
	if !ok || raw == nil {
		return defaultAuditMaxRounds, nil
	}
	value, err := coercePositiveInt(raw)
	if err != nil {
		return 0, fmt.Errorf("runtime_options.%s must be an integer between 1 and %d: %w", auditRuntimeOptionKey, hardAuditMaxRounds, err)
	}
	if value < 1 || value > hardAuditMaxRounds {
		return 0, fmt.Errorf("runtime_options.%s must be between 1 and %d, got %d", auditRuntimeOptionKey, hardAuditMaxRounds, value)
	}
	return value, nil
}

// IsOptInBuiltinTool reports tools that must not appear unless explicitly listed in tool_ids.
func IsOptInBuiltinTool(name string) bool {
	return strings.TrimSpace(name) == ChapterAuditToolName
}

// NarrativeToolSpecsForWorkflow returns tools safe for writing workflows (no nested LLM audit).
func NarrativeToolSpecsForWorkflow() []provider.ToolSpec {
	specs := NarrativeToolSpecs()
	filtered := make([]provider.ToolSpec, 0, len(specs))
	for _, spec := range specs {
		if IsOptInBuiltinTool(spec.Name) {
			continue
		}
		filtered = append(filtered, spec)
	}
	return filtered
}

func chapterAuditSystemPrompt() string {
	return strings.TrimSpace(`你是 AI 小说创作平台中的 Continuity Auditor Agent。
你的职责是审阅给定章节草稿，对照上下文包与可选规则审计结果，找出连续性、事实、角色、情节推进与叙事问题，并给出可执行的修改建议。
硬性约束：
1）只输出严格 JSON 对象，不要 Markdown、解释性前后缀或代码围栏；
2）不要重写整章正文，只给建议；
3）不要调用工具；
4）severity 只能是 info / warning / error；
5）status 只能是 passed / warning / failed；
6）若规则审计已有 error，status 不得为 passed。
JSON 形状：
{"status":"passed|warning|failed","summary":"...","issues":[{"type":"hard_conflict|soft_drift|style|pacing|character|plot|other","severity":"info|warning|error","message":"...","draft_excerpt":"...","suggestion":"...","evidence":[{"source_type":"...","source_id":"...","label":"...","excerpt":"..."}]}],"rewrite_hints":["..."]}`)
}

func chapterAuditUserPrompt(req ChapterAuditRequest, draft string, pack domain.ContextPack, rules *domain.ContinuityAudit) (string, error) {
	payload := map[string]any{
		"project_id":   strings.TrimSpace(req.ProjectID),
		"chapter_id":   strings.TrimSpace(req.ChapterID),
		"title":        strings.TrimSpace(req.Title),
		"brief":        strings.TrimSpace(req.Brief),
		"chapter_idea": strings.TrimSpace(req.ChapterIdea),
		"focus":        req.Focus,
		"draft":        draft,
		"context_pack": pack,
	}
	if rules != nil {
		payload["rules_audit"] = rules
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal chapter.audit prompt payload: %w", err)
	}
	return "请审阅以下草稿并返回严格 JSON：\n" + string(bytes), nil
}

func parseChapterAuditResponse(content string) (ChapterAuditResult, error) {
	raw := strings.TrimSpace(content)
	if raw == "" {
		return ChapterAuditResult{}, fmt.Errorf("chapter.audit model returned empty content")
	}
	raw = stripJSONFence(raw)
	var result ChapterAuditResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return ChapterAuditResult{}, fmt.Errorf("decode chapter.audit JSON: %w", err)
	}
	result.Status = strings.TrimSpace(strings.ToLower(result.Status))
	switch result.Status {
	case "passed", "warning", "failed":
	default:
		return ChapterAuditResult{}, fmt.Errorf("chapter.audit status must be passed|warning|failed, got %q", result.Status)
	}
	result.Summary = strings.TrimSpace(result.Summary)
	if result.Summary == "" {
		return ChapterAuditResult{}, fmt.Errorf("chapter.audit summary must not be empty")
	}
	if result.Issues == nil {
		result.Issues = []ChapterAuditIssue{}
	}
	for i := range result.Issues {
		issue := &result.Issues[i]
		issue.Type = strings.TrimSpace(issue.Type)
		issue.Severity = strings.TrimSpace(strings.ToLower(issue.Severity))
		issue.Message = strings.TrimSpace(issue.Message)
		if issue.Type == "" {
			return ChapterAuditResult{}, fmt.Errorf("chapter.audit issues[%d].type must not be empty", i)
		}
		switch issue.Severity {
		case "info", "warning", "error":
		default:
			return ChapterAuditResult{}, fmt.Errorf("chapter.audit issues[%d].severity must be info|warning|error", i)
		}
		if issue.Message == "" {
			return ChapterAuditResult{}, fmt.Errorf("chapter.audit issues[%d].message must not be empty", i)
		}
	}
	hints := make([]string, 0, len(result.RewriteHints))
	for _, hint := range result.RewriteHints {
		if trimmed := strings.TrimSpace(hint); trimmed != "" {
			hints = append(hints, trimmed)
		}
	}
	result.RewriteHints = hints
	return result, nil
}

func mergeAuditStatus(llmStatus, rulesStatus string) string {
	rank := func(status string) int {
		switch strings.TrimSpace(strings.ToLower(status)) {
		case "failed":
			return 3
		case "warning":
			return 2
		case "passed":
			return 1
		default:
			return 0
		}
	}
	if rank(rulesStatus) > rank(llmStatus) {
		return strings.TrimSpace(strings.ToLower(rulesStatus))
	}
	return strings.TrimSpace(strings.ToLower(llmStatus))
}

func stripJSONFence(content string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)
	if strings.HasPrefix(strings.ToLower(trimmed), "json") {
		trimmed = strings.TrimSpace(trimmed[4:])
	}
	if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return strings.TrimSpace(trimmed)
}

func truncateRunes(text string, maxRunes int) (string, bool, error) {
	if maxRunes <= 0 {
		return "", false, fmt.Errorf("max draft runes must be positive")
	}
	if utf8.RuneCountInString(text) <= maxRunes {
		return text, false, nil
	}
	runes := []rune(text)
	return string(runes[:maxRunes]), true, nil
}

func coercePositiveInt(raw any) (int, error) {
	switch value := raw.(type) {
	case int:
		return value, nil
	case int32:
		return int(value), nil
	case int64:
		return int(value), nil
	case float64:
		if value != float64(int(value)) {
			return 0, fmt.Errorf("got non-integer number %v", value)
		}
		return int(value), nil
	case float32:
		if value != float32(int(value)) {
			return 0, fmt.Errorf("got non-integer number %v", value)
		}
		return int(value), nil
	case json.Number:
		n, err := value.Int64()
		if err != nil {
			return 0, err
		}
		return int(n), nil
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return 0, fmt.Errorf("empty string")
		}
		var n int
		if _, err := fmt.Sscanf(trimmed, "%d", &n); err != nil {
			return 0, err
		}
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", raw)
	}
}
