package agent

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"aeonechoes/server/internal/domain"
)

var continuityNegativeMarkers = []string{"不能", "不可", "不得", "禁止", "无法", "不会"}
var continuityNegativeCues = []string{"不能", "不可", "不得", "禁止", "无法", "不会", "没", "没有", "未", "不再", "尚未"}
var continuityDeadStateMarkers = []string{"已死", "已经死了", "死了", "身亡", "死亡", "不在世", "不存在", "消失了", "已经消失", "早已死去"}
var continuityAliveStateMarkers = []string{"还活着", "仍然活着", "活着", "健在", "没有死", "并没有死"}
var continuityOwnershipMarkers = []string{"持有", "拥有", "拿着", "带着", "握着", "佩戴"}
var continuityKnowledgeNegativeMarkers = []string{"并不知道", "并不知", "尚不知道", "不知道", "未曾得知", "不知"}
var continuityKnowledgePositiveMarkers = []string{"已经得知", "知道", "知晓", "得知", "明白", "清楚"}
var continuityUnsupportedClaimMarkers = []string{"获得", "发现", "进入", "来自", "属于", "揭示", "取出", "打开", "启动", "开启", "解锁", "找到"}
var continuityUnsupportedClaimHeadKeywords = []string{
	"议会", "王庭", "神殿", "圣堂", "学宫", "回廊", "引擎", "装置", "机关", "计划", "协议", "密钥", "钥匙", "罗盘", "卷轴", "纹章", "印记", "核心", "碎片", "残页", "舰队", "军团", "祭坛", "古城", "哨站", "港口", "王城", "塔楼", "星门", "秘匣", "秘库", "秘卷",
}
var continuityUnsupportedClaimGenericCandidates = map[string]struct{}{
	"房间": {}, "院子": {}, "巷子": {}, "客栈": {}, "街道": {}, "走廊": {}, "门口": {}, "前厅": {}, "后厅": {}, "灯火": {}, "钟声": {}, "消息": {}, "真相": {}, "秘密": {}, "希望": {}, "机会": {},
}
var continuityUnsupportedClaimLeadPattern = regexp.MustCompile(`^[“”"'‘’「」『』\s]*(?:了|又|还|便|就|再|仍|尚|已|正|将)*(?:一位|一名|一座|一间|一扇|一台|一枚|一把|一柄|一卷|一页|一块|一份|一个|一条|一道|一封|一颗|一片|一处|这座|那座|这间|那间|这扇|那扇|这台|那台|这位|那位|某个|某座|某间|某位|数座|数台)?`)

var continuityKeywordStopwords = map[string]struct{}{
	"本章": {}, "目标": {}, "推进": {}, "围绕": {}, "处理": {}, "需要": {}, "继续": {}, "展开": {}, "场景": {}, "冲突": {},
	"转折": {}, "角色": {}, "状态": {}, "变化": {}, "写作": {}, "注意": {}, "章节": {}, "方案": {}, "简报": {}, "以及": {},
	"一个": {}, "这个": {}, "那个": {}, "他们": {}, "我们": {}, "自己": {}, "时候": {}, "开始": {}, "结束": {}, "必须": {},
	"主要": {}, "依据": {}, "相关": {}, "内容": {}, "剧情": {}, "故事": {}, "安排": {}, "发展": {},
}

// ContinuityAuditor checks a draft against deterministic continuity rules.
type ContinuityAuditor interface {
	Audit(input ContinuityAuditInput) (domain.ContinuityAudit, error)
}

// ContinuityAuditInput is the deterministic continuity audit input assembled by workflow orchestration.
type ContinuityAuditInput struct {
	Title       string
	Brief       string
	ChapterIdea string
	Draft       string
	ContextPack domain.ContextPack
}

// RuleBasedContinuityAuditor performs deterministic high-confidence continuity checks.
type RuleBasedContinuityAuditor struct{}

func NewRuleBasedContinuityAuditor() *RuleBasedContinuityAuditor {
	return &RuleBasedContinuityAuditor{}
}

