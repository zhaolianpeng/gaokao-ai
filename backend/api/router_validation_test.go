package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"gaokao-ai/backend/model"
)

func TestRecommendRejectsUnrealisticScore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(nil, nil, nil, nil, nil, nil, nil, nil, nil, "", "", 0)

	req := httptest.NewRequest(http.MethodPost, "/api/recommend", strings.NewReader(`{"province":"黑龙江","score":8608,"rank":1000,"subject":"physics"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}

	var body model.ErrorResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Error != "score out of realistic range" {
		t.Fatalf("expected score range error, got %q", body.Error)
	}
}

func TestAgentRecommendRejectsUnrealisticRank(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(nil, nil, nil, nil, nil, nil, nil, nil, nil, "", "", 0)

	req := httptest.NewRequest(http.MethodPost, "/api/agent-recommend", strings.NewReader(`{"student":{"province":"黑龙江","subject":"physics","score":620,"rank":3000000,"year":2025},"demand":"推荐院校"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}

	var body model.ErrorResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Error != "rank out of realistic range" {
		t.Fatalf("expected rank range error, got %q", body.Error)
	}
}

func TestScoreRankLookupRejectsUnrealisticScore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(nil, nil, nil, nil, nil, nil, nil, nil, nil, "", "", 0)

	req := httptest.NewRequest(http.MethodGet, "/api/score-rank?province=黑龙江&year=2025&subject=physics&score=8608", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}

	var body model.ErrorResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Error != "score out of realistic range" {
		t.Fatalf("expected score range error, got %q", body.Error)
	}
}
