package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) CreateChapter(req domain.CreateChapterRequest) (domain.Chapter, error) {
	if err := requireStore(s); err != nil {
		return domain.Chapter{}, err
	}
	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		return domain.Chapter{}, fmt.Errorf("create chapter project_id must not be empty")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return domain.Chapter{}, fmt.Errorf("chapter title must not be empty")
	}
	if _, err := s.GetProject(projectID); err != nil {
		return domain.Chapter{}, err
	}
	chapterID, err := s.NewID("chapter")
	if err != nil {
		return domain.Chapter{}, fmt.Errorf("generate chapter id: %w", err)
	}
	number := req.Number
	if number <= 0 {
		number, err = nextChapterNumber(s, projectID)
		if err != nil {
			return domain.Chapter{}, err
		}
	}
	n := now()
	status := req.Status
	if status == "" {
		status = domain.ChapterStatusDrafting
	}
	if !status.Valid() {
		return domain.Chapter{}, fmt.Errorf("chapter status %q is invalid", status)
	}
	metadata, err := jsonbOrEmptyObject(req.Metadata)
	if err != nil {
		return domain.Chapter{}, err
	}
	chapter := domain.Chapter{ID: chapterID, ProjectID: projectID, Number: number, Title: title, Status: status, Metadata: req.Metadata, CreatedAt: n, UpdatedAt: n}
	if _, err := s.pool.Exec(context.Background(), `INSERT INTO chapters(id, project_id, number, title, status, metadata, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, chapter.ID, chapter.ProjectID, chapter.Number, chapter.Title, chapter.Status, metadata, chapter.CreatedAt, chapter.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return domain.Chapter{}, repository.Conflict("chapter", chapter.ID, fmt.Sprintf("chapter number %d already exists in project %q", number, projectID), err)
		}
		return domain.Chapter{}, fmt.Errorf("insert chapter %q: %w", chapter.ID, err)
	}
	return chapter, nil
}

func (s *Store) UpdateChapter(req domain.UpdateChapterRequest) (domain.Chapter, error) {
	if err := requireStore(s); err != nil {
		return domain.Chapter{}, err
	}
	projectID := strings.TrimSpace(req.ProjectID)
	chapterID := strings.TrimSpace(req.ChapterID)
	if projectID == "" || chapterID == "" {
		return domain.Chapter{}, fmt.Errorf("update chapter project_id and chapter_id must not be empty")
	}
	if req.Number == nil && req.Title == nil && req.Status == nil && req.Metadata == nil {
		return domain.Chapter{}, fmt.Errorf("chapter update must include at least one field")
	}
	existing, err := s.GetChapter(chapterID)
	if err != nil {
		return domain.Chapter{}, err
	}
	if existing.ProjectID != projectID {
		return domain.Chapter{}, repository.NotFound("chapter", chapterID)
	}
	updated := existing
	if req.Number != nil {
		if *req.Number <= 0 {
			return domain.Chapter{}, fmt.Errorf("chapter number must be greater than zero")
		}
		updated.Number = *req.Number
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return domain.Chapter{}, fmt.Errorf("chapter title must not be empty")
		}
		updated.Title = title
	}
	if req.Status != nil {
		status := *req.Status
		if !status.Valid() {
			return domain.Chapter{}, fmt.Errorf("chapter status %q is invalid", status)
		}
		updated.Status = status
	}
	if req.Metadata != nil {
		metadata := make(map[string]string, len(updated.Metadata)+len(*req.Metadata))
		for key, value := range updated.Metadata {
			metadata[key] = value
		}
		for key, value := range *req.Metadata {
			metadata[key] = value
		}
		updated.Metadata = metadata
	}
	updated.UpdatedAt = now()
	metadata, err := jsonbOrEmptyObject(updated.Metadata)
	if err != nil {
		return domain.Chapter{}, err
	}
	result, err := s.pool.Exec(context.Background(), `UPDATE chapters SET number=$3, title=$4, status=$5, metadata=$6, updated_at=$7 WHERE id=$1 AND project_id=$2`, updated.ID, projectID, updated.Number, updated.Title, updated.Status, metadata, updated.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Chapter{}, repository.Conflict("chapter", chapterID, fmt.Sprintf("chapter number %d already exists in project %q", updated.Number, projectID), err)
		}
		return domain.Chapter{}, fmt.Errorf("update chapter %q: %w", updated.ID, err)
	}
	if result.RowsAffected() != 1 {
		return domain.Chapter{}, repository.NotFound("chapter", chapterID)
	}
	return updated, nil
}

func (s *Store) GetChapter(id string) (domain.Chapter, error) {
	if err := requireStore(s); err != nil {
		return domain.Chapter{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Chapter{}, fmt.Errorf("chapter id must not be empty")
	}
	row := s.pool.QueryRow(context.Background(), chapterSelectSQL()+` WHERE id=$1`, id)
	item, err := scanChapter(row)
	if err != nil {
		if isNoRows(err) {
			return domain.Chapter{}, repository.NotFound("chapter", id)
		}
		return domain.Chapter{}, fmt.Errorf("get chapter %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) ListChapters(projectID string) ([]domain.Chapter, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("list chapters project_id must not be empty")
	}
	if _, err := s.GetProject(projectID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(context.Background(), chapterSelectSQL()+` WHERE project_id=$1 ORDER BY number ASC, id ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list chapters for project %q: %w", projectID, err)
	}
	defer rows.Close()
	items := make([]domain.Chapter, 0)
	for rows.Next() {
		item, err := scanChapter(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chapters for project %q: %w", projectID, err)
	}
	return items, nil
}

