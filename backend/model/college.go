package model

type College struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Province      string `json:"province"`
	Level         string `json:"level"`
	Is985         bool   `json:"is_985"`
	Is211         bool   `json:"is_211"`
	IsDoubleFirst bool   `json:"is_double_first"`
}

type DashboardOverview struct {
	Province          string `json:"province"`
	Year              int    `json:"year"`
	Subject           string `json:"subject"`
	CollegeCount      int    `json:"college_count"`
	ProgramGroupCount int    `json:"program_group_count"`
	EnrollmentCount   int    `json:"enrollment_count"`
	MajorCount        int    `json:"major_count"`
	StatCount         int    `json:"stat_count"`
}

type ProvinceScoreLineItem struct {
	Province   string `json:"province"`
	Year       int    `json:"year"`
	Subject    string `json:"subject"`
	Batch      string `json:"batch"`
	Score      int    `json:"score"`
	SourceName string `json:"source_name"`
	SourceURL  string `json:"source_url"`
}

type ScoreRankLookup struct {
	Province     string `json:"province"`
	Year         int    `json:"year"`
	Subject      string `json:"subject"`
	QueryScore   int    `json:"query_score"`
	MatchedScore int    `json:"matched_score"`
	Rank         int    `json:"rank"`
	Count        int    `json:"count"`
	Diff         int    `json:"diff"`
	Exact        bool   `json:"exact"`
	Available    bool   `json:"available"`
}

type CollegeListFilter struct {
	Province string
	Year     int
	Subject  string
	Keyword  string
	SortMode string
	Page     int
	Limit    int
}

type CollegeListResponse struct {
	Items    []CollegeListItem `json:"items"`
	SortMode string            `json:"sortMode"`
	Page     int               `json:"page"`
	Limit    int               `json:"limit"`
	HasMore  bool              `json:"hasMore"`
}

type CollegeListItem struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	Province        string   `json:"province"`
	City            string   `json:"city"`
	Level           string   `json:"level"`
	Tags            []string `json:"tags"`
	SchoolLevelTags []string `json:"school_level_tags"`
	RecommendedRate string   `json:"recommended_postgraduate_rate"`
	Ranking         string   `json:"ranking"`
	GroupCount      int      `json:"group_count"`
	MajorCount      int      `json:"major_count"`
	MinGroupScore   int      `json:"min_group_score"`
	MinGroupRank    int      `json:"min_group_rank"`
}

type CollegeDetail struct {
	ID                       int                   `json:"id"`
	Name                     string                `json:"name"`
	Province                 string                `json:"province"`
	City                     string                `json:"city"`
	CityLevel                string                `json:"city_level"`
	Level                    string                `json:"level"`
	Tags                     []string              `json:"tags"`
	SchoolLevelTags          []string              `json:"school_level_tags"`
	Affiliation              string                `json:"affiliation"`
	SchoolType               string                `json:"school_type"`
	OwnershipType            string                `json:"ownership_type"`
	RecommendedRate          string                `json:"recommended_postgraduate_rate"`
	Ranking                  string                `json:"ranking"`
	TransferPolicy           string                `json:"transfer_policy"`
	AdmissionsCharterURL     string                `json:"admissions_charter_url"`
	SoftScienceGrade         string                `json:"softscience_grade"`
	SoftScienceRanking       string                `json:"softscience_ranking"`
	DisciplineEvaluation     string                `json:"discipline_evaluation"`
	MasterMajorCount         int                   `json:"master_major_count"`
	MasterMajorList          []string              `json:"master_major_list"`
	DoctorMajorCount         int                   `json:"doctor_major_count"`
	DoctorMajorList          []string              `json:"doctor_major_list"`
	ProgramGroups            []CollegeProgramGroup `json:"program_groups"`
	MajorPlans               []CollegeMajorPlan    `json:"major_plans"`
	HistoricalStatsAvailable []int                 `json:"historical_stats_available"`
}

