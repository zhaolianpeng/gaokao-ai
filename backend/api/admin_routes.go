package api

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"gaokao-ai/backend/logging"
	"gaokao-ai/backend/model"
	"gaokao-ai/backend/service"
)

const adminCookieName = "gaokao_mis_session"

var misPageRoutes = map[string]struct{}{
	"dashboard":      {},
	"colleges":       {},
	"province-lines": {},
	"score-ranks":    {},
	"students":       {},
	"orders":         {},
	"staff":          {},
	"volunteers":     {},
	"ai-tasks":       {},
	"payment-items":  {},
}

func serveMISPage(c *gin.Context) {
	page := strings.TrimSpace(c.Param("page"))
	if page != "" {
		if _, ok := misPageRoutes[page]; !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
			return
		}
	}
	c.File(filepath.Join("admin", "index.html"))
}

func registerAdminRoutes(r *gin.Engine, adminService *service.AdminService, payService *service.PayService) {
	if adminService == nil {
		return
	}

	r.GET("/mis", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/mis/dashboard")
	})
	r.GET("/mis/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/mis/dashboard")
	})
	r.GET("/mis/:page", serveMISPage)
	r.GET("/api/admin/console", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/mis/dashboard")
	})

	r.POST("/api/admin/login", func(c *gin.Context) {
		var req model.AdminLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logging.LogEvent("admin_login", map[string]any{"username": strings.TrimSpace(req.Username), "status": "bad_request", "error": err.Error()})
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		token, user, err := adminService.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			logging.LogEvent("admin_login", map[string]any{"username": strings.TrimSpace(req.Username), "status": "failed", "error": err.Error()})
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.SetCookie(adminCookieName, token, 86400*7, "/", "", false, true)
		logging.LogEvent("admin_login", map[string]any{"adminId": user.ID, "username": user.Username, "status": "success"})
		c.JSON(http.StatusOK, gin.H{"user": user})
	})

	adminGroup := r.Group("/api/admin")
	adminGroup.Use(adminAuthRequired(adminService), adminOperationLogger())
	{
		adminGroup.GET("/me", func(c *gin.Context) {
			user := c.MustGet("adminUser")
			c.JSON(http.StatusOK, gin.H{"user": user})
		})
		adminGroup.POST("/me", func(c *gin.Context) {
			user := c.MustGet("adminUser")
			c.JSON(http.StatusOK, gin.H{"user": user})
		})

		adminGroup.POST("/logout", func(c *gin.Context) {
			if user, ok := c.MustGet("adminUser").(*model.AdminUser); ok && user != nil {
				logging.LogEvent("admin_logout", map[string]any{"adminId": user.ID, "username": user.Username, "status": "success"})
			}
			adminService.Logout(readAdminToken(c))
			c.SetCookie(adminCookieName, "", -1, "/", "", false, true)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		adminGroup.GET("/dashboard", func(c *gin.Context) {
			data, err := adminService.Repo().GetDashboard(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, data)
		})
		adminGroup.POST("/dashboard", func(c *gin.Context) {
			data, err := adminService.Repo().GetDashboard(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, data)
		})

		adminGroup.GET("/colleges", func(c *gin.Context) {
			items, total, err := adminService.Repo().ListColleges(
				c.Request.Context(),
				c.Query("keyword"),
				c.Query("name"),
				c.Query("level"),
				c.Query("schoolType"),
				c.Query("ownershipType"),
				c.Query("province"),
				c.Query("city"),
				c.Query("is985"),
				c.Query("is211"),
				c.Query("isDoubleFirst"),
				parseInt(c.Query("page"), 1),
				parseInt(c.Query("limit"), 20),
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminCollege]{Items: items, Total: total, Page: parseInt(c.Query("page"), 1), Limit: parseInt(c.Query("limit"), 20)})
		})
		adminGroup.POST("/colleges/list", func(c *gin.Context) {
			items, total, err := adminService.Repo().ListColleges(
				c.Request.Context(),
				c.Query("keyword"),
				c.Query("name"),
				c.Query("level"),
				c.Query("schoolType"),
				c.Query("ownershipType"),
				c.Query("province"),
				c.Query("city"),
				c.Query("is985"),
				c.Query("is211"),
				c.Query("isDoubleFirst"),
				parseInt(c.Query("page"), 1),
				parseInt(c.Query("limit"), 20),
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminCollege]{Items: items, Total: total, Page: parseInt(c.Query("page"), 1), Limit: parseInt(c.Query("limit"), 20)})
		})

		adminGroup.POST("/colleges", func(c *gin.Context) {
			var req model.AdminCollege
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			id, err := adminService.Repo().SaveCollege(c.Request.Context(), req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"id": id, "ok": true})
		})

		adminGroup.GET("/province-lines", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListProvinceScoreLines(c.Request.Context(), c.Query("keyword"), c.Query("province"), parseInt(c.Query("year"), 0), c.Query("subject"), c.Query("batch"), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminProvinceScoreLine]{Items: items, Total: total, Page: page, Limit: limit})
		})
		adminGroup.POST("/province-lines/list", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListProvinceScoreLines(c.Request.Context(), c.Query("keyword"), c.Query("province"), parseInt(c.Query("year"), 0), c.Query("subject"), c.Query("batch"), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminProvinceScoreLine]{Items: items, Total: total, Page: page, Limit: limit})
		})

		adminGroup.POST("/province-lines", func(c *gin.Context) {
			var req model.AdminProvinceScoreLine
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			id, err := adminService.Repo().SaveProvinceScoreLine(c.Request.Context(), req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"id": id, "ok": true})
		})

		adminGroup.DELETE("/province-lines/:id", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}
			if err := adminService.Repo().DeleteProvinceScoreLine(c.Request.Context(), id); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		adminGroup.GET("/score-ranks", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListScoreRanks(c.Request.Context(), c.Query("keyword"), c.Query("province"), parseInt(c.Query("year"), 0), c.Query("subject"), parseInt(c.Query("score"), 0), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminScoreRank]{Items: items, Total: total, Page: page, Limit: limit})
		})
		adminGroup.POST("/score-ranks/list", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListScoreRanks(c.Request.Context(), c.Query("keyword"), c.Query("province"), parseInt(c.Query("year"), 0), c.Query("subject"), parseInt(c.Query("score"), 0), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminScoreRank]{Items: items, Total: total, Page: page, Limit: limit})
		})

		adminGroup.POST("/score-ranks", func(c *gin.Context) {
			var req model.AdminScoreRank
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			id, err := adminService.Repo().SaveScoreRank(c.Request.Context(), req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"id": id, "ok": true})
		})

		adminGroup.DELETE("/score-ranks/:id", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}
			if err := adminService.Repo().DeleteScoreRank(c.Request.Context(), id); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		adminGroup.GET("/students", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListStudents(c.Request.Context(), c.Query("keyword"), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminStudent]{Items: items, Total: total, Page: page, Limit: limit})
		})
		adminGroup.POST("/students/list", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListStudents(c.Request.Context(), c.Query("keyword"), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminStudent]{Items: items, Total: total, Page: page, Limit: limit})
		})

		adminGroup.POST("/students", func(c *gin.Context) {
			var req model.AdminStudent
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if req.ID <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
				return
			}
			if err := adminService.Repo().SaveStudent(c.Request.Context(), req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		adminGroup.GET("/staff", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListAdminUsers(c.Request.Context(), c.Query("keyword"), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminUser]{Items: items, Total: total, Page: page, Limit: limit})
		})
		adminGroup.POST("/staff/list", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListAdminUsers(c.Request.Context(), c.Query("keyword"), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminUser]{Items: items, Total: total, Page: page, Limit: limit})
		})

		adminGroup.POST("/staff", func(c *gin.Context) {
			var req struct {
				model.AdminUser
				Password string `json:"password"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			password := strings.TrimSpace(req.Password)
			if req.ID <= 0 && password == "" {
				password = "admin123"
			}
			passwordHash := ""
			if password != "" {
				var err error
				passwordHash, err = service.HashPassword(password)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
			id, err := adminService.Repo().SaveAdminUser(c.Request.Context(), req.AdminUser, passwordHash)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"id": id, "ok": true})
		})

		adminGroup.DELETE("/staff/:id", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}
			user := c.MustGet("adminUser").(*model.AdminUser)
			if user.ID == id {
				c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除当前登录账号"})
				return
			}
			if err := adminService.Repo().DeleteAdminUser(c.Request.Context(), id); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		adminGroup.GET("/tasks", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListTasks(c.Request.Context(), c.Query("keyword"), c.Query("taskType"), c.Query("status"), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminTask]{Items: items, Total: total, Page: page, Limit: limit})
		})
		adminGroup.POST("/tasks/list", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListTasks(c.Request.Context(), c.Query("keyword"), c.Query("taskType"), c.Query("status"), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminTask]{Items: items, Total: total, Page: page, Limit: limit})
		})

		adminGroup.GET("/orders", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListOrders(c.Request.Context(), c.Query("keyword"), c.Query("status"), c.Query("productId"), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminOrder]{Items: items, Total: total, Page: page, Limit: limit})
		})
		adminGroup.POST("/orders/list", func(c *gin.Context) {
			page := parseInt(c.Query("page"), 1)
			limit := parseInt(c.Query("limit"), 20)
			items, total, err := adminService.Repo().ListOrders(c.Request.Context(), c.Query("keyword"), c.Query("status"), c.Query("productId"), page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, model.AdminListResponse[model.AdminOrder]{Items: items, Total: total, Page: page, Limit: limit})
		})

		adminGroup.POST("/orders", func(c *gin.Context) {
			var req model.AdminOrder
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			id, err := adminService.Repo().SaveOrder(c.Request.Context(), req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"id": id, "ok": true})
		})

		adminGroup.POST("/orders/backfill", func(c *gin.Context) {
			if payService == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "pay service unavailable"})
				return
			}
			var req struct {
				StartDate string `json:"startDate"`
				EndDate   string `json:"endDate"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			result, err := payService.BackfillOrders(c.Request.Context(), req.StartDate, req.EndDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, result)
		})

		adminGroup.DELETE("/tasks/:id", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}
			if err := adminService.Repo().DeleteTask(c.Request.Context(), id); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		adminGroup.GET("/vip-products", func(c *gin.Context) {
			items, err := adminService.Repo().ListVIPProducts(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"items": items})
		})
		adminGroup.POST("/vip-products/list", func(c *gin.Context) {
			items, err := adminService.Repo().ListVIPProducts(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"items": items})
		})

		adminGroup.POST("/vip-products", func(c *gin.Context) {
			var req model.VIPProductConfig
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			id, err := adminService.Repo().SaveVIPProduct(c.Request.Context(), req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"id": id, "ok": true})
		})

		adminGroup.POST("/vip-products/:id/toggle", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}
			var req struct {
				Enabled bool `json:"enabled"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := adminService.Repo().SetVIPProductEnabled(c.Request.Context(), id, req.Enabled); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		adminGroup.DELETE("/vip-products/:id", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
				return
			}
			if err := adminService.Repo().DeleteVIPProduct(c.Request.Context(), id); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
	}
}

func adminAuthRequired(adminService *service.AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := adminService.CurrentUser(c.Request.Context(), readAdminToken(c))
		if err != nil {
			logging.LogEvent("admin_auth", map[string]any{"status": "failed", "error": err.Error(), "path": c.Request.URL.Path})
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Set("adminUser", user)
		c.Next()
	}
}

func readAdminToken(c *gin.Context) string {
	token, _ := c.Cookie(adminCookieName)
	return strings.TrimSpace(token)
}

func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed == 0 {
		return fallback
	}
	return parsed
}