func (a *RuleBasedContinuityAuditor) Audit(input ContinuityAuditInput) (domain.ContinuityAudit, error) {
	if a == nil {
		return domain.ContinuityAudit{}, fmt.Errorf("continuity auditor is not configured")
	}
	draft := strings.TrimSpace(input.Draft)
	if draft == "" {
		return domain.ContinuityAudit{}, fmt.Errorf("continuity audit draft must not be empty")
	}
	entityIndex := newContinuityEntityIndex(input.ContextPack.Entities)
	issues := make([]domain.ContinuityIssue, 0)
	issues = append(issues, a.auditWorldRuleConflicts(draft, input.ContextPack.WorldRules)...)
	issues = append(issues, a.auditFactConflicts(draft, input.ContextPack.Facts, entityIndex)...)
	issues = append(issues, a.auditUnsupportedNewClaims(draft, input.ContextPack)...)
	issues = append(issues, a.auditSoftDrift(input, draft, entityIndex)...)
	issues = append(issues, a.auditMissingFollowups(draft, input.ContextPack.PlotThreads, entityIndex)...)
	return domain.ContinuityAudit{Status: continuityAuditStatus(issues), Issues: issues}, nil
}

func (a *RuleBasedContinuityAuditor) auditWorldRuleConflicts(draft string, rules map[string]string) []domain.ContinuityIssue {
	if len(rules) == 0 {
		return nil
	}
	sentences := splitContinuitySentences(draft)
	keys := make([]string, 0, len(rules))
	for key := range rules {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	issues := make([]domain.ContinuityIssue, 0)
	for _, key := range keys {
		rule := strings.TrimSpace(rules[key])
		action := extractForbiddenAction(rule)
		if action == "" {
			continue
		}
		for _, sentence := range sentences {
			if !containsCompactText(sentence, action) || containsNegativeCue(sentence) {
				continue
			}
			label := strings.TrimSpace(key)
			if label == "" {
				label = trimToRunes(rule, 24)
			}
			issues = append(issues, domain.ContinuityIssue{
				Type:         "hard_conflict",
				Severity:     "error",
				Message:      fmt.Sprintf("草稿触发了世界规则禁止的动作：%s", action),
				DraftExcerpt: trimToRunes(strings.TrimSpace(sentence), 120),
				Suggestion:   fmt.Sprintf("改写该段，避免出现“%s”，或先修正上游世界规则。", action),
				Evidence: []domain.ContinuityEvidenceRef{{
					SourceType: "world_rule",
					SourceID:   key,
					Label:      label,
					Excerpt:    trimToRunes(rule, 120),
				}},
			})
			break
		}
	}
	return issues
}

func (a *RuleBasedContinuityAuditor) auditFactConflicts(draft string, facts []domain.Fact, entities continuityEntityIndex) []domain.ContinuityIssue {
	if len(facts) == 0 {
		return nil
	}
	sentences := splitContinuitySentences(draft)
	issues := make([]domain.ContinuityIssue, 0)
	for _, fact := range facts {
		if issue, ok := detectLifeStateConflict(fact, sentences, entities); ok {
			issues = append(issues, issue)
			continue
		}
		if issue, ok := detectOwnershipConflict(fact, sentences, entities); ok {
			issues = append(issues, issue)
			continue
		}
		if issue, ok := detectKnowledgeConflict(fact, sentences, entities); ok {
			issues = append(issues, issue)
			continue
		}
	}
	return issues
}

func (a *RuleBasedContinuityAuditor) auditUnsupportedNewClaims(draft string, pack domain.ContextPack) []domain.ContinuityIssue {
	sentences := splitContinuitySentences(draft)
	if len(sentences) == 0 {
		return nil
	}
	canon := newContinuityCanonMatcher(pack)
	issues := make([]domain.ContinuityIssue, 0)
	seenCandidates := map[string]struct{}{}
	for _, sentence := range sentences {
		marker, rest, ok := unsupportedClaimMarkerWindow(sentence)
		if !ok {
			continue
		}
		candidate, ok := extractUnsupportedClaimCandidate(rest)
		if !ok {
			continue
		}
		compactCandidate := compactContinuityText(candidate)
		if _, seen := seenCandidates[compactCandidate]; seen {
			continue
		}
		if canon.contains(candidate) {
			continue
		}
		seenCandidates[compactCandidate] = struct{}{}
		issues = append(issues, domain.ContinuityIssue{
			Type:         "unsupported_new_claim",
			Severity:     "warning",
			Message:      fmt.Sprintf("草稿引入了缺少上下文依据的新设定候选：%s", candidate),
			DraftExcerpt: trimToRunes(strings.TrimSpace(sentence), 120),
			Suggestion:   fmt.Sprintf("确认“%s”是否已在上下文中出现；若没有，改写为既有设定，或先在前文补充明确铺垫。", candidate),
			Evidence: []domain.ContinuityEvidenceRef{{
				SourceType: "draft_observation",
				Label:      "draft-only candidate",
				Excerpt:    trimToRunes(fmt.Sprintf("marker=%s candidate=%s", marker, candidate), 120),
			}},
		})
	}
	return issues
}

func (a *RuleBasedContinuityAuditor) auditSoftDrift(input ContinuityAuditInput, draft string, entities continuityEntityIndex) []domain.ContinuityIssue {
	sourceType, sourceID, sourceText := continuityGoalSource(input)
	if strings.TrimSpace(sourceText) == "" {
		return nil
	}
	goalText := joinNonEmpty([]string{input.Title, sourceText})
	keywords := extractContinuityKeywords(goalText, entities)
	if len(keywords) == 0 {
		return nil
	}
	hits, missing := continuityKeywordCoverage(draft, keywords)
	coverage := float64(len(hits)) / float64(len(keywords))
	if len(hits) >= 2 || coverage >= 0.35 || (len(keywords) <= 2 && len(hits) == 1) {
		return nil
	}
	issue := domain.ContinuityIssue{
		Type:         "soft_drift",
		Severity:     "warning",
		Message:      fmt.Sprintf("草稿与章节目标重合度过低，命中关键词 %d/%d", len(hits), len(keywords)),
		DraftExcerpt: trimToRunes(firstText(firstContinuitySentence(draft), draft), 120),
		Suggestion:   fmt.Sprintf("补强章节目标相关内容，优先覆盖这些关键词：%s", strings.Join(firstNStrings(missing, 3), "、")),
		Evidence: []domain.ContinuityEvidenceRef{{
			SourceType: sourceType,
			SourceID:   sourceID,
			Label:      sourceType,
			Excerpt:    trimToRunes(sourceText, 120),
		}},
	}
	return []domain.ContinuityIssue{issue}
}

func (a *RuleBasedContinuityAuditor) auditMissingFollowups(draft string, threads []domain.PlotThread, entities continuityEntityIndex) []domain.ContinuityIssue {
	if len(threads) == 0 {
		return nil
	}
	issues := make([]domain.ContinuityIssue, 0)
	compactDraft := compactContinuityText(draft)
	for _, thread := range threads {
		if isClosedThread(thread.Status) {
			continue
		}
		threadKeywords := extractContinuityKeywords(thread.Title, entities)
		relatedNames := entities.relatedEntityNames(thread.RelatedEntityIDs)
		if len(threadKeywords) == 0 && len(relatedNames) == 0 {
			continue
		}
		touched := false
		for _, keyword := range threadKeywords {
			if strings.Contains(compactDraft, compactContinuityText(keyword)) {
				touched = true
				break
			}
		}
		if !touched {
			for _, name := range relatedNames {
				if strings.Contains(compactDraft, compactContinuityText(name)) {
					touched = true
					break
				}
			}
		}
		if touched {
			continue
		}
		issues = append(issues, domain.ContinuityIssue{
			Type:         "missing_followup",
			Severity:     "warning",
			Message:      fmt.Sprintf("草稿未推进情节线：%s", firstText(thread.Title, thread.ID)),
			DraftExcerpt: trimToRunes(firstText(firstContinuitySentence(draft), draft), 120),
			Suggestion:   fmt.Sprintf("补一小段推进“%s”，或明确说明该线索为何暂缓。", firstText(thread.Title, thread.ID)),
			Evidence: []domain.ContinuityEvidenceRef{{
				SourceType: "plot_thread",
				SourceID:   thread.ID,
				Label:      firstText(thread.Title, "plot_thread"),
				Excerpt:    trimToRunes(joinNonEmpty([]string{thread.Title, thread.Summary}), 120),
			}},
		})
	}
	return issues
}

func continuityAuditStatus(issues []domain.ContinuityIssue) string {
	hasWarning := false
	for _, issue := range issues {
		if issue.Severity == "error" {
			return "failed"
		}
		if issue.Severity == "warning" {
			hasWarning = true
		}
	}
	if hasWarning {
		return "warning"
	}
	return "passed"
}

type continuityCanonMatcher struct {
	compactTexts []string
}

func newContinuityCanonMatcher(pack domain.ContextPack) continuityCanonMatcher {
	texts := make([]string, 0, len(pack.Entities)*2+len(pack.Facts)+len(pack.PlotThreads)*2+len(pack.ChapterSummaries)*2+len(pack.WorldRules))
	for _, entity := range pack.Entities {
		texts = appendContinuityCanonText(texts, entity.Name)
		for _, alias := range entity.Aliases {
			texts = appendContinuityCanonText(texts, alias)
		}
	}
	for _, fact := range pack.Facts {
		texts = appendContinuityCanonText(texts, fact.Claim)
	}
	for _, thread := range pack.PlotThreads {
		texts = appendContinuityCanonText(texts, thread.Title)
		texts = appendContinuityCanonText(texts, thread.Summary)
	}
	for _, summary := range pack.ChapterSummaries {
		texts = appendContinuityCanonText(texts, summary.Title)
		texts = appendContinuityCanonText(texts, summary.Summary)
	}
	for _, rule := range pack.WorldRules {
		texts = appendContinuityCanonText(texts, rule)
	}
	return continuityCanonMatcher{compactTexts: texts}
}

func appendContinuityCanonText(target []string, value string) []string {
	compact := compactContinuityText(value)
	if compact == "" {
		return target
	}
	return append(target, compact)
}

func (m continuityCanonMatcher) contains(candidate string) bool {
	compactCandidate := compactContinuityText(candidate)
	if compactCandidate == "" {
		return false
	}
	for _, text := range m.compactTexts {
		if strings.Contains(text, compactCandidate) {
			return true
		}
	}
	return false
}

func unsupportedClaimMarkerWindow(sentence string) (string, string, bool) {
	for _, marker := range continuityUnsupportedClaimMarkers {
		idx := strings.Index(sentence, marker)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(sentence[idx+len(marker):])
		if rest == "" {
			continue
		}
		return marker, rest, true
	}
	return "", "", false
}

func extractUnsupportedClaimCandidate(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false
	}
	trimmed = continuityUnsupportedClaimLeadPattern.ReplaceAllString(trimmed, "")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return "", false
	}
	runes := []rune(trimmed)
	for _, length := range []int{4, 3} {
		if len(runes) < length {
			continue
		}
		for start := 0; start+length <= len(runes); start++ {
			candidate := strings.TrimSpace(string(runes[start : start+length]))
			if isHighConfidenceUnsupportedClaimCandidate(candidate) {
				return candidate, true
			}
		}
	}
	return "", false
}

