package postgres

import (
	"context"
	"fmt"
	"strings"

	"aeonechoes/server/internal/domain"
)

func (s *Store) SaveWorkflow(workflow domain.AIWorkflow) (domain.AIWorkflow, error) {
	if err := requireStore(s); err != nil {
		return domain.AIWorkflow{}, err
	}
	if strings.TrimSpace(workflow.ProjectID) == "" || strings.TrimSpace(workflow.Kind) == "" {
		return domain.AIWorkflow{}, fmt.Errorf("workflow project_id and kind must not be empty")
	}
	if strings.TrimSpace(workflow.ID) == "" {
		id, err := s.NewID("workflow")
		if err != nil {
			return domain.AIWorkflow{}, fmt.Errorf("generate workflow id: %w", err)
		}
		workflow.ID = id
		workflow.CreatedAt = now()
	} else if existing, err := s.getWorkflow(workflow.ID); err == nil {
		workflow.CreatedAt = existing.CreatedAt
	} else if !strings.Contains(err.Error(), "not found") {
		return domain.AIWorkflow{}, err
	} else {
		workflow.CreatedAt = now()
	}
	workflow.UpdatedAt = now()
	steps, input, output, err := workflowJSON(workflow)
	if err != nil {
		return domain.AIWorkflow{}, err
	}
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO ai_workflows(id, project_id, kind, role, status, model_id, context_pack_id, steps, input, output, error, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (id) DO UPDATE SET project_id=EXCLUDED.project_id, kind=EXCLUDED.kind, role=EXCLUDED.role, status=EXCLUDED.status,
    model_id=EXCLUDED.model_id, context_pack_id=EXCLUDED.context_pack_id, steps=EXCLUDED.steps, input=EXCLUDED.input,
    output=EXCLUDED.output, error=EXCLUDED.error, updated_at=EXCLUDED.updated_at`, workflow.ID, workflow.ProjectID, workflow.Kind, string(workflow.Role), workflow.Status, workflow.ModelID, workflow.ContextPackID, steps, input, output, workflow.Error, workflow.CreatedAt, workflow.UpdatedAt)
	if err != nil {
		return domain.AIWorkflow{}, fmt.Errorf("save workflow %q: %w", workflow.ID, err)
	}
	return workflow, nil
}

func (s *Store) ListWorkflows(projectID string) ([]domain.AIWorkflow, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	query := workflowSelectSQL()
	args := make([]any, 0, 1)
	if cleanProjectID := strings.TrimSpace(projectID); cleanProjectID != "" {
		query += ` WHERE project_id=$1`
		args = append(args, cleanProjectID)
	}
	query += ` ORDER BY created_at ASC, id ASC`
	rows, err := s.pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	defer rows.Close()
	items := make([]domain.AIWorkflow, 0)
	for rows.Next() {
		item, err := scanWorkflow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflows: %w", err)
	}
	return items, nil
}

func (s *Store) GetWorkflow(id string) (domain.AIWorkflow, error) {
	if err := requireStore(s); err != nil {
		return domain.AIWorkflow{}, err
	}
	return s.getWorkflow(id)
}

func (s *Store) getWorkflow(id string) (domain.AIWorkflow, error) {
	row := s.pool.QueryRow(context.Background(), workflowSelectSQL()+` WHERE id=$1`, id)
	item, err := scanWorkflow(row)
	if err != nil {
		if isNoRows(err) {
			return domain.AIWorkflow{}, fmt.Errorf("workflow %q not found", id)
		}
		return domain.AIWorkflow{}, fmt.Errorf("get workflow %q: %w", id, err)
	}
	return item, nil
}

func workflowJSON(workflow domain.AIWorkflow) ([]byte, []byte, []byte, error) {
	steps, err := jsonbOrEmptyArray(workflow.Steps)
	if err != nil {
		return nil, nil, nil, err
	}
	input, err := jsonbOrEmptyObject(workflow.Input)
	if err != nil {
		return nil, nil, nil, err
	}
	output, err := jsonbOrEmptyObject(workflow.Output)
	if err != nil {
		return nil, nil, nil, err
	}
	return steps, input, output, nil
}

func workflowSelectSQL() string {
	return `SELECT id, project_id, kind, role, status, COALESCE(model_id, ''), context_pack_id, steps, input, output, error, created_at, updated_at FROM ai_workflows`
}

type workflowScanner interface{ Scan(dest ...any) error }

func scanWorkflow(scanner workflowScanner) (domain.AIWorkflow, error) {
	var item domain.AIWorkflow
	var role string
	var steps []byte
	var input []byte
	var output []byte
	if err := scanner.Scan(&item.ID, &item.ProjectID, &item.Kind, &role, &item.Status, &item.ModelID, &item.ContextPackID, &steps, &input, &output, &item.Error, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.AIWorkflow{}, err
	}
	var err error
	if item.Steps, err = unmarshalJSONB[[]domain.WorkflowStep](steps); err != nil {
		return domain.AIWorkflow{}, err
	}
	if item.Input, err = unmarshalJSONB[map[string]string](input); err != nil {
		return domain.AIWorkflow{}, err
	}
	if item.Output, err = unmarshalJSONB[map[string]string](output); err != nil {
		return domain.AIWorkflow{}, err
	}
	item.Role = domain.AgentRole(role)
	return item, nil
}
