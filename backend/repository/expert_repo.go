package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"gaokao-ai/backend/model"
)

type ExpertRepository struct {
	db *sql.DB
}

func NewExpertRepository(db *sql.DB) *ExpertRepository {
	return &ExpertRepository{db: db}
}

func (r *ExpertRepository) GetDashboardOverview(ctx context.Context, province string, year int, subject string) (model.DashboardOverview, error) {
	query := `
SELECT
  COUNT(DISTINCT cep.college_id) AS college_count,
  COUNT(DISTINCT cpg.id) AS group_count,
  COUNT(DISTINCT cep.id) AS enrollment_count,
  COUNT(DISTINCT cep.major_name) AS major_count,
  COUNT(DISTINCT cmas.id) AS stat_count
FROM college_enrollment_plan cep
LEFT JOIN college_program_group cpg ON cpg.id = cep.program_group_id
LEFT JOIN college_major_admission_stat cmas ON cmas.enrollment_plan_id = cep.id
WHERE (? = '' OR cep.province = ?)
	AND (? = 0 OR cep.year = ?)
	AND (? = '' OR cep.subject = ?)`

	var overview model.DashboardOverview
	overview.Province = province
	overview.Year = year
	overview.Subject = subject
	row := r.db.QueryRowContext(ctx, query, province, province, year, year, subject, subject)
	if err := row.Scan(&overview.CollegeCount, &overview.ProgramGroupCount, &overview.EnrollmentCount, &overview.MajorCount, &overview.StatCount); err != nil {
		return model.DashboardOverview{}, err
	}
	return overview, nil
}

