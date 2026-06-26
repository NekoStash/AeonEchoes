package agent

import (
	"testing"

	"aeonechoes/server/internal/domain"
)

func TestRuleBasedContinuityAuditorWorldRuleConflict(t *testing.T) {
	auditor := NewRuleBasedContinuityAuditor()
	audit, err := auditor.Audit(ContinuityAuditInput{
		Draft:       "夜幕里，法师直接施展飞行术越过城墙。",
		ContextPack: domain.ContextPack{WorldRules: map[string]string{"flight": "凡人不得施展飞行术"}},
	})
	if err != nil {
		t.Fatalf("Audit() error: %v", err)
	}
	issue := requireSingleIssue(t, audit)
	if audit.Status != "failed" {
		t.Fatalf("status = %q, want failed", audit.Status)
	}
	if issue.Type != "hard_conflict" || issue.Severity != "error" {
		t.Fatalf("unexpected issue: %+v", issue)
	}
	if issue.Evidence[0].SourceType != "world_rule" {
		t.Fatalf("unexpected evidence: %+v", issue.Evidence)
	}
}

func TestRuleBasedContinuityAuditorFactConflict(t *testing.T) {
	auditor := NewRuleBasedContinuityAuditor()
	audit, err := auditor.Audit(ContinuityAuditInput{
		Draft: "林烬明明不知道密道入口，却还是站在机关前犹豫。",
		ContextPack: domain.ContextPack{
			Entities: []domain.Entity{{ID: "char-lin", Name: "林烬", Type: "character"}},
			Facts:    []domain.Fact{{ID: "fact-1", EntityID: "char-lin", Claim: "林烬知道密道入口", Confidence: 1}},
		},
	})
	if err != nil {
		t.Fatalf("Audit() error: %v", err)
	}
	issue := requireSingleIssue(t, audit)
	if issue.Type != "hard_conflict" || issue.Severity != "error" {
		t.Fatalf("unexpected issue: %+v", issue)
	}
}

func TestRuleBasedContinuityAuditorSoftDrift(t *testing.T) {
	auditor := NewRuleBasedContinuityAuditor()
	audit, err := auditor.Audit(ContinuityAuditInput{
		ChapterIdea: "本章目标：林烬潜入灰塔，调查钟声异常，并找到灰烬钥匙。",
		Draft:       "雨夜里，集市商贩反复争吵价钱，旅人们讨论下一场节庆，谁也没有提起塔楼。",
	})
	if err != nil {
		t.Fatalf("Audit() error: %v", err)
	}
	issue := findIssueByType(t, audit, "soft_drift")
	if audit.Status != "warning" {
		t.Fatalf("status = %q, want warning", audit.Status)
	}
	if issue.Severity != "warning" {
		t.Fatalf("unexpected issue severity: %+v", issue)
	}
}

func TestRuleBasedContinuityAuditorMissingFollowup(t *testing.T) {
	auditor := NewRuleBasedContinuityAuditor()
	audit, err := auditor.Audit(ContinuityAuditInput{
		Draft: "林烬在码头与陌生人擦肩而过，随后独自回到客栈睡下。",
		ContextPack: domain.ContextPack{
			Entities:    []domain.Entity{{ID: "char-lin", Name: "林烬", Type: "character"}, {ID: "item-key", Name: "灰烬钥匙", Type: "item"}},
			PlotThreads: []domain.PlotThread{{ID: "thread-1", Title: "钥匙伏笔", Summary: "灰烬钥匙的真正用途", RelatedEntityIDs: []string{"item-key"}, Status: "open"}},
		},
	})
	if err != nil {
		t.Fatalf("Audit() error: %v", err)
	}
	issue := findIssueByType(t, audit, "missing_followup")
	if issue.Severity != "warning" {
		t.Fatalf("unexpected issue: %+v", issue)
	}
}

