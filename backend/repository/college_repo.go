package repository

import (
	"context"
	"database/sql"

	"gaokao-ai/backend/model"
)

type CollegeRepository struct {
	db *sql.DB
}

func NewCollegeRepository(db *sql.DB) *CollegeRepository {
	return &CollegeRepository{db: db}
}

func (r *CollegeRepository) ListAdmissionLines(ctx context.Context, province, subject string, year int, targetMajor string, limit int) ([]model.RecommendItem, error) {
	if year <= 0 {
		year = 2025
	}
	if limit <= 0 {
		limit = 500
	}
	keywordLike := ""
	if targetMajor != "" {
		keywordLike = "%" + targetMajor + "%"
	}

	query := `
SELECT
	c.id,
	c.name,
	cep.province,
	COALESCE(cpg.group_code, ''),
	COALESCE(cpg.group_name, ''),
	cep.batch,
	COALESCE(cpg.subject_requirement, cep.subject_requirement, '不限'),
	COALESCE(MAX(cpg.group_plan_count), SUM(cep.plan_count)) AS plan_count,
	COUNT(DISTINCT cep.id) AS major_count,
	GROUP_CONCAT(DISTINCT cep.major_name ORDER BY cep.major_name SEPARATOR '、') AS majors,
	COALESCE(MAX(CASE WHEN ? <> '' AND cep.major_name LIKE ? THEN cep.major_name ELSE '' END), '') AS matched_major,
	COALESCE(MIN(NULLIF(cmas.min_score, 0)), MIN(NULLIF(cpg.group_min_score, 0)), 0) AS min_score,
	COALESCE(MIN(NULLIF(cmas.min_rank, 0)), MIN(NULLIF(cpg.group_min_rank, 0)), 0) AS min_rank,
	COALESCE(CAST(AVG(NULLIF(cmas.min_score, 0)) AS SIGNED), 0) AS avg_score,
	CASE WHEN ? <> '' AND MAX(CASE WHEN cep.major_name LIKE ? THEN 1 ELSE 0 END) = 1 THEN 1 ELSE 0 END AS target_hit
FROM college_enrollment_plan cep
JOIN college c ON cep.college_id = c.id
LEFT JOIN college_program_group cpg ON cpg.id = cep.program_group_id
LEFT JOIN college_major_admission_stat cmas ON cmas.enrollment_plan_id = cep.id AND cmas.stat_year = cep.year
WHERE cep.province = ?
	AND cep.subject = ?
	AND cep.year = ?
GROUP BY c.id, c.name, cep.province, cpg.group_code, cpg.group_name, cep.batch, cpg.subject_requirement, cep.subject_requirement
ORDER BY target_hit DESC, CASE WHEN min_rank = 0 THEN 1 ELSE 0 END ASC, min_rank ASC, min_score DESC, plan_count DESC, c.id ASC
LIMIT ?;`

	rows, err := r.db.QueryContext(ctx, query, keywordLike, keywordLike, keywordLike, keywordLike, province, subject, year, limit)
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
