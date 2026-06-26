package postgres

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"aeonechoes/server/internal/domain"
)

func (s *Store) SaveWorldline(item domain.Worldline) (domain.Worldline, error) {
	if err := requireStore(s); err != nil {
		return domain.Worldline{}, err
	}
	if strings.TrimSpace(item.ProjectID) == "" {
		return domain.Worldline{}, fmt.Errorf("worldline project_id must not be empty")
	}
	if _, err := s.GetProject(item.ProjectID); err != nil {
		return domain.Worldline{}, err
	}
	if item.ID == "" {
		id, err := s.NewID("worldline")
		if err != nil {
			return domain.Worldline{}, fmt.Errorf("generate worldline id: %w", err)
		}
		item.ID = id
		item.CreatedAt = now()
	} else if existing, err := s.getWorldline(item.ID); err == nil {
		item.CreatedAt = existing.CreatedAt
	} else if !strings.Contains(err.Error(), "not found") {
		return domain.Worldline{}, err
	} else {
		item.CreatedAt = now()
	}
	item.UpdatedAt = now()
	metadata, err := jsonbOrEmptyObject(item.Metadata)
	if err != nil {
		return domain.Worldline{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO worldlines(id, project_id, name, summary, canonical, metadata, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (id) DO UPDATE SET project_id=EXCLUDED.project_id, name=EXCLUDED.name, summary=EXCLUDED.summary, canonical=EXCLUDED.canonical, metadata=EXCLUDED.metadata, updated_at=EXCLUDED.updated_at`, item.ID, item.ProjectID, item.Name, item.Summary, item.Canonical, metadata, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return domain.Worldline{}, fmt.Errorf("save worldline %q: %w", item.ID, err)
	}
	return item, nil
}

func (s *Store) SaveEntity(item domain.Entity) (domain.Entity, error) {
	if err := requireStore(s); err != nil {
		return domain.Entity{}, err
	}
	if strings.TrimSpace(item.ProjectID) == "" || strings.TrimSpace(item.Name) == "" {
		return domain.Entity{}, fmt.Errorf("entity project_id and name must not be empty")
	}
	if item.ID == "" {
		id, err := s.NewID("entity")
		if err != nil {
			return domain.Entity{}, fmt.Errorf("generate entity id: %w", err)
		}
		item.ID = id
		item.CreatedAt = now()
	} else if existing, err := s.getEntity(item.ID); err == nil {
		item.CreatedAt = existing.CreatedAt
	} else if !strings.Contains(err.Error(), "not found") {
		return domain.Entity{}, err
	} else {
		item.CreatedAt = now()
	}
	item.UpdatedAt = now()
	aliases, traits, metadata, err := entityJSON(item)
	if err != nil {
		return domain.Entity{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO narrative_entities(id, project_id, worldline_id, name, type, aliases, summary, traits, importance, status, metadata, created_at, updated_at)
VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (id) DO UPDATE SET project_id=EXCLUDED.project_id, worldline_id=EXCLUDED.worldline_id, name=EXCLUDED.name, type=EXCLUDED.type,
    aliases=EXCLUDED.aliases, summary=EXCLUDED.summary, traits=EXCLUDED.traits, importance=EXCLUDED.importance, status=EXCLUDED.status,
    metadata=EXCLUDED.metadata, updated_at=EXCLUDED.updated_at`, item.ID, item.ProjectID, item.WorldlineID, item.Name, item.Type, aliases, item.Summary, traits, item.Importance, item.Status, metadata, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return domain.Entity{}, fmt.Errorf("save entity %q: %w", item.ID, err)
	}
	return item, nil
}

func (s *Store) SaveFact(item domain.Fact) (domain.Fact, error) {
	if err := requireStore(s); err != nil {
		return domain.Fact{}, err
	}
	if strings.TrimSpace(item.ProjectID) == "" || strings.TrimSpace(item.Claim) == "" {
		return domain.Fact{}, fmt.Errorf("fact project_id and claim must not be empty")
	}
	if item.ID == "" {
		id, err := s.NewID("fact")
		if err != nil {
			return domain.Fact{}, fmt.Errorf("generate fact id: %w", err)
		}
		item.ID = id
		item.CreatedAt = now()
	} else if existing, err := s.getFact(item.ID); err == nil {
		item.CreatedAt = existing.CreatedAt
	} else if !strings.Contains(err.Error(), "not found") {
		return domain.Fact{}, err
	} else {
		item.CreatedAt = now()
	}
	item.UpdatedAt = now()
	metadata, err := jsonbOrEmptyObject(item.Metadata)
	if err != nil {
		return domain.Fact{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO narrative_facts(id, project_id, worldline_id, entity_id, chapter_id, chapter_version_id, claim, source, confidence, status, embedding_ref, metadata, created_at, updated_at)
VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
ON CONFLICT (id) DO UPDATE SET project_id=EXCLUDED.project_id, worldline_id=EXCLUDED.worldline_id, entity_id=EXCLUDED.entity_id,
    chapter_id=EXCLUDED.chapter_id, chapter_version_id=EXCLUDED.chapter_version_id, claim=EXCLUDED.claim, source=EXCLUDED.source,
    confidence=EXCLUDED.confidence, status=EXCLUDED.status, embedding_ref=EXCLUDED.embedding_ref, metadata=EXCLUDED.metadata, updated_at=EXCLUDED.updated_at`, item.ID, item.ProjectID, item.WorldlineID, item.EntityID, item.ChapterID, item.ChapterVersionID, item.Claim, item.Source, item.Confidence, item.Status, item.EmbeddingRef, metadata, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return domain.Fact{}, fmt.Errorf("save fact %q: %w", item.ID, err)
	}
	return item, nil
}

func (s *Store) SaveGraphEdge(item domain.GraphEdge) (domain.GraphEdge, error) {
	if err := requireStore(s); err != nil {
		return domain.GraphEdge{}, err
	}
	if strings.TrimSpace(item.ProjectID) == "" || strings.TrimSpace(item.SourceEntityID) == "" || strings.TrimSpace(item.TargetEntityID) == "" {
		return domain.GraphEdge{}, fmt.Errorf("graph edge project_id, source_entity_id and target_entity_id must not be empty")
	}
	if item.ID == "" {
		id, err := s.NewID("edge")
		if err != nil {
			return domain.GraphEdge{}, fmt.Errorf("generate graph edge id: %w", err)
		}
		item.ID = id
		item.CreatedAt = now()
	} else if existing, err := s.getEdge(item.ID); err == nil {
		item.CreatedAt = existing.CreatedAt
	} else if !strings.Contains(err.Error(), "not found") {
		return domain.GraphEdge{}, err
	} else {
		item.CreatedAt = now()
	}
	item.UpdatedAt = now()
	evidence, metadata, err := edgeJSON(item)
	if err != nil {
		return domain.GraphEdge{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO graph_edges(id, project_id, worldline_id, source_entity_id, target_entity_id, type, label, weight, evidence_fact_ids, metadata, created_at, updated_at)
VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (id) DO UPDATE SET project_id=EXCLUDED.project_id, worldline_id=EXCLUDED.worldline_id, source_entity_id=EXCLUDED.source_entity_id,
    target_entity_id=EXCLUDED.target_entity_id, type=EXCLUDED.type, label=EXCLUDED.label, weight=EXCLUDED.weight,
    evidence_fact_ids=EXCLUDED.evidence_fact_ids, metadata=EXCLUDED.metadata, updated_at=EXCLUDED.updated_at`, item.ID, item.ProjectID, item.WorldlineID, item.SourceEntityID, item.TargetEntityID, item.Type, item.Label, item.Weight, evidence, metadata, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return domain.GraphEdge{}, fmt.Errorf("save graph edge %q: %w", item.ID, err)
	}
	return item, nil
}

func (s *Store) SavePlotThread(item domain.PlotThread) (domain.PlotThread, error) {
	if err := requireStore(s); err != nil {
		return domain.PlotThread{}, err
	}
	if strings.TrimSpace(item.ProjectID) == "" || strings.TrimSpace(item.Title) == "" {
		return domain.PlotThread{}, fmt.Errorf("plot thread project_id and title must not be empty")
	}
	if item.ID == "" {
		id, err := s.NewID("thread")
		if err != nil {
			return domain.PlotThread{}, fmt.Errorf("generate plot thread id: %w", err)
		}
		item.ID = id
		item.CreatedAt = now()
	} else if existing, err := s.getPlotThread(item.ID); err == nil {
		item.CreatedAt = existing.CreatedAt
	} else if !strings.Contains(err.Error(), "not found") {
		return domain.PlotThread{}, err
	} else {
		item.CreatedAt = now()
	}
	item.UpdatedAt = now()
	related, metadata, err := plotThreadJSON(item)
	if err != nil {
		return domain.PlotThread{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO plot_threads(id, project_id, worldline_id, title, summary, status, priority, related_entity_ids, opened_chapter_id, closed_chapter_id, metadata, created_at, updated_at)
VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (id) DO UPDATE SET project_id=EXCLUDED.project_id, worldline_id=EXCLUDED.worldline_id, title=EXCLUDED.title,
    summary=EXCLUDED.summary, status=EXCLUDED.status, priority=EXCLUDED.priority, related_entity_ids=EXCLUDED.related_entity_ids,
    opened_chapter_id=EXCLUDED.opened_chapter_id, closed_chapter_id=EXCLUDED.closed_chapter_id, metadata=EXCLUDED.metadata, updated_at=EXCLUDED.updated_at`, item.ID, item.ProjectID, item.WorldlineID, item.Title, item.Summary, item.Status, item.Priority, related, item.OpenedChapterID, item.ClosedChapterID, metadata, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return domain.PlotThread{}, fmt.Errorf("save plot thread %q: %w", item.ID, err)
	}
	return item, nil
}

func (s *Store) ListEntities(projectID string) ([]domain.Entity, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(context.Background(), entitySelectSQL()+` WHERE project_id=$1 ORDER BY name ASC, id ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list entities for project %q: %w", projectID, err)
	}
	defer rows.Close()
	items := make([]domain.Entity, 0)
	for rows.Next() {
		item, err := scanEntity(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entities for project %q: %w", projectID, err)
	}
	return items, nil
}

func (s *Store) ListFacts(projectID string) ([]domain.Fact, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(context.Background(), factSelectSQL()+` WHERE project_id=$1 ORDER BY created_at ASC, id ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list facts for project %q: %w", projectID, err)
	}
	defer rows.Close()
	items := make([]domain.Fact, 0)
	for rows.Next() {
		item, err := scanFact(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate facts for project %q: %w", projectID, err)
	}
	return items, nil
}

func (s *Store) ListPlotThreads(projectID string) ([]domain.PlotThread, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(context.Background(), plotThreadSelectSQL()+` WHERE project_id=$1 ORDER BY priority DESC, created_at ASC, id ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list plot threads for project %q: %w", projectID, err)
	}
	defer rows.Close()
	items := make([]domain.PlotThread, 0)
	for rows.Next() {
		item, err := scanPlotThread(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plot threads for project %q: %w", projectID, err)
	}
	return items, nil
}

func (s *Store) ExpandGraph(projectID string, entityIDs []string, depth int) (domain.GraphExpansion, error) {
	if err := requireStore(s); err != nil {
		return domain.GraphExpansion{}, err
	}
	if strings.TrimSpace(projectID) == "" {
		return domain.GraphExpansion{}, fmt.Errorf("project_id must not be empty")
	}
	if depth < 0 {
		return domain.GraphExpansion{}, fmt.Errorf("graph expansion depth must not be negative")
	}
	if depth == 0 {
		depth = 1
	}
	if _, err := s.GetProject(projectID); err != nil {
		return domain.GraphExpansion{}, err
	}
	allEntities, err := s.ListEntities(projectID)
	if err != nil {
		return domain.GraphExpansion{}, err
	}
	allEdges, err := s.listGraphEdges(projectID)
	if err != nil {
		return domain.GraphExpansion{}, err
	}
	allFacts, err := s.ListFacts(projectID)
	if err != nil {
		return domain.GraphExpansion{}, err
	}
	entityByID := make(map[string]domain.Entity, len(allEntities))
	for _, entity := range allEntities {
		entityByID[entity.ID] = entity
	}
	selected := map[string]bool{}
	frontier := map[string]bool{}
	if len(entityIDs) == 0 {
		for id := range entityByID {
			frontier[id] = true
		}
	} else {
		for _, id := range entityIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				frontier[id] = true
			}
		}
	}
	selectedEdges := map[string]bool{}
	for d := 0; d < depth && len(frontier) > 0; d++ {
		next := map[string]bool{}
		for id := range frontier {
			if _, ok := entityByID[id]; !ok {
				continue
			}
			selected[id] = true
			for _, edge := range allEdges {
				if edge.SourceEntityID == id || edge.TargetEntityID == id {
					selectedEdges[edge.ID] = true
					other := edge.TargetEntityID
					if other == id {
						other = edge.SourceEntityID
					}
					if !selected[other] {
						next[other] = true
					}
				}
			}
		}
		frontier = next
	}
	entities := make([]domain.Entity, 0, len(selected))
	for id := range selected {
		if entity, ok := entityByID[id]; ok {
			entities = append(entities, entity)
		}
	}
	edges := make([]domain.GraphEdge, 0, len(selectedEdges))
	factIDs := map[string]bool{}
	for _, edge := range allEdges {
		if selectedEdges[edge.ID] {
			edges = append(edges, edge)
			for _, factID := range edge.EvidenceFactIDs {
				factIDs[factID] = true
			}
		}
	}
	facts := make([]domain.Fact, 0)
	for _, fact := range allFacts {
		if factIDs[fact.ID] || selected[fact.EntityID] {
			facts = append(facts, fact)
		}
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].Name < entities[j].Name })
	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })
	sort.Slice(facts, func(i, j int) bool { return facts[i].CreatedAt.Before(facts[j].CreatedAt) })
	return domain.GraphExpansion{ProjectID: projectID, Depth: depth, Entities: entities, Edges: edges, Facts: facts}, nil
}

