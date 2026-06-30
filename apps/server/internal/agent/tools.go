package agent

import (
	"fmt"
	"strconv"
	"strings"

	"aeonechoes/server/internal/domain"
)

// ToolStore is the persistence surface used by provider tool calls.
type ToolStore interface {
	NewID(prefix string) (string, error)
	SaveEntity(item domain.Entity) (domain.Entity, error)
	SaveGraphEdge(item domain.GraphEdge) (domain.GraphEdge, error)
	SavePlotThread(item domain.PlotThread) (domain.PlotThread, error)
	ListEntities(projectID string) ([]domain.Entity, error)
	ListPlotThreads(projectID string) ([]domain.PlotThread, error)
	ExpandGraph(projectID string, entityIDs []string, depth int) (domain.GraphExpansion, error)
	EnsureChapter(req domain.ChapterEnsureRequest) (domain.Chapter, error)
	GetChapter(id string) (domain.Chapter, error)
	ListChapters(projectID string) ([]domain.Chapter, error)
	ListChapterVersions(projectID, chapterID string) ([]domain.ChapterVersion, error)
}

// GraphRepository is the graph retrieval surface used by novel-specific tools.
type GraphRepository interface {
	ExpandGraph(projectID string, entityIDs []string, depth int) (domain.GraphExpansion, error)
	ListEntities(projectID string) ([]domain.Entity, error)
	ListFacts(projectID string) ([]domain.Fact, error)
	ListPlotThreads(projectID string) ([]domain.PlotThread, error)
	ListChapters(projectID string) ([]domain.Chapter, error)
	ListChapterVersions(projectID, chapterID string) ([]domain.ChapterVersion, error)
	GetStoryBible(projectID string) (domain.StoryBible, error)
}

// ToolRuntime exposes novel-specific retrieval tools to agents.
type ToolRuntime struct {
	repo GraphRepository
}

func NewToolRuntime(repo GraphRepository) *ToolRuntime {
	return &ToolRuntime{repo: repo}
}

func (t *ToolRuntime) ExpandGraph(projectID string, entityIDs []string, depth int) (domain.GraphExpansion, error) {
	if t == nil || t.repo == nil {
		return domain.GraphExpansion{}, fmt.Errorf("tool runtime is not configured")
	}
	return t.repo.ExpandGraph(projectID, entityIDs, depth)
}

// ContextPackBuilder constructs compact role-specific context, never a full novel dump.
type ContextPackBuilder struct {
	repo  GraphRepository
	tools *ToolRuntime
	ids   IDSource
}

// IDSource is the tiny ID generator surface needed by context pack construction.
type IDSource interface {
	NewID(prefix string) (string, error)
}

func NewContextPackBuilder(repo GraphRepository, tools *ToolRuntime, ids IDSource) *ContextPackBuilder {
	return &ContextPackBuilder{repo: repo, tools: tools, ids: ids}
}

func (b *ContextPackBuilder) Build(projectID, chapterID string, role domain.AgentRole, query string, tokenBudget int) (domain.ContextPack, error) {
	return b.BuildWithSelection(projectID, chapterID, role, query, tokenBudget, nil, nil)
}

func (b *ContextPackBuilder) BuildWithSelection(projectID, chapterID string, role domain.AgentRole, query string, tokenBudget int, selection *ContextSelection, contextNodeIDs []string) (domain.ContextPack, error) {
	if b == nil || b.repo == nil || b.tools == nil || b.ids == nil {
		return domain.ContextPack{}, fmt.Errorf("context pack builder is not configured")
	}
	if strings.TrimSpace(projectID) == "" {
		return domain.ContextPack{}, fmt.Errorf("context pack project_id must not be empty")
	}
	if tokenBudget <= 0 {
		tokenBudget = 4000
	}
	selection = normalizeContextSelection(selection)
	bible, err := b.repo.GetStoryBible(projectID)
	if err != nil {
		return domain.ContextPack{}, err
	}
	contextPackID, err := b.ids.NewID("context_pack")
	if err != nil {
		return domain.ContextPack{}, fmt.Errorf("generate context pack id: %w", err)
	}
	includeWorldRules := shouldIncludeWorldRules(selection)
	metadata := buildSelectionMetadata(selection, contextNodeIDs)
	if selection == nil || !selection.hasStructuredFilters() {
		return b.buildAutoContextPack(contextPackID, bible, projectID, chapterID, role, query, tokenBudget, includeWorldRules, metadata)
	}
	return b.buildSelectedContextPack(contextPackID, bible, projectID, chapterID, role, query, tokenBudget, selection, includeWorldRules, metadata)
}