func TestRuleBasedContinuityAuditorUnsupportedNewClaim(t *testing.T) {
	auditor := NewRuleBasedContinuityAuditor()
	audit, err := auditor.Audit(ContinuityAuditInput{
		Draft: "林烬打开星门密钥，古城祭坛随之亮起。",
		ContextPack: domain.ContextPack{
			Entities: []domain.Entity{{ID: "char-lin", Name: "林烬", Type: "character"}},
		},
	})
	if err != nil {
		t.Fatalf("Audit() error: %v", err)
	}
	issue := requireSingleIssue(t, audit)
	if audit.Status != "warning" {
		t.Fatalf("status = %q, want warning", audit.Status)
	}
	if issue.Type != "unsupported_new_claim" || issue.Severity != "warning" {
		t.Fatalf("unexpected issue: %+v", issue)
	}
	if issue.Evidence[0].SourceType != "draft_observation" || issue.Evidence[0].Label != "draft-only candidate" {
		t.Fatalf("unexpected evidence: %+v", issue.Evidence)
	}
}

func TestRuleBasedContinuityAuditorUnsupportedNewClaimSkipsKnownEntity(t *testing.T) {
	auditor := NewRuleBasedContinuityAuditor()
	audit, err := auditor.Audit(ContinuityAuditInput{
		Draft: "林烬打开星门密钥，古城祭坛随之亮起。",
		ContextPack: domain.ContextPack{
			Entities: []domain.Entity{{ID: "char-lin", Name: "林烬", Type: "character"}, {ID: "item-gate", Name: "星门密钥", Type: "item"}},
		},
	})
	if err != nil {
		t.Fatalf("Audit() error: %v", err)
	}
	if hasIssueType(audit, "unsupported_new_claim") {
		t.Fatalf("unexpected unsupported_new_claim: %+v", audit.Issues)
	}
}

func TestRuleBasedContinuityAuditorUnsupportedNewClaimSkipsKnownFactOrChapterSummary(t *testing.T) {
	auditor := NewRuleBasedContinuityAuditor()
	t.Run("known fact", func(t *testing.T) {
		audit, err := auditor.Audit(ContinuityAuditInput{
			Draft: "林烬发现古城祭坛后停下脚步。",
			ContextPack: domain.ContextPack{
				Entities: []domain.Entity{{ID: "char-lin", Name: "林烬", Type: "character"}},
				Facts:    []domain.Fact{{ID: "fact-1", Claim: "古城祭坛埋在灰塔地下", Confidence: 1}},
			},
		})
		if err != nil {
			t.Fatalf("Audit() error: %v", err)
		}
		if hasIssueType(audit, "unsupported_new_claim") {
			t.Fatalf("unexpected unsupported_new_claim from fact-backed draft: %+v", audit.Issues)
		}
	})

	t.Run("known chapter summary", func(t *testing.T) {
		audit, err := auditor.Audit(ContinuityAuditInput{
			Draft: "林烬进入回廊引擎前先熄了灯。",
			ContextPack: domain.ContextPack{
				Entities:         []domain.Entity{{ID: "char-lin", Name: "林烬", Type: "character"}},
				ChapterSummaries: []domain.ChapterSummary{{ChapterID: "ch-1", ChapterVersionID: "cv-1", Title: "前章", Summary: "林烬已经确认回廊引擎的位置"}},
			},
		})
		if err != nil {
			t.Fatalf("Audit() error: %v", err)
		}
		if hasIssueType(audit, "unsupported_new_claim") {
			t.Fatalf("unexpected unsupported_new_claim from summary-backed draft: %+v", audit.Issues)
		}
	})
}

func TestRuleBasedContinuityAuditorUnsupportedNewClaimSkipsOrdinaryNarration(t *testing.T) {
	auditor := NewRuleBasedContinuityAuditor()
	audit, err := auditor.Audit(ContinuityAuditInput{
		Draft: "林烬走进房间，听见风吹过窗缝。",
		ContextPack: domain.ContextPack{
			Entities: []domain.Entity{{ID: "char-lin", Name: "林烬", Type: "character"}},
		},
	})
	if err != nil {
		t.Fatalf("Audit() error: %v", err)
	}
	if hasIssueType(audit, "unsupported_new_claim") {
		t.Fatalf("unexpected unsupported_new_claim: %+v", audit.Issues)
	}
}

