package postgres

import (
	"context"
	"fmt"
	"strings"

	"aeonechoes/server/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateProject(project domain.Project, bible domain.StoryBible) (domain.Project, domain.StoryBible, error) {
	if err := requireStore(s); err != nil {
		return domain.Project{}, domain.StoryBible{}, err
	}
	if strings.TrimSpace(project.Title) == "" {
		return domain.Project{}, domain.StoryBible{}, fmt.Errorf("project title must not be empty")
	}
	if strings.TrimSpace(project.ID) == "" {
		id, err := s.NewID("project")
		if err != nil {
			return domain.Project{}, domain.StoryBible{}, fmt.Errorf("generate project id: %w", err)
		}
		project.ID = id
	}
	if strings.TrimSpace(bible.ID) == "" {
		id, err := s.NewID("bible")
		if err != nil {
			return domain.Project{}, domain.StoryBible{}, fmt.Errorf("generate story bible id: %w", err)
		}
		bible.ID = id
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
	if bible.Version == 0 {
		bible.Version = 1
	}
	bible.CreatedAt = n
	seed, projectMetadata, err := projectJSON(project)
	if err != nil {
		return domain.Project{}, domain.StoryBible{}, err
	}
	bibleJSON, err := storyBibleJSON(bible)
	if err != nil {
		return domain.Project{}, domain.StoryBible{}, err
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return domain.Project{}, domain.StoryBible{}, fmt.Errorf("begin create project %q: %w", project.ID, err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(context.Background(), `
INSERT INTO projects(id, title, slug, status, seed, active_story_bible_id, default_worldline_id, metadata, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,NULL,$6,$7,$8,$9)`, project.ID, project.Title, project.Slug, project.Status, seed, project.DefaultWorldlineID, projectMetadata, project.CreatedAt, project.UpdatedAt); err != nil {
		return domain.Project{}, domain.StoryBible{}, fmt.Errorf("insert project %q: %w", project.ID, err)
	}
	if err := insertStoryBibleTx(tx, bible, bibleJSON); err != nil {
		return domain.Project{}, domain.StoryBible{}, err
	}
	if _, err := tx.Exec(context.Background(), `UPDATE projects SET active_story_bible_id=$2, updated_at=$3 WHERE id=$1`, project.ID, bible.ID, project.UpdatedAt); err != nil {
		return domain.Project{}, domain.StoryBible{}, fmt.Errorf("set active story bible for project %q: %w", project.ID, err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return domain.Project{}, domain.StoryBible{}, fmt.Errorf("commit create project %q: %w", project.ID, err)
	}
	return project, bible, nil
}

func (s *Store) GetProject(id string) (domain.Project, error) {
	if err := requireStore(s); err != nil {
		return domain.Project{}, err
	}
	row := s.pool.QueryRow(context.Background(), projectSelectSQL()+` WHERE id=$1`, id)
	item, err := scanProject(row)
	if err != nil {
		if isNoRows(err) {
			return domain.Project{}, fmt.Errorf("project %q not found", id)
		}
		return domain.Project{}, fmt.Errorf("get project %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) ListProjects() ([]domain.Project, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(context.Background(), projectSelectSQL()+` ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Project, 0)
	for rows.Next() {
		item, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return items, nil
}

func (s *Store) GetStoryBible(projectID string) (domain.StoryBible, error) {
	if err := requireStore(s); err != nil {
		return domain.StoryBible{}, err
	}
	project, err := s.GetProject(projectID)
	if err != nil {
		return domain.StoryBible{}, err
	}
	if strings.TrimSpace(project.ActiveStoryBibleID) == "" {
		return domain.StoryBible{}, fmt.Errorf("project %q has no active story bible", projectID)
	}
	row := s.pool.QueryRow(context.Background(), storyBibleSelectSQL()+` WHERE id=$1`, project.ActiveStoryBibleID)
	item, err := scanStoryBible(row)
	if err != nil {
		if isNoRows(err) {
			return domain.StoryBible{}, fmt.Errorf("active story bible %q not found", project.ActiveStoryBibleID)
		}
		return domain.StoryBible{}, fmt.Errorf("get story bible %q: %w", project.ActiveStoryBibleID, err)
	}
	return item, nil
}

func (s *Store) UpdateStoryBible(projectID string, bible domain.StoryBible) (domain.StoryBible, error) {
	if err := requireStore(s); err != nil {
		return domain.StoryBible{}, err
	}
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.StoryBible{}, fmt.Errorf("begin update story bible for project %q: %w", projectID, err)
	}
	defer tx.Rollback(ctx)

	project, err := scanProject(tx.QueryRow(ctx, projectSelectSQL()+` WHERE id=$1 FOR UPDATE`, projectID))
	if err != nil {
		if isNoRows(err) {
			return domain.StoryBible{}, fmt.Errorf("project %q not found", projectID)
		}
		return domain.StoryBible{}, fmt.Errorf("get project %q for story bible update: %w", projectID, err)
	}
	id, err := s.NewID("bible")
	if err != nil {
		return domain.StoryBible{}, fmt.Errorf("generate story bible id: %w", err)
	}
	version, err := nextBibleVersionTx(ctx, tx, projectID)
	if err != nil {
		return domain.StoryBible{}, err
	}
	bible.ID = id
	bible.ProjectID = projectID
	bible.Version = version
	bible.CreatedAt = now()
	bibleJSON, err := storyBibleJSON(bible)
	if err != nil {
		return domain.StoryBible{}, err
	}
	if err := insertStoryBibleTx(tx, bible, bibleJSON); err != nil {
		return domain.StoryBible{}, err
	}
	project.ActiveStoryBibleID = bible.ID
	project.UpdatedAt = now()
	if _, err := tx.Exec(ctx, `UPDATE projects SET active_story_bible_id=$2, updated_at=$3 WHERE id=$1`, projectID, bible.ID, project.UpdatedAt); err != nil {
		return domain.StoryBible{}, fmt.Errorf("update active story bible for project %q: %w", projectID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.StoryBible{}, fmt.Errorf("commit story bible update for project %q: %w", projectID, err)
	}
	return bible, nil
}

func (s *Store) nextBibleVersion(projectID string) (int, error) {
	return nextBibleVersionTx(context.Background(), s.pool, projectID)
}

func nextBibleVersionTx(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, projectID string) (int, error) {
	var version int
	if err := querier.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM story_bible_versions WHERE project_id=$1`, projectID).Scan(&version); err != nil {
		return 0, fmt.Errorf("next story bible version for project %q: %w", projectID, err)
	}
	return version, nil
}

type storyBibleJSONValues struct {
	themes        []byte
	rules         []byte
	worldlineIDs  []byte
	entityIDs     []byte
	plotThreadIDs []byte
	sourceSeed    []byte
}

func projectJSON(project domain.Project) ([]byte, []byte, error) {
	seed, err := jsonbOrEmptyObject(project.Seed)
	if err != nil {
		return nil, nil, err
	}
	metadata, err := jsonbOrEmptyObject(project.Metadata)
	if err != nil {
		return nil, nil, err
	}
	return seed, metadata, nil
}

func storyBibleJSON(bible domain.StoryBible) (storyBibleJSONValues, error) {
	themes, err := jsonbOrEmptyArray(bible.Themes)
	if err != nil {
		return storyBibleJSONValues{}, err
	}
	rules, err := jsonbOrEmptyObject(bible.Rules)
	if err != nil {
		return storyBibleJSONValues{}, err
	}
	worldlineIDs, err := jsonbOrEmptyArray(bible.WorldlineIDs)
	if err != nil {
		return storyBibleJSONValues{}, err
	}
	entityIDs, err := jsonbOrEmptyArray(bible.EntityIDs)
	if err != nil {
		return storyBibleJSONValues{}, err
	}
	plotThreadIDs, err := jsonbOrEmptyArray(bible.PlotThreadIDs)
	if err != nil {
		return storyBibleJSONValues{}, err
	}
	sourceSeed, err := jsonbOrEmptyObject(bible.SourceSeed)
	if err != nil {
		return storyBibleJSONValues{}, err
	}
	return storyBibleJSONValues{themes: themes, rules: rules, worldlineIDs: worldlineIDs, entityIDs: entityIDs, plotThreadIDs: plotThreadIDs, sourceSeed: sourceSeed}, nil
}

func insertStoryBibleTx(tx pgx.Tx, bible domain.StoryBible, values storyBibleJSONValues) error {
	_, err := tx.Exec(context.Background(), `
INSERT INTO story_bible_versions(id, project_id, version, title, logline, synopsis, genre, tone, audience, language, themes, rules,
    worldline_ids, entity_ids, plot_thread_ids, source_seed, genesis_workflow_id, approved, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`, bible.ID, bible.ProjectID, bible.Version, bible.Title, bible.Logline, bible.Synopsis, bible.Genre, bible.Tone, bible.Audience, bible.Language, values.themes, values.rules, values.worldlineIDs, values.entityIDs, values.plotThreadIDs, values.sourceSeed, bible.GenesisWorkflowID, bible.Approved, bible.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert story bible %q: %w", bible.ID, err)
	}
	return nil
}

func projectSelectSQL() string {
	return `SELECT id, title, slug, status, seed, active_story_bible_id, default_worldline_id, metadata, created_at, updated_at FROM projects`
}

func storyBibleSelectSQL() string {
	return `SELECT id, project_id, version, title, logline, synopsis, genre, tone, audience, language, themes, rules, worldline_ids, entity_ids, plot_thread_ids, source_seed, genesis_workflow_id, approved, created_at FROM story_bible_versions`
}

type projectScanner interface{ Scan(dest ...any) error }

type storyBibleScanner interface{ Scan(dest ...any) error }

func scanProject(scanner projectScanner) (domain.Project, error) {
	var item domain.Project
	var seed []byte
	var metadata []byte
	var activeStoryBibleID *string
	if err := scanner.Scan(&item.ID, &item.Title, &item.Slug, &item.Status, &seed, &activeStoryBibleID, &item.DefaultWorldlineID, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.Project{}, err
	}
	parsedSeed, err := unmarshalJSONB[domain.ProjectSeed](seed)
	if err != nil {
		return domain.Project{}, err
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.Project{}, err
	}
	item.Seed = parsedSeed
	if activeStoryBibleID != nil {
		item.ActiveStoryBibleID = *activeStoryBibleID
	}
	item.Metadata = parsedMetadata
	return item, nil
}

func scanStoryBible(scanner storyBibleScanner) (domain.StoryBible, error) {
	var item domain.StoryBible
	var themes []byte
	var rules []byte
	var worldlineIDs []byte
	var entityIDs []byte
	var plotThreadIDs []byte
	var sourceSeed []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.Version, &item.Title, &item.Logline, &item.Synopsis, &item.Genre, &item.Tone, &item.Audience, &item.Language, &themes, &rules, &worldlineIDs, &entityIDs, &plotThreadIDs, &sourceSeed, &item.GenesisWorkflowID, &item.Approved, &item.CreatedAt); err != nil {
		return domain.StoryBible{}, err
	}
	var err error
	if item.Themes, err = unmarshalJSONB[[]string](themes); err != nil {
		return domain.StoryBible{}, err
	}
	if item.Rules, err = unmarshalJSONB[map[string]string](rules); err != nil {
		return domain.StoryBible{}, err
	}
	if item.WorldlineIDs, err = unmarshalJSONB[[]string](worldlineIDs); err != nil {
		return domain.StoryBible{}, err
	}
	if item.EntityIDs, err = unmarshalJSONB[[]string](entityIDs); err != nil {
		return domain.StoryBible{}, err
	}
	if item.PlotThreadIDs, err = unmarshalJSONB[[]string](plotThreadIDs); err != nil {
		return domain.StoryBible{}, err
	}
	if item.SourceSeed, err = unmarshalJSONB[domain.ProjectSeed](sourceSeed); err != nil {
		return domain.StoryBible{}, err
	}
	return item, nil
}