func isHighConfidenceUnsupportedClaimCandidate(candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	runes := []rune(candidate)
	if len(runes) < 2 || len(runes) > 4 {
		return false
	}
	if _, blocked := continuityUnsupportedClaimGenericCandidates[candidate]; blocked {
		return false
	}
	for _, keyword := range continuityUnsupportedClaimHeadKeywords {
		if strings.HasPrefix(candidate, keyword) || strings.HasSuffix(candidate, keyword) {
			return true
		}
	}
	return false
}

func continuityAuditMetadata(audit domain.ContinuityAudit) map[string]string {
	errorCount := 0
	warningCount := 0
	for _, issue := range audit.Issues {
		switch issue.Severity {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		}
	}
	return map[string]string{
		"status":        audit.Status,
		"issue_count":   fmt.Sprintf("%d", len(audit.Issues)),
		"error_count":   fmt.Sprintf("%d", errorCount),
		"warning_count": fmt.Sprintf("%d", warningCount),
	}
}

type continuityEntityNameRef struct {
	EntityID string
	Name     string
}

type continuityEntityIndex struct {
	byID             map[string]domain.Entity
	primaryNameByID  map[string]string
	allNames         []continuityEntityNameRef
	matchedAliasByID map[string]string
}

func newContinuityEntityIndex(entities []domain.Entity) continuityEntityIndex {
	index := continuityEntityIndex{
		byID:             make(map[string]domain.Entity, len(entities)),
		primaryNameByID:  make(map[string]string, len(entities)),
		allNames:         make([]continuityEntityNameRef, 0, len(entities)*2),
		matchedAliasByID: make(map[string]string, len(entities)),
	}
	seenNames := map[string]struct{}{}
	for _, entity := range entities {
		index.byID[entity.ID] = entity
		primary := strings.TrimSpace(firstText(entity.Name))
		if primary == "" && len(entity.Aliases) > 0 {
			primary = strings.TrimSpace(entity.Aliases[0])
		}
		if primary != "" {
			index.primaryNameByID[entity.ID] = primary
			compact := compactContinuityText(primary)
			if _, ok := seenNames[compact]; !ok {
				seenNames[compact] = struct{}{}
				index.allNames = append(index.allNames, continuityEntityNameRef{EntityID: entity.ID, Name: primary})
			}
		}
		for _, alias := range entity.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			compact := compactContinuityText(alias)
			if _, ok := seenNames[compact]; ok {
				continue
			}
			seenNames[compact] = struct{}{}
			index.allNames = append(index.allNames, continuityEntityNameRef{EntityID: entity.ID, Name: alias})
		}
	}
	sort.Slice(index.allNames, func(i, j int) bool {
		left := []rune(index.allNames[i].Name)
		right := []rune(index.allNames[j].Name)
		if len(left) == len(right) {
			return index.allNames[i].Name < index.allNames[j].Name
		}
		return len(left) > len(right)
	})
	return index
}

