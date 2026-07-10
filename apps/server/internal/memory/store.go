package memory

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/repository"
)

var _ repository.AppStore = (*Store)(nil)

// Store is a thread-safe in-memory repository used for development and tests.
type Store struct {
	mu sync.RWMutex

	seq int64

	providers       map[string]domain.ProviderConfig
	models          map[string]domain.ModelConfig
	projects        map[string]domain.Project
	storyBibles     map[string]domain.StoryBible
	worldlines      map[string]domain.Worldline
	entities        map[string]domain.Entity
	facts           map[string]domain.Fact
	edges           map[string]domain.GraphEdge
	plotThreads     map[string]domain.PlotThread
	chapters        map[string]domain.Chapter
	chapterVersions map[string]domain.ChapterVersion
	indexJobs       map[string]domain.IndexJob
	workflows       map[string]domain.AIWorkflow
	settings        map[string]domain.AppSetting

	agentConfigs    map[string]domain.AgentConfig
	agentRuns       map[string]domain.AgentRun
	skillSources    map[string]domain.SkillSource
	skills          map[string]domain.Skill
	mcpServers      map[string]domain.MCPServerConfig
	toolDefinitions map[string]domain.ToolDefinition
	toolInvocations map[string]domain.ToolInvocation
}

func NewStore() *Store {
	return &Store{
		providers:       make(map[string]domain.ProviderConfig),
		models:          make(map[string]domain.ModelConfig),
		projects:        make(map[string]domain.Project),
		storyBibles:     make(map[string]domain.StoryBible),
		worldlines:      make(map[string]domain.Worldline),
		entities:        make(map[string]domain.Entity),
		facts:           make(map[string]domain.Fact),
		edges:           make(map[string]domain.GraphEdge),
		plotThreads:     make(map[string]domain.PlotThread),
		chapters:        make(map[string]domain.Chapter),
		chapterVersions: make(map[string]domain.ChapterVersion),
		indexJobs:       make(map[string]domain.IndexJob),
		workflows:       make(map[string]domain.AIWorkflow),
		settings:        make(map[string]domain.AppSetting),
		agentConfigs:    make(map[string]domain.AgentConfig),
		agentRuns:       make(map[string]domain.AgentRun),
		skillSources:    make(map[string]domain.SkillSource),
		skills:          make(map[string]domain.Skill),
		mcpServers:      make(map[string]domain.MCPServerConfig),
		toolDefinitions: make(map[string]domain.ToolDefinition),
		toolInvocations: make(map[string]domain.ToolInvocation),
	}
}

func (s *Store) NewID(prefix string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nextIDLocked(prefix), nil
}

func (s *Store) nextIDLocked(prefix string) string {
	s.seq++
	clean := strings.TrimSpace(prefix)
	if clean == "" {
		clean = "id"
	}
	return fmt.Sprintf("%s_%06d", clean, s.seq)
}

func now() time.Time { return time.Now().UTC() }

