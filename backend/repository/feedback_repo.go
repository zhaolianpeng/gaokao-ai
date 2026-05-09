package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gaokao-ai/backend/model"
)

type FeedbackRepository struct {
	db     *sql.DB
	driver string
}

func NewFeedbackRepository(db *sql.DB, driver string) (*FeedbackRepository, error) {
	repo := &FeedbackRepository{db: db, driver: strings.ToLower(strings.TrimSpace(driver))}
	if err := repo.ensureTable(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *FeedbackRepository) ensureTable() error {
	if r.driver == "" || r.driver == "mysql" {
		_, err := r.db.Exec(`
			CREATE TABLE IF NOT EXISTS mini_feedback (
				id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
				content TEXT NOT NULL,
				contact VARCHAR(255) NOT NULL DEFAULT '',
				page VARCHAR(128) NOT NULL DEFAULT '',
				backend_base_url VARCHAR(255) NOT NULL DEFAULT '',
				phone VARCHAR(64) NOT NULL DEFAULT '',
				nickname VARCHAR(128) NOT NULL DEFAULT '',
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			return fmt.Errorf("create mini_feedback table: %w", err)
		}
		return nil
	}
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS mini_feedback (
			id BIGSERIAL PRIMARY KEY,
			content TEXT NOT NULL,
			contact VARCHAR(255) NOT NULL DEFAULT '',
			page VARCHAR(128) NOT NULL DEFAULT '',
			backend_base_url VARCHAR(255) NOT NULL DEFAULT '',
			phone VARCHAR(64) NOT NULL DEFAULT '',
			nickname VARCHAR(128) NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("create mini_feedback table: %w", err)
	}
	return nil
}

func (r *FeedbackRepository) Create(ctx context.Context, record model.FeedbackRecord) (int, error) {
	query := `
		INSERT INTO mini_feedback (content, contact, page, backend_base_url, phone, nickname)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	args := []any{record.Content, record.Contact, record.Page, record.BackendBaseURL, record.Phone, record.Nickname}
	if r.driver != "" && r.driver != "mysql" {
		query = `
			INSERT INTO mini_feedback (content, contact, page, backend_base_url, phone, nickname)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`
		var id int
		if err := r.db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert mini_feedback: %w", err)
		}
		return id, nil
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("insert mini_feedback: %w", err)
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}
