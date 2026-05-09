package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"gaokao-ai/backend/logging"
	"gaokao-ai/backend/model"
	"gaokao-ai/backend/repository"
)

type RecommendService struct {
	repo  *repository.CollegeRepository
	cache *ResultCache
}

func NewRecommendService(repo *repository.CollegeRepository, cache *ResultCache) *RecommendService {
	return &RecommendService{repo: repo, cache: cache}
}

func (s *RecommendService) Recommend(ctx context.Context, req model.RecommendRequest) (model.RecommendResponse, error) {
	cacheKey := fmt.Sprintf("recommend:%s:%d:%d:%s:%d:%s", strings.TrimSpace(req.Province), req.Score, req.Rank, strings.TrimSpace(req.Subject), req.Year, strings.TrimSpace(req.TargetMajor))
	return rememberJSON(ctx, s.cache, cacheKey, func() (model.RecommendResponse, error) {
		logging.LogEvent("recommend_start", map[string]any{"province": req.Province, "subject": req.Subject, "year": req.Year, "score": req.Score, "rank": req.Rank, "targetMajor": req.TargetMajor})
		if req.Province == "" {
			req.Province = "黑龙江"
		}
		if req.Year == 0 {
			req.Year = 2025
		}
		items, err := s.repo.ListAdmissionLines(ctx, req.Province, req.Subject, req.Year, req.TargetMajor, 1000)
		if err != nil {
			return model.RecommendResponse{}, err
		}

		chong := make([]model.RecommendItem, 0)
		wen := make([]model.RecommendItem, 0)
		bao := make([]model.RecommendItem, 0)
		bands := buildRecommendBands(req.Rank)
		logging.LogEvent("recommend_bands", map[string]any{"rank": req.Rank, "chongLower": bands.ChongLower, "wenUpper": bands.WenUpper, "baoUpper": bands.BaoUpper, "candidateCount": len(items)})

		for _, item := range items {
			if item.MinRank == 0 {
				continue
			}
			diff := item.MinRank - req.Rank
			tag, ok := classifyByRankDiff(diff, bands)
			if !ok {
				continue
			}
			item.Probability = estimateProbability(diff)
			item.RecommendationReason = buildReason(req, item, diff)

			switch tag {
			case "chong":
				item.Tag = tag
				chong = append(chong, item)
			case "wen":
				item.Tag = tag
				wen = append(wen, item)
			case "bao":
				item.Tag = tag
				bao = append(bao, item)
			}
		}

		sort.Slice(chong, func(i, j int) bool {
			return absRankDiff(chong[i].MinRank, req.Rank) < absRankDiff(chong[j].MinRank, req.Rank)
		})
		sort.Slice(wen, func(i, j int) bool {
			return absRankDiff(wen[i].MinRank, req.Rank) < absRankDiff(wen[j].MinRank, req.Rank)
		})
		sort.Slice(bao, func(i, j int) bool {
			return absRankDiff(bao[i].MinRank, req.Rank) < absRankDiff(bao[j].MinRank, req.Rank)
		})

		return model.RecommendResponse{
			Chong: trim(chong, 10),
			Wen:   trim(wen, 20),
			Bao:   trim(bao, 20),
		}, nil
	})
}

func buildReason(req model.RecommendRequest, item model.RecommendItem, rankDiff int) string {
	reason := item.RecommendationReason
	if reason == "" {
		reason = "按黑龙江专业组最低位次匹配"
	}
	if req.TargetMajor != "" && item.MatchedMajor != "" {
		reason += "；命中意向专业：" + item.MatchedMajor
	}
	if rankDiff < 0 {
		reason += "；当前定位偏冲刺"
	} else if rankDiff <= buildRecommendBands(req.Rank).WenUpper {
		reason += "；当前定位偏稳妥"
	} else {
		reason += "；当前定位偏保底"
	}
	return reason
}

type recommendBands struct {
	ChongLower int
	WenUpper   int
	BaoUpper   int
}

func buildRecommendBands(rank int) recommendBands {
	base := int(math.Max(800, math.Min(5000, float64(rank)*0.12)))
	wenUpper := int(math.Max(1200, math.Min(8000, float64(rank)*0.18)))
	baoUpper := int(math.Max(2500, math.Min(15000, float64(rank)*0.35)))
	return recommendBands{
		ChongLower: -base,
		WenUpper:   wenUpper,
		BaoUpper:   baoUpper,
	}
}

func classifyByRankDiff(rankDiff int, bands recommendBands) (string, bool) {
	switch {
	case rankDiff < bands.ChongLower:
		return "", false
	case rankDiff < 0:
		return "chong", true
	case rankDiff <= bands.WenUpper:
		return "wen", true
	case rankDiff <= bands.BaoUpper:
		return "bao", true
	default:
		return "", false
	}
}

func absRankDiff(minRank, userRank int) int {
	diff := minRank - userRank
	if diff < 0 {
		return -diff
	}
	return diff
}

func trim(items []model.RecommendItem, size int) []model.RecommendItem {
	if len(items) <= size {
		return items
	}
	return items[:size]
}

func estimateProbability(rankDiff int) float64 {
	switch {
	case rankDiff > 5000:
		return 0.90
	case rankDiff > 2000:
		return 0.70
	case rankDiff >= 0:
		return 0.50
	case rankDiff >= -1000:
		return 0.35
	default:
		return 0.30
	}
}
