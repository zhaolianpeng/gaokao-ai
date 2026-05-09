package model

type FeedbackSubmitRequest struct {
	Content        string `json:"content" binding:"required"`
	Contact        string `json:"contact"`
	Page           string `json:"page"`
	BackendBaseURL string `json:"backendBaseUrl"`
	Phone          string `json:"phone"`
	Nickname       string `json:"nickname"`
}

type FeedbackRecord struct {
	ID             int
	Content        string
	Contact        string
	Page           string
	BackendBaseURL string
	Phone          string
	Nickname       string
}

type FeedbackSubmitResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}
