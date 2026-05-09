package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gaokao-ai/backend/model"
)

type AuthRepository struct {
	db *observedDB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: observeDB(db)}
}

func (r *AuthRepository) GetUserByID(ctx context.Context, userID int) (*model.AuthUserRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, openid, phone, nickname, COALESCE(avatar_url, ''), login_type, created_at, updated_at, last_login_at
		FROM mini_auth_user
		WHERE id = ?
	`, userID)
	return scanAuthUser(row)
}

func (r *AuthRepository) GetUserByOpenID(ctx context.Context, openid string) (*model.AuthUserRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, openid, phone, nickname, COALESCE(avatar_url, ''), login_type, created_at, updated_at, last_login_at
		FROM mini_auth_user
		WHERE openid = ?
	`, openid)
	return scanAuthUser(row)
}

func (r *AuthRepository) UpsertWechatUser(ctx context.Context, openid, phone string) (*model.AuthUserRecord, bool, error) {
	existing, err := r.GetUserByOpenID(ctx, openid)
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}
	if existing == nil {
		result, execErr := r.db.ExecContext(ctx, `
			INSERT INTO mini_auth_user (openid, phone, nickname, avatar_url, login_type, created_at, updated_at, last_login_at)
			VALUES (?, ?, '', '', 'wechat-phone', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, openid, phone)
		if execErr != nil {
			return nil, false, fmt.Errorf("insert mini_auth_user: %w", execErr)
		}
		insertedID, _ := result.LastInsertId()
		createdUser, getErr := r.GetUserByID(ctx, int(insertedID))
		if getErr != nil {
			return nil, false, getErr
		}
		return createdUser, true, nil
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE mini_auth_user
		SET phone = ?, updated_at = CURRENT_TIMESTAMP, last_login_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, phone, existing.ID)
	if err != nil {
		return nil, false, fmt.Errorf("update mini_auth_user login: %w", err)
	}
	updatedUser, getErr := r.GetUserByID(ctx, existing.ID)
	if getErr != nil {
		return nil, false, getErr
	}
	return updatedUser, false, nil
}

func (r *AuthRepository) UpdateProfile(ctx context.Context, userID int, phone, nickname, avatarURL string) (*model.AuthUserRecord, error) {
	_, err := r.db.ExecContext(ctx, `
		UPDATE mini_auth_user
		SET phone = CASE WHEN ? <> '' THEN ? ELSE phone END,
			nickname = ?,
			avatar_url = CASE WHEN ? <> '' THEN ? ELSE avatar_url END,
			updated_at = CURRENT_TIMESTAMP,
			last_login_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, phone, phone, nickname, avatarURL, avatarURL, userID)
	if err != nil {
		return nil, fmt.Errorf("update mini_auth_user profile: %w", err)
	}
	return r.GetUserByID(ctx, userID)
}

func scanAuthUser(scanner interface{ Scan(dest ...any) error }) (*model.AuthUserRecord, error) {
	var record model.AuthUserRecord
	err := scanner.Scan(
		&record.ID,
		&record.OpenID,
		&record.Phone,
		&record.Nickname,
		&record.AvatarURL,
		&record.LoginType,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.LastLoginAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	return &record, nil
}