func (i continuityEntityIndex) primaryName(id string) string {
	if name := strings.TrimSpace(i.primaryNameByID[strings.TrimSpace(id)]); name != "" {
		return name
	}
	return ""
}

func (i continuityEntityIndex) firstMatchedPrimaryName(text string) string {
	for _, ref := range i.allNames {
		if containsCompactText(text, ref.Name) {
			return i.primaryName(ref.EntityID)
		}
	}
	return ""
}

func (i continuityEntityIndex) matchedNames(text string) []continuityEntityNameRef {
	matched := make([]continuityEntityNameRef, 0)
	seenEntityID := map[string]struct{}{}
	for _, ref := range i.allNames {
		if !containsCompactText(text, ref.Name) {
			continue
		}
		if _, ok := seenEntityID[ref.EntityID]; ok {
			continue
		}
		seenEntityID[ref.EntityID] = struct{}{}
		matched = append(matched, ref)
	}
	return matched
}

func (i continuityEntityIndex) relatedEntityNames(ids []string) []string {
	seen := map[string]struct{}{}
	items := make([]string, 0, len(ids))
	for _, id := range ids {
		name := i.primaryName(id)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		items = append(items, name)
	}
	return items
}

func detectLifeStateConflict(fact domain.Fact, sentences []string, entities continuityEntityIndex) (domain.ContinuityIssue, bool) {
	entityName, state, ok := parseLifeStateFact(fact, entities)
	if !ok {
		return domain.ContinuityIssue{}, false
	}
	for _, sentence := range sentences {
		if !containsCompactText(sentence, entityName) {
			continue
		}
		switch state {
		case "dead":
			if !containsAnyCompactText(sentence, continuityAliveStateMarkers) {
				continue
			}
		case "alive":
			if !containsAnyCompactText(sentence, continuityDeadStateMarkers) || containsNegativeCue(sentence) {
				continue
			}
		default:
			continue
		}
		return domain.ContinuityIssue{
			Type:         "hard_conflict",
			Severity:     "error",
			Message:      fmt.Sprintf("草稿与既有生死/存在事实冲突：%s", trimToRunes(fact.Claim, 80)),
			DraftExcerpt: trimToRunes(strings.TrimSpace(sentence), 120),
			Suggestion:   fmt.Sprintf("校正 %s 的生死/存在状态，或先在前文明确交代状态变化。", entityName),
			Evidence: []domain.ContinuityEvidenceRef{{
				SourceType: "fact",
				SourceID:   fact.ID,
				Label:      entityName,
				Excerpt:    trimToRunes(fact.Claim, 120),
			}},
		}, true
	}
	return domain.ContinuityIssue{}, false
}

