package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"

	"gaokao-ai/backend/model"
	"gaokao-ai/backend/repository"
)

type AdminService struct {
	repo     *repository.AdminRepository
	mu       sync.RWMutex
	sessions map[string]int
}

func NewAdminService(repo *repository.AdminRepository) *AdminService {
	return &AdminService{repo: repo, sessions: make(map[string]int)}
}

func (s *AdminService) EnsureBootstrap(ctx context.Context) error {
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.EnsureBootstrap(ctx, string(hash))
}

func (s *AdminService) ShouldShowVIPEntry(ctx context.Context) (bool, error) {
	return s.repo.ShouldShowVIPEntry(ctx)
}

func (s *AdminService) ShareGateConfig(ctx context.Context) (*model.ShareGateConfigResponse, error) {
	return s.repo.GetShareGateConfig(ctx)
}

func (s *AdminService) Login(ctx context.Context, username, password string) (string, *model.AdminUser, error) {
	user, err := s.repo.GetAdminUserAuthByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return "", nil, fmt.Errorf("用户名或密码错误")
	}
	if user.Status != "enabled" {
		return "", nil, fmt.Errorf("账号已被禁用")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, fmt.Errorf("用户名或密码错误")
	}
	if err := s.repo.TouchAdminLogin(ctx, user.ID); err != nil {
		return "", nil, err
	}
	token, err := randomToken(32)
	if err != nil {
		return "", nil, err
	}
	s.mu.Lock()
	s.sessions[token] = user.ID
	s.mu.Unlock()
	current, err := s.repo.GetAdminUserByID(ctx, user.ID)
	if err != nil {
		return "", nil, err
	}
	return token, current, nil
}

func (s *AdminService) CurrentUser(ctx context.Context, token string) (*model.AdminUser, error) {
	s.mu.RLock()
	userID, ok := s.sessions[strings.TrimSpace(token)]
	s.mu.RUnlock()
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("未登录或会话已过期")
	}
	return s.repo.GetAdminUserByID(ctx, userID)
}

func (s *AdminService) Logout(token string) {
	s.mu.Lock()
	delete(s.sessions, strings.TrimSpace(token))
	s.mu.Unlock()
}

func (s *AdminService) Repo() *repository.AdminRepository {
	return s.repo
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func randomToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
