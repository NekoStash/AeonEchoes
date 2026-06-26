package extractor

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"aeonechoes/server/internal/domain"
)

// ExtractResult contains deterministic, reviewable knowledge derived from a chapter version.
type ExtractResult struct {
	Facts       []domain.Fact       `json:"facts"`
	Entities    []domain.Entity     `json:"entities"`
	Edges       []domain.GraphEdge  `json:"edges"`
	PlotThreads []domain.PlotThread `json:"plot_threads"`
}

// Extractor extracts structured narrative knowledge from chapter text.
type Extractor interface {
	ExtractChapter(version domain.ChapterVersion) (ExtractResult, error)
}

// DeterministicExtractor is a conservative marker-based extractor used as a testable baseline extractor.
// Supported markers:
//
//	[[人物:林烬]] [[地点:黑曜档案馆]] [[组织:星图会]] [[物品:灰烬钥匙]] [[伏笔:第三见证人]]
//	[[伏笔:第三见证人|回收提示]]
//	[[关系:林烬->灰烬钥匙:持有]]
type DeterministicExtractor struct{}

func NewDeterministicExtractor() *DeterministicExtractor { return &DeterministicExtractor{} }

var (
	entityMarkerRE = regexp.MustCompile(`\[\[(人物|地点|组织|物品|伏笔):([^\]|]+)(?:\|([^\]]+))?\]\]`)
	relationRE     = regexp.MustCompile(`\[\[关系:([^\]-]+)->([^:\]]+):([^\]]+)\]\]`)
)

func (e *DeterministicExtractor) ExtractChapter(version domain.ChapterVersion) (ExtractResult, error) {
	if strings.TrimSpace(version.ProjectID) == "" || strings.TrimSpace(version.ID) == "" {
		return ExtractResult{}, fmt.Errorf("chapter version project_id and id must not be empty")
	}
	text := strings.TrimSpace(strings.Join([]string{version.Title, version.Summary, version.Content}, "\n"))
	if text == "" {
		return ExtractResult{}, fmt.Errorf("chapter version content surface is empty")
	}
	result := ExtractResult{}
	result.Facts = append(result.Facts, domain.Fact{
		ProjectID:        version.ProjectID,
		ChapterID:        version.ChapterID,
		ChapterVersionID: version.ID,
		Claim:            fmt.Sprintf("章节《%s》摘要：%s", firstText(version.Title, "未命名章节"), summarize(version)),
		Source:           version.ID,
		Confidence:       1,
		Status:           "proposed",
		Metadata:         map[string]string{"extractor": "deterministic", "fact_type": "chapter_summary"},
	})

	entityByKey := map[string]domain.Entity{}
	addEntity := func(markerType, name, hint string) domain.Entity {
		name = strings.TrimSpace(name)
		entityType := mapMarkerType(markerType)
		key := entityType + ":" + name
		if existing, ok := entityByKey[key]; ok {
			return existing
		}
		entity := domain.Entity{
			ProjectID:  version.ProjectID,
			Name:       name,
			Type:       entityType,
			Summary:    firstText(strings.TrimSpace(hint), fmt.Sprintf("由章节《%s》标记抽取。", firstText(version.Title, "未命名章节"))),
			Importance: 50,
			Status:     "proposed",
			Metadata:   map[string]string{"extractor": "deterministic", "source_chapter_version_id": version.ID},
		}
		entityByKey[key] = entity
		return entity
	}

	for _, match := range entityMarkerRE.FindAllStringSubmatch(text, -1) {
		markerType, name, hint := match[1], match[2], ""
		if len(match) > 3 {
			hint = match[3]
		}
		entity := addEntity(markerType, name, hint)
		if markerType == "伏笔" {
			result.PlotThreads = appendUniqueThread(result.PlotThreads, domain.PlotThread{
				ProjectID:        version.ProjectID,
				Title:            entity.Name,
				Summary:          firstText(hint, "章节中埋入的伏笔，需要后续跟踪回收。"),
				Status:           "open",
				Priority:         50,
				OpenedChapterID:  version.ChapterID,
				RelatedEntityIDs: nil,
				Metadata:         map[string]string{"extractor": "deterministic", "source_chapter_version_id": version.ID},
			})
		}
	}

	for _, match := range relationRE.FindAllStringSubmatch(text, -1) {
		sourceName, targetName, label := strings.TrimSpace(match[1]), strings.TrimSpace(match[2]), strings.TrimSpace(match[3])
		source := addEntity("人物", sourceName, "关系标记中的源实体。")
		target := addEntity("物品", targetName, "关系标记中的目标实体。")
		result.Edges = append(result.Edges, domain.GraphEdge{
			ProjectID:       version.ProjectID,
			SourceEntityID:  source.Name,
			TargetEntityID:  target.Name,
			Type:            normalizeRelationType(label),
			Label:           label,
			Weight:          1,
			EvidenceFactIDs: nil,
			Metadata:        map[string]string{"extractor": "deterministic", "source_chapter_version_id": version.ID, "source_entity_name": source.Name, "target_entity_name": target.Name},
		})
	}

	for _, entity := range entityByKey {
		result.Entities = append(result.Entities, entity)
	}
	return result, nil
}

func mapMarkerType(markerType string) string {
	switch markerType {
	case "人物":
		return "character"
	case "地点":
		return "place"
	case "组织":
		return "organization"
	case "物品":
		return "item"
	case "伏笔":
		return "clue"
	default:
		return "concept"
	}
}

func normalizeRelationType(label string) string {
	label = strings.TrimSpace(label)
	switch label {
	case "持有", "拥有":
		return "owns"
	case "认识", "知道":
		return "knows"
	case "敌对", "仇恨":
		return "opposes"
	case "保护":
		return "protects"
	default:
		return "related_to"
	}
}

func summarize(version domain.ChapterVersion) string {
	if strings.TrimSpace(version.Summary) != "" {
		return trimRunes(version.Summary, 180)
	}
	return trimRunes(version.Content, 180)
}

func trimRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func appendUniqueThread(items []domain.PlotThread, item domain.PlotThread) []domain.PlotThread {
	for _, existing := range items {
		if existing.Title == item.Title {
			return items
		}
	}
	return append(items, item)
}
