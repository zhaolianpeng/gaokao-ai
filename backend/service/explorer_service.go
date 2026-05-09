package service

import (
	"context"
	"fmt"
	"strings"

	"gaokao-ai/backend/model"
	"gaokao-ai/backend/repository"
)

type ExplorerService struct {
	repo  *repository.ExpertRepository
	cache *ResultCache
}

func NewExplorerService(repo *repository.ExpertRepository, cache *ResultCache) *ExplorerService {
	return &ExplorerService{repo: repo, cache: cache}
}

func (s *ExplorerService) GetDashboardOverview(ctx context.Context, province string, year int, subject string) (model.DashboardOverview, error) {
	key := fmt.Sprintf("dashboard:%s:%d:%s", strings.TrimSpace(province), year, strings.TrimSpace(subject))
	return rememberJSON(ctx, s.cache, key, func() (model.DashboardOverview, error) {
		return s.repo.GetDashboardOverview(ctx, province, year, subject)
	})
}

func (s *ExplorerService) GetProvinceScoreLines(ctx context.Context, province string, year int, subject string) ([]model.ProvinceScoreLineItem, error) {
	key := fmt.Sprintf("province-lines:%s:%d:%s", strings.TrimSpace(province), year, strings.TrimSpace(subject))
	return rememberJSON(ctx, s.cache, key, func() ([]model.ProvinceScoreLineItem, error) {
		return s.repo.GetProvinceScoreLines(ctx, province, year, subject)
	})
}

func (s *ExplorerService) LookupScoreRank(ctx context.Context, province string, year int, subject string, score int) (model.ScoreRankLookup, error) {
	key := fmt.Sprintf("score-rank:%s:%d:%s:%d", strings.TrimSpace(province), year, strings.TrimSpace(subject), score)
	return rememberJSON(ctx, s.cache, key, func() (model.ScoreRankLookup, error) {
		return s.repo.LookupScoreRank(ctx, province, year, subject, score)
	})
}

func (s *ExplorerService) ListColleges(ctx context.Context, filter model.CollegeListFilter) (model.CollegeListResponse, error) {
	key := fmt.Sprintf("colleges:%s:%d:%s:%s:%s:%d:%d", strings.TrimSpace(filter.Province), filter.Year, strings.TrimSpace(filter.Subject), strings.TrimSpace(filter.Keyword), strings.TrimSpace(filter.SortMode), filter.Page, filter.Limit)
	return rememberJSON(ctx, s.cache, key, func() (model.CollegeListResponse, error) {
		items, hasMore, err := s.repo.ListColleges(ctx, filter)
		if err != nil {
			return model.CollegeListResponse{}, err
		}
		return model.CollegeListResponse{
			Items:    items,
			SortMode: filter.SortMode,
			Page:     filter.Page,
			Limit:    filter.Limit,
			HasMore:  hasMore,
		}, nil
	})
}

func (s *ExplorerService) GetCollegeDetail(ctx context.Context, collegeID int, province string, year int, subject string) (model.CollegeDetail, error) {
	key := fmt.Sprintf("college-detail:%d:%s:%d:%s", collegeID, strings.TrimSpace(province), year, strings.TrimSpace(subject))
	return rememberJSON(ctx, s.cache, key, func() (model.CollegeDetail, error) {
		return s.repo.GetCollegeDetail(ctx, collegeID, province, year, subject)
	})
}