func detectOwnershipConflict(fact domain.Fact, sentences []string, entities continuityEntityIndex) (domain.ContinuityIssue, bool) {
	ownerName, itemName, ok := parseOwnershipFact(fact, entities)
	if !ok {
		return domain.ContinuityIssue{}, false
	}
	checkedEntityIDs := map[string]struct{}{}
	for _, sentence := range sentences {
		if !containsCompactText(sentence, itemName) || containsNegativeCue(sentence) {
			continue
		}
		for _, ref := range entities.allNames {
			primary := entities.primaryName(ref.EntityID)
			if primary == "" || primary == ownerName || primary == itemName {
				continue
			}
			if _, ok := checkedEntityIDs[ref.EntityID]; ok {
				continue
			}
			if !ownershipClaimMatches(sentence, ref.Name, itemName) {
				continue
			}
			checkedEntityIDs[ref.EntityID] = struct{}{}
			return domain.ContinuityIssue{
				Type:         "hard_conflict",
				Severity:     "error",
				Message:      fmt.Sprintf("草稿与既有持有/归属事实冲突：%s 应由 %s 持有", itemName, ownerName),
				DraftExcerpt: trimToRunes(strings.TrimSpace(sentence), 120),
				Suggestion:   fmt.Sprintf("保持 %s 仍由 %s 持有，或先明确交代转移过程。", itemName, ownerName),
				Evidence: []domain.ContinuityEvidenceRef{{
					SourceType: "fact",
					SourceID:   fact.ID,
					Label:      ownerName,
					Excerpt:    trimToRunes(fact.Claim, 120),
				}},
			}, true
		}
	}
	return domain.ContinuityIssue{}, false
}

