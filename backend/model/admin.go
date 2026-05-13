package model

import "time"

type AdminUser struct {
	ID          int       `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	Phone       string    `json:"phone"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	LastLoginAt time.Time `json:"lastLoginAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AdminUserAuth struct {
	AdminUser
	PasswordHash string
}

type AdminLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AdminPaginationRequest struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

func (r AdminPaginationRequest) Normalized() (int, int) {
	page := r.Page
	limit := r.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return page, limit
}

type AdminCollegeListRequest struct {
	AdminPaginationRequest
	Keyword       string `form:"keyword"`
	Name          string `form:"name"`
	Level         string `form:"level"`
	SchoolType    string `form:"schoolType"`
	OwnershipType string `form:"ownershipType"`
	Province      string `form:"province"`
	City          string `form:"city"`
	Is985         string `form:"is985"`
	Is211         string `form:"is211"`
	IsDoubleFirst string `form:"isDoubleFirst"`
}

type AdminProvinceScoreLineListRequest struct {
	AdminPaginationRequest
	Keyword  string `form:"keyword"`
	Province string `form:"province"`
	Year     int    `form:"year"`
	Subject  string `form:"subject"`
	Batch    string `form:"batch"`
}

type AdminScoreRankListRequest struct {
	AdminPaginationRequest
	Keyword  string `form:"keyword"`
	Province string `form:"province"`
	Year     int    `form:"year"`
	Subject  string `form:"subject"`
	Score    int    `form:"score"`
}

type AdminStudentListRequest struct {
	AdminPaginationRequest
	Keyword string `form:"keyword"`
}

type AdminProfileOptionListRequest struct {
	AdminPaginationRequest
	Keyword    string `form:"keyword"`
	OptionType string `form:"optionType"`
}

type AdminUserListRequest struct {
	AdminPaginationRequest
	Keyword string `form:"keyword"`
}

type AdminTaskListRequest struct {
	AdminPaginationRequest
	Keyword  string `form:"keyword"`
	TaskType string `form:"taskType"`
	Status   string `form:"status"`
}

type AdminOrderListRequest struct {
	AdminPaginationRequest
	Keyword   string `form:"keyword"`
	Status    string `form:"status"`
	ProductID string `form:"productId"`
}

type AdminStaffSaveRequest struct {
	AdminUser
	Password string `json:"password"`
}

type AdminOrderBackfillRequest struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type AdminEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type AdminDashboard struct {
	CollegeCount      int `json:"collegeCount"`
	ProvinceLineCount int `json:"provinceLineCount"`
	ScoreRankCount    int `json:"scoreRankCount"`
	StudentCount      int `json:"studentCount"`
	StaffCount        int `json:"staffCount"`
	VolunteerCount    int `json:"volunteerCount"`
	AITaskCount       int `json:"aiTaskCount"`
	VIPProductCount   int `json:"vipProductCount"`
}

type AdminCollege struct {
	ID                          int       `json:"id"`
	Name                        string    `json:"name"`
	Province                    string    `json:"province"`
	City                        string    `json:"city"`
	Level                       string    `json:"level"`
	Is985                       bool      `json:"is985"`
	Is211                       bool      `json:"is211"`
	IsDoubleFirst               bool      `json:"isDoubleFirst"`
	Website                     string    `json:"website"`
	Ranking                     string    `json:"ranking"`
	SchoolType                  string    `json:"schoolType"`
	OwnershipType               string    `json:"ownershipType"`
	RecommendedPostgraduateRate string    `json:"recommendedPostgraduateRate"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}

