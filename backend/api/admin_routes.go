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
	"dashboard":       {},
	"colleges":        {},
	"province-lines":  {},
	"score-ranks":     {},
	"students":        {},
	"profile-options": {},
	"orders":          {},
	"staff":           {},
	"volunteers":      {},
	"ai-tasks":        {},
	"vip-entry":       {},
	"share-gate":      {},
	"payment-items":   {},
}

func serveMISPage(c *gin.Context) {
	page := strings.TrimSpace(c.Param("page"))
	if page != "" {
		if _, ok := misPageRoutes[page]; !ok {
			writeError(c, http.StatusNotFound, "page not found")
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
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		token, user, err := adminService.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			logging.LogEvent("admin_login", map[string]any{"username": strings.TrimSpace(req.Username), "status": "failed", "error": err.Error()})
			writeError(c, http.StatusUnauthorized, err.Error())
			return
		}
		c.SetCookie(adminCookieName, token, 86400*7, "/", "", false, true)
		logging.LogEvent("admin_login", map[string]any{"adminId": user.ID, "username": user.Username, "status": "success"})
		c.JSON(http.StatusOK, model.AdminUserResponse{User: user})
	})

	adminGroup := r.Group("/api/admin")
	adminGroup.Use(adminAuthRequired(adminService), adminOperationLogger())
	{
		adminGroup.GET("/me", func(c *gin.Context) {
			user, _ := c.MustGet("adminUser").(*model.AdminUser)
			c.JSON(http.StatusOK, model.AdminUserResponse{User: user})
		})
		adminGroup.POST("/me", func(c *gin.Context) {
			user, _ := c.MustGet("adminUser").(*model.AdminUser)
			c.JSON(http.StatusOK, model.AdminUserResponse{User: user})
		})

		adminGroup.POST("/logout", func(c *gin.Context) {
			if user, ok := c.MustGet("adminUser").(*model.AdminUser); ok && user != nil {
				logging.LogEvent("admin_logout", map[string]any{"adminId": user.ID, "username": user.Username, "status": "success"})
			}
			adminService.Logout(readAdminToken(c))
			c.SetCookie(adminCookieName, "", -1, "/", "", false, true)
			writeAdminOK(c)
		})

		adminGroup.GET("/dashboard", func(c *gin.Context) {
			data, err := adminService.Repo().GetDashboard(c.Request.Context())
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			c.JSON(http.StatusOK, data)
		})
		adminGroup.POST("/dashboard", func(c *gin.Context) {
			data, err := adminService.Repo().GetDashboard(c.Request.Context())
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			c.JSON(http.StatusOK, data)
		})

		adminGroup.GET("/colleges", func(c *gin.Context) {
			var req model.AdminCollegeListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListColleges(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})
		adminGroup.POST("/colleges/list", func(c *gin.Context) {
			var req model.AdminCollegeListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListColleges(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})

		adminGroup.POST("/colleges", func(c *gin.Context) {
			var req model.AdminCollege
			if !bindJSON(c, &req) {
				return
			}
			id, err := adminService.Repo().SaveCollege(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminID(c, id)
		})

		adminGroup.GET("/province-lines", func(c *gin.Context) {
			var req model.AdminProvinceScoreLineListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListProvinceScoreLines(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})
		adminGroup.POST("/province-lines/list", func(c *gin.Context) {
			var req model.AdminProvinceScoreLineListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListProvinceScoreLines(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})

		adminGroup.POST("/province-lines", func(c *gin.Context) {
			var req model.AdminProvinceScoreLine
			if !bindJSON(c, &req) {
				return
			}
			id, err := adminService.Repo().SaveProvinceScoreLine(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminID(c, id)
		})

		adminGroup.DELETE("/province-lines/:id", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				writeError(c, http.StatusBadRequest, "invalid id")
				return
			}
			if err := adminService.Repo().DeleteProvinceScoreLine(c.Request.Context(), id); err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminOK(c)
		})

		adminGroup.GET("/score-ranks", func(c *gin.Context) {
			var req model.AdminScoreRankListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListScoreRanks(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})
		adminGroup.POST("/score-ranks/list", func(c *gin.Context) {
			var req model.AdminScoreRankListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListScoreRanks(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})

		adminGroup.POST("/score-ranks", func(c *gin.Context) {
			var req model.AdminScoreRank
			if !bindJSON(c, &req) {
				return
			}
			id, err := adminService.Repo().SaveScoreRank(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminID(c, id)
		})

		adminGroup.DELETE("/score-ranks/:id", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				writeError(c, http.StatusBadRequest, "invalid id")
				return
			}
			if err := adminService.Repo().DeleteScoreRank(c.Request.Context(), id); err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminOK(c)
		})

		adminGroup.GET("/students", func(c *gin.Context) {
			var req model.AdminStudentListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListStudents(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})
		adminGroup.POST("/students/list", func(c *gin.Context) {
			var req model.AdminStudentListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListStudents(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})

		adminGroup.POST("/students", func(c *gin.Context) {
			var req model.AdminStudent
			if !bindJSON(c, &req) {
				return
			}
			if strings.TrimSpace(req.ID) == "" {
				writeError(c, http.StatusBadRequest, "invalid student id")
				return
			}
			if err := adminService.Repo().SaveStudent(c.Request.Context(), req); err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminOK(c)
		})

		adminGroup.GET("/profile-options", func(c *gin.Context) {
			var req model.AdminProfileOptionListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListProfileOptions(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})
		adminGroup.POST("/profile-options/list", func(c *gin.Context) {
			var req model.AdminProfileOptionListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListProfileOptions(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})
		adminGroup.POST("/profile-options", func(c *gin.Context) {
			var req model.AdminProfileOption
			if !bindJSON(c, &req) {
				return
			}
			id, err := adminService.Repo().SaveProfileOption(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			c.JSON(http.StatusOK, model.AdminIDOnlyResponse{ID: id})
		})
		adminGroup.DELETE("/profile-options/:id", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				writeError(c, http.StatusBadRequest, "invalid id")
				return
			}
			if err := adminService.Repo().DeleteProfileOption(c.Request.Context(), id); err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminOK(c)
		})

		adminGroup.GET("/staff", func(c *gin.Context) {
			var req model.AdminUserListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListAdminUsers(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})
		adminGroup.POST("/staff/list", func(c *gin.Context) {
			var req model.AdminUserListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListAdminUsers(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})

		adminGroup.POST("/staff", func(c *gin.Context) {
			var req model.AdminStaffSaveRequest
			if !bindJSON(c, &req) {
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
					writeError(c, http.StatusInternalServerError, err.Error())
					return
				}
			}
			id, err := adminService.Repo().SaveAdminUser(c.Request.Context(), req.AdminUser, passwordHash)
			if err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminID(c, id)
		})

		adminGroup.DELETE("/staff/:id", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				writeError(c, http.StatusBadRequest, "invalid id")
				return
			}
			user := c.MustGet("adminUser").(*model.AdminUser)
			if user.ID == id {
				writeError(c, http.StatusBadRequest, "不能删除当前登录账号")
				return
			}
			if err := adminService.Repo().DeleteAdminUser(c.Request.Context(), id); err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminOK(c)
		})

		adminGroup.GET("/tasks", func(c *gin.Context) {
			var req model.AdminTaskListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListTasks(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})
		adminGroup.POST("/tasks/list", func(c *gin.Context) {
			var req model.AdminTaskListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListTasks(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})

		adminGroup.GET("/orders", func(c *gin.Context) {
			var req model.AdminOrderListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListOrders(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})
		adminGroup.POST("/orders/list", func(c *gin.Context) {
			var req model.AdminOrderListRequest
			if !bindQuery(c, &req) {
				return
			}
			items, total, err := adminService.Repo().ListOrders(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminList(c, req.AdminPaginationRequest, items, total)
		})

		adminGroup.POST("/orders", func(c *gin.Context) {
			var req model.AdminOrder
			if !bindJSON(c, &req) {
				return
			}
			id, err := adminService.Repo().SaveOrder(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminID(c, id)
		})

		adminGroup.POST("/orders/backfill", func(c *gin.Context) {
			if payService == nil {
				writeError(c, http.StatusServiceUnavailable, "pay service unavailable")
				return
			}
			var req model.AdminOrderBackfillRequest
			if !bindJSON(c, &req) {
				return
			}
			result, err := payService.BackfillOrders(c.Request.Context(), req.StartDate, req.EndDate)
			if err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			c.JSON(http.StatusOK, result)
		})

		adminGroup.DELETE("/tasks/:id", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				writeError(c, http.StatusBadRequest, "invalid id")
				return
			}
			if err := adminService.Repo().DeleteTask(c.Request.Context(), id); err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminOK(c)
		})

		adminGroup.GET("/vip-products", func(c *gin.Context) {
			items, err := adminService.Repo().ListVIPProducts(c.Request.Context())
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminItems(c, items)
		})
		adminGroup.POST("/vip-products/list", func(c *gin.Context) {
			items, err := adminService.Repo().ListVIPProducts(c.Request.Context())
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			writeAdminItems(c, items)
		})

		adminGroup.POST("/vip-products", func(c *gin.Context) {
			var req model.VIPProductConfig
			if !bindJSON(c, &req) {
				return
			}
			id, err := adminService.Repo().SaveVIPProduct(c.Request.Context(), req)
			if err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminID(c, id)
		})

		adminGroup.GET("/vip-entry-config", func(c *gin.Context) {
			item, err := adminService.Repo().GetVIPEntryControlConfig(c.Request.Context())
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			c.JSON(http.StatusOK, item)
		})

		adminGroup.POST("/vip-entry-config", func(c *gin.Context) {
			var req model.VIPEntryControlConfig
			if !bindJSON(c, &req) {
				return
			}
			if err := adminService.Repo().SaveVIPEntryControlConfig(c.Request.Context(), req.ShowVIPEntry); err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminOK(c)
		})

		adminGroup.GET("/share-gate-config", func(c *gin.Context) {
			item, err := adminService.Repo().GetShareGateControlConfig(c.Request.Context())
			if err != nil {
				writeError(c, http.StatusInternalServerError, err.Error())
				return
			}
			c.JSON(http.StatusOK, item)
		})

		adminGroup.POST("/share-gate-config", func(c *gin.Context) {
			var req model.ShareGateControlConfig
			if !bindJSON(c, &req) {
				return
			}
			if err := adminService.Repo().SaveShareGateControlConfig(
				c.Request.Context(),
				req.RequireShareForAIReport,
				req.RequireShareForCollegeMajor,
				req.RequireShareForRecommendResult,
				req.RequireShareForPlanCompare,
			); err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminOK(c)
		})

		adminGroup.POST("/vip-products/:id/toggle", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				writeError(c, http.StatusBadRequest, "invalid id")
				return
			}
			var req model.AdminEnabledRequest
			if !bindJSON(c, &req) {
				return
			}
			if err := adminService.Repo().SetVIPProductEnabled(c.Request.Context(), id, req.Enabled); err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminOK(c)
		})

		adminGroup.DELETE("/vip-products/:id", func(c *gin.Context) {
			id := parseInt(c.Param("id"), 0)
			if id <= 0 {
				writeError(c, http.StatusBadRequest, "invalid id")
				return
			}
			if err := adminService.Repo().DeleteVIPProduct(c.Request.Context(), id); err != nil {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeAdminOK(c)
		})
	}
}

func bindAdminQuery(c *gin.Context, req any) bool {
	return bindQuery(c, req)
}

func writeAdminList[T any](c *gin.Context, pager model.AdminPaginationRequest, items []T, total int) {
	page, limit := pager.Normalized()
	c.JSON(http.StatusOK, model.AdminListResponse[T]{Items: items, Total: total, Page: page, Limit: limit})
}

func writeAdminItems[T any](c *gin.Context, items []T) {
	c.JSON(http.StatusOK, model.AdminItemsResponse[T]{Items: items})
}

func writeAdminOK(c *gin.Context) {
	c.JSON(http.StatusOK, model.AdminOKResponse{OK: true})
}

func writeAdminID(c *gin.Context, id int) {
	c.JSON(http.StatusOK, model.AdminIDResponse{ID: id, OK: true})
}

func adminAuthRequired(adminService *service.AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := adminService.CurrentUser(c.Request.Context(), readAdminToken(c))
		if err != nil {
			logging.LogEvent("admin_auth", map[string]any{"status": "failed", "error": err.Error(), "path": c.Request.URL.Path})
			abortWithError(c, http.StatusUnauthorized, err.Error())
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