type CollegeProgramGroup struct {
	GroupCode          string `json:"group_code"`
	GroupName          string `json:"group_name"`
	Batch              string `json:"batch"`
	BatchRemark        string `json:"batch_remark"`
	Category           string `json:"category"`
	SubjectRequirement string `json:"subject_requirement"`
	PlanCount          int    `json:"plan_count"`
	MinScore           int    `json:"min_score"`
	MinRank            int    `json:"min_rank"`
}

type CollegeMajorPlan struct {
	ID                 int                  `json:"id"`
	MajorCode          string               `json:"major_code"`
	MajorName          string               `json:"major_name"`
	MajorFullName      string               `json:"major_full_name"`
	Batch              string               `json:"batch"`
	BatchRemark        string               `json:"batch_remark"`
	GroupCode          string               `json:"group_code"`
	GroupName          string               `json:"group_name"`
	SubjectRequirement string               `json:"subject_requirement"`
	PlanCount          int                  `json:"plan_count"`
	StudyYears         string               `json:"study_years"`
	TuitionFee         string               `json:"tuition_fee"`
	MajorCategory      string               `json:"major_category"`
	DisciplineCategory string               `json:"discipline_category"`
	MajorStrength      string               `json:"major_strength"`
	MasterPoints       []string             `json:"master_points"`
	DoctorPoints       []string             `json:"doctor_points"`
	AdmissionStats     []MajorAdmissionStat `json:"admission_stats"`
}

type MajorAdmissionStat struct {
	Year          int    `json:"year"`
	LegacyBatch   string `json:"legacy_batch"`
	PlanCount     int    `json:"plan_count"`
	AdmittedCount int    `json:"admitted_count"`
	MinScore      int    `json:"min_score"`
	MinRank       int    `json:"min_rank"`
	MaxScore      int    `json:"max_score"`
	MaxRank       int    `json:"max_rank"`
}

type CollegeAdmissionLine struct {
	ID        int    `json:"id"`
	CollegeID int    `json:"college_id"`
	Province  string `json:"province"`
	Year      int    `json:"year"`
	Subject   string `json:"subject"`
	Major     string `json:"major"`
	MinScore  int    `json:"min_score"`
	MinRank   int    `json:"min_rank"`
	AvgScore  int    `json:"avg_score"`
}

type RecommendItem struct {
	CollegeID            int     `json:"college_id"`
	CollegeName          string  `json:"college_name"`
	Province             string  `json:"province"`
	GroupCode            string  `json:"group_code"`
	GroupName            string  `json:"group_name"`
	Batch                string  `json:"batch"`
	SubjectRequirement   string  `json:"subject_requirement"`
	PlanCount            int     `json:"plan_count"`
	MajorCount           int     `json:"major_count"`
	Major                string  `json:"major"`
	MatchedMajor         string  `json:"matched_major"`
	RecommendationReason string  `json:"recommendation_reason"`
	MinScore             int     `json:"min_score"`
	MinRank              int     `json:"min_rank"`
	AvgScore             int     `json:"avg_score"`
	Probability          float64 `json:"probability"`
	Tag                  string  `json:"tag"`
}

type RecommendRequest struct {
	Province    string `json:"province" binding:"required"`
	Score       int    `json:"score" binding:"required"`
	Rank        int    `json:"rank" binding:"required"`
	Subject     string `json:"subject" binding:"required"`
	Year        int    `json:"year"`
	TargetMajor string `json:"targetMajor"`
	Notes       string `json:"notes"`
}

type RecommendResponse struct {
	Chong []RecommendItem `json:"chong"`
	Wen   []RecommendItem `json:"wen"`
	Bao   []RecommendItem `json:"bao"`
}

type AIAnalyzeRequest struct {
	Student    RecommendRequest  `json:"student" binding:"required"`
	Recommend  RecommendResponse `json:"recommend" binding:"required"`
	ExtraNotes string            `json:"extra_notes"`
}