func (b *ContextPackBuilder) buildAutoContextPack(contextPackID string, bible domain.StoryBible, projectID, chapterID string, role domain.AgentRole, query string, tokenBudget int, includeWorldRules bool, metadata map[string]string) (domain.ContextPack, error) {
	expansion, err := b.tools.ExpandGraph(projectID, nil, 1)
	if err != nil {
		return domain.ContextPack{}, err
	}
	facts, err := b.repo.ListFacts(projectID)
	if err != nil {
		return domain.ContextPack{}, err
	}
	if len(facts) > 20 {
		facts = facts[:20]
	}
	versions, err := b.repo.ListChapterVersions(projectID, chapterID)
	if err != nil {
		return domain.ContextPack{}, err
	}
	threads, err := b.repo.ListPlotThreads(projectID)
	if err != nil {
		return domain.ContextPack{}, err
	}
	metadata["selection_chapter_summary_count"] = strconv.Itoa(len(buildChapterSummaries(versions, chapterID, nil, nil)))
	metadata["selection_entity_count"] = strconv.Itoa(len(limitEntities(expansion.Entities, 20)))
	metadata["selection_fact_count"] = strconv.Itoa(len(facts))
	metadata["selection_plot_thread_count"] = strconv.Itoa(len(limitThreads(threads, 12)))
	return domain.ContextPack{
		ID:               contextPackID,
		ProjectID:        projectID,
		ChapterID:        chapterID,
		Role:             role,
		TokenBudget:      tokenBudget,
		Query:            query,
		StoryBibleID:     bible.ID,
		WorldRules:       copyWorldRules(bible.Rules, includeWorldRules),
		Facts:            facts,
		Entities:         limitEntities(expansion.Entities, 20),
		Edges:            limitEdges(expansion.Edges, 30),
		PlotThreads:      limitThreads(threads, 12),
		ChapterSummaries: buildChapterSummaries(versions, chapterID, nil, nil),
		ToolTrace:        []string{"selection.mode=auto", "graph.expand depth=1", "facts.limit=20", "chapter_summaries.limit=8"},
		Metadata:         metadata,
		CreatedAt:        nowUTC(),
	}, nil
}

