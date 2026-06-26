package extractor

import (
	"testing"

	"aeonechoes/server/internal/domain"
)

func TestDeterministicExtractorExtractsMarkedKnowledge(t *testing.T) {
	ex := NewDeterministicExtractor()
	result, err := ex.ExtractChapter(domain.ChapterVersion{
		ID:        "cv_1",
		ProjectID: "project_1",
		ChapterID: "chapter_1",
		Title:     "空白目录",
		Summary:   "林烬发现灰烬钥匙。",
		Content:   "[[人物:林烬]] 在 [[地点:黑曜档案馆]] 找到 [[物品:灰烬钥匙]]。[[关系:林烬->灰烬钥匙:持有]] [[伏笔:第三见证人|缺席者会以档案页形式出现]]",
	})
	if err != nil {
		t.Fatalf("ExtractChapter() error: %v", err)
	}
	if len(result.Facts) != 1 || result.Facts[0].Metadata["fact_type"] != "chapter_summary" {
		t.Fatalf("expected summary fact, got %+v", result.Facts)
	}
	if len(result.Entities) != 4 {
		t.Fatalf("expected 4 unique entities, got %d: %+v", len(result.Entities), result.Entities)
	}
	if len(result.Edges) != 1 || result.Edges[0].Label != "持有" || result.Edges[0].Type != "owns" {
		t.Fatalf("expected owns edge, got %+v", result.Edges)
	}
	if len(result.PlotThreads) != 1 || result.PlotThreads[0].Title != "第三见证人" {
		t.Fatalf("expected plot thread, got %+v", result.PlotThreads)
	}
}

func TestDeterministicExtractorAlwaysEmitsSummaryFact(t *testing.T) {
	ex := NewDeterministicExtractor()
	result, err := ex.ExtractChapter(domain.ChapterVersion{ID: "cv_1", ProjectID: "project_1", ChapterID: "chapter_1", Title: "无标记章节", Content: "这里没有任何显式标记，但仍需要摘要事实。"})
	if err != nil {
		t.Fatalf("ExtractChapter() error: %v", err)
	}
	if len(result.Facts) != 1 {
		t.Fatalf("expected one summary fact, got %+v", result.Facts)
	}
	if len(result.Entities) != 0 || len(result.Edges) != 0 || len(result.PlotThreads) != 0 {
		t.Fatalf("expected no marked knowledge, got %+v", result)
	}
}