func (s *Store) listGraphEdges(projectID string) ([]domain.GraphEdge, error) {
	rows, err := s.pool.Query(context.Background(), edgeSelectSQL()+` WHERE project_id=$1 ORDER BY id ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list graph edges for project %q: %w", projectID, err)
	}
	defer rows.Close()
	items := make([]domain.GraphEdge, 0)
	for rows.Next() {
		item, err := scanGraphEdge(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate graph edges for project %q: %w", projectID, err)
	}
	return items, nil
}

func (s *Store) getWorldline(id string) (domain.Worldline, error) {
	row := s.pool.QueryRow(context.Background(), `SELECT id, project_id, name, summary, canonical, metadata, created_at, updated_at FROM worldlines WHERE id=$1`, id)
	var item domain.Worldline
	var metadata []byte
	if err := row.Scan(&item.ID, &item.ProjectID, &item.Name, &item.Summary, &item.Canonical, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if isNoRows(err) {
			return domain.Worldline{}, fmt.Errorf("worldline %q not found", id)
		}
		return domain.Worldline{}, fmt.Errorf("get worldline %q: %w", id, err)
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.Worldline{}, err
	}
	item.Metadata = parsedMetadata
	return item, nil
}

func (s *Store) getEntity(id string) (domain.Entity, error) {
	row := s.pool.QueryRow(context.Background(), entitySelectSQL()+` WHERE id=$1`, id)
	item, err := scanEntity(row)
	if err != nil {
		if isNoRows(err) {
			return domain.Entity{}, fmt.Errorf("entity %q not found", id)
		}
		return domain.Entity{}, fmt.Errorf("get entity %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) getFact(id string) (domain.Fact, error) {
	row := s.pool.QueryRow(context.Background(), factSelectSQL()+` WHERE id=$1`, id)
	item, err := scanFact(row)
	if err != nil {
		if isNoRows(err) {
			return domain.Fact{}, fmt.Errorf("fact %q not found", id)
		}
		return domain.Fact{}, fmt.Errorf("get fact %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) getEdge(id string) (domain.GraphEdge, error) {
	row := s.pool.QueryRow(context.Background(), edgeSelectSQL()+` WHERE id=$1`, id)
	item, err := scanGraphEdge(row)
	if err != nil {
		if isNoRows(err) {
			return domain.GraphEdge{}, fmt.Errorf("graph edge %q not found", id)
		}
		return domain.GraphEdge{}, fmt.Errorf("get graph edge %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) getPlotThread(id string) (domain.PlotThread, error) {
	row := s.pool.QueryRow(context.Background(), plotThreadSelectSQL()+` WHERE id=$1`, id)
	item, err := scanPlotThread(row)
	if err != nil {
		if isNoRows(err) {
			return domain.PlotThread{}, fmt.Errorf("plot thread %q not found", id)
		}
		return domain.PlotThread{}, fmt.Errorf("get plot thread %q: %w", id, err)
	}
	return item, nil
}

func entityJSON(item domain.Entity) ([]byte, []byte, []byte, error) {
	aliases, err := jsonbOrEmptyArray(item.Aliases)
	if err != nil {
		return nil, nil, nil, err
	}
	traits, err := jsonbOrEmptyObject(item.Traits)
	if err != nil {
		return nil, nil, nil, err
	}
	metadata, err := jsonbOrEmptyObject(item.Metadata)
	if err != nil {
		return nil, nil, nil, err
	}
	return aliases, traits, metadata, nil
}

func edgeJSON(item domain.GraphEdge) ([]byte, []byte, error) {
	evidence, err := jsonbOrEmptyArray(item.EvidenceFactIDs)
	if err != nil {
		return nil, nil, err
	}
	metadata, err := jsonbOrEmptyObject(item.Metadata)
	if err != nil {
		return nil, nil, err
	}
	return evidence, metadata, nil
}

func plotThreadJSON(item domain.PlotThread) ([]byte, []byte, error) {
	related, err := jsonbOrEmptyArray(item.RelatedEntityIDs)
	if err != nil {
		return nil, nil, err
	}
	metadata, err := jsonbOrEmptyObject(item.Metadata)
	if err != nil {
		return nil, nil, err
	}
	return related, metadata, nil
}

func entitySelectSQL() string {
	return `SELECT id, project_id, COALESCE(worldline_id, ''), name, type, aliases, summary, traits, importance, status, metadata, created_at, updated_at FROM narrative_entities`
}

func factSelectSQL() string {
	return `SELECT id, project_id, COALESCE(worldline_id, ''), COALESCE(entity_id, ''), chapter_id, chapter_version_id, claim, source, confidence, status, embedding_ref, metadata, created_at, updated_at FROM narrative_facts`
}

func edgeSelectSQL() string {
	return `SELECT id, project_id, COALESCE(worldline_id, ''), source_entity_id, target_entity_id, type, label, weight, evidence_fact_ids, metadata, created_at, updated_at FROM graph_edges`
}

func plotThreadSelectSQL() string {
	return `SELECT id, project_id, COALESCE(worldline_id, ''), title, summary, status, priority, related_entity_ids, opened_chapter_id, closed_chapter_id, metadata, created_at, updated_at FROM plot_threads`
}

type graphScanner interface{ Scan(dest ...any) error }

func scanEntity(scanner graphScanner) (domain.Entity, error) {
	var item domain.Entity
	var aliases []byte
	var traits []byte
	var metadata []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.WorldlineID, &item.Name, &item.Type, &aliases, &item.Summary, &traits, &item.Importance, &item.Status, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.Entity{}, err
	}
	var err error
	if item.Aliases, err = unmarshalJSONB[[]string](aliases); err != nil {
		return domain.Entity{}, err
	}
	if item.Traits, err = unmarshalJSONB[map[string]string](traits); err != nil {
		return domain.Entity{}, err
	}
	if item.Metadata, err = unmarshalJSONB[map[string]string](metadata); err != nil {
		return domain.Entity{}, err
	}
	return item, nil
}

func scanFact(scanner graphScanner) (domain.Fact, error) {
	var item domain.Fact
	var metadata []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.WorldlineID, &item.EntityID, &item.ChapterID, &item.ChapterVersionID, &item.Claim, &item.Source, &item.Confidence, &item.Status, &item.EmbeddingRef, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.Fact{}, err
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.Fact{}, err
	}
	item.Metadata = parsedMetadata
	return item, nil
}

func scanGraphEdge(scanner graphScanner) (domain.GraphEdge, error) {
	var item domain.GraphEdge
	var evidence []byte
	var metadata []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.WorldlineID, &item.SourceEntityID, &item.TargetEntityID, &item.Type, &item.Label, &item.Weight, &evidence, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.GraphEdge{}, err
	}
	var err error
	if item.EvidenceFactIDs, err = unmarshalJSONB[[]string](evidence); err != nil {
		return domain.GraphEdge{}, err
	}
	if item.Metadata, err = unmarshalJSONB[map[string]string](metadata); err != nil {
		return domain.GraphEdge{}, err
	}
	return item, nil
}

func scanPlotThread(scanner graphScanner) (domain.PlotThread, error) {
	var item domain.PlotThread
	var related []byte
	var metadata []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.WorldlineID, &item.Title, &item.Summary, &item.Status, &item.Priority, &related, &item.OpenedChapterID, &item.ClosedChapterID, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.PlotThread{}, err
	}
	var err error
	if item.RelatedEntityIDs, err = unmarshalJSONB[[]string](related); err != nil {
		return domain.PlotThread{}, err
	}
	if item.Metadata, err = unmarshalJSONB[map[string]string](metadata); err != nil {
		return domain.PlotThread{}, err
	}
	return item, nil
}
