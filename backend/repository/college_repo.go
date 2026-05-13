package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gaokao-ai/backend/model"
)

type CollegeRepository struct {
	db *observedDB
}

func NewCollegeRepository(db *sql.DB) *CollegeRepository {
	return &CollegeRepository{db: observeDB(db)}
}

func buildTargetMajorKeywords(targetMajor string) []string {
	trimmed := strings.TrimSpace(targetMajor)
	if trimmed == "" {
		return nil
	}
	replacer := strings.NewReplacer("，", " ", "、", " ", "；", " ", ";", " ", "、", " ", "/", " ", "|", " ", "（", " ", "）", " ", "(", " ", ")", " ")
	normalized := replacer.Replace(trimmed)
	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		parts = []string{trimmed}
	}
	seen := map[string]struct{}{}
	keywords := make([]string, 0, len(parts)*2)
	appendKeyword := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		keywords = append(keywords, value)
	}
	for _, part := range parts {
		appendKeyword(part)
		for _, suffix := range []string{"专业", "类", "方向"} {
			if strings.HasSuffix(part, suffix) {
				appendKeyword(strings.TrimSpace(strings.TrimSuffix(part, suffix)))
			}
		}
	}
	if len(keywords) == 0 {
		appendKeyword(trimmed)
	}
	return keywords
}

func (r *CollegeRepository) ListAdmissionLines(ctx context.Context, province, subject string, year int, targetMajor string, targetRank int, limit int) ([]model.RecommendItem, error) {
	if year <= 0 {
		year = 2025
	}
	if limit <= 0 {
		limit = 500
	}
	keywords := buildTargetMajorKeywords(targetMajor)
	matchedMajorExpr := "''"
	targetHitExpr := "0"
	queryArgs := make([]any, 0, len(keywords)*3+5)
	if len(keywords) > 0 {
		parts := make([]string, 0, len(keywords))
		matchedArgs := make([]any, 0, len(keywords))
		targetHitArgs := make([]any, 0, len(keywords))
		for _, keyword := range keywords {
			parts = append(parts, "cep.major_name LIKE ?")
			likeValue := "%" + keyword + "%"
			matchedArgs = append(matchedArgs, likeValue)
			targetHitArgs = append(targetHitArgs, likeValue)
		}
		condition := strings.Join(parts, " OR ")
		matchedMajorExpr = fmt.Sprintf("COALESCE(MAX(CASE WHEN %s THEN cep.major_name ELSE '' END), '')", condition)
		targetHitExpr = fmt.Sprintf("CASE WHEN MAX(CASE WHEN %s THEN 1 ELSE 0 END) = 1 THEN 1 ELSE 0 END", condition)
		queryArgs = append(queryArgs, matchedArgs...)
		queryArgs = append(queryArgs, targetHitArgs...)
	}

	query := fmt.Sprintf(`
SELECT *
FROM (
	SELECT
		c.id,
		c.name,
		cep.province,
		COALESCE(c.city, '') AS city,
		COALESCE(cpg.group_code, '') AS group_code,
		COALESCE(cpg.group_name, '') AS group_name,
		cep.batch,
		COALESCE(cpg.subject_requirement, cep.subject_requirement, '不限') AS subject_requirement,
		COALESCE(MAX(cpg.group_plan_count), SUM(cep.plan_count)) AS plan_count,
		COUNT(DISTINCT cep.id) AS major_count,
		GROUP_CONCAT(DISTINCT cep.major_name ORDER BY cep.major_name SEPARATOR '、') AS majors,
		%s AS matched_major,
		COALESCE(MIN(NULLIF(cmas.min_score, 0)), MIN(NULLIF(cpg.group_min_score, 0)), 0) AS min_score,
		COALESCE(MIN(NULLIF(cmas.min_rank, 0)), MIN(NULLIF(cpg.group_min_rank, 0)), 0) AS min_rank,
		COALESCE(CAST(AVG(NULLIF(cmas.min_score, 0)) AS SIGNED), 0) AS avg_score,
		COALESCE(MIN(CASE WHEN cmas.stat_year = ? AND cmas.min_rank > 0 THEN cmas.min_rank END), 0) AS rank_last_year,
		COALESCE(MIN(CASE WHEN cmas.stat_year = ? AND cmas.min_rank > 0 THEN cmas.min_rank END), 0) AS rank_two_years_ago,
		COALESCE(MIN(CASE WHEN cmas.stat_year = ? AND cmas.min_rank > 0 THEN cmas.min_rank END), 0) AS rank_three_years_ago,
		%s AS target_hit
	FROM college_enrollment_plan cep
	JOIN college c ON cep.college_id = c.id
	LEFT JOIN college_program_group cpg ON cpg.id = cep.program_group_id
	LEFT JOIN college_major_admission_stat cmas ON cmas.enrollment_plan_id = cep.id AND cmas.stat_year BETWEEN ? AND ?
	WHERE cep.province = ?
		AND cep.subject = ?
		AND cep.year = ?
	GROUP BY c.id, c.name, cep.province, cpg.group_code, cpg.group_name, cep.batch, cpg.subject_requirement, cep.subject_requirement
) AS grouped_lines
ORDER BY
	target_hit DESC,
	CASE WHEN min_rank = 0 THEN 1 ELSE 0 END ASC,
	CASE
		WHEN ? > 0 AND min_rank > 0 THEN ABS(min_rank - ?)
		ELSE min_rank
	END ASC,
	min_rank ASC,
	min_score DESC,
	plan_count DESC,
	id ASC
LIMIT ?;`, matchedMajorExpr, targetHitExpr)

	queryArgs = append(queryArgs, year-1, year-2, year-3, year-3, year)
	queryArgs = append(queryArgs, province, subject, year, targetRank, targetRank, limit)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.RecommendItem, 0)
	for rows.Next() {
		var item model.RecommendItem
		var targetHit int
		if err := rows.Scan(
			&item.CollegeID,
			&item.CollegeName,
			&item.Province,
			&item.City,
			&item.GroupCode,
			&item.GroupName,
			&item.Batch,
			&item.SubjectRequirement,
			&item.PlanCount,
			&item.MajorCount,
			&item.Major,
			&item.MatchedMajor,
			&item.MinScore,
			&item.MinRank,
			&item.AvgScore,
			&item.RankLastYear,
			&item.RankTwoYearsAgo,
			&item.RankThreeYearsAgo,
			&targetHit,
		); err != nil {
			return nil, err
		}
		if targetHit == 1 && item.MatchedMajor != "" {
			item.RecommendationReason = "命中意向专业，优先保留该专业组"
		} else if item.MinRank > 0 {
			item.RecommendationReason = "按黑龙江 2025 专业组最低位次匹配"
		} else {
			item.RecommendationReason = "当前专业组缺少有效位次，按计划与专业数保留为备选"
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
