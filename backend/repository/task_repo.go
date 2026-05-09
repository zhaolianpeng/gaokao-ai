package repository

import (
	"context"
	"database/sql"
	"fmt"

	"gaokao-ai/backend/model"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) CreateTask(ctx context.Context, title string, studentJSON, templatesJSON, recommendJSON []byte, demand, taskType string) (int, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_recommend_task (title, student, demand, templates, suggestions, status, provider, error_message, attempt_count, task_type, recommend)
		VALUES (?, ?, ?, ?, JSON_ARRAY(), 'pending', '', '', 0, ?, ?)
	`, title, string(studentJSON), demand, string(templatesJSON), taskType, string(recommendJSON))
	if err != nil {
		return 0, fmt.Errorf("insert task: %w", err)
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (r *TaskRepository) MarkProcessing(ctx context.Context, taskID int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE agent_recommend_task
		SET status = 'processing', attempt_count = attempt_count + 1, started_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, taskID)
	return err
}

func (r *TaskRepository) CompleteTask(ctx context.Context, taskID int, report, provider string, suggestionsJSON []byte) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE agent_recommend_task
		SET status = 'succeeded', report = ?, provider = ?, suggestions = ?, error_message = '', completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, report, provider, string(suggestionsJSON), taskID)
	return err
}

func (r *TaskRepository) FailTask(ctx context.Context, taskID int, errorMessage string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE agent_recommend_task
		SET status = 'failed', error_message = ?, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, errorMessage, taskID)
	return err
}

func (r *TaskRepository) GetTask(ctx context.Context, taskID int) (*model.TaskRecord, error) {
	var record model.TaskRecord
	var report sql.NullString
	var provider sql.NullString
	var errorMessage sql.NullString
	var startedAt sql.NullTime
	var completedAt sql.NullTime
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, student, demand, templates, suggestions, status, report, provider, error_message, attempt_count, task_type, recommend, created_at, updated_at, started_at, completed_at
		FROM agent_recommend_task
		WHERE id = ?
	`, taskID)
	err := row.Scan(
		&record.ID,
		&record.Title,
		&record.StudentJSON,
		&record.Demand,
		&record.TemplatesJSON,
		&record.SuggestionsJSON,
		&record.Status,
		&report,
		&provider,
		&errorMessage,
		&record.AttemptCount,
		&record.TaskType,
		&record.RecommendJSON,
		&record.CreatedAt,
		&record.UpdatedAt,
		&startedAt,
		&completedAt,
	)
	if err != nil {
		return nil, err
	}
	record.Report = report.String
	record.Provider = provider.String
	record.ErrorMessage = errorMessage.String
	if startedAt.Valid {
		record.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		record.CompletedAt = &completedAt.Time
	}
	return &record, nil
}