func detectKnowledgeConflict(fact domain.Fact, sentences []string, entities continuityEntityIndex) (domain.ContinuityIssue, bool) {
	entityName, proposition, knows, ok := parseKnowledgeFact(fact, entities)
	if !ok {
		return domain.ContinuityIssue{}, false
	}
	compactEntity := compactContinuityText(entityName)
	for _, sentence := range sentences {
		compactSentence := compactContinuityText(sentence)
		if !strings.Contains(compactSentence, compactEntity) || !strings.Contains(compactSentence, proposition) {
			continue
		}
		if knows {
			if !containsAnyCompactToken(compactSentence, continuityKnowledgeNegativeMarkers) {
				continue
			}
		} else {
			if !containsAnyCompactToken(compactSentence, continuityKnowledgePositiveMarkers) || containsAnyCompactToken(compactSentence, continuityKnowledgeNegativeMarkers) {
				continue
			}
		}
		return domain.ContinuityIssue{
			Type:         "hard_conflict",
			Severity:     "error",
			Message:      fmt.Sprintf("草稿与既有认知状态事实冲突：%s", trimToRunes(fact.Claim, 80)),
			DraftExcerpt: trimToRunes(strings.TrimSpace(sentence), 120),
			Suggestion:   fmt.Sprintf("校正 %s 对该信息的认知状态，或先补充新的获知/失忆过程。", entityName),
			Evidence: []domain.ContinuityEvidenceRef{{
				SourceType: "fact",
				SourceID:   fact.ID,
				Label:      entityName,
				Excerpt:    trimToRunes(fact.Claim, 120),
			}},
		}, true
	}
	return domain.ContinuityIssue{}, false
}

func parseLifeStateFact(fact domain.Fact, entities continuityEntityIndex) (string, string, bool) {
	entityName := entities.primaryName(fact.EntityID)
	if entityName == "" {
		entityName = entities.firstMatchedPrimaryName(fact.Claim)
	}
	if entityName == "" {
		return "", "", false
	}
	for _, marker := range continuityDeadStateMarkers {
		if containsCompactSequence(fact.Claim, entityName, marker) {
			return entityName, "dead", true
		}
	}
	for _, marker := range continuityAliveStateMarkers {
		if containsCompactSequence(fact.Claim, entityName, marker) {
			return entityName, "alive", true
		}
	}
	status := strings.ToLower(strings.TrimSpace(entities.byID[fact.EntityID].Status))
	switch status {
	case "dead", "deceased", "destroyed", "gone":
		return entityName, "dead", true
	case "alive", "active":
		return entityName, "alive", true
	default:
		return "", "", false
	}
}

func parseOwnershipFact(fact domain.Fact, entities continuityEntityIndex) (string, string, bool) {
	ownerName := entities.primaryName(fact.EntityID)
	matched := entities.matchedNames(fact.Claim)
	if ownerName != "" {
		for _, ref := range matched {
			itemName := entities.primaryName(ref.EntityID)
			if itemName == "" || itemName == ownerName {
				continue
			}
			if ownershipClaimMatches(fact.Claim, ownerName, ref.Name) {
				return ownerName, itemName, true
			}
		}
	}
	for _, ownerRef := range matched {
		ownerPrimary := entities.primaryName(ownerRef.EntityID)
		if ownerPrimary == "" {
			continue
		}
		for _, itemRef := range matched {
			itemPrimary := entities.primaryName(itemRef.EntityID)
			if itemPrimary == "" || itemPrimary == ownerPrimary || itemRef.EntityID == ownerRef.EntityID {
				continue
			}
			if ownershipClaimMatches(fact.Claim, ownerRef.Name, itemRef.Name) {
				return ownerPrimary, itemPrimary, true
			}
		}
	}
	return "", "", false
}

