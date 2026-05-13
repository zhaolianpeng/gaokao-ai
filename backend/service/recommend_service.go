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
	cacheKey := fmt.Sprintf("recommend:v3:%s:%d:%d:%s:%d:%s", strings.TrimSpace(req.Province), req.Score, req.Rank, strings.TrimSpace(req.Subject), req.Year, strings.TrimSpace(req.TargetMajor))
	return rememberJSON(ctx, s.cache, cacheKey, func() (model.RecommendResponse, error) {
		logging.LogEvent("recommend_start", map[string]any{"province": req.Province, "subject": req.Subject, "year": req.Year, "score": req.Score, "rank": req.Rank, "targetMajor": req.TargetMajor})
		if req.Province == "" {
			req.Province = "黑龙江"
		}
		if req.Year == 0 {
			req.Year = 2025
		}
		items, err := s.repo.ListAdmissionLines(ctx, req.Province, req.Subject, req.Year, req.TargetMajor, req.Rank, 1000)
		if err != nil {
			return model.RecommendResponse{}, err
		}

		chong := make([]model.RecommendItem, 0)
		wen := make([]model.RecommendItem, 0)
		bao := make([]model.RecommendItem, 0)
		bands := buildRecommendBands(req.Rank)
		logging.LogEvent("recommend_bands", map[string]any{"rank": req.Rank, "chongLowerRatio": bands.ChongLowerRatio, "wenLowerRatio": bands.WenLowerRatio, "wenUpperRatio": bands.WenUpperRatio, "baoUpperRatio": bands.BaoUpperRatio, "candidateCount": len(items)})

		for _, item := range items {
			if item.MinRank == 0 {
				continue
			}
			benchmarkRank := buildBenchmarkRank(item)
			if benchmarkRank == 0 {
				continue
			}
			diff := benchmarkRank - req.Rank
			tag, ok := classifyByRankDiff(diff, benchmarkRank, bands)
			if !ok {
				continue
			}
			item.Probability = estimateProbability(req, item, diff)
			item.ProbabilityLabel = buildProbabilityLabel(item.Probability)
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
			return compareBucketItems(chong[i], chong[j], req.Rank, req.Score, "chong")
		})
		sort.Slice(wen, func(i, j int) bool {
			return compareBucketItems(wen[i], wen[j], req.Rank, req.Score, "wen")
		})
		sort.Slice(bao, func(i, j int) bool {
			return compareBucketItems(bao[i], bao[j], req.Rank, req.Score, "bao")
		})

		return model.RecommendResponse{
			Chong: trim(chong, 10),
			Wen:   trim(wen, 20),
			Bao:   trim(bao, 10),
		}, nil
	})
}

func buildReason(req model.RecommendRequest, item model.RecommendItem, rankDiff int) string {
	parts := make([]string, 0, 4)
	historyText := buildHistoryRankSummary(item)
	if historyText != "" {
		parts = append(parts, historyText)
	}
	if item.WeightedRank > 0 && req.Rank > 0 {
		if rankDiff < 0 {
			parts = append(parts, fmt.Sprintf("你当前位次 %d，略高于该组近年主流录取位次 %d，适合作为冲刺尝试。", req.Rank, item.WeightedRank))
		} else if normalizedRankGap(item.WeightedRank, req.Rank) <= 0.08 {
			parts = append(parts, fmt.Sprintf("你当前位次 %d，与该组近年主流录取位次 %d 接近，家长一般会把这类学校放进主力稳妥区。", req.Rank, item.WeightedRank))
		} else {
			parts = append(parts, fmt.Sprintf("你当前位次 %d，明显优于该组近年主流录取位次 %d，这类学校更适合承担保底职责。", req.Rank, item.WeightedRank))
		}
	}
	if req.Score > 0 && item.ScoreLastYear > 0 {
		scoreDiff := req.Score - item.ScoreLastYear
		if scoreDiff >= 8 {
			parts = append(parts, fmt.Sprintf("按去年最低分看，你高出约 %d 分，分数层面相对更稳。", scoreDiff))
		} else if scoreDiff >= -3 {
			parts = append(parts, fmt.Sprintf("按去年最低分看，你和该组只差 %d 分以内，属于可以重点比较的区间。", absInt(scoreDiff)))
		} else {
			parts = append(parts, fmt.Sprintf("按去年最低分看，你还低约 %d 分，更适合少量前置冲刺。", -scoreDiff))
		}
	}
	if req.TargetMajor != "" && item.MatchedMajor != "" {
		parts = append(parts, "该组已直接命中你的意向专业："+item.MatchedMajor+"。")
	} else if req.TargetMajor != "" {
		parts = append(parts, "学校层次可以参考，但正式填报前还要逐一核查组内专业是否覆盖你的意向方向。")
	}
	parts = append(parts, "综合判断："+item.ProbabilityLabel+"。")
	return strings.Join(parts, "")
}

type recommendBands struct {
	ChongLowerRatio float64
	WenLowerRatio   float64
	WenUpperRatio   float64
	BaoUpperRatio   float64
	ChongLowerAbs   int
}

func buildRecommendBands(rank int) recommendBands {
	chongLowerAbs := int(math.Max(800, math.Min(6000, float64(rank)*0.18)))
	return recommendBands{
		ChongLowerRatio: -0.18,
		WenLowerRatio:   -0.05,
		WenUpperRatio:   0.08,
		BaoUpperRatio:   0.30,
		ChongLowerAbs:   chongLowerAbs,
	}
}

func normalizedRankGap(minRank, userRank int) float64 {
	if minRank <= 0 || userRank <= 0 {
		return 0
	}
	return float64(minRank-userRank) / float64(minRank)
}

