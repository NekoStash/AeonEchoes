package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"
)

const defaultToolLoopMaxRounds = 6

// ToolExecutionRecord is a stable trace item for one backend tool execution.
type ToolExecutionRecord struct {
	CallID    string          `json:"call_id,omitempty"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Result    json.RawMessage `json:"result"`
}

// ToolLoopResult is the final model response plus executed backend tool trace.
type ToolLoopResult struct {
	Response  provider.ModelResponse `json:"response"`
	Trace     []string               `json:"tool_trace,omitempty"`
	ToolCalls []ToolExecutionRecord  `json:"tool_calls,omitempty"`
}

// ToolExecutor executes whitelisted narrative graph tools emitted by text models.
type ToolExecutor struct {
	store ToolStore
}

func NewToolExecutor(store ToolStore) *ToolExecutor {
	return &ToolExecutor{store: store}
}

func RunToolLoop(ctx context.Context, client provider.TextModelClient, baseReq provider.TextRequest, executor *ToolExecutor, maxRounds int) (ToolLoopResult, error) {
	if client == nil {
		return ToolLoopResult{}, fmt.Errorf("tool loop text client is not configured")
	}
	if executor == nil {
		return ToolLoopResult{}, fmt.Errorf("tool loop executor is not configured")
	}
	if len(baseReq.Tools) == 0 {
		return ToolLoopResult{}, fmt.Errorf("tool loop requires at least one tool spec")
	}
	if maxRounds <= 0 {
		maxRounds = defaultToolLoopMaxRounds
	}
	messages := append([]provider.Message{}, baseReq.Messages...)
	if strings.TrimSpace(baseReq.UserPrompt) != "" {
		messages = append(messages, provider.Message{Role: "user", Content: baseReq.UserPrompt})
	}
	req := baseReq
	req.UserPrompt = ""
	trace := make([]string, 0)
	records := make([]ToolExecutionRecord, 0)
	var final provider.ModelResponse
	for round := 1; round <= maxRounds; round++ {
		req.Messages = messages
		resp, err := client.Generate(ctx, req)
		if err != nil {
			return ToolLoopResult{}, err
		}
		final = resp
		if len(resp.ToolCalls) == 0 {
			return ToolLoopResult{Response: resp, Trace: trace, ToolCalls: records}, nil
		}
		assistant := provider.Message{Role: "assistant", Content: resp.Content, ToolCalls: resp.ToolCalls}
		messages = append(messages, assistant)
		for _, call := range resp.ToolCalls {
			result, err := executor.Execute(ctx, call)
			if err != nil {
				return ToolLoopResult{}, err
			}
			payload, err := json.Marshal(result)
			if err != nil {
				return ToolLoopResult{}, fmt.Errorf("marshal tool %q result: %w", call.Name, err)
			}
			trace = append(trace, fmt.Sprintf("round=%d tool=%s call_id=%s", round, call.Name, strings.TrimSpace(call.ID)))
			records = append(records, ToolExecutionRecord{CallID: call.ID, Name: call.Name, Arguments: call.Arguments, Result: payload})
			messages = append(messages, provider.Message{Role: "tool", Name: call.Name, ToolCallID: call.ID, Content: string(payload)})
		}
	}
	return ToolLoopResult{}, fmt.Errorf("tool loop exceeded max rounds %d; last response finish_reason=%q content_len=%d tool_calls=%d", maxRounds, final.FinishReason, len(final.Content), len(final.ToolCalls))
}

func NarrativeToolSpecs() []provider.ToolSpec {
	return []provider.ToolSpec{
		toolSpec("character.search", "Search project character entities by id, name, alias, summary or traits.", map[string]any{
			"project_id": strSchema("Project id"),
			"query":      strSchema("Optional name or text query"),
			"limit":      intSchema("Maximum results"),
		}),
		toolSpec("character.upsert", "Create or update one character entity by stable id or name.", map[string]any{
			"project_id":   strSchema("Project id"),
			"id":           strSchema("Optional entity id"),
			"name":         strSchema("Character name"),
			"aliases":      arrayStringSchema("Aliases"),
			"summary":      strSchema("Character summary"),
			"traits":       objectStringSchema("Character traits"),
			"importance":   intSchema("Importance score"),
			"status":       strSchema("Status"),
			"worldline_id": strSchema("Optional worldline id"),
			"metadata":     objectStringSchema("Metadata"),
		}, "project_id", "name"),
		toolSpec("relationship.search", "Search relationship graph edges.", map[string]any{
			"project_id": strSchema("Project id"),
			"source_id":  strSchema("Optional source entity id"),
			"target_id":  strSchema("Optional target entity id"),
			"type":       strSchema("Optional relationship type"),
			"query":      strSchema("Optional label query"),
			"limit":      intSchema("Maximum results"),
		}),
		toolSpec("relationship.upsert", "Create or update a relationship graph edge idempotently.", map[string]any{
			"project_id":   strSchema("Project id"),
			"id":           strSchema("Optional edge id"),
			"source_id":    strSchema("Source entity id"),
			"target_id":    strSchema("Target entity id"),
			"type":         strSchema("Relationship type"),
			"label":        strSchema("Relationship label"),
			"weight":       numSchema("Weight"),
			"worldline_id": strSchema("Optional worldline id"),
			"metadata":     objectStringSchema("Metadata"),
		}, "project_id", "source_id", "target_id", "type"),
		toolSpec("event.search", "Search event entities.", map[string]any{
			"project_id": strSchema("Project id"),
			"query":      strSchema("Optional text query"),
			"limit":      intSchema("Maximum results"),
		}),
		toolSpec("event.upsert", "Create or update an event entity and optional participant edges.", map[string]any{
			"project_id":        strSchema("Project id"),
			"id":                strSchema("Optional event id"),
			"name":              strSchema("Event name"),
			"summary":           strSchema("Event summary"),
			"participants":      arrayStringSchema("Participant entity ids"),
			"chronology_key":    strSchema("Chronology key"),
			"worldline_id":      strSchema("Optional worldline id"),
			"source_chapter_id": strSchema("Optional source chapter id"),
			"metadata":          objectStringSchema("Metadata"),
		}, "project_id", "name"),
		toolSpec("timeline.range", "List timeline and time_node entities ordered by chronology_key.", map[string]any{
			"project_id": strSchema("Project id"),
			"start":      strSchema("Optional start chronology key"),
			"end":        strSchema("Optional end chronology key"),
			"limit":      intSchema("Maximum results"),
		}),
		toolSpec("timeline.node.upsert", "Create or update a time_node entity and optional timeline containment edge.", map[string]any{
			"project_id":     strSchema("Project id"),
			"id":             strSchema("Optional node id"),
			"timeline_id":    strSchema("Optional timeline entity id"),
			"name":           strSchema("Node name"),
			"summary":        strSchema("Node summary"),
			"chronology_key": strSchema("Chronology key"),
			"time_scope":     strSchema("Time scope"),
			"metadata":       objectStringSchema("Metadata"),
		}, "project_id", "name"),
		toolSpec("timeline.node.create_before", "Create a prequel time_node before an anchor chronology key.", map[string]any{
			"project_id":  strSchema("Project id"),
			"anchor_id":   strSchema("Anchor time_node id"),
			"anchor_key":  strSchema("Anchor chronology key"),
			"timeline_id": strSchema("Optional timeline entity id"),
			"name":        strSchema("Node name"),
			"summary":     strSchema("Node summary"),
			"metadata":    objectStringSchema("Metadata"),
		}, "project_id", "name"),
		toolSpec("timeline.node.create_after", "Create a time_node after an anchor chronology key.", map[string]any{
			"project_id":  strSchema("Project id"),
			"anchor_id":   strSchema("Anchor time_node id"),
			"anchor_key":  strSchema("Anchor chronology key"),
			"timeline_id": strSchema("Optional timeline entity id"),
			"name":        strSchema("Node name"),
			"summary":     strSchema("Node summary"),
			"metadata":    objectStringSchema("Metadata"),
		}, "project_id", "name"),
		toolSpec("plot_thread.search", "Search plot threads.", map[string]any{
			"project_id": strSchema("Project id"),
			"query":      strSchema("Optional title or summary query"),
			"status":     strSchema("Optional status"),
			"limit":      intSchema("Maximum results"),
		}),
		toolSpec("plot_thread.upsert", "Create or update a plot thread idempotently by id or title.", map[string]any{
			"project_id":         strSchema("Project id"),
			"id":                 strSchema("Optional thread id"),
			"title":              strSchema("Thread title"),
			"summary":            strSchema("Thread summary"),
			"status":             strSchema("Status"),
			"priority":           intSchema("Priority"),
			"related_entity_ids": arrayStringSchema("Related entity ids"),
			"opened_chapter_id":  strSchema("Opened chapter id"),
			"closed_chapter_id":  strSchema("Closed chapter id"),
			"metadata":           objectStringSchema("Metadata"),
		}, "project_id", "title"),
		toolSpec("chapter.list", "List project chapters.", map[string]any{
			"project_id": strSchema("Project id"),
		}),
		toolSpec("chapter.get_range", "Return chapters and latest versions in a number range.", map[string]any{
			"project_id": strSchema("Project id"),
			"start":      intSchema("Start chapter number"),
			"end":        intSchema("End chapter number"),
		}),
		toolSpec("graph.expand", "Expand narrative graph from optional entity ids.", map[string]any{
			"project_id": strSchema("Project id"),
			"entity_ids": arrayStringSchema("Seed entity ids"),
			"depth":      intRangeSchema("Expansion depth", 1, 4),
		}, "project_id"),
	}
}

func (e *ToolExecutor) Execute(ctx context.Context, call provider.ToolCall) (any, error) {
	if e == nil || e.store == nil {
		return nil, fmt.Errorf("tool executor is not configured")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	name := strings.TrimSpace(call.Name)
	if name == "" {
		return nil, fmt.Errorf("tool call name must not be empty")
	}
	args, err := decodeToolArgs(call.Arguments)
	if err != nil {
		return nil, fmt.Errorf("decode tool %q arguments: %w", name, err)
	}
	switch name {
	case "character.search":
		return e.searchEntities(args, []string{"character"})
	case "character.upsert":
		return e.upsertEntity(args, "character")
	case "relationship.search":
		return e.searchRelationships(args)
	case "relationship.upsert":
		return e.upsertRelationship(args, "relationship")
	case "event.search":
		return e.searchEntities(args, []string{"event"})
	case "event.upsert":
		return e.upsertEvent(args)
	case "timeline.range":
		return e.timelineRange(args)
	case "timeline.node.upsert":
		return e.upsertTimelineNode(args, "")
	case "timeline.node.create_before":
		return e.upsertTimelineNode(args, "before")
	case "timeline.node.create_after":
		return e.upsertTimelineNode(args, "after")
	case "plot_thread.search":
		return e.searchPlotThreads(args)
	case "plot_thread.upsert":
		return e.upsertPlotThread(args)
	case "chapter.list":
		return e.listChapters(args)
	case "chapter.get_range":
		return e.chapterRange(args)
	case "graph.expand":
		return e.graphExpand(args)
	default:
		return nil, fmt.Errorf("tool %q is not whitelisted", name)
	}
}

func (e *ToolExecutor) searchEntities(args map[string]any, types []string) (map[string]any, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return nil, err
	}
	query := strings.ToLower(optionalString(args, "query"))
	limit := positiveLimit(optionalInt(args, "limit"), 20)
	entities, err := e.store.ListEntities(projectID)
	if err != nil {
		return nil, err
	}
	typeSet := map[string]struct{}{}
	for _, typ := range types {
		typeSet[typ] = struct{}{}
	}
	matched := make([]domain.Entity, 0)
	for _, entity := range entities {
		if len(typeSet) > 0 {
			if _, ok := typeSet[entity.Type]; !ok {
				continue
			}
		}
		if query != "" && !entityMatches(entity, query) {
			continue
		}
		matched = append(matched, entity)
		if len(matched) >= limit {
			break
		}
	}
	return map[string]any{"items": matched, "count": len(matched)}, nil
}

func (e *ToolExecutor) upsertEntity(args map[string]any, entityType string) (map[string]any, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return nil, err
	}
	name, err := requireToolString(args, "name")
	if err != nil {
		return nil, err
	}
	entities, err := e.store.ListEntities(projectID)
	if err != nil {
		return nil, err
	}
	id := optionalString(args, "id")
	var entity domain.Entity
	action := "created"
	if id != "" {
		for _, existing := range entities {
			if existing.ID == id {
				entity = existing
				action = "updated"
				break
			}
		}
	}
	if entity.ID == "" {
		for _, existing := range entities {
			if existing.Type == entityType && strings.EqualFold(strings.TrimSpace(existing.Name), name) {
				entity = existing
				action = "updated"
				break
			}
		}
	}
	if entity.ID == "" && id != "" {
		entity.ID = id
	}
	entity.ProjectID = projectID
	entity.Name = name
	entity.Type = entityType
	entity.WorldlineID = optionalString(args, "worldline_id")
	if entity.WorldlineID == "" && action == "updated" {
		// keep existing worldline id
		for _, existing := range entities {
			if existing.ID == entity.ID {
				entity.WorldlineID = existing.WorldlineID
				break
			}
		}
	}
	if aliases := optionalStringSlice(args, "aliases"); len(aliases) > 0 {
		entity.Aliases = aliases
	}
	if summary := optionalString(args, "summary"); summary != "" {
		entity.Summary = summary
	}
	if traits := optionalStringMap(args, "traits"); len(traits) > 0 {
		entity.Traits = mergeStringMap(entity.Traits, traits)
	}
	if metadata := optionalStringMap(args, "metadata"); len(metadata) > 0 {
		entity.Metadata = mergeStringMap(entity.Metadata, metadata)
	}
	if importance := optionalInt(args, "importance"); importance > 0 {
		entity.Importance = importance
	}
	if status := optionalString(args, "status"); status != "" {
		entity.Status = status
	} else if entity.Status == "" {
		entity.Status = "active"
	}
	saved, err := e.store.SaveEntity(entity)
	if err != nil {
		return nil, err
	}
	return map[string]any{"action": action, "entity": saved}, nil
}

func (e *ToolExecutor) searchRelationships(args map[string]any) (map[string]any, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return nil, err
	}
	expansion, err := e.store.ExpandGraph(projectID, nil, 1)
	if err != nil {
		return nil, err
	}
	sourceID := optionalString(args, "source_id")
	targetID := optionalString(args, "target_id")
	typ := optionalString(args, "type")
	query := strings.ToLower(optionalString(args, "query"))
	limit := positiveLimit(optionalInt(args, "limit"), 20)
	items := make([]domain.GraphEdge, 0)
	for _, edge := range expansion.Edges {
		if sourceID != "" && edge.SourceEntityID != sourceID {
			continue
		}
		if targetID != "" && edge.TargetEntityID != targetID {
			continue
		}
		if typ != "" && edge.Type != typ {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(edge.Label+" "+edge.Type), query) {
			continue
		}
		items = append(items, edge)
		if len(items) >= limit {
			break
		}
	}
	return map[string]any{"items": items, "count": len(items)}, nil
}

func (e *ToolExecutor) upsertRelationship(args map[string]any, defaultType string) (map[string]any, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return nil, err
	}
	sourceID := firstToolString(args, "source_id", "source_entity_id")
	targetID := firstToolString(args, "target_id", "target_entity_id")
	if strings.TrimSpace(sourceID) == "" || strings.TrimSpace(targetID) == "" {
		return nil, fmt.Errorf("relationship.upsert source_id and target_id must not be empty")
	}
	typ := optionalString(args, "type")
	if typ == "" {
		typ = defaultType
	}
	expansion, err := e.store.ExpandGraph(projectID, nil, 1)
	if err != nil {
		return nil, err
	}
	id := optionalString(args, "id")
	var edge domain.GraphEdge
	action := "created"
	knownEntities := map[string]struct{}{}
	for _, entity := range expansion.Entities {
		knownEntities[entity.ID] = struct{}{}
	}
	if _, ok := knownEntities[sourceID]; !ok {
		return nil, fmt.Errorf("relationship.upsert source_id %q not found", sourceID)
	}
	if _, ok := knownEntities[targetID]; !ok {
		return nil, fmt.Errorf("relationship.upsert target_id %q not found", targetID)
	}
	for _, existing := range expansion.Edges {
		if id != "" && existing.ID == id {
			edge = existing
			action = "updated"
			break
		}
		if id == "" && existing.SourceEntityID == sourceID && existing.TargetEntityID == targetID && existing.Type == typ {
			edge = existing
			action = "updated"
			break
		}
	}
	if edge.ID == "" && id != "" {
		edge.ID = id
	}
	edge.ProjectID = projectID
	edge.SourceEntityID = sourceID
	edge.TargetEntityID = targetID
	edge.Type = typ
	if label := optionalString(args, "label"); label != "" {
		edge.Label = label
	}
	if edge.Label == "" {
		edge.Label = typ
	}
	if weight := optionalFloat(args, "weight"); weight > 0 {
		edge.Weight = weight
	} else if edge.Weight <= 0 {
		edge.Weight = 1
	}
	if worldlineID := optionalString(args, "worldline_id"); worldlineID != "" {
		edge.WorldlineID = worldlineID
	}
	if evidence := optionalStringSlice(args, "evidence_fact_ids"); len(evidence) > 0 {
		edge.EvidenceFactIDs = evidence
	}
	if metadata := optionalStringMap(args, "metadata"); len(metadata) > 0 {
		edge.Metadata = mergeStringMap(edge.Metadata, metadata)
	}
	saved, err := e.store.SaveGraphEdge(edge)
	if err != nil {
		return nil, err
	}
	return map[string]any{"action": action, "edge": saved}, nil
}

func (e *ToolExecutor) upsertEvent(args map[string]any) (map[string]any, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return nil, err
	}
	result, err := e.upsertEntityWithMetadata(args, "event", func(metadata map[string]string) map[string]string {
		if chronologyKey := optionalString(args, "chronology_key"); chronologyKey != "" {
			metadata["chronology_key"] = chronologyKey
		}
		if chapterID := optionalString(args, "source_chapter_id"); chapterID != "" {
			metadata["source_chapter_id"] = chapterID
		}
		return metadata
	})
	if err != nil {
		return nil, err
	}
	entity, ok := result["entity"].(domain.Entity)
	if !ok {
		return nil, fmt.Errorf("event.upsert internal entity result missing")
	}
	participants := optionalStringSlice(args, "participants")
	edges := make([]domain.GraphEdge, 0, len(participants))
	for _, participantID := range participants {
		edgeResult, err := e.upsertRelationship(map[string]any{
			"project_id": projectID,
			"source_id":  entity.ID,
			"target_id":  participantID,
			"type":       "event_participant",
			"label":      "参与事件",
		}, "event_participant")
		if err != nil {
			return nil, err
		}
		if edge, ok := edgeResult["edge"].(domain.GraphEdge); ok {
			edges = append(edges, edge)
		}
	}
	result["participant_edges"] = edges
	return result, nil
}

func (e *ToolExecutor) upsertEntityWithMetadata(args map[string]any, entityType string, enrich func(map[string]string) map[string]string) (map[string]any, error) {
	metadata := optionalStringMap(args, "metadata")
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata = enrich(metadata)
	args["metadata"] = metadata
	return e.upsertEntity(args, entityType)
}

func (e *ToolExecutor) timelineRange(args map[string]any) (map[string]any, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return nil, err
	}
	start := optionalString(args, "start")
	end := optionalString(args, "end")
	limit := positiveLimit(optionalInt(args, "limit"), 50)
	entities, err := e.store.ListEntities(projectID)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Entity, 0)
	for _, entity := range entities {
		if entity.Type != "time_node" && entity.Type != "timeline" {
			continue
		}
		key := chronologyKey(entity)
		if start != "" && key < start {
			continue
		}
		if end != "" && key > end {
			continue
		}
		items = append(items, entity)
	}
	sort.Slice(items, func(i, j int) bool {
		ki := chronologyKey(items[i])
		kj := chronologyKey(items[j])
		if ki == kj {
			return items[i].ID < items[j].ID
		}
		return ki < kj
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return map[string]any{"items": items, "count": len(items)}, nil
}

func (e *ToolExecutor) upsertTimelineNode(args map[string]any, mode string) (map[string]any, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return nil, err
	}
	metadata := optionalStringMap(args, "metadata")
	if metadata == nil {
		metadata = map[string]string{}
	}
	chronology := optionalString(args, "chronology_key")
	if chronology == "" {
		anchorKey, err := e.resolveAnchorKey(projectID, optionalString(args, "anchor_id"), optionalString(args, "anchor_key"))
		if err != nil {
			return nil, err
		}
		switch mode {
		case "before":
			chronology = predecessorChronologyKey(anchorKey)
			metadata["time_scope"] = "prequel"
		case "after":
			chronology = successorChronologyKey(anchorKey)
		default:
			return nil, fmt.Errorf("timeline node chronology_key must not be empty")
		}
	}
	if scope := optionalString(args, "time_scope"); scope != "" {
		metadata["time_scope"] = scope
	}
	metadata["chronology_key"] = chronology
	args["metadata"] = metadata
	result, err := e.upsertEntity(args, "time_node")
	if err != nil {
		return nil, err
	}
	node, ok := result["entity"].(domain.Entity)
	if !ok {
		return nil, fmt.Errorf("timeline.node internal entity result missing")
	}
	if timelineID := optionalString(args, "timeline_id"); timelineID != "" {
		edgeResult, err := e.upsertRelationship(map[string]any{
			"project_id": projectID,
			"source_id":  timelineID,
			"target_id":  node.ID,
			"type":       "timeline_contains",
			"label":      "包含时间节点",
		}, "timeline_contains")
		if err != nil {
			return nil, err
		}
		result["timeline_edge"] = edgeResult["edge"]
	}
	return result, nil
}

func (e *ToolExecutor) resolveAnchorKey(projectID, anchorID, anchorKey string) (string, error) {
	if strings.TrimSpace(anchorKey) != "" {
		return strings.TrimSpace(anchorKey), nil
	}
	if strings.TrimSpace(anchorID) == "" {
		return "", fmt.Errorf("timeline anchor_id or anchor_key must not be empty")
	}
	entities, err := e.store.ListEntities(projectID)
	if err != nil {
		return "", err
	}
	for _, entity := range entities {
		if entity.ID == anchorID {
			key := chronologyKey(entity)
			if key == "" {
				return "", fmt.Errorf("anchor time node %q has no chronology_key", anchorID)
			}
			return key, nil
		}
	}
	return "", fmt.Errorf("anchor time node %q not found", anchorID)
}

func (e *ToolExecutor) searchPlotThreads(args map[string]any) (map[string]any, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return nil, err
	}
	query := strings.ToLower(optionalString(args, "query"))
	status := optionalString(args, "status")
	limit := positiveLimit(optionalInt(args, "limit"), 20)
	threads, err := e.store.ListPlotThreads(projectID)
	if err != nil {
		return nil, err
	}
	items := make([]domain.PlotThread, 0)
	for _, thread := range threads {
		if status != "" && thread.Status != status {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(thread.Title+" "+thread.Summary), query) {
			continue
		}
		items = append(items, thread)
		if len(items) >= limit {
			break
		}
	}
	return map[string]any{"items": items, "count": len(items)}, nil
}

func (e *ToolExecutor) upsertPlotThread(args map[string]any) (map[string]any, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return nil, err
	}
	title, err := requireToolString(args, "title")
	if err != nil {
		return nil, err
	}
	threads, err := e.store.ListPlotThreads(projectID)
	if err != nil {
		return nil, err
	}
	id := optionalString(args, "id")
	var thread domain.PlotThread
	action := "created"
	for _, existing := range threads {
		if (id != "" && existing.ID == id) || (id == "" && strings.EqualFold(strings.TrimSpace(existing.Title), title)) {
			thread = existing
			action = "updated"
			break
		}
	}
	if thread.ID == "" && id != "" {
		thread.ID = id
	}
	thread.ProjectID = projectID
	thread.Title = title
	if summary := optionalString(args, "summary"); summary != "" {
		thread.Summary = summary
	}
	if status := optionalString(args, "status"); status != "" {
		thread.Status = status
	} else if thread.Status == "" {
		thread.Status = "open"
	}
	if priority := optionalInt(args, "priority"); priority != 0 {
		thread.Priority = priority
	}
	if ids := optionalStringSlice(args, "related_entity_ids"); len(ids) > 0 {
		thread.RelatedEntityIDs = ids
	}
	if opened := optionalString(args, "opened_chapter_id"); opened != "" {
		thread.OpenedChapterID = opened
	}
	if closed := optionalString(args, "closed_chapter_id"); closed != "" {
		thread.ClosedChapterID = closed
	}
	if metadata := optionalStringMap(args, "metadata"); len(metadata) > 0 {
		thread.Metadata = mergeStringMap(thread.Metadata, metadata)
	}
	saved, err := e.store.SavePlotThread(thread)
	if err != nil {
		return nil, err
	}
	return map[string]any{"action": action, "plot_thread": saved}, nil
}

func (e *ToolExecutor) listChapters(args map[string]any) (map[string]any, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return nil, err
	}
	chapters, err := e.store.ListChapters(projectID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": chapters, "count": len(chapters)}, nil
}

func (e *ToolExecutor) chapterRange(args map[string]any) (map[string]any, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return nil, err
	}
	start := optionalInt(args, "start")
	end := optionalInt(args, "end")
	if start <= 0 {
		start = 1
	}
	if end <= 0 {
		end = start
	}
	if end < start {
		return nil, fmt.Errorf("chapter.get_range end must be >= start")
	}
	chapters, err := e.store.ListChapters(projectID)
	if err != nil {
		return nil, err
	}
	versions, err := e.store.ListChapterVersions(projectID, "")
	if err != nil {
		return nil, err
	}
	latestByChapter := map[string]domain.ChapterVersion{}
	for _, version := range versions {
		if _, ok := latestByChapter[version.ChapterID]; !ok {
			latestByChapter[version.ChapterID] = version
		}
	}
	items := make([]map[string]any, 0)
	for _, chapter := range chapters {
		if chapter.Number < start || chapter.Number > end {
			continue
		}
		items = append(items, map[string]any{"chapter": chapter, "latest_version": latestByChapter[chapter.ID]})
	}
	return map[string]any{"items": items, "count": len(items)}, nil
}

func (e *ToolExecutor) graphExpand(args map[string]any) (domain.GraphExpansion, error) {
	projectID, err := requireToolString(args, "project_id")
	if err != nil {
		return domain.GraphExpansion{}, err
	}
	depth := 1
	if _, provided := args["depth"]; provided {
		depth = optionalInt(args, "depth")
		if depth < 1 || depth > 4 {
			return domain.GraphExpansion{}, fmt.Errorf("tool argument depth must be between 1 and 4")
		}
	}
	return e.store.ExpandGraph(projectID, optionalStringSlice(args, "entity_ids"), depth)
}

func decodeToolArgs(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args == nil {
		return nil, fmt.Errorf("arguments must be a JSON object")
	}
	return args, nil
}

func toolSpec(name, description string, properties map[string]any, required ...string) provider.ToolSpec {
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	payload, err := json.Marshal(schema)
	if err != nil {
		panic(fmt.Sprintf("marshal tool schema %s: %v", name, err))
	}
	return provider.ToolSpec{Name: name, Description: description, Parameters: payload}
}

func strSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}
func intSchema(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}
func intRangeSchema(description string, minimum, maximum int) map[string]any {
	return map[string]any{"type": "integer", "description": description, "minimum": minimum, "maximum": maximum}
}
func numSchema(description string) map[string]any {
	return map[string]any{"type": "number", "description": description}
}
func arrayStringSchema(description string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": description}
}
func objectStringSchema(description string) map[string]any {
	return map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}, "description": description}
}

func requireToolString(args map[string]any, key string) (string, error) {
	value := optionalString(args, key)
	if value == "" {
		return "", fmt.Errorf("tool argument %s must not be empty", key)
	}
	return value, nil
}

func optionalString(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	s, ok := value.(string)
	if !ok {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return strings.TrimSpace(s)
}

func firstToolString(args map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := optionalString(args, key); value != "" {
			return value
		}
	}
	return ""
}

func optionalInt(args map[string]any, key string) int {
	value, ok := args[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		var i int
		_, _ = fmt.Sscanf(fmt.Sprint(value), "%d", &i)
		return i
	}
}

func optionalFloat(args map[string]any, key string) float64 {
	value, ok := args[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		var f float64
		_, _ = fmt.Sscanf(fmt.Sprint(value), "%f", &f)
		return f
	}
}

func optionalStringSlice(args map[string]any, key string) []string {
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}
	items := make([]string, 0)
	switch v := value.(type) {
	case []string:
		for _, item := range v {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				items = append(items, trimmed)
			}
		}
	case []any:
		for _, item := range v {
			if trimmed := strings.TrimSpace(fmt.Sprint(item)); trimmed != "" {
				items = append(items, trimmed)
			}
		}
	}
	return items
}

func optionalStringMap(args map[string]any, key string) map[string]string {
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}
	result := map[string]string{}
	switch v := value.(type) {
	case map[string]string:
		for key, val := range v {
			if strings.TrimSpace(key) != "" {
				result[key] = val
			}
		}
	case map[string]any:
		for key, val := range v {
			if strings.TrimSpace(key) != "" && val != nil {
				result[key] = fmt.Sprint(val)
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func mergeStringMap(base, overlay map[string]string) map[string]string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	merged := make(map[string]string, len(base)+len(overlay))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		merged[key] = value
	}
	return merged
}

func entityMatches(entity domain.Entity, query string) bool {
	if strings.Contains(strings.ToLower(entity.ID+" "+entity.Name+" "+entity.Summary+" "+entity.Type), query) {
		return true
	}
	for _, alias := range entity.Aliases {
		if strings.Contains(strings.ToLower(alias), query) {
			return true
		}
	}
	for key, value := range entity.Traits {
		if strings.Contains(strings.ToLower(key+" "+value), query) {
			return true
		}
	}
	for key, value := range entity.Metadata {
		if strings.Contains(strings.ToLower(key+" "+value), query) {
			return true
		}
	}
	return false
}

func chronologyKey(entity domain.Entity) string {
	if entity.Metadata == nil {
		return ""
	}
	return strings.TrimSpace(entity.Metadata["chronology_key"])
}

func predecessorChronologyKey(anchor string) string {
	anchor = strings.TrimSpace(anchor)
	if anchor == "" {
		return "!prequel:000000"
	}
	return "!prequel:" + anchor + ":001"
}

func successorChronologyKey(anchor string) string {
	anchor = strings.TrimSpace(anchor)
	if anchor == "" {
		return "after:000001"
	}
	return anchor + ":after:001"
}

func positiveLimit(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	if value > 200 {
		return 200
	}
	return value
}
