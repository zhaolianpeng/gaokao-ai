package model

import "time"

type AuthUser struct {
	ID          string `json:"id"`
	OpenID      string `json:"openid"`
	Phone       string `json:"phone"`
	Nickname    string `json:"nickname"`
	AvatarURL   string `json:"avatarUrl"`
	LoginType   string `json:"loginType"`
	StorageMode string `json:"storageMode"`
	Created     bool   `json:"created"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
	LastLoginAt int64  `json:"lastLoginAt"`
}

type AuthUserRecord struct {
	ID          int
	OpenID      string
	Phone       string
	Nickname    string
	AvatarURL   string
	LoginType   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt time.Time
}

type WechatLoginRequest struct {
	Code      string `json:"code" binding:"required"`
	LoginCode string `json:"loginCode" binding:"required"`
}

type WechatProfileUpdateRequest struct {
	UserID    string `json:"userId" binding:"required"`
	Phone     string `json:"phone"`
	Nickname  string `json:"nickname" binding:"required"`
	AvatarURL string `json:"avatarUrl" binding:"required"`
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
	OrderID   string              `json:"orderId"`
	Payment   WechatPaymentParams `json:"payment"`
	ProductID string              `json:"productId"`
}

type TaskStudent struct {
	Province     string `json:"province"`
	Subject      string `json:"subject"`
	AnalysisYear string `json:"analysisYear"`
	Score        int    `json:"score"`
	Rank         int    `json:"rank"`
	TargetMajor  string `json:"targetMajor"`
	Notes        string `json:"notes"`
	Year         int    `json:"year"`
}

type AgentRecommendRequest struct {
	Student   TaskStudent `json:"student" binding:"required"`
	Demand    string      `json:"demand" binding:"required"`
	Templates []string    `json:"templates"`
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