func nextChapterNumber(s *Store, projectID string) (int, error) {
	var next int
	if err := s.pool.QueryRow(context.Background(), `SELECT COALESCE(MAX(number), 0) + 1 FROM chapters WHERE project_id=$1`, projectID).Scan(&next); err != nil {
		return 0, fmt.Errorf("next chapter number for project %q: %w", projectID, err)
	}
	return next, nil
}

func (s *Store) SaveChapterVersion(version domain.ChapterVersion) (domain.ChapterVersion, domain.IndexJob, error) {
	if err := requireStore(s); err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, err
	}
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
	if _, err := s.GetProject(version.ProjectID); err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, err
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("begin save chapter version: %w", err)
	}
	defer tx.Rollback(context.Background())
	chapterID := version.ChapterID
	var chapterProjectID string
	if err := tx.QueryRow(context.Background(), `SELECT project_id FROM chapters WHERE id=$1`, chapterID).Scan(&chapterProjectID); err != nil {
		if isNoRows(err) {
			return domain.ChapterVersion{}, domain.IndexJob{}, repository.NotFound("chapter", chapterID)
		}
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("get chapter %q: %w", chapterID, err)
	}
	if chapterProjectID != version.ProjectID {
		return domain.ChapterVersion{}, domain.IndexJob{}, repository.NotFound("chapter", chapterID)
	}
	if strings.TrimSpace(version.ID) == "" {
		id, err := s.NewID("chapter_version")
		if err != nil {
			return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("generate chapter version id: %w", err)
		}
		version.ID = id
	}
	var existingVersionID string
	err = tx.QueryRow(context.Background(), `SELECT id FROM chapter_versions WHERE id=$1`, version.ID).Scan(&existingVersionID)
	if err == nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, repository.Conflict("chapter version", version.ID, fmt.Sprintf("chapter version %q already exists", version.ID), nil)
	}
	if !isNoRows(err) {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("check chapter version %q existence: %w", version.ID, err)
	}
	version.ParentVersionID = strings.TrimSpace(version.ParentVersionID)
	if err := validateChapterVersionParentTx(tx, version); err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, err
	}
	chapterVersion, err := nextChapterVersionTx(tx, chapterID)
	if err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, err
	}
	version.Version = chapterVersion
	version.CreatedAt = now()
	if version.IndexStatus == "" {
		version.IndexStatus = "pending"
	}
	metadata, err := jsonbOrEmptyObject(version.Metadata)
	if err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, err
	}
	if _, err := tx.Exec(context.Background(), `
INSERT INTO chapter_versions(id, project_id, chapter_id, parent_version_id, version, title, content, summary, author_role, source_workflow_id, index_status, metadata, created_at)
VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12,$13)`, version.ID, version.ProjectID, version.ChapterID, version.ParentVersionID, version.Version, version.Title, version.Content, version.Summary, string(version.AuthorRole), version.SourceWorkflowID, version.IndexStatus, metadata, version.CreatedAt); err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("insert chapter version %q: %w", version.ID, err)
	}
	if _, err := tx.Exec(context.Background(), `
UPDATE index_jobs
SET status='superseded', error='superseded by newer pending job', updated_at=$3
WHERE project_id=$1 AND chapter_id=$2 AND status='pending'`, version.ProjectID, version.ChapterID, now()); err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("supersede pending index jobs for chapter %q: %w", version.ChapterID, err)
	}
	indexJobID, err := s.NewID("index_job")
	if err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("generate index job id: %w", err)
	}
	job := domain.IndexJob{ID: indexJobID, ProjectID: version.ProjectID, ChapterID: version.ChapterID, ChapterVersionID: version.ID, Kind: "chapter_reindex", Status: "pending", Payload: map[string]string{"trigger": "chapter_version_saved"}, CreatedAt: now(), UpdatedAt: now()}
	payload, err := jsonbOrEmptyObject(job.Payload)
	if err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, err
	}
	if _, err := tx.Exec(context.Background(), `
INSERT INTO index_jobs(id, project_id, chapter_id, chapter_version_id, kind, status, attempts, error, payload, scheduled_at, started_at, completed_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, job.ID, job.ProjectID, job.ChapterID, job.ChapterVersionID, job.Kind, job.Status, job.Attempts, job.Error, payload, job.ScheduledAt, job.StartedAt, job.CompletedAt, job.CreatedAt, job.UpdatedAt); err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("insert index job %q: %w", job.ID, err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		return domain.ChapterVersion{}, domain.IndexJob{}, fmt.Errorf("commit chapter version %q: %w", version.ID, err)
	}
	return version, job, nil
}

func (s *Store) GetChapterVersion(id string) (domain.ChapterVersion, error) {
	if err := requireStore(s); err != nil {
		return domain.ChapterVersion{}, err
	}
	row := s.pool.QueryRow(context.Background(), chapterVersionSelectSQL()+` WHERE id=$1`, id)
	item, err := scanChapterVersion(row)
	if err != nil {
		if isNoRows(err) {
			return domain.ChapterVersion{}, fmt.Errorf("chapter version %q not found", id)
		}
		return domain.ChapterVersion{}, fmt.Errorf("get chapter version %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) UpdateChapterVersionIndexStatus(id, status string) (domain.ChapterVersion, error) {
	if err := requireStore(s); err != nil {
		return domain.ChapterVersion{}, err
	}
	if strings.TrimSpace(id) == "" || strings.TrimSpace(status) == "" {
		return domain.ChapterVersion{}, fmt.Errorf("chapter version id and index status must not be empty")
	}
	result, err := s.pool.Exec(context.Background(), `UPDATE chapter_versions SET index_status=$2 WHERE id=$1`, id, status)
	if err != nil {
		return domain.ChapterVersion{}, fmt.Errorf("update chapter version %q index status: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return domain.ChapterVersion{}, fmt.Errorf("chapter version %q not found", id)
	}
	return s.GetChapterVersion(id)
}

func (s *Store) ListChapterVersions(projectID, chapterID string) ([]domain.ChapterVersion, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	projectID = strings.TrimSpace(projectID)
	chapterID = strings.TrimSpace(chapterID)
	if projectID == "" {
		return nil, fmt.Errorf("list chapter versions project_id must not be empty")
	}
	if _, err := s.GetProject(projectID); err != nil {
		return nil, err
	}
	if chapterID != "" {
		chapter, err := s.GetChapter(chapterID)
		if err != nil {
			return nil, err
		}
		if chapter.ProjectID != projectID {
			return nil, repository.NotFound("chapter", chapterID)
		}
	}
	query := chapterVersionSelectSQL() + ` WHERE project_id=$1`
	args := []any{projectID}
	if chapterID != "" {
		query += ` AND chapter_id=$2`
		args = append(args, chapterID)
	}
	query += ` ORDER BY chapter_id ASC, version DESC, created_at DESC, id DESC`
	rows, err := s.pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list chapter versions for project %q: %w", projectID, err)
	}
	defer rows.Close()
	items := make([]domain.ChapterVersion, 0)
	for rows.Next() {
		item, err := scanChapterVersion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chapter versions for project %q: %w", projectID, err)
	}
	return items, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func validateChapterVersionParentTx(tx pgx.Tx, version domain.ChapterVersion) error {
	if version.ParentVersionID == "" {
		return nil
	}
	if version.ParentVersionID == version.ID {
		return fmt.Errorf("chapter version %q cannot reference itself as parent", version.ID)
	}
	currentID := version.ParentVersionID
	visited := map[string]struct{}{version.ID: {}}
	for {
		if _, seen := visited[currentID]; seen {
			return fmt.Errorf("chapter version parent chain contains a cycle at %q", currentID)
		}
		visited[currentID] = struct{}{}
		var projectID string
		var chapterID string
		var parentVersionID *string
		err := tx.QueryRow(context.Background(), `SELECT project_id, chapter_id, parent_version_id FROM chapter_versions WHERE id=$1`, currentID).Scan(&projectID, &chapterID, &parentVersionID)
		if err != nil {
			if isNoRows(err) {
				if currentID == version.ParentVersionID {
					return repository.NotFound("chapter version", currentID)
				}
				return fmt.Errorf("chapter version parent chain references missing version %q", currentID)
			}
			return fmt.Errorf("get chapter version parent %q: %w", currentID, err)
		}
		if projectID != version.ProjectID || chapterID != version.ChapterID {
			return fmt.Errorf("chapter version parent chain crosses project or chapter at %q", currentID)
		}
		if parentVersionID == nil || strings.TrimSpace(*parentVersionID) == "" {
			return nil
		}
		currentID = strings.TrimSpace(*parentVersionID)
	}
}

func nextChapterVersionTx(tx pgx.Tx, chapterID string) (int, error) {
	var next int
	if err := tx.QueryRow(context.Background(), `SELECT COALESCE(MAX(version), 0) + 1 FROM chapter_versions WHERE chapter_id=$1`, chapterID).Scan(&next); err != nil {
		return 0, fmt.Errorf("next chapter version for chapter %q: %w", chapterID, err)
	}
	return next, nil
}

func chapterSelectSQL() string {
	return `SELECT id, project_id, number, title, status, metadata, created_at, updated_at FROM chapters`
}

func chapterVersionSelectSQL() string {
	return `SELECT id, project_id, chapter_id, COALESCE(parent_version_id, ''), version, title, content, summary, author_role, source_workflow_id, index_status, metadata, created_at FROM chapter_versions`
}

type chapterVersionScanner interface{ Scan(dest ...any) error }

func scanChapter(scanner chapterVersionScanner) (domain.Chapter, error) {
	var item domain.Chapter
	var metadata []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.Number, &item.Title, &item.Status, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.Chapter{}, err
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.Chapter{}, err
	}
	item.Metadata = parsedMetadata
	return item, nil
}

func scanChapterVersion(scanner chapterVersionScanner) (domain.ChapterVersion, error) {
	var item domain.ChapterVersion
	var authorRole string
	var metadata []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.ChapterID, &item.ParentVersionID, &item.Version, &item.Title, &item.Content, &item.Summary, &authorRole, &item.SourceWorkflowID, &item.IndexStatus, &metadata, &item.CreatedAt); err != nil {
		return domain.ChapterVersion{}, err
	}
	parsedMetadata, err := unmarshalJSONB[map[string]string](metadata)
	if err != nil {
		return domain.ChapterVersion{}, err
	}
	item.AuthorRole = domain.AgentRole(authorRole)
	item.Metadata = parsedMetadata
	return item, nil
}
