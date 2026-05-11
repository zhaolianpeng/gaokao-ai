package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gaokao-ai/backend/model"
)

type AuthRepository struct {
	db *observedDB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: observeDB(db)}
}

func (r *AuthRepository) GetUserByID(ctx context.Context, userID string) (*model.AuthUserRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT object_id, openid, phone, nickname, COALESCE(avatar_url, ''), COALESCE(id_card, ''), COALESCE(school_name, ''), COALESCE(school_year, ''), COALESCE(class_name, ''), COALESCE(student_no, ''), COALESCE(from_recommend, 0), login_type, created_at, updated_at, last_login_at
		FROM mini_auth_user
		WHERE object_id = ?
	`, userID)
	return scanAuthUser(row)
}

func (r *AuthRepository) GetUserByLegacyID(ctx context.Context, userID int) (*model.AuthUserRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT object_id, openid, phone, nickname, COALESCE(avatar_url, ''), COALESCE(id_card, ''), COALESCE(school_name, ''), COALESCE(school_year, ''), COALESCE(class_name, ''), COALESCE(student_no, ''), COALESCE(from_recommend, 0), login_type, created_at, updated_at, last_login_at
		FROM mini_auth_user
		WHERE id = ?
	`, userID)
	return scanAuthUser(row)
}

func (r *AuthRepository) GetUserByOpenID(ctx context.Context, openid string) (*model.AuthUserRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT object_id, openid, phone, nickname, COALESCE(avatar_url, ''), COALESCE(id_card, ''), COALESCE(school_name, ''), COALESCE(school_year, ''), COALESCE(class_name, ''), COALESCE(student_no, ''), COALESCE(from_recommend, 0), login_type, created_at, updated_at, last_login_at
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
		objectID, objectErr := newObjectID()
		if objectErr != nil {
			return nil, false, fmt.Errorf("generate object id: %w", objectErr)
		}
		result, execErr := r.db.ExecContext(ctx, `
			INSERT INTO mini_auth_user (object_id, openid, phone, nickname, avatar_url, login_type, created_at, updated_at, last_login_at)
			VALUES (?, ?, ?, '', '', 'wechat-phone', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, objectID, openid, phone)
		if execErr != nil {
			return nil, false, fmt.Errorf("insert mini_auth_user: %w", execErr)
		}
		_ = result
		createdUser, getErr := r.GetUserByID(ctx, objectID)
		if getErr != nil {
			return nil, false, getErr
		}
		return createdUser, true, nil
	}

	targetID := strings.TrimSpace(existing.ID)
	if targetID == "" {
		legacyUser, legacyErr := r.lookupUserObjectIDByOpenID(ctx, openid)
		if legacyErr != nil {
			return nil, false, legacyErr
		}
		targetID = legacyUser
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE mini_auth_user
		SET phone = ?, updated_at = CURRENT_TIMESTAMP, last_login_at = CURRENT_TIMESTAMP
		WHERE object_id = ?
	`, phone, targetID)
	if err != nil {
		return nil, false, fmt.Errorf("update mini_auth_user login: %w", err)
	}
	updatedUser, getErr := r.GetUserByID(ctx, targetID)
	if getErr != nil {
		return nil, false, getErr
	}
	return updatedUser, false, nil
}

func (r *AuthRepository) UpdateProfile(ctx context.Context, userID string, req model.WechatProfileUpdateRequest) (*model.AuthUserRecord, error) {
	assignments := []string{"nickname = ?"}
	args := []any{strings.TrimSpace(req.Nickname)}

	if req.Phone != nil {
		assignments = append(assignments, "phone = ?")
		args = append(args, strings.TrimSpace(*req.Phone))
	}
	if req.AvatarURL != nil {
		assignments = append(assignments, "avatar_url = ?")
		args = append(args, strings.TrimSpace(*req.AvatarURL))
	}
	if req.IDCard != nil {
		assignments = append(assignments, "id_card = ?")
		args = append(args, strings.ToUpper(strings.TrimSpace(*req.IDCard)))
	}
	if req.SchoolName != nil {
		assignments = append(assignments, "school_name = ?")
		args = append(args, strings.TrimSpace(*req.SchoolName))
	}
	if req.SchoolYear != nil {
		assignments = append(assignments, "school_year = ?")
		args = append(args, strings.TrimSpace(*req.SchoolYear))
	}
	if req.ClassName != nil {
		assignments = append(assignments, "class_name = ?")
		args = append(args, strings.TrimSpace(*req.ClassName))
	}
	if req.StudentNo != nil {
		assignments = append(assignments, "student_no = ?")
		args = append(args, strings.TrimSpace(*req.StudentNo))
	}
	if req.FromRecommend != nil {
		assignments = append(assignments, "from_recommend = ?")
		args = append(args, *req.FromRecommend)
	}

	assignments = append(assignments, "updated_at = CURRENT_TIMESTAMP", "last_login_at = CURRENT_TIMESTAMP")
	args = append(args, userID)
	_, err := r.db.ExecContext(ctx, `
		UPDATE mini_auth_user
		SET `+strings.Join(assignments, ", ")+`
		WHERE object_id = ?
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("update mini_auth_user profile: %w", err)
	}
	return r.GetUserByID(ctx, userID)
}

func (r *AuthRepository) lookupUserObjectIDByOpenID(ctx context.Context, openid string) (string, error) {
	trimmedOpenID := strings.TrimSpace(openid)
	if trimmedOpenID == "" {
		return "", fmt.Errorf("missing openid")
	}
	var objectID string
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(object_id, '') FROM mini_auth_user WHERE openid = ?`, trimmedOpenID).Scan(&objectID); err != nil {
		return "", err
	}
	objectID = strings.TrimSpace(objectID)
	if objectID != "" {
		return objectID, nil
	}
	generated, err := newObjectID()
	if err != nil {
		return "", err
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE mini_auth_user SET object_id = ?, updated_at = CURRENT_TIMESTAMP WHERE openid = ?`, generated, trimmedOpenID); err != nil {
		return "", err
	}
	return generated, nil
}

func (r *AuthRepository) ResolveUserID(ctx context.Context, rawUserID string) (string, error) {
	trimmed := strings.TrimSpace(rawUserID)
	if trimmed == "" {
		return "", fmt.Errorf("invalid user id")
	}
	if len(trimmed) == 24 {
		return trimmed, nil
	}
	legacyID, err := strconv.Atoi(trimmed)
	if err != nil || legacyID <= 0 {
		return "", fmt.Errorf("invalid user id")
	}
	record, err := r.GetUserByLegacyID(ctx, legacyID)
	if err != nil {
		return "", err
	}
	if record == nil || strings.TrimSpace(record.ID) == "" {
		return "", fmt.Errorf("invalid user id")
	}
	return strings.TrimSpace(record.ID), nil
}

func scanAuthUser(scanner interface{ Scan(dest ...any) error }) (*model.AuthUserRecord, error) {
	var record model.AuthUserRecord
	err := scanner.Scan(
		&record.ID,
		&record.OpenID,
		&record.Phone,
		&record.Nickname,
		&record.AvatarURL,
		&record.IDCard,
		&record.SchoolName,
		&record.SchoolYear,
		&record.ClassName,
		&record.StudentNo,
		&record.FromRecommend,
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
