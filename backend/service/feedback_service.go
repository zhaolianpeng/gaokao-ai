package service

import (
	"context"
	"fmt"
	"strings"

	"gaokao-ai/backend/model"
	"gaokao-ai/backend/repository"
)

type FeedbackService struct {
	repo *repository.FeedbackRepository
}

func NewFeedbackService(repo *repository.FeedbackRepository) *FeedbackService {
	return &FeedbackService{repo: repo}
}

func (s *FeedbackService) Submit(ctx context.Context, req model.FeedbackSubmitRequest) (*model.FeedbackSubmitResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, fmt.Errorf("missing feedback content")
	}
	id, err := s.repo.Create(ctx, model.FeedbackRecord{
		Content:        content,
		Contact:        strings.TrimSpace(req.Contact),
		Page:           strings.TrimSpace(req.Page),
		BackendBaseURL: strings.TrimSpace(req.BackendBaseURL),
		Phone:          strings.TrimSpace(req.Phone),
		Nickname:       strings.TrimSpace(req.Nickname),
	})
	if err != nil {
		return nil, err
	}
	return &model.FeedbackSubmitResponse{ID: fmt.Sprintf("%d", id), Message: "提交成功"}, nil
}