func parseKnowledgeFact(fact domain.Fact, entities continuityEntityIndex) (string, string, bool, bool) {
	entityName := entities.primaryName(fact.EntityID)
	if entityName == "" {
		entityName = entities.firstMatchedPrimaryName(fact.Claim)
	}
	if entityName == "" {
		return "", "", false, false
	}
	compactClaim := compactContinuityText(fact.Claim)
	compactEntity := compactContinuityText(entityName)
	for _, marker := range continuityKnowledgeNegativeMarkers {
		compactMarker := compactContinuityText(marker)
		idx := strings.Index(compactClaim, compactEntity+compactMarker)
		if idx < 0 {
			continue
		}
		proposition := compactClaim[idx+len(compactEntity)+len(compactMarker):]
		if len([]rune(proposition)) < 2 {
			continue
		}
		return entityName, proposition, false, true
	}
	for _, marker := range continuityKnowledgePositiveMarkers {
		compactMarker := compactContinuityText(marker)
		idx := strings.Index(compactClaim, compactEntity+compactMarker)
		if idx < 0 {
			continue
		}
		proposition := compactClaim[idx+len(compactEntity)+len(compactMarker):]
		if len([]rune(proposition)) < 2 {
			continue
		}
		return entityName, proposition, true, true
	}
	return "", "", false, false
}

func continuityGoalSource(input ContinuityAuditInput) (string, string, string) {
	if strings.TrimSpace(input.ChapterIdea) != "" {
		return "chapter_idea", "", strings.TrimSpace(input.ChapterIdea)
	}
	if strings.TrimSpace(input.Brief) != "" {
		return "brief", "", strings.TrimSpace(input.Brief)
	}
	if len(input.ContextPack.ChapterSummaries) == 0 {
		return "", "", ""
	}
	parts := make([]string, 0, len(input.ContextPack.ChapterSummaries))
	for _, summary := range input.ContextPack.ChapterSummaries {
		parts = append(parts, joinNonEmpty([]string{summary.Title, summary.Summary}))
	}
	return "chapter_summary", input.ContextPack.ChapterSummaries[0].ChapterVersionID, joinNonEmpty(parts)
}

func continuityKeywordCoverage(draft string, keywords []string) ([]string, []string) {
	compactDraft := compactContinuityText(draft)
	hits := make([]string, 0)
	missing := make([]string, 0)
	for _, keyword := range keywords {
		if strings.Contains(compactDraft, compactContinuityText(keyword)) {
			hits = append(hits, keyword)
			continue
		}
		missing = append(missing, keyword)
	}
	return hits, missing
}

func extractForbiddenAction(rule string) string {
	rule = strings.TrimSpace(rule)
	for _, marker := range continuityNegativeMarkers {
		idx := strings.Index(rule, marker)
		if idx < 0 {
			continue
		}
		action := strings.TrimSpace(rule[idx+len(marker):])
		action = strings.Trim(action, " ：:，,。；;、!！?？\t\r\n‘’“”\"'()（）[]【】")
		action = strings.TrimPrefix(action, "被")
		if len([]rune(action)) >= 2 {
			return action
		}
	}
	return ""
}

func splitContinuitySentences(text string) []string {
	parts := strings.FieldsFunc(text, func(r rune) bool {
		switch r {
		case '\n', '\r', '。', '！', '？', '!', '?', '；', ';':
			return true
		default:
			return false
		}
	})
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	if len(items) == 0 {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return nil
		}
		return []string{trimmed}
	}
	return items
}

func firstContinuitySentence(text string) string {
	sentences := splitContinuitySentences(text)
	if len(sentences) == 0 {
		return ""
	}
	return sentences[0]
}

func containsNegativeCue(text string) bool {
	return containsAnyCompactText(text, continuityNegativeCues)
}

