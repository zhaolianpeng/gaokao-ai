package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"gaokao-ai/backend/model"
	"gaokao-ai/backend/service"
)

func TestAdminLoginBindingReturnsStructuredError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerAdminRoutes(r, service.NewAdminService(nil), nil)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(`{}`))
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
	if strings.TrimSpace(body.Error) == "" {
		t.Fatal("expected error message in structured response")
	}
}

func TestAdminAuthMiddlewareReturnsStructuredError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerAdminRoutes(r, service.NewAdminService(nil), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}

	var body model.ErrorResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if strings.TrimSpace(body.Error) == "" {
		t.Fatal("expected auth middleware to return structured error message")
	}
}

func TestBindAdminQueryReturnsStructuredErrorOnInvalidInt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test-admin-query", func(c *gin.Context) {
		var req model.AdminCollegeListRequest
		if !bindAdminQuery(c, &req) {
			return
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test-admin-query?page=abc", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}

	var body model.ErrorResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if strings.TrimSpace(body.Error) == "" {
		t.Fatal("expected query binding failure to return structured error")
	}
}

func TestWriteAdminListUsesNormalizedPaginationDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test-admin-list", func(c *gin.Context) {
		var req model.AdminUserListRequest
		if !bindAdminQuery(c, &req) {
			return
		}
		writeAdminList(c, req.AdminPaginationRequest, []model.AdminUser{{ID: 1, Username: "tester"}}, 1)
	})

	req := httptest.NewRequest(http.MethodGet, "/test-admin-list", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var body model.AdminListResponse[model.AdminUser]
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Page != 1 {
		t.Fatalf("expected default page 1, got %d", body.Page)
	}
	if body.Limit != 20 {
		t.Fatalf("expected default limit 20, got %d", body.Limit)
	}
	if len(body.Items) != 1 || body.Items[0].Username != "tester" {
		t.Fatal("expected list response items to be preserved")
	}
}