func (r *ExpertRepository) GetProvinceScoreLines(ctx context.Context, province string, year int, subject string) ([]model.ProvinceScoreLineItem, error) {
	query := `
SELECT province, year, subject, batch, score, source_name, source_url
FROM province_score_line
WHERE (? = '' OR province = ?)
  AND (? = 0 OR year = ?)
  AND (? = '' OR subject = ?)
ORDER BY subject ASC, score DESC, batch ASC`

	rows, err := r.db.QueryContext(ctx, query, province, province, year, year, subject, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ProvinceScoreLineItem, 0)
	for rows.Next() {
		var item model.ProvinceScoreLineItem
		if err := rows.Scan(&item.Province, &item.Year, &item.Subject, &item.Batch, &item.Score, &item.SourceName, &item.SourceURL); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ExpertRepository) LookupScoreRank(ctx context.Context, province string, year int, subject string, score int) (model.ScoreRankLookup, error) {
	result := model.ScoreRankLookup{
		Province:   province,
		Year:       year,
		Subject:    subject,
		QueryScore: score,
	}
	query := `
SELECT province, year, subject, score, ` + "`rank`" + ` AS rank_value, ` + "`count`" + ` AS count_value,
	   CASE WHEN score = ? THEN TRUE ELSE FALSE END AS exact,
	   ABS(score - ?) AS diff
FROM score_rank
WHERE province = ?
  AND year = ?
  AND subject = ?
ORDER BY
  CASE
	WHEN score = ? THEN 0
	WHEN score < ? THEN 1
    ELSE 2
  END,
	ABS(score - ?) ASC,
  score DESC
LIMIT 1`

	err := r.db.QueryRowContext(ctx, query, score, score, province, year, subject, score, score, score).Scan(
		&result.Province,
		&result.Year,
		&result.Subject,
		&result.MatchedScore,
		&result.Rank,
		&result.Count,
		&result.Exact,
		&result.Diff,
	)
	if err == sql.ErrNoRows {
		result.Available = false
		return result, nil
	}
	if err != nil {
		return model.ScoreRankLookup{}, err
	}
	result.Available = true
	result.Diff = int(math.Abs(float64(result.MatchedScore - score)))
	return result, nil
}

func (r *ExpertRepository) ListColleges(ctx context.Context, filter model.CollegeListFilter) ([]model.CollegeListItem, bool, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	filter.SortMode = normalizeCollegeSortMode(filter.SortMode)
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit
	keyword := strings.TrimSpace(filter.Keyword)
	keywordLike := ""
	if keyword != "" {
		keywordLike = "%" + keyword + "%"
	}

	orderClause := `CASE WHEN min_group_rank = 0 THEN 1 ELSE 0 END ASC,
	min_group_rank ASC,
	min_group_score DESC,
	c.id ASC`
	if filter.SortMode == "tier" {
		orderClause = `
	CASE
		WHEN c.is_985 = 1 THEN 4
		WHEN c.is_985 = 0 AND c.is_211 = 1 AND c.is_double_first = 1 AND ` + buildRankingValueExpr() + ` > 0 AND ` + buildRankingValueExpr() + ` <= 60 THEN 3
		WHEN c.is_211 = 1 THEN 2
		WHEN c.is_double_first = 1 THEN 1
		ELSE 0
	END DESC,
	CASE WHEN min_group_rank = 0 THEN 1 ELSE 0 END ASC,
	min_group_rank ASC,
	min_group_score DESC,
	CASE WHEN ` + buildRankingValueExpr() + ` > 0 THEN ` + buildRankingValueExpr() + ` ELSE 999999 END ASC,
	c.id ASC`
	}

	query := `
SELECT
  c.id,
  c.name,
  c.province,
  c.city,
  c.level,
	c.is_985,
	c.is_211,
	c.is_double_first,
  c.tags,
  c.school_level_tags,
	  COALESCE(CAST(c.recommended_postgraduate_rate AS CHAR), ''),
  c.ranking,
  COUNT(DISTINCT cep.program_group_id) AS group_count,
  COUNT(DISTINCT cep.id) AS major_count,
  COALESCE(MIN(cpg.group_min_score), 0) AS min_group_score,
  COALESCE(MIN(NULLIF(cpg.group_min_rank, 0)), 0) AS min_group_rank
FROM college c
JOIN college_enrollment_plan cep ON cep.college_id = c.id
LEFT JOIN college_program_group cpg ON cpg.id = cep.program_group_id
WHERE (? = '' OR cep.province = ?)
	AND (? = 0 OR cep.year = ?)
	AND (? = '' OR cep.subject = ?)
	AND (? = '' OR c.name LIKE ? OR cep.major_name LIKE ?)
GROUP BY c.id, c.name, c.province, c.city, c.level, c.is_985, c.is_211, c.is_double_first, c.tags, c.school_level_tags, c.recommended_postgraduate_rate, c.ranking
ORDER BY ` + orderClause + `
LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, filter.Province, filter.Province, filter.Year, filter.Year, filter.Subject, filter.Subject, keywordLike, keywordLike, keywordLike, filter.Limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	items := make([]model.CollegeListItem, 0)
	for rows.Next() {
		var item model.CollegeListItem
		var is985 bool
		var is211 bool
		var isDoubleFirst bool
		var tagsRaw []byte
		var schoolLevelRaw []byte
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Province,
			&item.City,
			&item.Level,
			&is985,
			&is211,
			&isDoubleFirst,
			&tagsRaw,
			&schoolLevelRaw,
			&item.RecommendedRate,
			&item.Ranking,
			&item.GroupCount,
			&item.MajorCount,
			&item.MinGroupScore,
			&item.MinGroupRank,
		); err != nil {
			return nil, false, err
		}
		item.Tags = decodeStringArray(tagsRaw)
		item.Tags = prependSchoolTierTags(item.Tags, is985, is211, isDoubleFirst)
		item.SchoolLevelTags = decodeStringArray(schoolLevelRaw)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(items) > filter.Limit
	if hasMore {
		items = items[:filter.Limit]
	}
	return items, hasMore, nil
}

func normalizeCollegeSortMode(mode string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	switch mode {
	case "admission":
		return "admission"
	default:
		return "tier"
	}
}

func buildRankingValueExpr() string {
	return `CASE
		WHEN c.ranking REGEXP '^[0-9]+$' THEN CAST(c.ranking AS UNSIGNED)
		ELSE 0
	END`
}

func prependSchoolTierTags(tags []string, is985, is211, isDoubleFirst bool) []string {
	result := make([]string, 0, len(tags)+3)
	appendIfMissing := func(label string, enabled bool) {
		if !enabled {
			return
		}
		for _, item := range result {
			if item == label {
				return
			}
		}
		result = append(result, label)
	}

	appendIfMissing("985", is985)
	appendIfMissing("211", is211)
	appendIfMissing("双一流", isDoubleFirst)
	for _, item := range tags {
		exists := false
		for _, current := range result {
			if current == item {
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, item)
		}
	}
	return result
}

func (r *ExpertRepository) GetCollegeDetail(ctx context.Context, collegeID int, province string, year int, subject string) (model.CollegeDetail, error) {
	query := `
SELECT
  id, name, province, city, city_level, level, tags, school_level_tags,
  affiliation, school_type, ownership_type,
	  COALESCE(CAST(recommended_postgraduate_rate AS CHAR), ''),
  ranking, transfer_policy, admissions_charter_url,
  softscience_grade, softscience_ranking, discipline_evaluation,
  master_major_count, master_major_list, doctor_major_count, doctor_major_list
FROM college
WHERE id = ?`

	var detail model.CollegeDetail
	var tagsRaw, schoolLevelRaw, masterRaw, doctorRaw []byte
	row := r.db.QueryRowContext(ctx, query, collegeID)
	if err := row.Scan(
		&detail.ID,
		&detail.Name,
		&detail.Province,
		&detail.City,
		&detail.CityLevel,
		&detail.Level,
		&tagsRaw,
		&schoolLevelRaw,
		&detail.Affiliation,
		&detail.SchoolType,
		&detail.OwnershipType,
		&detail.RecommendedRate,
		&detail.Ranking,
		&detail.TransferPolicy,
		&detail.AdmissionsCharterURL,
		&detail.SoftScienceGrade,
		&detail.SoftScienceRanking,
		&detail.DisciplineEvaluation,
		&detail.MasterMajorCount,
		&masterRaw,
		&detail.DoctorMajorCount,
		&doctorRaw,
	); err != nil {
		return model.CollegeDetail{}, err
	}
	detail.Tags = decodeStringArray(tagsRaw)
	detail.SchoolLevelTags = decodeStringArray(schoolLevelRaw)
	detail.MasterMajorList = decodeStringArray(masterRaw)
	detail.DoctorMajorList = decodeStringArray(doctorRaw)

	programGroups, err := r.listProgramGroups(ctx, collegeID, province, year, subject)
	if err != nil {
		return model.CollegeDetail{}, err
	}
	detail.ProgramGroups = programGroups

	majorPlans, years, err := r.listMajorPlans(ctx, collegeID, province, year, subject)
	if err != nil {
		return model.CollegeDetail{}, err
	}
	detail.MajorPlans = majorPlans
	detail.HistoricalStatsAvailable = years
	return detail, nil
}

func (r *ExpertRepository) listProgramGroups(ctx context.Context, collegeID int, province string, year int, subject string) ([]model.CollegeProgramGroup, error) {
	query := `
SELECT group_code, group_name, batch, batch_remark, category, subject_requirement, group_plan_count, group_min_score, group_min_rank
FROM college_program_group
WHERE college_id = ?
	AND (? = '' OR province = ?)
	AND (? = 0 OR year = ?)
	AND (? = '' OR subject = ?)
ORDER BY CASE WHEN group_min_rank = 0 THEN 1 ELSE 0 END ASC, group_min_rank ASC, group_code ASC`

	rows, err := r.db.QueryContext(ctx, query, collegeID, province, province, year, year, subject, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.CollegeProgramGroup, 0)
	for rows.Next() {
		var item model.CollegeProgramGroup
		if err := rows.Scan(&item.GroupCode, &item.GroupName, &item.Batch, &item.BatchRemark, &item.Category, &item.SubjectRequirement, &item.PlanCount, &item.MinScore, &item.MinRank); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ExpertRepository) listMajorPlans(ctx context.Context, collegeID int, province string, year int, subject string) ([]model.CollegeMajorPlan, []int, error) {
	query := `
SELECT
  cep.id,
  cep.major_code,
  cep.major_name,
  cep.major_full_name,
  cep.batch,
  cep.batch_remark,
  COALESCE(cpg.group_code, ''),
  COALESCE(cpg.group_name, ''),
  cep.subject_requirement,
  cep.plan_count,
  cep.study_years,
  cep.tuition_fee,
  cep.major_category,
  cep.discipline_category,
  COALESCE(cm.major_strength, ''),
  cm.master_points,
  cm.doctor_points
FROM college_enrollment_plan cep
LEFT JOIN college_program_group cpg ON cpg.id = cep.program_group_id
LEFT JOIN college_major cm ON cm.college_id = cep.college_id AND cm.major_name = cep.major_name
WHERE cep.college_id = ?
	AND (? = '' OR cep.province = ?)
	AND (? = 0 OR cep.year = ?)
	AND (? = '' OR cep.subject = ?)
ORDER BY COALESCE(cpg.group_min_rank, 0) ASC, cep.plan_count DESC, cep.id ASC`

	rows, err := r.db.QueryContext(ctx, query, collegeID, province, province, year, year, subject, subject)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	items := make([]model.CollegeMajorPlan, 0)
	yearSet := map[int]struct{}{}
	for rows.Next() {
		var item model.CollegeMajorPlan
		var masterRaw, doctorRaw []byte
		if err := rows.Scan(
			&item.ID,
			&item.MajorCode,
			&item.MajorName,
			&item.MajorFullName,
			&item.Batch,
			&item.BatchRemark,
			&item.GroupCode,
			&item.GroupName,
			&item.SubjectRequirement,
			&item.PlanCount,
			&item.StudyYears,
			&item.TuitionFee,
			&item.MajorCategory,
			&item.DisciplineCategory,
			&item.MajorStrength,
			&masterRaw,
			&doctorRaw,
		); err != nil {
			return nil, nil, err
		}
		item.MasterPoints = decodeStringArray(masterRaw)
		item.DoctorPoints = decodeStringArray(doctorRaw)
		stats, years, err := r.listAdmissionStats(ctx, item.ID)
		if err != nil {
			return nil, nil, err
		}
		item.AdmissionStats = stats
		for _, year := range years {
			yearSet[year] = struct{}{}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return items, mapKeysSorted(yearSet), nil
}

func (r *ExpertRepository) listAdmissionStats(ctx context.Context, enrollmentPlanID int) ([]model.MajorAdmissionStat, []int, error) {
	query := `
SELECT stat_year, legacy_batch, plan_count, admitted_count, min_score, min_rank, max_score, max_rank
FROM college_major_admission_stat
WHERE enrollment_plan_id = ?
ORDER BY stat_year DESC`

	rows, err := r.db.QueryContext(ctx, query, enrollmentPlanID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	items := make([]model.MajorAdmissionStat, 0)
	years := make([]int, 0)
	for rows.Next() {
		var item model.MajorAdmissionStat
		if err := rows.Scan(&item.Year, &item.LegacyBatch, &item.PlanCount, &item.AdmittedCount, &item.MinScore, &item.MinRank, &item.MaxScore, &item.MaxRank); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
		years = append(years, item.Year)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return items, years, nil
}

func decodeStringArray(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	items := make([]string, 0)
	if err := json.Unmarshal(raw, &items); err != nil {
		return []string{}
	}
	return items
}

func mapKeysSorted(yearSet map[int]struct{}) []int {
	if len(yearSet) == 0 {
		return []int{}
	}
	years := make([]int, 0, len(yearSet))
	for year := range yearSet {
		years = append(years, year)
	}
	for i := 0; i < len(years); i++ {
		for j := i + 1; j < len(years); j++ {
			if years[i] < years[j] {
				years[i], years[j] = years[j], years[i]
			}
		}
	}
	return years
}

func DebugCollegeFilter(filter model.CollegeListFilter) string {
	return fmt.Sprintf("province=%s year=%d subject=%s keyword=%s limit=%d", filter.Province, filter.Year, filter.Subject, filter.Keyword, filter.Limit)
}
