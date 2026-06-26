package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
)

func (s *Store) CreateIndexJob(job domain.IndexJob) (domain.IndexJob, error) {
	if err := requireStore(s); err != nil {
		return domain.IndexJob{}, err
	}
	if strings.TrimSpace(job.ProjectID) == "" || strings.TrimSpace(job.Kind) == "" {
		return domain.IndexJob{}, fmt.Errorf("index job project_id and kind must not be empty")
	}
	if strings.TrimSpace(job.ID) == "" {
		id, err := s.NewID("index_job")
		if err != nil {
			return domain.IndexJob{}, fmt.Errorf("generate index job id: %w", err)
		}
		job.ID = id
	}
	n := now()
	job.CreatedAt = n
	job.UpdatedAt = n
	if job.Status == "" {
		job.Status = "pending"
	}
	payload, err := jsonbOrEmptyObject(job.Payload)
	if err != nil {
		return domain.IndexJob{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO index_jobs(id, project_id, chapter_id, chapter_version_id, kind, status, attempts, error, payload, scheduled_at, started_at, completed_at, created_at, updated_at)
VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, job.ID, job.ProjectID, job.ChapterID, job.ChapterVersionID, job.Kind, job.Status, job.Attempts, job.Error, payload, job.ScheduledAt, job.StartedAt, job.CompletedAt, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return domain.IndexJob{}, fmt.Errorf("insert index job %q: %w", job.ID, err)
	}
	return job, nil
}

func (s *Store) GetIndexJob(id string) (domain.IndexJob, error) {
	if err := requireStore(s); err != nil {
		return domain.IndexJob{}, err
	}
	row := s.pool.QueryRow(context.Background(), indexJobSelectSQL()+` WHERE id=$1`, id)
	item, err := scanIndexJob(row)
	if err != nil {
		if isNoRows(err) {
			return domain.IndexJob{}, fmt.Errorf("index job %q not found", id)
		}
		return domain.IndexJob{}, fmt.Errorf("get index job %q: %w", id, err)
	}
	return item, nil
}

func (s *Store) UpdateIndexJobStatus(id, status, errorMessage string) (domain.IndexJob, error) {
	if err := requireStore(s); err != nil {
		return domain.IndexJob{}, err
	}
	if strings.TrimSpace(id) == "" || strings.TrimSpace(status) == "" {
		return domain.IndexJob{}, fmt.Errorf("index job id and status must not be empty")
	}
	n := now()
	query := `UPDATE index_jobs SET status=$2, error=$3, updated_at=$4`
	args := []any{id, status, errorMessage, n}
	switch status {
	case "running":
		query += `, started_at=$4, attempts=attempts+1`
	case "completed", "failed":
		query += `, completed_at=$4`
	}
	query += ` WHERE id=$1`
	result, err := s.pool.Exec(context.Background(), query, args...)
	if err != nil {
		return domain.IndexJob{}, fmt.Errorf("update index job %q status: %w", id, err)
	}
	if result.RowsAffected() != 1 {
		return domain.IndexJob{}, fmt.Errorf("index job %q not found", id)
	}
	return s.GetIndexJob(id)
}

func (s *Store) ListIndexJobs(projectID string) ([]domain.IndexJob, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	query := indexJobSelectSQL()
	args := []any{}
	if strings.TrimSpace(projectID) != "" {
		query += ` WHERE project_id=$1`
		args = append(args, projectID)
	}
	query += ` ORDER BY created_at DESC, id DESC`
	return s.queryIndexJobs(query, args...)
}

func (s *Store) ListPendingIndexJobs(projectID string, limit int) ([]domain.IndexJob, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	query := indexJobSelectSQL() + ` WHERE status='pending'`
	args := []any{}
	if strings.TrimSpace(projectID) != "" {
		query += ` AND project_id=$1`
		args = append(args, projectID)
	}
	query += ` ORDER BY created_at ASC, id ASC`
	if limit > 0 {
		args = append(args, limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	return s.queryIndexJobs(query, args...)
}

func (s *Store) queryIndexJobs(sql string, args ...any) ([]domain.IndexJob, error) {
	rows, err := s.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query index jobs: %w", err)
	}
	defer rows.Close()
	items := make([]domain.IndexJob, 0)
	for rows.Next() {
		item, err := scanIndexJob(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate index jobs: %w", err)
	}
	return items, nil
}

func indexJobSelectSQL() string {
	return `SELECT id, project_id, COALESCE(chapter_id, ''), COALESCE(chapter_version_id, ''), kind, status, attempts, error, payload, created_at, updated_at, scheduled_at, started_at, completed_at FROM index_jobs`
}

type indexJobScanner interface{ Scan(dest ...any) error }

func scanIndexJob(scanner indexJobScanner) (domain.IndexJob, error) {
	var item domain.IndexJob
	var payload []byte
	var scheduledAt *time.Time
	var startedAt *time.Time
	var completedAt *time.Time
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.ChapterID, &item.ChapterVersionID, &item.Kind, &item.Status, &item.Attempts, &item.Error, &payload, &item.CreatedAt, &item.UpdatedAt, &scheduledAt, &startedAt, &completedAt); err != nil {
		return domain.IndexJob{}, err
	}
	parsedPayload, err := unmarshalJSONB[map[string]string](payload)
	if err != nil {
		return domain.IndexJob{}, err
	}
	item.Payload = parsedPayload
	item.ScheduledAt = scheduledAt
	item.StartedAt = startedAt
	item.CompletedAt = completedAt
	return item, nil
}