type AdminProvinceScoreLine struct {
	ID         int       `json:"id"`
	Province   string    `json:"province"`
	Year       int       `json:"year"`
	Subject    string    `json:"subject"`
	Batch      string    `json:"batch"`
	Score      int       `json:"score"`
	SourceName string    `json:"sourceName"`
	SourceURL  string    `json:"sourceUrl"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type AdminScoreRank struct {
	ID        int       `json:"id"`
	Province  string    `json:"province"`
	Year      int       `json:"year"`
	Subject   string    `json:"subject"`
	Score     int       `json:"score"`
	Rank      int       `json:"rank"`
	Count     int       `json:"count"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AdminStudent struct {
	ID            string    `json:"id"`
	OpenID        string    `json:"openId"`
	Phone         string    `json:"phone"`
	Nickname      string    `json:"nickname"`
	AvatarURL     string    `json:"avatarUrl"`
	IDCard        string    `json:"idCard"`
	SchoolName    string    `json:"schoolName"`
	SchoolYear    string    `json:"schoolYear"`
	ClassName     string    `json:"className"`
	StudentNo     string    `json:"studentNo"`
	FromRecommend bool      `json:"fromRecommend"`
	LoginType     string    `json:"loginType"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	LastLoginAt   time.Time `json:"lastLoginAt"`
}

type AdminProfileOption struct {
	ID          int       `json:"id"`
	OptionType  string    `json:"optionType"`
	OptionLabel string    `json:"optionLabel"`
	OptionValue string    `json:"optionValue"`
	SortOrder   int       `json:"sortOrder"`
	Enabled     bool      `json:"enabled"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ProfileOptionItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type ProfileOptionCatalogResponse struct {
	Schools     []ProfileOptionItem `json:"schools"`
	SchoolYears []ProfileOptionItem `json:"schoolYears"`
	ClassNames  []ProfileOptionItem `json:"classNames"`
}

type AdminTask struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	TaskType     string    `json:"taskType"`
	Status       string    `json:"status"`
	Provider     string    `json:"provider"`
	Demand       string    `json:"demand"`
	Student      string    `json:"student"`
	Report       string    `json:"report"`
	ErrorMessage string    `json:"errorMessage"`
	AttemptCount int       `json:"attemptCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	CompletedAt  time.Time `json:"completedAt"`
}

type AdminOrder struct {
	ID             int        `json:"id"`
	OrderID        string     `json:"orderId"`
	UserID         string     `json:"userId"`
	UserNickname   string     `json:"userNickname"`
	UserPhone      string     `json:"userPhone"`
	OpenID         string     `json:"openId"`
	ProductID      string     `json:"productId"`
	ProductName    string     `json:"productName"`
	Content        string     `json:"content"`
	AmountFen      int        `json:"amountFen"`
	Status         string     `json:"status"`
	PaymentChannel string     `json:"paymentChannel"`
	PrepayID       string     `json:"prepayId"`
	TransactionID  string     `json:"transactionId"`
	Remark         string     `json:"remark"`
	PaidAt         *time.Time `json:"paidAt"`
	ExpiresAt      *time.Time `json:"expiresAt"`
	EffectiveFrom  *time.Time `json:"effectiveFrom"`
	EffectiveUntil *time.Time `json:"effectiveUntil"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type VIPProductConfig struct {
	ID           int        `json:"id"`
	ProductID    string     `json:"productId"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	AmountFen    int        `json:"amountFen"`
	Enabled      bool       `json:"enabled"`
	ValidityType string     `json:"validityType"`
	ValidTimes   int        `json:"validTimes"`
	ValidFrom    *time.Time `json:"validFrom"`
	ValidUntil   *time.Time `json:"validUntil"`
	OrderCount   int        `json:"orderCount"`
	SortOrder    int        `json:"sortOrder"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type VIPEntryControlConfig struct {
	ShowVIPEntry bool      `json:"showVipEntry"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type VIPEntryConfigResponse struct {
	ShowVIPEntry bool `json:"showVipEntry"`
}

type ShareGateControlConfig struct {
	RequireShareForAIReport        bool      `json:"requireShareForAiReport"`
	RequireShareForCollegeMajor    bool      `json:"requireShareForCollegeMajor"`
	RequireShareForRecommendResult bool      `json:"requireShareForRecommendResult"`
	RequireShareForPlanCompare     bool      `json:"requireShareForPlanCompare"`
	UpdatedAt                      time.Time `json:"updatedAt"`
}

type ShareGateConfigResponse struct {
	RequireShareForAIReport        bool `json:"requireShareForAiReport"`
	RequireShareForCollegeMajor    bool `json:"requireShareForCollegeMajor"`
	RequireShareForRecommendResult bool `json:"requireShareForRecommendResult"`
	RequireShareForPlanCompare     bool `json:"requireShareForPlanCompare"`
}

type AdminListResponse[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

type AdminUserResponse struct {
	User *AdminUser `json:"user"`
}

type AdminItemsResponse[T any] struct {
	Items []T `json:"items"`
}

type AdminOKResponse struct {
	OK bool `json:"ok"`
}

type AdminIDResponse struct {
	ID int  `json:"id"`
	OK bool `json:"ok"`
}

type AdminIDOnlyResponse struct {
	ID int `json:"id"`
}