func containsAnyCompactText(text string, candidates []string) bool {
	compact := compactContinuityText(text)
	return containsAnyCompactToken(compact, candidates)
}

func containsAnyCompactToken(compactTextValue string, candidates []string) bool {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if strings.Contains(compactTextValue, compactContinuityText(candidate)) {
			return true
		}
	}
	return false
}

func containsCompactText(text string, candidate string) bool {
	candidate = compactContinuityText(candidate)
	if candidate == "" {
		return false
	}
	return strings.Contains(compactContinuityText(text), candidate)
}

func containsCompactSequence(text string, parts ...string) bool {
	var builder strings.Builder
	for _, part := range parts {
		builder.WriteString(compactContinuityText(part))
	}
	target := builder.String()
	if target == "" {
		return false
	}
	return strings.Contains(compactContinuityText(text), target)
}

func compactContinuityText(text string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(text)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Han, r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func ownershipClaimMatches(text, ownerName, itemName string) bool {
	for _, marker := range continuityOwnershipMarkers {
		if containsCompactSequence(text, ownerName, marker, itemName) {
			return true
		}
	}
	return containsCompactSequence(text, itemName, "属于", ownerName) ||
		containsCompactSequence(text, itemName, "在", ownerName, "手中") ||
		containsCompactSequence(text, ownerName, "的", itemName)
}

func extractContinuityKeywords(text string, entities continuityEntityIndex) []string {
	collector := newContinuityKeywordCollector(10)
	for _, ref := range entities.allNames {
		if containsCompactText(text, ref.Name) {
			collector.add(entities.primaryName(ref.EntityID))
		}
	}
	var asciiBuilder strings.Builder
	var hanBuilder strings.Builder
	flushASCII := func() {
		if asciiBuilder.Len() == 0 {
			return
		}
		word := asciiBuilder.String()
		asciiBuilder.Reset()
		if len([]rune(word)) >= 4 {
			collector.add(word)
		}
	}
	flushHan := func() {
		if hanBuilder.Len() == 0 {
			return
		}
		sequence := hanBuilder.String()
		hanBuilder.Reset()
		addHanKeywordVariants(sequence, collector)
	}
	for _, r := range text {
		switch {
		case unicode.Is(unicode.Han, r):
			flushASCII()
			hanBuilder.WriteRune(r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			flushHan()
			asciiBuilder.WriteRune(unicode.ToLower(r))
		default:
			flushASCII()
			flushHan()
		}
	}
	flushASCII()
	flushHan()
	return collector.items
}

func addHanKeywordVariants(sequence string, collector *continuityKeywordCollector) {
	runes := []rune(strings.TrimSpace(sequence))
	if len(runes) < 2 {
		return
	}
	if len(runes) <= 6 {
		collector.add(string(runes))
	}
	for size := 2; size <= 3; size++ {
		for start := 0; start+size <= len(runes); start++ {
			collector.add(string(runes[start : start+size]))
		}
	}
}

type continuityKeywordCollector struct {
	items []string
	seen  map[string]struct{}
	limit int
}

func newContinuityKeywordCollector(limit int) *continuityKeywordCollector {
	return &continuityKeywordCollector{items: make([]string, 0, limit), seen: map[string]struct{}{}, limit: limit}
}

func (c *continuityKeywordCollector) add(value string) {
	value = strings.TrimSpace(value)
	if value == "" || isContinuityStopKeyword(value) {
		return
	}
	compact := compactContinuityText(value)
	if compact == "" {
		return
	}
	if _, ok := c.seen[compact]; ok {
		return
	}
	if len(c.items) >= c.limit {
		return
	}
	c.seen[compact] = struct{}{}
	c.items = append(c.items, value)
}

func isContinuityStopKeyword(value string) bool {
	if _, ok := continuityKeywordStopwords[value]; ok {
		return true
	}
	runes := []rune(value)
	if len(runes) < 2 {
		return true
	}
	stopRuneCount := 0
	for _, r := range runes {
		if strings.ContainsRune("的了着是和与并将把从向于在中为又就都而后前里外", r) {
			stopRuneCount++
		}
	}
	return stopRuneCount == len(runes)
}

func isClosedThread(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "closed", "resolved", "done", "completed":
		return true
	default:
		return false
	}
}

func firstNStrings(items []string, n int) []string {
	if len(items) <= n {
		return items
	}
	return items[:n]
}
