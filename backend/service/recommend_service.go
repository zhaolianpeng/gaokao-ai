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
	cacheKey := fmt.Sprintf("recommend:v4:%s:%d:%d:%s:%d:%s", strings.TrimSpace(req.Province), req.Score, req.Rank, strings.TrimSpace(req.Subject), req.Year, strings.TrimSpace(req.TargetMajor))
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
		provinceLines, err := s.repo.GetProvinceScoreLines(ctx, req.Province, []int{req.Year, req.Year - 1}, normalizeRecommendLookupSubject(req.Year, req.Subject))
		if err != nil {
			return model.RecommendResponse{}, err
		}
		lineMap := buildProvinceLineMap(provinceLines)

		chong := make([]model.RecommendItem, 0)
		jiaoChong := make([]model.RecommendItem, 0)
		wen := make([]model.RecommendItem, 0)
		jiaoBao := make([]model.RecommendItem, 0)
		bao := make([]model.RecommendItem, 0)
		bands := buildRecommendBands(req.Rank)
		logging.LogEvent("recommend_bands", map[string]any{"rank": req.Rank, "maxChongRatio": bands.MaxChongRatio, "candidateCount": len(items)})

		for _, item := range items {
			benchmarkRank := buildBenchmarkRank(item)
			if benchmarkRank == 0 {
				continue
			}
			item.WeightedRank = benchmarkRank
			item.CurrentLineScore = lookupProvinceLineScore(lineMap, req.Year, item.Batch)
			item.LastYearLineScore = lookupProvinceLineScore(lineMap, req.Year-1, item.Batch)
			if req.Score > 0 && item.CurrentLineScore > 0 {
				item.CurrentLineDiff = req.Score - item.CurrentLineScore
			}
			if item.ScoreLastYear > 0 && item.LastYearLineScore > 0 {
				item.LastYearLineDiff = item.ScoreLastYear - item.LastYearLineScore
			}
			if item.CurrentLineScore > 0 && item.LastYearLineScore > 0 && req.Score > 0 && item.ScoreLastYear > 0 {
				item.LineDiffGap = item.CurrentLineDiff - item.LastYearLineDiff
			}
			diff := benchmarkRank - req.Rank
			if !shouldKeepCandidate(item, req, diff, bands) {
				continue
			}
			item.Probability = estimateProbability(req, item, diff)
			item.ProbabilityLabel = buildProbabilityLabel(item.Probability)
			tag := classifyByProbability(item.Probability)
			item.RecommendationReason = buildReason(req, item, diff)

			switch tag {
			case "chong":
				item.Tag = tag
				chong = append(chong, item)
			case "jiaochong":
				item.Tag = tag
				jiaoChong = append(jiaoChong, item)
			case "wen":
				item.Tag = tag
				wen = append(wen, item)
			case "jiaobao":
				item.Tag = tag
				jiaoBao = append(jiaoBao, item)
			case "bao":
				item.Tag = tag
				bao = append(bao, item)
			}
		}

		sort.Slice(chong, func(i, j int) bool {
			return compareBucketItems(chong[i], chong[j], req.Rank, req.Score, "chong")
		})
		sort.Slice(jiaoChong, func(i, j int) bool {
			return compareBucketItems(jiaoChong[i], jiaoChong[j], req.Rank, req.Score, "jiaochong")
		})
		sort.Slice(wen, func(i, j int) bool {
			return compareBucketItems(wen[i], wen[j], req.Rank, req.Score, "wen")
		})
		sort.Slice(jiaoBao, func(i, j int) bool {
			return compareBucketItems(jiaoBao[i], jiaoBao[j], req.Rank, req.Score, "jiaobao")
		})
		sort.Slice(bao, func(i, j int) bool {
			return compareBucketItems(bao[i], bao[j], req.Rank, req.Score, "bao")
		})

		return model.RecommendResponse{
			Chong:     trim(chong, 8),
			JiaoChong: trim(jiaoChong, 12),
			Wen:       trim(wen, 18),
			JiaoBao:   trim(jiaoBao, 12),
			Bao:       trim(bao, 8),
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
			parts = append(parts, fmt.Sprintf("你当前位次 %d，略高于该组近年主流录取位次 %d，位次层面偏冲。", req.Rank, item.WeightedRank))
		} else if normalizedRankGap(item.WeightedRank, req.Rank) <= 0.05 {
			parts = append(parts, fmt.Sprintf("你当前位次 %d，与该组近年主流录取位次 %d 比较接近，位次层面偏稳。", req.Rank, item.WeightedRank))
		} else {
			parts = append(parts, fmt.Sprintf("你当前位次 %d，明显优于该组近年主流录取位次 %d，位次层面更偏保。", req.Rank, item.WeightedRank))
		}
	}
	if item.CurrentLineScore > 0 && item.LastYearLineScore > 0 && req.Score > 0 && item.ScoreLastYear > 0 {
		parts = append(parts, fmt.Sprintf("按线差法看，你今年高出省控线 %d 分；该组去年最低分高出省控线 %d 分，线差差值为 %d 分。", item.CurrentLineDiff, item.LastYearLineDiff, item.LineDiffGap))
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
	MaxChongRatio float64
	MaxScoreGap   int
}