func buildBenchmarkRank(item model.RecommendItem) int {
	ranks := []struct {
		value  int
		weight float64
	}{
		{value: item.RankLastYear, weight: 0.5},
		{value: item.RankTwoYearsAgo, weight: 0.3},
		{value: item.RankThreeYearsAgo, weight: 0.2},
	}
	var total float64
	var weighted float64
	for _, rank := range ranks {
		if rank.value <= 0 {
			continue
		}
		total += rank.weight
		weighted += float64(rank.value) * rank.weight
	}
	if total > 0 {
		return int(math.Round(weighted / total))
	}
	return item.MinRank
}

func classifyByRankDiff(rankDiff int, minRank int, bands recommendBands) (string, bool) {
	ratio := normalizedRankGap(minRank, minRank-rankDiff)
	switch {
	case ratio < bands.ChongLowerRatio && rankDiff < -bands.ChongLowerAbs:
		return "", false
	case ratio < bands.WenLowerRatio:
		return "chong", true
	case ratio <= bands.WenUpperRatio:
		return "wen", true
	case ratio <= bands.BaoUpperRatio:
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

func bucketTargetGap(bucket string) float64 {
	switch bucket {
	case "chong":
		return -0.08
	case "bao":
		return 0.16
	default:
		return 0.02
	}
}

func bucketTargetScoreGap(bucket string) int {
	switch bucket {
	case "chong":
		return -2
	case "bao":
		return 12
	default:
		return 3
	}
}

func compareBucketItems(left, right model.RecommendItem, userRank int, userScore int, bucket string) bool {
	leftFit := math.Abs(normalizedRankGap(buildBenchmarkRank(left), userRank) - bucketTargetGap(bucket))
	rightFit := math.Abs(normalizedRankGap(buildBenchmarkRank(right), userRank) - bucketTargetGap(bucket))
	if leftFit != rightFit {
		return leftFit < rightFit
	}
	leftScoreFit := scoreFitDistance(left.ScoreLastYear, userScore, bucket)
	rightScoreFit := scoreFitDistance(right.ScoreLastYear, userScore, bucket)
	if leftScoreFit != rightScoreFit {
		return leftScoreFit < rightScoreFit
	}
	if left.Probability != right.Probability {
		return left.Probability > right.Probability
	}
	if (left.MatchedMajor != "") != (right.MatchedMajor != "") {
		return left.MatchedMajor != ""
	}
	if left.PlanCount != right.PlanCount {
		return left.PlanCount > right.PlanCount
	}
	return absRankDiff(left.MinRank, userRank) < absRankDiff(right.MinRank, userRank)
}

func scoreFitDistance(lastYearScore int, userScore int, bucket string) int {
	if lastYearScore <= 0 || userScore <= 0 {
		return 999
	}
	return absInt((userScore - lastYearScore) - bucketTargetScoreGap(bucket))
}

func estimateProbability(req model.RecommendRequest, item model.RecommendItem, rankDiff int) float64 {
	benchmarkRank := buildBenchmarkRank(item)
	ratio := normalizedRankGap(benchmarkRank, benchmarkRank-rankDiff)
	probability := 0.5
	switch {
	case ratio >= 0.20:
		probability = 0.92
	case ratio >= 0.12:
		probability = 0.85
	case ratio >= 0.05:
		probability = 0.76
	case ratio >= -0.02:
		probability = 0.64
	case ratio >= -0.08:
		probability = 0.48
	case ratio >= -0.15:
		probability = 0.34
	default:
		probability = 0.22
	}
	if item.MatchedMajor != "" {
		probability += 0.03
	}
	if req.Score > 0 && item.ScoreLastYear > 0 {
		scoreDiff := req.Score - item.ScoreLastYear
		switch {
		case scoreDiff >= 12:
			probability += 0.06
		case scoreDiff >= 6:
			probability += 0.04
		case scoreDiff >= 0:
			probability += 0.02
		case scoreDiff >= -5:
			probability -= 0.01
		default:
			probability -= 0.04
		}
	}
	if item.RankLastYear > 0 && item.RankTwoYearsAgo > 0 && item.RankThreeYearsAgo > 0 {
		probability += 0.02
	}
	if item.PlanCount >= 5 {
		probability += 0.02
	}
	if item.MajorCount <= 1 {
		probability -= 0.02
	}
	return math.Max(0.05, math.Min(0.97, probability))
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func buildProbabilityLabel(probability float64) string {
	switch {
	case probability >= 0.85:
		return "保底把握较高，适合承担兜底角色"
	case probability >= 0.68:
		return "稳妥把握较强，适合作为主力志愿"
	case probability >= 0.45:
		return "有一定机会录取，适合作为冲稳之间的过渡"
	default:
		return "冲刺性质更强，建议少量放在前面尝试"
	}
}

func buildHistoryRankSummary(item model.RecommendItem) string {
	parts := make([]string, 0, 4)
	if item.RankLastYear > 0 {
		parts = append(parts, fmt.Sprintf("去年 %d", item.RankLastYear))
	}
	if item.RankTwoYearsAgo > 0 {
		parts = append(parts, fmt.Sprintf("前年 %d", item.RankTwoYearsAgo))
	}
	if item.RankThreeYearsAgo > 0 {
		parts = append(parts, fmt.Sprintf("三年前 %d", item.RankThreeYearsAgo))
	}
	if len(parts) == 0 {
		return ""
	}
	if item.WeightedRank > 0 {
		return fmt.Sprintf("该专业组近 3 年最低位次分别为%s，加权后参考位次约 %d。", strings.Join(parts, "、"), item.WeightedRank)
	}
	return fmt.Sprintf("该专业组近 3 年最低位次分别为%s。", strings.Join(parts, "、"))
}