func (b *ContextPackBuilder) buildSelectedContextPack(contextPackID string, bible domain.StoryBible, projectID, chapterID string, role domain.AgentRole, query string, tokenBudget int, selection *ContextSelection, includeWorldRules bool, metadata map[string]string) (domain.ContextPack, error) {
	allEntities, err := b.repo.ListEntities(projectID)
	if err != nil {
		return domain.ContextPack{}, err
	}
	allFacts, err := b.repo.ListFacts(projectID)
	if err != nil {
		return domain.ContextPack{}, err
	}
	allVersions, err := b.repo.ListChapterVersions(projectID, "")
	if err != nil {
		return domain.ContextPack{}, err
	}
	threads, err := b.repo.ListPlotThreads(projectID)
	if err != nil {
		return domain.ContextPack{}, err
	}
	versionByID, versionIDsByChapterID, err := indexChapterVersions(selection.ChapterIDs, allVersions)
	if err != nil {
		return domain.ContextPack{}, err
	}
	selectedCharacters, err := resolveCharacterSelection(selection, allEntities)
	if err != nil {
		return domain.ContextPack{}, err
	}
	var expansion domain.GraphExpansion
	if len(selectedCharacters.idsInOrder) > 0 {
		expansion, err = b.tools.ExpandGraph(projectID, selectedCharacters.idsInOrder, 1)
		if err != nil {
			return domain.ContextPack{}, err
		}
	} else {
		expansion = domain.GraphExpansion{ProjectID: projectID, Depth: 1}
	}
	selectedChapterIDs := stringSetFromSlice(selection.ChapterIDs)
	selectedVersionIDs := stringSet{}
	for _, chapterID := range selection.ChapterIDs {
		for _, versionID := range versionIDsByChapterID[chapterID] {
			selectedVersionIDs.add(versionID)
		}
	}
	selectedEntityIDs := stringSet{}
	for _, entity := range selectedCharacters.entities {
		selectedEntityIDs.add(entity.ID)
		addEntitySourceChapter(entity, versionByID, selectedChapterIDs, selectedVersionIDs)
	}
	for _, entity := range expansion.Entities {
		selectedEntityIDs.add(entity.ID)
		addEntitySourceChapter(entity, versionByID, selectedChapterIDs, selectedVersionIDs)
	}
	entities := filterEntitiesForSelection(allEntities, selectedEntityIDs, selectedVersionIDs)
	for _, entity := range entities {
		selectedEntityIDs.add(entity.ID)
		addEntitySourceChapter(entity, versionByID, selectedChapterIDs, selectedVersionIDs)
	}
	facts := filterFactsForSelection(allFacts, selectedChapterIDs, selectedVersionIDs, selectedEntityIDs)
	if len(facts) > 20 {
		facts = facts[:20]
	}
	for _, fact := range facts {
		if strings.TrimSpace(fact.EntityID) != "" {
			selectedEntityIDs.add(fact.EntityID)
		}
		if strings.TrimSpace(fact.ChapterID) != "" {
			selectedChapterIDs.add(fact.ChapterID)
		}
		if strings.TrimSpace(fact.ChapterVersionID) != "" {
			selectedVersionIDs.add(fact.ChapterVersionID)
			if version, ok := versionByID[fact.ChapterVersionID]; ok && strings.TrimSpace(version.ChapterID) != "" {
				selectedChapterIDs.add(version.ChapterID)
			}
		}
	}
	entities = filterEntitiesForSelection(allEntities, selectedEntityIDs, selectedVersionIDs)
	edges := filterEdgesForSelection(expansion.Edges, selectedEntityIDs)
	plotThreads := filterThreadsForSelection(threads, selectedEntityIDs, selectedChapterIDs, selectedVersionIDs)
	summaries := buildChapterSummaries(allVersions, chapterID, selectedChapterIDs, selectedVersionIDs)
	metadata["selection_chapter_summary_count"] = strconv.Itoa(len(summaries))
	metadata["selection_entity_count"] = strconv.Itoa(len(entities))
	metadata["selection_fact_count"] = strconv.Itoa(len(facts))
	metadata["selection_plot_thread_count"] = strconv.Itoa(len(plotThreads))
	toolTrace := []string{"selection.mode=explicit", "graph.expand depth=1 seeded_by=character_selection", "facts.filter=chapter_or_character", "chapter_summaries.filter=selection"}
	return domain.ContextPack{
		ID:               contextPackID,
		ProjectID:        projectID,
		ChapterID:        chapterID,
		Role:             role,
		TokenBudget:      tokenBudget,
		Query:            query,
		StoryBibleID:     bible.ID,
		WorldRules:       copyWorldRules(bible.Rules, includeWorldRules),
		Facts:            facts,
		Entities:         limitEntities(entities, 20),
		Edges:            limitEdges(edges, 30),
		PlotThreads:      limitThreads(plotThreads, 12),
		ChapterSummaries: summaries,
		ToolTrace:        toolTrace,
		Metadata:         metadata,
		CreatedAt:        nowUTC(),
	}, nil
}