func buildRecommendBands(rank int) recommendBands {
	return recommendBands{
		MaxChongRatio: -0.30,
		MaxScoreGap:   -18,
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

func shouldKeepCandidate(item model.RecommendItem, req model.RecommendRequest, rankDiff int, bands recommendBands) bool {
	ratio := normalizedRankGap(item.WeightedRank, req.Rank)
	if ratio < bands.MaxChongRatio {
		return false
	}
	if item.CurrentLineScore > 0 && item.LastYearLineScore > 0 && item.LineDiffGap < bands.MaxScoreGap {
		return false
	}
	return true
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
		return -0.14
	case "jiaochong":
		return -0.06
	case "jiaobao":
		return 0.10
	case "bao":
		return 0.20
	default:
		return 0.02
	}
}

func bucketTargetScoreGap(bucket string) int {
	switch bucket {
	case "chong":
		return -8
	case "jiaochong":
		return -2
	case "jiaobao":
		return 10
	case "bao":
		return 18
	default:
		return 3
	}
}

func bucketTargetProbability(bucket string) float64 {
	switch bucket {
	case "chong":
		return 0.22
	case "jiaochong":
		return 0.40
	case "jiaobao":
		return 0.79
	case "bao":
		return 0.92
	default:
		return 0.60
	}
}

func compareBucketItems(left, right model.RecommendItem, userRank int, userScore int, bucket string) bool {
	leftRankFit := math.Abs(normalizedRankGap(buildBenchmarkRank(left), userRank) - bucketTargetGap(bucket))
	rightRankFit := math.Abs(normalizedRankGap(buildBenchmarkRank(right), userRank) - bucketTargetGap(bucket))
	leftScoreFit := 999.0
	if left.CurrentLineScore > 0 && left.LastYearLineScore > 0 {
		leftScoreFit = scoreFitDistance(left.LineDiffGap, bucket)
	}
	rightScoreFit := 999.0
	if right.CurrentLineScore > 0 && right.LastYearLineScore > 0 {
		rightScoreFit = scoreFitDistance(right.LineDiffGap, bucket)
	}
	leftProbFit := math.Abs(left.Probability - bucketTargetProbability(bucket))
	rightProbFit := math.Abs(right.Probability - bucketTargetProbability(bucket))
	leftFit := leftRankFit*0.55 + leftScoreFit*0.03 + leftProbFit*0.42
	rightFit := rightRankFit*0.55 + rightScoreFit*0.03 + rightProbFit*0.42
	if leftFit != rightFit {
		return leftFit < rightFit
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
	return absRankDiff(buildBenchmarkRank(left), userRank) < absRankDiff(buildBenchmarkRank(right), userRank)
}

func scoreFitDistance(lineDiffGap int, bucket string) float64 {
	return math.Abs(float64(lineDiffGap - bucketTargetScoreGap(bucket)))
}

func estimateProbability(req model.RecommendRequest, item model.RecommendItem, rankDiff int) float64 {
	benchmarkRank := buildBenchmarkRank(item)
	ratio := normalizedRankGap(benchmarkRank, benchmarkRank-rankDiff)
	rankProbability := 0.5
	switch {
	case ratio >= 0.20:
		rankProbability = 0.92
	case ratio >= 0.12:
		rankProbability = 0.84
	case ratio >= 0.05:
		rankProbability = 0.72
	case ratio >= -0.02:
		rankProbability = 0.58
	case ratio >= -0.08:
		rankProbability = 0.42
	case ratio >= -0.15:
		rankProbability = 0.26
	default:
		rankProbability = 0.12
	}
	lineProbability := 0.5
	hasLineProbability := item.CurrentLineScore > 0 && item.LastYearLineScore > 0
	if hasLineProbability {
		switch {
		case item.LineDiffGap >= 15:
			lineProbability = 0.92
		case item.LineDiffGap >= 8:
			lineProbability = 0.82
		case item.LineDiffGap >= 3:
			lineProbability = 0.70
		case item.LineDiffGap >= -2:
			lineProbability = 0.56
		case item.LineDiffGap >= -8:
			lineProbability = 0.40
		case item.LineDiffGap >= -15:
			lineProbability = 0.24
		default:
			lineProbability = 0.10
		}
	}
	extraProbability := 0.52
	if item.MatchedMajor != "" {
		extraProbability += 0.08
	}
	if item.RankLastYear > 0 && item.RankTwoYearsAgo > 0 && item.RankThreeYearsAgo > 0 {
		extraProbability += 0.04
	}
	if item.PlanCount >= 5 {
		extraProbability += 0.04
	}
	if item.MajorCount <= 1 {
		extraProbability -= 0.03
	}
	if hasLineProbability {
		return math.Max(0.05, math.Min(0.97, rankProbability*0.65+lineProbability*0.25+extraProbability*0.10))
	}
	return math.Max(0.05, math.Min(0.97, rankProbability*0.82+extraProbability*0.18))
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
	case probability >= 0.70:
		return "较保区间，录取把握较高"
	case probability >= 0.68:
		return "稳妥把握较强，适合作为主力志愿"
	case probability >= 0.45:
		return "较冲区间，有一定机会录取"
	default:
		return "冲刺性质更强，建议少量放在前面尝试"
	}
}

func classifyByProbability(probability float64) string {
	switch {
	case probability < 0.30:
		return "chong"
	case probability < 0.50:
		return "jiaochong"
	case probability < 0.70:
		return "wen"
	case probability < 0.85:
		return "jiaobao"
	default:
		return "bao"
	}
}

func buildProvinceLineMap(items []model.ProvinceScoreLineItem) map[string]int {
	result := make(map[string]int, len(items))
	for _, item := range items {
		for _, key := range provinceLineKeys(item.Year, item.Batch) {
			result[key] = item.Score
		}
	}
	return result
}

func provinceLineKey(year int, batch string) string {
	return fmt.Sprintf("%d::%s", year, strings.TrimSpace(batch))
}

func provinceLineKeys(year int, batch string) []string {
	normalizedBatch := strings.TrimSpace(batch)
	if year <= 0 || normalizedBatch == "" {
		return nil
	}
	seen := map[string]struct{}{}
	keys := make([]string, 0, 4)
	appendKey := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := provinceLineKey(year, value)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	appendKey(normalizedBatch)
	compact := strings.NewReplacer("普通", "", "本科", "本科", "高职（专科）", "专科", "（", "", "）", "").Replace(normalizedBatch)
	if strings.Contains(normalizedBatch, "本科") {
		appendKey("本科批")
		appendKey("普通本科批")
		appendKey("本科普通批")
	}
	if strings.Contains(normalizedBatch, "特殊类型") {
		appendKey("特殊类型招生资格线")
		appendKey("普通特殊类型招生资格线")
	}
	if strings.Contains(normalizedBatch, "专科") || strings.Contains(normalizedBatch, "高职") {
		appendKey("高职（专科）批")
		appendKey("普通高职（专科）批")
		appendKey("专科批")
	}
	if compact != normalizedBatch {
		appendKey(compact)
	}
	return keys
}

func lookupProvinceLineScore(lineMap map[string]int, year int, batch string) int {
	if year <= 0 || strings.TrimSpace(batch) == "" {
		return 0
	}
	for _, key := range provinceLineKeys(year, batch) {
		if score, ok := lineMap[key]; ok {
			return score
		}
	}
	return 0
}

func normalizeRecommendLookupSubject(year int, subject string) string {
	subject = strings.TrimSpace(subject)
	if year > 0 && year <= 2023 {
		switch subject {
		case "历史", "文科", "历史类":
			return "文科"
		case "物理", "理科", "物理类":
			return "理科"
		}
	}
	switch subject {
	case "历史", "文科":
		return "历史类"
	case "物理", "理科":
		return "物理类"
	default:
		return subject
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
