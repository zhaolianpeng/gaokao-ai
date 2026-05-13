package model

import (
	"fmt"
	"time"
)

type AuthUser struct {
	ID            string `json:"id"`
	OpenID        string `json:"openid"`
	Phone         string `json:"phone"`
	Nickname      string `json:"nickname"`
	AvatarURL     string `json:"avatarUrl"`
	IDCard        string `json:"idCard"`
	SchoolName    string `json:"schoolName"`
	SchoolYear    string `json:"schoolYear"`
	ClassName     string `json:"className"`
	StudentNo     string `json:"studentNo"`
	FromRecommend bool   `json:"fromRecommend"`
	LoginType     string `json:"loginType"`
	StorageMode   string `json:"storageMode"`
	Created       bool   `json:"created"`
	CreatedAt     int64  `json:"createdAt"`
	UpdatedAt     int64  `json:"updatedAt"`
	LastLoginAt   int64  `json:"lastLoginAt"`
}

type AuthUserRecord struct {
	ID            string
	OpenID        string
	Phone         string
	Nickname      string
	AvatarURL     string
	IDCard        string
	SchoolName    string
	SchoolYear    string
	ClassName     string
	StudentNo     string
	FromRecommend bool
	LoginType     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastLoginAt   time.Time
}

type WechatLoginRequest struct {
	Code      string `json:"code" binding:"required"`
	LoginCode string `json:"loginCode" binding:"required"`
}

type WechatProfileUpdateRequest struct {
	UserID        string  `json:"userId" binding:"required"`
	Phone         *string `json:"phone"`
	Nickname      string  `json:"nickname" binding:"required"`
	AvatarURL     *string `json:"avatarUrl"`
	IDCard        *string `json:"idCard"`
	SchoolName    *string `json:"schoolName"`
	SchoolYear    *string `json:"schoolYear"`
	ClassName     *string `json:"className"`
	StudentNo     *string `json:"studentNo"`
	FromRecommend *bool   `json:"fromRecommend"`
}

type WechatPayRequest struct {
	UserID    string `json:"userId" binding:"required"`
	ProductID string `json:"productId" binding:"required"`
	OpenID    string `json:"openId"`
	OrderID   string `json:"orderId" binding:"required"`
}

type WechatPayConfirmRequest struct {
	UserID    string `json:"userId" binding:"required"`
	ProductID string `json:"productId" binding:"required"`
	OrderID   string `json:"orderId" binding:"required"`
}

type WechatPaymentParams struct {
	AppID     string `json:"appId"`
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

type WechatPayDebug struct {
	Nonce     string `json:"nonce"`
	PrepayID  string `json:"prepayId"`
	TimeStamp string `json:"timestamp"`
}

type WechatPayResponse struct {
	AmountFen int                 `json:"amountFen"`
	Debug     WechatPayDebug      `json:"debug"`
	ExpiresAt int64               `json:"expiresAt"`
	OrderID   string              `json:"orderId"`
	Payment   WechatPaymentParams `json:"payment"`
	ProductID string              `json:"productId"`
}

type WechatVIPMembershipRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type VIPMembershipStatusResponse struct {
	Active       bool   `json:"active"`
	OrderID      string `json:"orderId"`
	ProductID    string `json:"productId"`
	ProductName  string `json:"productName"`
	LevelType    string `json:"levelType"`
	LevelText    string `json:"levelText"`
	StatusText   string `json:"statusText"`
	ValidityText string `json:"validityText"`
	StartAt      int64  `json:"startAt"`
	EndAt        int64  `json:"endAt"`
	PaidAt       int64  `json:"paidAt"`
	StartText    string `json:"startText"`
	EndText      string `json:"endText"`
}

type TaskStudent struct {
	Province      string `json:"province"`
	Subject       string `json:"subject"`
	AnalysisYear  string `json:"analysisYear"`
	Score         int    `json:"score"`
	Rank          int    `json:"rank"`
	TargetMajor   string `json:"targetMajor"`
	Notes         string `json:"notes"`
	SchoolName    string `json:"schoolName"`
	SchoolYear    string `json:"schoolYear"`
	ClassName     string `json:"className"`
	FromRecommend bool   `json:"fromRecommend"`
	Year          int    `json:"year"`
}

func (s TaskStudent) Validate() error {
	if err := ValidateGaokaoScore(s.Score); err != nil {
		return err
	}
	if err := ValidateGaokaoRank(s.Rank); err != nil {
		return err
	}
	if s.Year < 0 {
		return fmt.Errorf("invalid year")
	}
	return nil
}

type AgentRecommendRequest struct {
	Student   TaskStudent `json:"student" binding:"required"`
	Demand    string      `json:"demand" binding:"required"`
	Templates []string    `json:"templates"`
}

func (r AgentRecommendRequest) Validate() error {
	return r.Student.Validate()
}

type AgentSuggestion struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Keyword string `json:"keyword"`
	Subject string `json:"subject"`
}

type TaskRecord struct {
	ID              int
	Title           string
	StudentJSON     []byte
	Demand          string
	TemplatesJSON   []byte
	SuggestionsJSON []byte
	Status          string
	Report          string
	Provider        string
	ErrorMessage    string
	AttemptCount    int
	TaskType        string
	RecommendJSON   []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
}

type TaskStatusResponse struct {
	TaskID       string            `json:"taskId"`
	Title        string            `json:"title"`
	Status       string            `json:"status"`
	Ready        bool              `json:"ready"`
	Failed       bool              `json:"failed"`
	ErrorMessage string            `json:"errorMessage,omitempty"`
	Report       string            `json:"report,omitempty"`
	Provider     string            `json:"provider,omitempty"`
	Student      *TaskStudent      `json:"student,omitempty"`
	Suggestions  []AgentSuggestion `json:"suggestions,omitempty"`
}