func (s *Store) CreateProvider(cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	if !cfg.Type.Valid() {
		return domain.ProviderConfig{}, fmt.Errorf("provider type %q is not supported", cfg.Type)
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return domain.ProviderConfig{}, fmt.Errorf("provider name must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(cfg.ID) == "" {
		cfg.ID = s.nextIDLocked("provider")
	}
	if _, exists := s.providers[cfg.ID]; exists {
		return domain.ProviderConfig{}, fmt.Errorf("provider %q already exists", cfg.ID)
	}
	n := now()
	cfg.CreatedAt = n
	cfg.UpdatedAt = n
	if cfg.DefaultRequestTimeoutSec <= 0 {
		cfg.DefaultRequestTimeoutSec = 60
	}
	s.providers[cfg.ID] = cfg
	return cfg, nil
}

func (s *Store) UpdateProvider(id string, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	if strings.TrimSpace(id) == "" {
		return domain.ProviderConfig{}, fmt.Errorf("provider id must not be empty")
	}
	if !cfg.Type.Valid() {
		return domain.ProviderConfig{}, fmt.Errorf("provider type %q is not supported", cfg.Type)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.providers[id]
	if !ok {
		return domain.ProviderConfig{}, fmt.Errorf("provider %q not found", id)
	}
	cfg.ID = id
	cfg.CreatedAt = existing.CreatedAt
	cfg.UpdatedAt = now()
	s.providers[id] = cfg
	return cfg, nil
}

func (s *Store) GetProvider(id string) (domain.ProviderConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.providers[id]
	if !ok {
		return domain.ProviderConfig{}, fmt.Errorf("provider %q not found", id)
	}
	return cfg, nil
}

func (s *Store) DeleteProvider(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.providers[id]; !ok {
		return fmt.Errorf("provider %q not found", id)
	}
	delete(s.providers, id)
	for modelID, model := range s.models {
		if model.ProviderID == id {
			delete(s.models, modelID)
		}
	}
	return nil
}

func (s *Store) ListProviders() ([]domain.ProviderConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.ProviderConfig, 0, len(s.providers))
	for _, item := range s.providers {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (s *Store) TouchProviderModelRefresh(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.providers[id]
	if !ok {
		return fmt.Errorf("provider %q not found", id)
	}
	n := now()
	cfg.LastModelRefreshAt = &n
	cfg.UpdatedAt = n
	s.providers[id] = cfg
	return nil
}

func (s *Store) CreateModel(cfg domain.ModelConfig) (domain.ModelConfig, error) {
	if strings.TrimSpace(cfg.ProviderID) == "" {
		return domain.ModelConfig{}, fmt.Errorf("model provider_id must not be empty")
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return domain.ModelConfig{}, fmt.Errorf("model name must not be empty")
	}
	if !cfg.Kind.Valid() {
		return domain.ModelConfig{}, fmt.Errorf("model kind %q is invalid", cfg.Kind)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	providerCfg, ok := s.providers[cfg.ProviderID]
	if !ok {
		return domain.ModelConfig{}, fmt.Errorf("provider %q not found for model", cfg.ProviderID)
	}
	if strings.TrimSpace(cfg.ID) == "" {
		cfg.ID = s.nextIDLocked("model")
	}
	if _, exists := s.models[cfg.ID]; exists {
		return domain.ModelConfig{}, fmt.Errorf("model %q already exists", cfg.ID)
	}
	cfg.ProviderType = providerCfg.Type
	n := now()
	cfg.CreatedAt = n
	cfg.UpdatedAt = n
	if cfg.RoutingWeight == 0 {
		cfg.RoutingWeight = 100
	}
	if cfg.DefaultForKind {
		s.clearDefaultForKindLocked(cfg.Kind, cfg.ID)
	}
	s.models[cfg.ID] = cfg
	return cfg, nil
}

func (s *Store) UpdateModel(id string, cfg domain.ModelConfig) (domain.ModelConfig, error) {
	if strings.TrimSpace(id) == "" {
		return domain.ModelConfig{}, fmt.Errorf("model id must not be empty")
	}
	if !cfg.Kind.Valid() {
		return domain.ModelConfig{}, fmt.Errorf("model kind %q is invalid", cfg.Kind)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.models[id]
	if !ok {
		return domain.ModelConfig{}, fmt.Errorf("model %q not found", id)
	}
	providerCfg, ok := s.providers[cfg.ProviderID]
	if !ok {
		return domain.ModelConfig{}, fmt.Errorf("provider %q not found for model", cfg.ProviderID)
	}
	cfg.ID = id
	cfg.ProviderType = providerCfg.Type
	cfg.CreatedAt = existing.CreatedAt
	cfg.UpdatedAt = now()
	if cfg.DefaultForKind {
		s.clearDefaultForKindLocked(cfg.Kind, id)
	}
	s.models[id] = cfg
	return cfg, nil
}

func (s *Store) UpsertModel(cfg domain.ModelConfig) (domain.ModelConfig, error) {
	if strings.TrimSpace(cfg.ID) == "" {
		return s.CreateModel(cfg)
	}
	s.mu.RLock()
	_, exists := s.models[cfg.ID]
	s.mu.RUnlock()
	if exists {
		return s.UpdateModel(cfg.ID, cfg)
	}
	return s.CreateModel(cfg)
}

func (s *Store) clearDefaultForKindLocked(kind domain.ModelKind, exceptID string) {
	for id, model := range s.models {
		if id == exceptID {
			continue
		}
		if model.Kind == kind && model.DefaultForKind {
			model.DefaultForKind = false
			model.UpdatedAt = now()
			s.models[id] = model
		}
	}
}

func (s *Store) GetModel(id string) (domain.ModelConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.models[id]
	if !ok {
		return domain.ModelConfig{}, fmt.Errorf("model %q not found", id)
	}
	return cfg, nil
}

func (s *Store) DeleteModel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.models[id]; !ok {
		return fmt.Errorf("model %q not found", id)
	}
	delete(s.models, id)
	return nil
}

func (s *Store) ListModels() ([]domain.ModelConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.ModelConfig, 0, len(s.models))
	for _, item := range s.models {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (s *Store) ListModelsByKind(kind domain.ModelKind) ([]domain.ModelConfig, error) {
	items, err := s.ListModels()
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.ModelConfig, 0, len(items))
	for _, item := range items {
		if item.Kind == kind && item.Enabled {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (s *Store) CreateProject(project domain.Project, bible domain.StoryBible) (domain.Project, domain.StoryBible, error) {
	if strings.TrimSpace(project.Title) == "" {
		return domain.Project{}, domain.StoryBible{}, fmt.Errorf("project title must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(project.ID) == "" {
		project.ID = s.nextIDLocked("project")
	}
	if strings.TrimSpace(bible.ID) == "" {
		bible.ID = s.nextIDLocked("bible")
	}
	if bible.ProjectID == "" {
		bible.ProjectID = project.ID
	}
	if bible.ProjectID != project.ID {
		return domain.Project{}, domain.StoryBible{}, fmt.Errorf("story bible project_id %q does not match project %q", bible.ProjectID, project.ID)
	}
	n := now()
	project.CreatedAt = n
	project.UpdatedAt = n
	if project.Status == "" {
		project.Status = "active"
	}
	project.ActiveStoryBibleID = bible.ID
	bible.CreatedAt = n
	if bible.Version == 0 {
		bible.Version = 1
	}
	s.projects[project.ID] = project
	s.storyBibles[bible.ID] = bible
	return project, bible, nil
}

func (s *Store) GetProject(id string) (domain.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[id]
	if !ok {
		return domain.Project{}, repository.NotFound("project", id)
	}
	return project, nil
}

func (s *Store) ListProjects() ([]domain.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Project, 0, len(s.projects))
	for _, item := range s.projects {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (s *Store) GetStoryBible(projectID string) (domain.StoryBible, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[projectID]
	if !ok {
		return domain.StoryBible{}, fmt.Errorf("project %q not found", projectID)
	}
	bible, ok := s.storyBibles[project.ActiveStoryBibleID]
	if !ok {
		return domain.StoryBible{}, fmt.Errorf("active story bible %q not found", project.ActiveStoryBibleID)
	}
	return bible, nil
}

func (s *Store) UpdateStoryBible(projectID string, bible domain.StoryBible) (domain.StoryBible, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	project, ok := s.projects[projectID]
	if !ok {
		return domain.StoryBible{}, fmt.Errorf("project %q not found", projectID)
	}
	bible.ID = s.nextIDLocked("bible")
	bible.ProjectID = projectID
	bible.Version = s.nextBibleVersionLocked(projectID)
	bible.CreatedAt = now()
	s.storyBibles[bible.ID] = bible
	project.ActiveStoryBibleID = bible.ID
	project.UpdatedAt = now()
	s.projects[projectID] = project
	return bible, nil
}

func (s *Store) nextBibleVersionLocked(projectID string) int {
	version := 0
	for _, bible := range s.storyBibles {
		if bible.ProjectID == projectID && bible.Version > version {
			version = bible.Version
		}
	}
	return version + 1
}

func (s *Store) GetSetting(scope, key string) (domain.AppSetting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.settings[settingStorageKey(scope, key)]
	if !ok {
		return domain.AppSetting{}, fmt.Errorf("setting %q/%q not found", strings.TrimSpace(scope), strings.TrimSpace(key))
	}
	return item, nil
}

func (s *Store) UpsertSetting(setting domain.AppSetting) (domain.AppSetting, error) {
	setting.Scope = strings.TrimSpace(setting.Scope)
	setting.Key = strings.TrimSpace(setting.Key)
	if setting.Scope == "" || setting.Key == "" {
		return domain.AppSetting{}, fmt.Errorf("setting scope and key must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	setting.UpdatedAt = now()
	s.settings[settingStorageKey(setting.Scope, setting.Key)] = setting
	return setting, nil
}

func (s *Store) ListSettings(scope string) ([]domain.AppSetting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cleanScope := strings.TrimSpace(scope)
	items := make([]domain.AppSetting, 0)
	for _, item := range s.settings {
		if cleanScope != "" && item.Scope != cleanScope {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Scope != items[j].Scope {
			return items[i].Scope < items[j].Scope
		}
		if items[i].Key != items[j].Key {
			return items[i].Key < items[j].Key
		}
		return items[i].UpdatedAt.Before(items[j].UpdatedAt)
	})
	return items, nil
}

func settingStorageKey(scope, key string) string {
	return strings.TrimSpace(scope) + ":" + strings.TrimSpace(key)
}

func (s *Store) SaveWorldline(item domain.Worldline) (domain.Worldline, error) {
	if strings.TrimSpace(item.ProjectID) == "" {
		return domain.Worldline{}, fmt.Errorf("worldline project_id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[item.ProjectID]; !ok {
		return domain.Worldline{}, fmt.Errorf("project %q not found", item.ProjectID)
	}
	if item.ID == "" {
		item.ID = s.nextIDLocked("worldline")
		item.CreatedAt = now()
	} else if existing, ok := s.worldlines[item.ID]; ok {
		item.CreatedAt = existing.CreatedAt
	} else {
		item.CreatedAt = now()
	}
	item.UpdatedAt = now()
	s.worldlines[item.ID] = item
	return item, nil
}

func (s *Store) SaveEntity(item domain.Entity) (domain.Entity, error) {
	if strings.TrimSpace(item.ProjectID) == "" || strings.TrimSpace(item.Name) == "" {
		return domain.Entity{}, fmt.Errorf("entity project_id and name must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == "" {
		item.ID = s.nextIDLocked("entity")
		item.CreatedAt = now()
	} else if existing, ok := s.entities[item.ID]; ok {
		item.CreatedAt = existing.CreatedAt
	} else {
		item.CreatedAt = now()
	}
	item.UpdatedAt = now()
	s.entities[item.ID] = item
	return item, nil
}

func (s *Store) SaveFact(item domain.Fact) (domain.Fact, error) {
	if strings.TrimSpace(item.ProjectID) == "" || strings.TrimSpace(item.Claim) == "" {
		return domain.Fact{}, fmt.Errorf("fact project_id and claim must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == "" {
		item.ID = s.nextIDLocked("fact")
		item.CreatedAt = now()
	} else if existing, ok := s.facts[item.ID]; ok {
		item.CreatedAt = existing.CreatedAt
	} else {
		item.CreatedAt = now()
	}
	item.UpdatedAt = now()
	s.facts[item.ID] = item
	return item, nil
}

func (s *Store) SaveGraphEdge(item domain.GraphEdge) (domain.GraphEdge, error) {
	if strings.TrimSpace(item.ProjectID) == "" || strings.TrimSpace(item.SourceEntityID) == "" || strings.TrimSpace(item.TargetEntityID) == "" {
		return domain.GraphEdge{}, fmt.Errorf("graph edge project_id, source_entity_id and target_entity_id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == "" {
		item.ID = s.nextIDLocked("edge")
		item.CreatedAt = now()
	} else if existing, ok := s.edges[item.ID]; ok {
		item.CreatedAt = existing.CreatedAt
	} else {
		item.CreatedAt = now()
	}
	item.UpdatedAt = now()
	s.edges[item.ID] = item
	return item, nil
}

func (s *Store) SavePlotThread(item domain.PlotThread) (domain.PlotThread, error) {
	if strings.TrimSpace(item.ProjectID) == "" || strings.TrimSpace(item.Title) == "" {
		return domain.PlotThread{}, fmt.Errorf("plot thread project_id and title must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == "" {
		item.ID = s.nextIDLocked("thread")
		item.CreatedAt = now()
	} else if existing, ok := s.plotThreads[item.ID]; ok {
		item.CreatedAt = existing.CreatedAt
	} else {
		item.CreatedAt = now()
	}
	item.UpdatedAt = now()
	s.plotThreads[item.ID] = item
	return item, nil
}

func (s *Store) ListEntities(projectID string) ([]domain.Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Entity, 0)
	for _, item := range s.entities {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (s *Store) ListFacts(projectID string) ([]domain.Fact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Fact, 0)
	for _, item := range s.facts {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (s *Store) ListPlotThreads(projectID string) ([]domain.PlotThread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.PlotThread, 0)
	for _, item := range s.plotThreads {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Priority > items[j].Priority })
	return items, nil
}

func (s *Store) ExpandGraph(projectID string, entityIDs []string, depth int) (domain.GraphExpansion, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.GraphExpansion{}, fmt.Errorf("project_id must not be empty")
	}
	if depth < 1 || depth > 4 {
		return domain.GraphExpansion{}, fmt.Errorf("graph expansion depth must be between 1 and 4")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.projects[projectID]; !ok {
		return domain.GraphExpansion{}, fmt.Errorf("project %q not found", projectID)
	}
	selected := map[string]bool{}
	frontier := map[string]bool{}
	if len(entityIDs) == 0 {
		for id, entity := range s.entities {
			if entity.ProjectID == projectID {
				frontier[id] = true
			}
		}
	} else {
		for _, id := range entityIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				return domain.GraphExpansion{}, fmt.Errorf("graph expansion entity_ids must not contain empty values")
			}
			entity, ok := s.entities[id]
			if !ok || entity.ProjectID != projectID {
				return domain.GraphExpansion{}, repository.NotFound("entity", id)
			}
			frontier[id] = true
		}
	}
	selectedEdges := map[string]bool{}
	for d := 0; d < depth && len(frontier) > 0; d++ {
		next := map[string]bool{}
		for id := range frontier {
			entity, ok := s.entities[id]
			if !ok || entity.ProjectID != projectID {
				continue
			}
			selected[id] = true
			for edgeID, edge := range s.edges {
				if edge.ProjectID != projectID {
					continue
				}
				if edge.SourceEntityID == id || edge.TargetEntityID == id {
					other := edge.TargetEntityID
					if other == id {
						other = edge.SourceEntityID
					}
					otherEntity, ok := s.entities[other]
					if !ok || otherEntity.ProjectID != projectID {
						return domain.GraphExpansion{}, fmt.Errorf("graph edge %q references missing entity %q", edgeID, other)
					}
					selectedEdges[edgeID] = true
					if !selected[other] {
						next[other] = true
					}
				}
			}
		}
		frontier = next
	}
	for id := range frontier {
		entity, ok := s.entities[id]
		if !ok || entity.ProjectID != projectID {
			return domain.GraphExpansion{}, fmt.Errorf("graph expansion references missing entity %q", id)
		}
		selected[id] = true
	}
	entities := make([]domain.Entity, 0, len(selected))
	for id := range selected {
		if entity, ok := s.entities[id]; ok {
			entities = append(entities, entity)
		}
	}
	edges := make([]domain.GraphEdge, 0, len(selectedEdges))
	factIDs := map[string]bool{}
	for id := range selectedEdges {
		if edge, ok := s.edges[id]; ok {
			edges = append(edges, edge)
			for _, factID := range edge.EvidenceFactIDs {
				factIDs[factID] = true
			}
		}
	}
	facts := make([]domain.Fact, 0)
	for _, fact := range s.facts {
		if fact.ProjectID != projectID {
			continue
		}
		if factIDs[fact.ID] || selected[fact.EntityID] {
			facts = append(facts, fact)
		}
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].Name < entities[j].Name })
	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })
	sort.Slice(facts, func(i, j int) bool { return facts[i].CreatedAt.Before(facts[j].CreatedAt) })
	return domain.GraphExpansion{ProjectID: projectID, Depth: depth, Entities: entities, Edges: edges, Facts: facts, GeneratedAt: now()}, nil
}

func (s *Store) CreateChapter(req domain.CreateChapterRequest) (domain.Chapter, error) {
	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		return domain.Chapter{}, fmt.Errorf("create chapter project_id must not be empty")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return domain.Chapter{}, fmt.Errorf("chapter title must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[projectID]; !ok {
		return domain.Chapter{}, repository.NotFound("project", projectID)
	}
	number := req.Number
	if number <= 0 {
		number = s.nextChapterNumberLocked(projectID)
	}
	for _, existing := range s.chapters {
		if existing.ProjectID == projectID && existing.Number == number {
			return domain.Chapter{}, repository.Conflict("chapter", existing.ID, fmt.Sprintf("chapter number %d already exists in project %q", number, projectID), nil)
		}
	}
	status := req.Status
	if status == "" {
		status = domain.ChapterStatusDrafting
	}
	if !status.Valid() {
		return domain.Chapter{}, fmt.Errorf("chapter status %q is invalid", status)
	}
	n := now()
	chapter := domain.Chapter{ID: s.nextIDLocked("chapter"), ProjectID: projectID, Number: number, Title: title, Status: status, Metadata: copyStringMap(req.Metadata), CreatedAt: n, UpdatedAt: n}
	s.chapters[chapter.ID] = chapter
	return chapter, nil
}

func (s *Store) UpdateChapter(req domain.UpdateChapterRequest) (domain.Chapter, error) {
	projectID := strings.TrimSpace(req.ProjectID)
	chapterID := strings.TrimSpace(req.ChapterID)
	if projectID == "" || chapterID == "" {
		return domain.Chapter{}, fmt.Errorf("update chapter project_id and chapter_id must not be empty")
	}
	if req.Number == nil && req.Title == nil && req.Status == nil && req.Metadata == nil {
		return domain.Chapter{}, fmt.Errorf("chapter update must include at least one field")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.chapters[chapterID]
	if !ok || existing.ProjectID != projectID {
		return domain.Chapter{}, repository.NotFound("chapter", chapterID)
	}
	if req.Number != nil {
		if *req.Number <= 0 {
			return domain.Chapter{}, fmt.Errorf("chapter number must be greater than zero")
		}
		if *req.Number != existing.Number {
			for _, candidate := range s.chapters {
				if candidate.ID != chapterID && candidate.ProjectID == projectID && candidate.Number == *req.Number {
					return domain.Chapter{}, repository.Conflict("chapter", chapterID, fmt.Sprintf("chapter number %d already exists in project %q", *req.Number, projectID), nil)
				}
			}
			existing.Number = *req.Number
		}
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return domain.Chapter{}, fmt.Errorf("chapter title must not be empty")
		}
		existing.Title = title
	}
	if req.Status != nil {
		status := *req.Status
		if !status.Valid() {
			return domain.Chapter{}, fmt.Errorf("chapter status %q is invalid", status)
		}
		existing.Status = status
	}
	if req.Metadata != nil {
		metadata := copyStringMap(existing.Metadata)
		if metadata == nil {
			metadata = map[string]string{}
		}
		for key, value := range *req.Metadata {
			metadata[key] = value
		}
		existing.Metadata = metadata
	}
	existing.UpdatedAt = now()
	s.chapters[chapterID] = existing
	return existing, nil
}

func (s *Store) GetChapter(id string) (domain.Chapter, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Chapter{}, fmt.Errorf("chapter id must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	chapter, ok := s.chapters[id]
	if !ok {
		return domain.Chapter{}, repository.NotFound("chapter", id)
	}
	return chapter, nil
}

func (s *Store) ListChapters(projectID string) ([]domain.Chapter, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("list chapters project_id must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.projects[projectID]; !ok {
		return nil, repository.NotFound("project", projectID)
	}
	items := make([]domain.Chapter, 0)
	for _, item := range s.chapters {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Number == items[j].Number {
			return items[i].ID < items[j].ID
		}
		return items[i].Number < items[j].Number
	})
	return items, nil
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func (s *Store) SaveChapterVersion(version domain.ChapterVersion) (domain.ChapterVersion, domain.IndexJob, error) {
	version.ProjectID = strings.TrimSpace(version.ProjectID)
	if version.ProjectID == "" {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("chapter version project_id must not be empty")
	}
	version.Title = strings.TrimSpace(version.Title)
	if version.Title == "" {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("chapter version title must not be empty")
	}
	if strings.TrimSpace(version.Content) == "" {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("chapter version content must not be empty")
	}
	if !version.AuthorRole.Valid() {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("chapter version author_role %q is invalid", version.AuthorRole)
	}
	version.ChapterID = strings.TrimSpace(version.ChapterID)
	if version.ChapterID == "" {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("chapter version chapter_id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[version.ProjectID]; !ok {
		return domain.ChapterVersion{}, domain.IndexJob{}, repository.NotFound("project", version.ProjectID)
	}
	chapter, ok := s.chapters[version.ChapterID]
	if !ok || chapter.ProjectID != version.ProjectID {
		return domain.ChapterVersion{}, domain.IndexJob{}, repository.NotFound("chapter", version.ChapterID)
	}
	if strings.TrimSpace(version.ID) == "" {
		version.ID = s.nextIDLocked("chapter_version")
	} else if _, exists := s.chapterVersions[version.ID]; exists {
		return domain.ChapterVersion{}, domain.IndexJob{}, repository.Conflict("chapter version", version.ID, fmt.Sprintf("chapter version %q already exists", version.ID), nil)
	}
	version.ParentVersionID = strings.TrimSpace(version.ParentVersionID)
	if err := s.validateChapterVersionParentLocked(version); err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, err
	}
	version.Version = s.nextChapterVersionLocked(version.ChapterID)
	version.CreatedAt = now()
	if version.IndexStatus == "" {
		version.IndexStatus = "pending"
	}
	s.chapterVersions[version.ID] = version
	s.supersedePendingIndexJobsLocked(version.ProjectID, version.ChapterID)
	jobCreatedAt := now()
	job := domain.IndexJob{ID: s.nextIDLocked("index_job"), ProjectID: version.ProjectID, ChapterID: version.ChapterID, ChapterVersionID: version.ID, Kind: "chapter_reindex", Status: "pending", Payload: map[string]string{"trigger": "chapter_version_saved"}, CreatedAt: jobCreatedAt, UpdatedAt: jobCreatedAt}
	s.indexJobs[job.ID] = job
	return version, job, nil
}

func (s *Store) validateChapterVersionParentLocked(version domain.ChapterVersion) error {
	if version.ParentVersionID == "" {
		return nil
	}
	if version.ParentVersionID == version.ID {
		return fmt.Errorf("chapter version %q cannot reference itself as parent", version.ID)
	}
	parent, ok := s.chapterVersions[version.ParentVersionID]
	if !ok {
		return repository.NotFound("chapter version", version.ParentVersionID)
	}
	if parent.ProjectID != version.ProjectID || parent.ChapterID != version.ChapterID {
		return fmt.Errorf("chapter version parent %q must belong to project %q and chapter %q", version.ParentVersionID, version.ProjectID, version.ChapterID)
	}
	visited := map[string]struct{}{version.ID: {}}
	current := parent
	for {
		if _, seen := visited[current.ID]; seen {
			return fmt.Errorf("chapter version parent chain contains a cycle at %q", current.ID)
		}
		visited[current.ID] = struct{}{}
		if current.ProjectID != version.ProjectID || current.ChapterID != version.ChapterID {
			return fmt.Errorf("chapter version parent chain crosses project or chapter at %q", current.ID)
		}
		if strings.TrimSpace(current.ParentVersionID) == "" {
			return nil
		}
		next, ok := s.chapterVersions[current.ParentVersionID]
		if !ok {
			return fmt.Errorf("chapter version parent chain references missing version %q", current.ParentVersionID)
		}
		current = next
	}
}

func (s *Store) supersedePendingIndexJobsLocked(projectID, chapterID string) {
	for id, job := range s.indexJobs {
		if job.ProjectID != projectID || job.ChapterID != chapterID {
			continue
		}
		if job.Status != "pending" {
			continue
		}
		job.Status = "superseded"
		job.Error = "superseded by newer pending job"
		job.UpdatedAt = now()
		s.indexJobs[id] = job
	}
}

func (s *Store) nextChapterNumberLocked(projectID string) int {
	maxNumber := 0
	for _, chapter := range s.chapters {
		if chapter.ProjectID == projectID && chapter.Number > maxNumber {
			maxNumber = chapter.Number
		}
	}
	return maxNumber + 1
}

func (s *Store) nextChapterVersionLocked(chapterID string) int {
	maxVersion := 0
	for _, version := range s.chapterVersions {
		if version.ChapterID == chapterID && version.Version > maxVersion {
			maxVersion = version.Version
		}
	}
	return maxVersion + 1
}

func (s *Store) GetChapterVersion(id string) (domain.ChapterVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.chapterVersions[id]
	if !ok {
		return domain.ChapterVersion{}, fmt.Errorf("chapter version %q not found", id)
	}
	return item, nil
}

func (s *Store) UpdateChapterVersionIndexStatus(id, status string) (domain.ChapterVersion, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(status) == "" {
		return domain.ChapterVersion{}, fmt.Errorf("chapter version id and index status must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.chapterVersions[id]
	if !ok {
		return domain.ChapterVersion{}, fmt.Errorf("chapter version %q not found", id)
	}
	item.IndexStatus = status
	s.chapterVersions[id] = item
	return item, nil
}

func (s *Store) ListChapterVersions(projectID, chapterID string) ([]domain.ChapterVersion, error) {
	projectID = strings.TrimSpace(projectID)
	chapterID = strings.TrimSpace(chapterID)
	if projectID == "" {
		return nil, fmt.Errorf("list chapter versions project_id must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.projects[projectID]; !ok {
		return nil, repository.NotFound("project", projectID)
	}
	if chapterID != "" {
		chapter, ok := s.chapters[chapterID]
		if !ok || chapter.ProjectID != projectID {
			return nil, repository.NotFound("chapter", chapterID)
		}
	}
	items := make([]domain.ChapterVersion, 0)
	for _, item := range s.chapterVersions {
		if item.ProjectID != projectID {
			continue
		}
		if chapterID != "" && item.ChapterID != chapterID {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ChapterID == items[j].ChapterID {
			return items[i].Version > items[j].Version
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) CreateIndexJob(job domain.IndexJob) (domain.IndexJob, error) {
	if strings.TrimSpace(job.ProjectID) == "" || strings.TrimSpace(job.Kind) == "" {
		return domain.IndexJob{}, fmt.Errorf("index job project_id and kind must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(job.ID) == "" {
		job.ID = s.nextIDLocked("index_job")
	}
	n := now()
	job.CreatedAt = n
	job.UpdatedAt = n
	if job.Status == "" {
		job.Status = "pending"
	}
	s.indexJobs[job.ID] = job
	return job, nil
}

func (s *Store) UpdateIndexJobStatus(id, status, errorMessage string) (domain.IndexJob, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(status) == "" {
		return domain.IndexJob{}, fmt.Errorf("index job id and status must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.indexJobs[id]
	if !ok {
		return domain.IndexJob{}, fmt.Errorf("index job %q not found", id)
	}
	job.Status = status
	job.Error = errorMessage
	job.UpdatedAt = now()
	switch status {
	case "running":
		t := now()
		job.StartedAt = &t
		job.Attempts++
	case "completed", "failed":
		t := now()
		job.CompletedAt = &t
	}
	s.indexJobs[id] = job
	return job, nil
}

func (s *Store) GetIndexJob(id string) (domain.IndexJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.indexJobs[id]
	if !ok {
		return domain.IndexJob{}, fmt.Errorf("index job %q not found", id)
	}
	return item, nil
}

func (s *Store) ListIndexJobs(filter repository.IndexJobFilter) ([]domain.IndexJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	projectID := strings.TrimSpace(filter.ProjectID)
	status := strings.TrimSpace(filter.Status)
	items := make([]domain.IndexJob, 0)
	for _, item := range s.indexJobs {
		if projectID != "" && item.ProjectID != projectID {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if filter.Limit > 0 && len(items) > filter.Limit {
		items = items[:filter.Limit]
	}
	return items, nil
}

func (s *Store) ListPendingIndexJobs(projectID string, limit int) ([]domain.IndexJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.IndexJob, 0)
	for _, item := range s.indexJobs {
		if item.Status != "pending" {
			continue
		}
		if projectID != "" && item.ProjectID != projectID {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Store) SaveWorkflow(workflow domain.AIWorkflow) (domain.AIWorkflow, error) {
	if strings.TrimSpace(workflow.ProjectID) == "" || strings.TrimSpace(workflow.Kind) == "" {
		return domain.AIWorkflow{}, fmt.Errorf("workflow project_id and kind must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(workflow.ID) == "" {
		workflow.ID = s.nextIDLocked("workflow")
		workflow.CreatedAt = now()
	} else if existing, ok := s.workflows[workflow.ID]; ok {
		workflow.CreatedAt = existing.CreatedAt
	} else {
		workflow.CreatedAt = now()
	}
	workflow.UpdatedAt = now()
	s.workflows[workflow.ID] = workflow
	return workflow, nil
}

func (s *Store) ListWorkflows(projectID string) ([]domain.AIWorkflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cleanProjectID := strings.TrimSpace(projectID)
	items := make([]domain.AIWorkflow, 0)
	for _, item := range s.workflows {
		if cleanProjectID != "" && item.ProjectID != cleanProjectID {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) GetWorkflow(id string) (domain.AIWorkflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.workflows[strings.TrimSpace(id)]
	if !ok {
		return domain.AIWorkflow{}, fmt.Errorf("workflow %q not found", strings.TrimSpace(id))
	}
	return item, nil
}
