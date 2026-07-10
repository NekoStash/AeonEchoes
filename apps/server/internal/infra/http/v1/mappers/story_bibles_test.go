package mappers

import (
	"testing"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
)

func TestStoryBibleDTOFromDomainDoesNotSynthesizeCollectionsOrReadLegacyChapters(t *testing.T) {
	bible := domain.StoryBible{
		Title: "空设定集",
		Rules: map[string]string{"canon": "真实规则不得伪装成伏笔"},
		SourceSeed: domain.ProjectSeed{
			Premise:        "测试空集合",
			MainCharacters: []string{"不应伪造的角色"},
			TargetChapters: 12,
			Metadata: map[string]string{
				"story_bible_chapters": `[{"id":"legacy-1","title":"旧规划","status":"planned","summary":"不应读取"}]`,
			},
		},
	}

	result, err := StoryBibleDTOFromDomain(bible)
	if err != nil {
		t.Fatalf("StoryBibleDTOFromDomain() error: %v", err)
	}
	if result.Characters == nil || result.Foreshadows == nil || result.ChapterPlan == nil {
		t.Fatalf("story bible collections must be present: %+v", result)
	}
	if len(result.Characters) != 0 || len(result.Foreshadows) != 0 || len(result.ChapterPlan) != 0 {
		t.Fatalf("story bible collections were synthesized or read from legacy metadata: %+v", result)
	}
}

func TestStoryBibleDTOToDomainRejectsInvalidChapterStatus(t *testing.T) {
	_, err := StoryBibleDTOToDomain(dto.StoryBibleDTO{
		Title:       "非法规划",
		Premise:     "测试状态枚举",
		Characters:  []dto.StoryBibleCharacterDTO{},
		Foreshadows: []dto.StoryBibleForeshadowDTO{},
		ChapterPlan: []dto.StoryBibleChapterPlanDTO{{ID: "plan-1", Title: "第一章", Status: "draft", Summary: "旧状态"}},
		SourceSeed:  dto.ProjectSeedDTO{Title: "非法规划", Premise: "测试状态枚举"},
	})
	if err == nil {
		t.Fatalf("StoryBibleDTOToDomain(invalid chapter status) error = nil")
	}
}

func TestStoryBibleDTOToDomainWritesOnlyCanonicalChapterPlan(t *testing.T) {
	bible, err := StoryBibleDTOToDomain(dto.StoryBibleDTO{
		Title:       "规范规划",
		Premise:     "测试 canonical chapter_plan",
		Characters:  []dto.StoryBibleCharacterDTO{},
		Foreshadows: []dto.StoryBibleForeshadowDTO{},
		ChapterPlan: []dto.StoryBibleChapterPlanDTO{{ID: "plan-1", Title: "第一章", Status: "planned", Summary: "仅规划"}},
		SourceSeed:  dto.ProjectSeedDTO{Title: "规范规划", Premise: "测试 canonical chapter_plan"},
	})
	if err != nil {
		t.Fatalf("StoryBibleDTOToDomain() error: %v", err)
	}
	if bible.SourceSeed.Metadata[storyBibleChapterPlanMetadataKey] == "" {
		t.Fatalf("canonical chapter plan metadata was not written: %+v", bible.SourceSeed.Metadata)
	}
	if _, exists := bible.SourceSeed.Metadata["story_bible_chapters"]; exists {
		t.Fatalf("legacy chapter metadata was written: %+v", bible.SourceSeed.Metadata)
	}
}
