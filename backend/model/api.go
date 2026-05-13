package model

import "fmt"

const (
	MaxGaokaoScore = 900
	MaxGaokaoRank  = 2000000
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type ItemsResponse[T any] struct {
	Items []T `json:"items"`
}

type TaskStatusRequest struct {
	TaskID string `json:"taskId" form:"taskId" binding:"required"`
}

type DashboardOverviewRequest struct {
	Province string `json:"province" form:"province"`
	Year     int    `json:"year" form:"year"`
	Subject  string `json:"subject" form:"subject"`
}

func (r *DashboardOverviewRequest) Normalize() {
	if r.Year <= 0 {
		r.Year = 2025
	}
}

type ProvinceLinesRequest struct {
	Province string `json:"province" form:"province"`
	Year     int    `json:"year" form:"year"`
	Subject  string `json:"subject" form:"subject"`
}

func (r *ProvinceLinesRequest) Normalize() {
	if r.Year <= 0 {
		r.Year = 2025
	}
	if r.Province == "" {
		r.Province = "黑龙江"
	}
}

type ScoreRankRequest struct {
	Province string `json:"province" form:"province"`
	Year     int    `json:"year" form:"year"`
	Subject  string `json:"subject" form:"subject"`
	Score    int    `json:"score" form:"score"`
}

func (r *ScoreRankRequest) Normalize() {
	if r.Year <= 0 {
		r.Year = 2025
	}
	if r.Province == "" {
		r.Province = "黑龙江"
	}
}

func (r ScoreRankRequest) Validate() error {
	if err := ValidateGaokaoScore(r.Score); err != nil {
		return err
	}
	return nil
}

type RankScoreRequest struct {
	Province string `json:"province" form:"province"`
	Year     int    `json:"year" form:"year"`
	Subject  string `json:"subject" form:"subject"`
	Rank     int    `json:"rank" form:"rank"`
}

func (r *RankScoreRequest) Normalize() {
	if r.Year <= 0 {
		r.Year = 2025
	}
	if r.Province == "" {
		r.Province = "黑龙江"
	}
}

func (r RankScoreRequest) Validate() error {
	if err := ValidateGaokaoRank(r.Rank); err != nil {
		return err
	}
	return nil
}

type ExplorerCollegeListRequest struct {
	Province string `json:"province" form:"province"`
	Year     int    `json:"year" form:"year"`
	Subject  string `json:"subject" form:"subject"`
	Keyword  string `json:"keyword" form:"keyword"`
	SortMode string `json:"sortMode" form:"sortMode"`
	Page     int    `json:"page" form:"page"`
	Limit    int    `json:"limit" form:"limit"`
}

func (r *ExplorerCollegeListRequest) Normalize() {
	if r.Year <= 0 {
		r.Year = 2025
	}
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.Limit <= 0 {
		r.Limit = 20
	}
	if r.SortMode == "" {
		r.SortMode = "tier"
	}
}

type CollegeDetailRequest struct {
	Province string `json:"province" form:"province"`
	Year     int    `json:"year" form:"year"`
	Subject  string `json:"subject" form:"subject"`
}

func (r *CollegeDetailRequest) Normalize() {
	if r.Year <= 0 {
		r.Year = 2025
	}
}

func ValidateGaokaoScore(score int) error {
	if score <= 0 {
		return fmt.Errorf("invalid score")
	}
	if score > MaxGaokaoScore {
		return fmt.Errorf("score out of realistic range")
	}
	return nil
}

func ValidateGaokaoRank(rank int) error {
	if rank <= 0 {
		return fmt.Errorf("invalid rank")
	}
	if rank > MaxGaokaoRank {
		return fmt.Errorf("rank out of realistic range")
	}
	return nil
}