func TestRuleBasedContinuityAuditorPassed(t *testing.T) {
	auditor := NewRuleBasedContinuityAuditor()
	audit, err := auditor.Audit(ContinuityAuditInput{
		Title: "第八章",
		Brief: "围绕林烬和灰烬钥匙潜入灰塔。",
		Draft: "林烬把灰烬钥匙藏进袖口，悄悄潜入灰塔，沿着回廊调查钟声的来源。",
		ContextPack: domain.ContextPack{
			Entities:    []domain.Entity{{ID: "char-lin", Name: "林烬", Type: "character"}, {ID: "item-key", Name: "灰烬钥匙", Type: "item"}},
			Facts:       []domain.Fact{{ID: "fact-1", EntityID: "char-lin", Claim: "林烬持有灰烬钥匙", Confidence: 1}},
			PlotThreads: []domain.PlotThread{{ID: "thread-1", Title: "钟声异常", RelatedEntityIDs: []string{"char-lin"}, Status: "open"}},
		},
	})
	if err != nil {
		t.Fatalf("Audit() error: %v", err)
	}
	if audit.Status != "passed" {
		t.Fatalf("status = %q, want passed; issues=%+v", audit.Status, audit.Issues)
	}
	if len(audit.Issues) != 0 {
		t.Fatalf("issues len = %d, want 0", len(audit.Issues))
	}
}

func TestRuleBasedContinuityAuditorFailedWhenErrorAndWarningExist(t *testing.T) {
	auditor := NewRuleBasedContinuityAuditor()
	audit, err := auditor.Audit(ContinuityAuditInput{
		ChapterIdea: "本章目标：林烬进入灰塔，调查钟声异常。",
		Draft:       "苏九拿着灰烬钥匙离开集市，完全不提灰塔。",
		ContextPack: domain.ContextPack{
			Entities:    []domain.Entity{{ID: "char-lin", Name: "林烬", Type: "character"}, {ID: "char-su", Name: "苏九", Type: "character"}, {ID: "item-key", Name: "灰烬钥匙", Type: "item"}},
			Facts:       []domain.Fact{{ID: "fact-1", EntityID: "char-lin", Claim: "林烬持有灰烬钥匙", Confidence: 1}},
			PlotThreads: []domain.PlotThread{{ID: "thread-1", Title: "塔楼异动", RelatedEntityIDs: []string{"char-lin"}, Status: "open"}},
		},
	})
	if err != nil {
		t.Fatalf("Audit() error: %v", err)
	}
	if audit.Status != "failed" {
		t.Fatalf("status = %q, want failed; issues=%+v", audit.Status, audit.Issues)
	}
	if findIssueByType(t, audit, "hard_conflict").Severity != "error" {
		t.Fatalf("expected hard_conflict error: %+v", audit.Issues)
	}
	if findIssueByType(t, audit, "missing_followup").Severity != "warning" {
		t.Fatalf("expected missing_followup warning: %+v", audit.Issues)
	}
}

func requireSingleIssue(t *testing.T, audit domain.ContinuityAudit) domain.ContinuityIssue {
	t.Helper()
	if len(audit.Issues) != 1 {
		t.Fatalf("issues len = %d, want 1; issues=%+v", len(audit.Issues), audit.Issues)
	}
	return audit.Issues[0]
}

func findIssueByType(t *testing.T, audit domain.ContinuityAudit, issueType string) domain.ContinuityIssue {
	t.Helper()
	for _, issue := range audit.Issues {
		if issue.Type == issueType {
			return issue
		}
	}
	t.Fatalf("issue type %q not found in %+v", issueType, audit.Issues)
	return domain.ContinuityIssue{}
}

func hasIssueType(audit domain.ContinuityAudit, issueType string) bool {
	for _, issue := range audit.Issues {
		if issue.Type == issueType {
			return true
		}
	}
	return false
}