func limitEntities(items []domain.Entity, n int) []domain.Entity {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func limitEdges(items []domain.GraphEdge, n int) []domain.GraphEdge {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func limitThreads(items []domain.PlotThread, n int) []domain.PlotThread {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func trimToRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func normalizeContextSelection(selection *ContextSelection) *ContextSelection {
	if selection == nil {
		return nil
	}
	normalized := &ContextSelection{
		ChapterIDs:        dedupeTrimmedStrings(selection.ChapterIDs),
		CharacterIDs:      dedupeTrimmedStrings(selection.CharacterIDs),
		CharacterNames:    dedupeTrimmedStrings(selection.CharacterNames),
		IncludeWorldRules: selection.IncludeWorldRules,
	}
	if !normalized.hasStructuredFilters() && normalized.IncludeWorldRules == nil {
		return nil
	}
	return normalized
}

func (s *ContextSelection) hasStructuredFilters() bool {
	return s != nil && (len(s.ChapterIDs) > 0 || len(s.CharacterIDs) > 0 || len(s.CharacterNames) > 0)
}

func shouldIncludeWorldRules(selection *ContextSelection) bool {
	if selection == nil || selection.IncludeWorldRules == nil {
		return true
	}
	return *selection.IncludeWorldRules
}

func buildSelectionMetadata(selection *ContextSelection, contextNodeIDs []string) map[string]string {
	metadata := map[string]string{}
	if selection == nil {
		metadata["selection_mode"] = "auto"
	} else {
		metadata["selection_mode"] = "explicit"
		if len(selection.ChapterIDs) > 0 {
			metadata["selection_chapter_ids"] = strings.Join(selection.ChapterIDs, ",")
		}
		if len(selection.CharacterIDs) > 0 {
			metadata["selection_character_ids"] = strings.Join(selection.CharacterIDs, ",")
		}
		if len(selection.CharacterNames) > 0 {
			metadata["selection_character_names"] = strings.Join(selection.CharacterNames, ",")
		}
		metadata["selection_include_world_rules"] = strconv.FormatBool(shouldIncludeWorldRules(selection))
	}
	contextNodeIDs = dedupeTrimmedStrings(contextNodeIDs)
	if len(contextNodeIDs) > 0 {
		metadata["context_node_ids_compat"] = strings.Join(contextNodeIDs, ",")
	}
	return metadata
}

func buildChapterSummaries(versions []domain.ChapterVersion, chapterID string, selectedChapterIDs, selectedVersionIDs stringSet) []domain.ChapterSummary {
	summaries := make([]domain.ChapterSummary, 0, len(versions))
	for _, version := range versions {
		if !shouldIncludeVersion(version, chapterID, selectedChapterIDs, selectedVersionIDs) {
			continue
		}
		if strings.TrimSpace(version.Summary) == "" && version.ChapterID != chapterID {
			continue
		}
		summaries = append(summaries, domain.ChapterSummary{ChapterID: version.ChapterID, ChapterVersionID: version.ID, Title: version.Title, Summary: firstText(version.Summary, trimToRunes(version.Content, 240))})
		if len(summaries) >= 8 {
			break
		}
	}
	return summaries
}

func shouldIncludeVersion(version domain.ChapterVersion, defaultChapterID string, selectedChapterIDs, selectedVersionIDs stringSet) bool {
	if len(selectedChapterIDs) > 0 || len(selectedVersionIDs) > 0 {
		return selectedChapterIDs.has(version.ChapterID) || selectedVersionIDs.has(version.ID)
	}
	if strings.TrimSpace(defaultChapterID) == "" {
		return true
	}
	return version.ChapterID == defaultChapterID
}

type characterSelectionResult struct {
	entities   []domain.Entity
	idsInOrder []string
}

func resolveCharacterSelection(selection *ContextSelection, entities []domain.Entity) (characterSelectionResult, error) {
	result := characterSelectionResult{}
	if selection == nil {
		return result, nil
	}
	byID := map[string]domain.Entity{}
	byName := map[string]domain.Entity{}
	for _, entity := range entities {
		byID[entity.ID] = entity
		if entity.Type != "character" {
			continue
		}
		byName[strings.ToLower(strings.TrimSpace(entity.Name))] = entity
		for _, alias := range entity.Aliases {
			trimmed := strings.ToLower(strings.TrimSpace(alias))
			if trimmed != "" {
				byName[trimmed] = entity
			}
		}
	}
	seen := stringSet{}
	for _, id := range selection.CharacterIDs {
		entity, ok := byID[id]
		if !ok {
			return characterSelectionResult{}, fmt.Errorf("context selection character_id %q not found", id)
		}
		if entity.Type != "character" {
			return characterSelectionResult{}, fmt.Errorf("context selection entity %q is not a character", id)
		}
		if !seen.has(entity.ID) {
			seen.add(entity.ID)
			result.entities = append(result.entities, entity)
			result.idsInOrder = append(result.idsInOrder, entity.ID)
		}
	}
	for _, name := range selection.CharacterNames {
		entity, ok := byName[strings.ToLower(name)]
		if !ok {
			return characterSelectionResult{}, fmt.Errorf("context selection character_name %q not found", name)
		}
		if !seen.has(entity.ID) {
			seen.add(entity.ID)
			result.entities = append(result.entities, entity)
			result.idsInOrder = append(result.idsInOrder, entity.ID)
		}
	}
	return result, nil
}

func indexChapterVersions(selectedChapterIDs []string, versions []domain.ChapterVersion) (map[string]domain.ChapterVersion, map[string][]string, error) {
	versionByID := map[string]domain.ChapterVersion{}
	versionIDsByChapterID := map[string][]string{}
	availableChapterIDs := stringSet{}
	for _, version := range versions {
		versionByID[version.ID] = version
		versionIDsByChapterID[version.ChapterID] = append(versionIDsByChapterID[version.ChapterID], version.ID)
		availableChapterIDs.add(version.ChapterID)
	}
	for _, chapterID := range selectedChapterIDs {
		if !availableChapterIDs.has(chapterID) {
			return nil, nil, fmt.Errorf("context selection chapter_id %q not found", chapterID)
		}
	}
	return versionByID, versionIDsByChapterID, nil
}

func addEntitySourceChapter(entity domain.Entity, versionByID map[string]domain.ChapterVersion, selectedChapterIDs, selectedVersionIDs stringSet) {
	versionID := strings.TrimSpace(entity.Metadata["source_chapter_version_id"])
	if versionID == "" {
		return
	}
	selectedVersionIDs.add(versionID)
	if version, ok := versionByID[versionID]; ok && strings.TrimSpace(version.ChapterID) != "" {
		selectedChapterIDs.add(version.ChapterID)
	}
}

func filterEntitiesForSelection(allEntities []domain.Entity, selectedEntityIDs, selectedVersionIDs stringSet) []domain.Entity {
	if len(selectedEntityIDs) == 0 && len(selectedVersionIDs) == 0 {
		return nil
	}
	filtered := make([]domain.Entity, 0)
	for _, entity := range allEntities {
		if selectedEntityIDs.has(entity.ID) || selectedVersionIDs.has(strings.TrimSpace(entity.Metadata["source_chapter_version_id"])) {
			filtered = append(filtered, entity)
		}
	}
	return filtered
}

func filterFactsForSelection(allFacts []domain.Fact, selectedChapterIDs, selectedVersionIDs, selectedEntityIDs stringSet) []domain.Fact {
	filtered := make([]domain.Fact, 0)
	for _, fact := range allFacts {
		if selectedChapterIDs.has(fact.ChapterID) || selectedVersionIDs.has(fact.ChapterVersionID) || selectedEntityIDs.has(fact.EntityID) {
			filtered = append(filtered, fact)
		}
	}
	return filtered
}

func filterEdgesForSelection(edges []domain.GraphEdge, selectedEntityIDs stringSet) []domain.GraphEdge {
	if len(selectedEntityIDs) == 0 {
		return nil
	}
	filtered := make([]domain.GraphEdge, 0, len(edges))
	for _, edge := range edges {
		if selectedEntityIDs.has(edge.SourceEntityID) || selectedEntityIDs.has(edge.TargetEntityID) {
			filtered = append(filtered, edge)
		}
	}
	return filtered
}

func filterThreadsForSelection(items []domain.PlotThread, selectedEntityIDs, selectedChapterIDs, selectedVersionIDs stringSet) []domain.PlotThread {
	filtered := make([]domain.PlotThread, 0)
	for _, item := range items {
		if selectedChapterIDs.has(item.OpenedChapterID) || selectedChapterIDs.has(item.ClosedChapterID) || selectedVersionIDs.has(strings.TrimSpace(item.Metadata["source_chapter_version_id"])) {
			filtered = append(filtered, item)
			continue
		}
		for _, entityID := range item.RelatedEntityIDs {
			if selectedEntityIDs.has(entityID) {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

func copyWorldRules(rules map[string]string, include bool) map[string]string {
	if !include || len(rules) == 0 {
		return nil
	}
	copied := make(map[string]string, len(rules))
	for key, value := range rules {
		copied[key] = value
	}
	return copied
}

func dedupeTrimmedStrings(values []string) []string {
	seen := map[string]bool{}
	items := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		items = append(items, trimmed)
	}
	return items
}

type stringSet map[string]struct{}

func stringSetFromSlice(values []string) stringSet {
	set := stringSet{}
	for _, value := range values {
		set.add(value)
	}
	return set
}

func (s stringSet) add(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	s[value] = struct{}{}
}

func (s stringSet) has(value string) bool {
	if len(s) == 0 {
		return false
	}
	_, ok := s[strings.TrimSpace(value)]
	return ok
}
