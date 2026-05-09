package api

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gaokao-ai/backend/logging"
	"gaokao-ai/backend/model"
)

func adminOperationLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		userValue, exists := c.Get("adminUser")
		if !exists {
			return
		}
		user, ok := userValue.(*model.AdminUser)
		if !ok || user == nil {
			return
		}

		logging.LogEvent("admin_operation", map[string]any{
			"adminId":     user.ID,
			"username":    user.Username,
			"displayName": user.DisplayName,
			"role":        user.Role,
			"method":      c.Request.Method,
			"path":        c.FullPath(),
			"rawPath":     c.Request.URL.Path,
			"query":       c.Request.URL.RawQuery,
			"resource":    adminOperationResource(c.FullPath()),
			"action":      adminOperationAction(c.Request.Method, c.FullPath()),
			"status":      c.Writer.Status(),
			"latencyMs":   time.Since(startedAt).Milliseconds(),
			"pathParams":  paramsToMap(c.Params),
		})
	}
}

func adminOperationAction(method, path string) string {
	trimmedPath := strings.Trim(strings.TrimSpace(path), "/")
	if trimmedPath == "api/admin/logout" {
		return "logout"
	}
	if strings.HasSuffix(trimmedPath, "/toggle") {
		return "toggle"
	}
	if strings.HasSuffix(trimmedPath, "/backfill") {
		return "backfill"
	}
	if strings.HasSuffix(trimmedPath, "/list") {
		return "list"
	}
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET":
		return "view"
	case "POST":
		return "submit"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return strings.ToLower(strings.TrimSpace(method))
	}
}

func adminOperationResource(path string) string {
	trimmedPath := strings.Trim(strings.TrimSpace(path), "/")
	if trimmedPath == "" {
		return "admin"
	}
	parts := strings.Split(trimmedPath, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return parts[len(parts)-1]
}

func paramsToMap(params gin.Params) map[string]string {
	if len(params) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(params))
	for _, item := range params {
		result[item.Key] = item.Value
	}
	return result
}
