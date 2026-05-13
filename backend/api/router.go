package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gaokao-ai/backend/logging"
	"gaokao-ai/backend/model"
	"gaokao-ai/backend/service"
)

const maxAvatarUploadSize = 5 << 20

func NewRouter(recommendService *service.RecommendService, aiService *service.AIService, explorerService *service.ExplorerService, authService *service.AuthService, payService *service.PayService, taskService *service.TaskService, feedbackService *service.FeedbackService, adminService *service.AdminService, trustedProxies []string, uploadDir, publicBaseURL string, logBodyLimitBytes int) *gin.Engine {
	r := gin.New()
	r.Use(logging.RequestResponseLogger(logBodyLimitBytes), gin.RecoveryWithWriter(gin.DefaultErrorWriter))
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "set trusted proxies failed: %v\n", err)
	}
	uploadDir = strings.TrimSpace(uploadDir)
	if uploadDir == "" {
		uploadDir = "uploads"
	}
	r.Static("/uploads", uploadDir)
	registerAdminRoutes(r, adminService, payService)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.HEAD("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	r.POST("/api/recommend", func(c *gin.Context) {
		var req model.RecommendRequest
		if !bindJSON(c, &req) {
			return
		}
		if req.Year == 0 {
			req.Year = 2025
		}
		if req.Province == "" {
			req.Province = "黑龙江"
		}
		if err := req.Validate(); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}

		resp, err := recommendService.Recommend(c.Request.Context(), req)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, resp)
	})

	r.POST("/api/analyze", func(c *gin.Context) {
		var req model.AIAnalyzeRequest
		if !bindJSON(c, &req) {
			return
		}
		if err := req.Validate(); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}

		report, err := aiService.Analyze(c.Request.Context(), req)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"report": report})
	})

	r.POST("/api/analyze-task", func(c *gin.Context) {
		var req model.AIAnalyzeRequest
		if !bindJSON(c, &req) {
			return
		}
		if err := req.Validate(); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		if taskService == nil {
			writeError(c, http.StatusServiceUnavailable, "task service unavailable")
			return
		}
		taskID, status, err := taskService.SubmitAnalyzeTask(c.Request.Context(), req)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"taskId": taskID, "status": status, "title": "AI 志愿分析报告"})
	})

	r.GET("/api/analyze/task", func(c *gin.Context) {
		var req model.TaskStatusRequest
		if !bindQuery(c, &req) {
			return
		}
		if taskService == nil {
			writeError(c, http.StatusServiceUnavailable, "task service unavailable")
			return
		}
		result, err := taskService.GetTaskStatus(c.Request.Context(), req.TaskID)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/analyze/task", func(c *gin.Context) {
		if taskService == nil {
			writeError(c, http.StatusServiceUnavailable, "task service unavailable")
			return
		}
		var req model.TaskStatusRequest
		if !bindJSON(c, &req) {
			return
		}
		result, err := taskService.GetTaskStatus(c.Request.Context(), req.TaskID)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/auth/wx-login", func(c *gin.Context) {
		var req model.WechatLoginRequest
		if !bindJSON(c, &req) {
			return
		}
		if authService == nil {
			writeError(c, http.StatusServiceUnavailable, "auth service unavailable")
			return
		}
		user, err := authService.Login(c.Request.Context(), req)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, user)
	})

	r.POST("/api/auth/wx-profile", func(c *gin.Context) {
		var req model.WechatProfileUpdateRequest
		if !bindJSON(c, &req) {
			return
		}
		if authService == nil {
			writeError(c, http.StatusServiceUnavailable, "auth service unavailable")
			return
		}
		user, err := authService.UpdateProfile(c.Request.Context(), req)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": user})
	})

	loadProfileOptions := func(c *gin.Context) {
		if adminService == nil {
			writeError(c, http.StatusServiceUnavailable, "admin service unavailable")
			return
		}
		items, err := adminService.Repo().ListEnabledProfileOptions(c.Request.Context())
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, items)
	}
	r.GET("/api/profile-options", loadProfileOptions)
	r.POST("/api/profile-options", loadProfileOptions)

	r.POST("/api/auth/wx-avatar", func(c *gin.Context) {
		if authService == nil {
			writeError(c, http.StatusServiceUnavailable, "auth service unavailable")
			return
		}
		userID := strings.TrimSpace(c.PostForm("userId"))
		if userID == "" {
			writeError(c, http.StatusBadRequest, "userId required")
			return
		}
		file, header, err := c.Request.FormFile("avatar")
		if err != nil {
			writeError(c, http.StatusBadRequest, "avatar file required")
			return
		}
		defer file.Close()
		if header.Size <= 0 || header.Size > maxAvatarUploadSize {
			writeError(c, http.StatusBadRequest, "avatar file too large")
			return
		}
		user, err := authService.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		avatarURL, err := saveAvatarFile(file, header, uploadDir, buildPublicBaseURL(c, publicBaseURL))
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		updatedUser, err := authService.UpdateProfile(c.Request.Context(), model.WechatProfileUpdateRequest{
			UserID:    user.ID,
			Phone:     &user.Phone,
			Nickname:  user.Nickname,
			AvatarURL: &avatarURL,
		})
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": updatedUser, "avatarUrl": avatarURL})
	})

	r.POST("/api/vip/pay", func(c *gin.Context) {
		var req model.WechatPayRequest
		if !bindJSON(c, &req) {
			return
		}
		if payService == nil {
			writeError(c, http.StatusServiceUnavailable, "pay service unavailable")
			return
		}
		result, err := payService.CreatePayment(c.Request.Context(), req)
		if err != nil {
			statusCode := http.StatusBadRequest
			if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "missing") {
				statusCode = http.StatusInternalServerError
			}
			writeError(c, statusCode, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/vip/products", func(c *gin.Context) {
		if adminService == nil {
			writeError(c, http.StatusServiceUnavailable, "admin service unavailable")
			return
		}
		items, err := adminService.Repo().ListVIPProducts(c.Request.Context())
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		visibleItems := make([]model.VIPProductConfig, 0, len(items))
		for _, item := range items {
			if !item.Enabled {
				continue
			}
			visibleItems = append(visibleItems, item)
		}
		c.JSON(http.StatusOK, visibleItems)
	})

	r.POST("/api/vip/membership", func(c *gin.Context) {
		var req model.WechatVIPMembershipRequest
		if !bindJSON(c, &req) {
			return
		}
		if payService == nil {
			writeError(c, http.StatusServiceUnavailable, "pay service unavailable")
			return
		}
		status, err := payService.GetMembership(c.Request.Context(), req.UserID)
		if err != nil {
			statusCode := http.StatusBadRequest
			if !strings.Contains(err.Error(), "invalid") {
				statusCode = http.StatusInternalServerError
			}
			writeError(c, statusCode, err.Error())
			return
		}
		c.JSON(http.StatusOK, status)
	})

	r.GET("/api/vip/entry-config", func(c *gin.Context) {
		if adminService == nil {
			writeError(c, http.StatusServiceUnavailable, "admin service unavailable")
			return
		}
		showVIPEntry, err := adminService.ShouldShowVIPEntry(c.Request.Context())
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, model.VIPEntryConfigResponse{ShowVIPEntry: showVIPEntry})
	})

	r.POST("/api/vip/entry-config", func(c *gin.Context) {
		if adminService == nil {
			writeError(c, http.StatusServiceUnavailable, "admin service unavailable")
			return
		}
		showVIPEntry, err := adminService.ShouldShowVIPEntry(c.Request.Context())
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, model.VIPEntryConfigResponse{ShowVIPEntry: showVIPEntry})
	})

	r.GET("/api/share-gate-config", func(c *gin.Context) {
		if adminService == nil {
			writeError(c, http.StatusServiceUnavailable, "admin service unavailable")
			return
		}
		config, err := adminService.ShareGateConfig(c.Request.Context())
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, config)
	})

	r.POST("/api/share-gate-config", func(c *gin.Context) {
		if adminService == nil {
			writeError(c, http.StatusServiceUnavailable, "admin service unavailable")
			return
		}
		config, err := adminService.ShareGateConfig(c.Request.Context())
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, config)
	})

	r.POST("/api/vip/pay/confirm", func(c *gin.Context) {
		var req model.WechatPayConfirmRequest
		if !bindJSON(c, &req) {
			return
		}
		if payService == nil {
			writeError(c, http.StatusServiceUnavailable, "pay service unavailable")
			return
		}
		result, err := payService.ConfirmPayment(c.Request.Context(), req)
		if err != nil {
			statusCode := http.StatusBadRequest
			if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "mismatch") && !strings.Contains(err.Error(), "超时关闭") {
				statusCode = http.StatusInternalServerError
			}
			writeError(c, statusCode, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/vip/pay/notify", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "成功"})
	})

	r.POST("/api/about/feedback", func(c *gin.Context) {
		var req model.FeedbackSubmitRequest
		if !bindJSON(c, &req) {
			return
		}
		if feedbackService == nil {
			writeError(c, http.StatusServiceUnavailable, "feedback service unavailable")
			return
		}
		result, err := feedbackService.Submit(c.Request.Context(), req)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/agent-recommend", func(c *gin.Context) {
		var req model.AgentRecommendRequest
		if !bindJSON(c, &req) {
			return
		}
		if err := req.Validate(); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		if taskService == nil {
			writeError(c, http.StatusServiceUnavailable, "task service unavailable")
			return
		}
		taskID, status, err := taskService.SubmitAgentRecommend(c.Request.Context(), req)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"taskId": taskID, "status": status, "title": "AI 智能体报考建议"})
	})

	r.GET("/api/agent-recommend/task", func(c *gin.Context) {
		var req model.TaskStatusRequest
		if !bindQuery(c, &req) {
			return
		}
		if taskService == nil {
			writeError(c, http.StatusServiceUnavailable, "task service unavailable")
			return
		}
		result, err := taskService.GetTaskStatus(c.Request.Context(), req.TaskID)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/agent-recommend/task", func(c *gin.Context) {
		if taskService == nil {
			writeError(c, http.StatusServiceUnavailable, "task service unavailable")
			return
		}
		var req model.TaskStatusRequest
		if !bindJSON(c, &req) {
			return
		}
		result, err := taskService.GetTaskStatus(c.Request.Context(), req.TaskID)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.GET("/api/dashboard/overview", func(c *gin.Context) {
		var req model.DashboardOverviewRequest
		if !bindQuery(c, &req) {
			return
		}
		req.Normalize()
		overview, err := explorerService.GetDashboardOverview(c.Request.Context(), req.Province, req.Year, req.Subject)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, overview)
	})

	r.GET("/api/province-lines", func(c *gin.Context) {
		var req model.ProvinceLinesRequest
		if !bindQuery(c, &req) {
			return
		}
		req.Normalize()
		subject := normalizeLookupSubject(req.Year, req.Subject)
		items, err := explorerService.GetProvinceScoreLines(c.Request.Context(), req.Province, req.Year, subject)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		writeItems(c, items)
	})

	r.POST("/api/province-lines", func(c *gin.Context) {
		var req model.ProvinceLinesRequest
		if !bindJSON(c, &req) {
			return
		}
		req.Normalize()
		subject := normalizeLookupSubject(req.Year, req.Subject)
		items, err := explorerService.GetProvinceScoreLines(c.Request.Context(), strings.TrimSpace(req.Province), req.Year, subject)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		writeItems(c, items)
	})

	r.GET("/api/score-rank", func(c *gin.Context) {
		var req model.ScoreRankRequest
		if !bindQuery(c, &req) {
			return
		}
		req.Normalize()
		if err := req.Validate(); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		subject := normalizeLookupSubject(req.Year, req.Subject)
		if subject == "" {
			writeError(c, http.StatusBadRequest, "invalid subject")
			return
		}
		result, err := explorerService.LookupScoreRank(c.Request.Context(), req.Province, req.Year, subject, req.Score)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/score-rank", func(c *gin.Context) {
		var req model.ScoreRankRequest
		if !bindJSON(c, &req) {
			return
		}
		req.Normalize()
		if err := req.Validate(); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		subject := normalizeLookupSubject(req.Year, req.Subject)
		if subject == "" {
			writeError(c, http.StatusBadRequest, "invalid subject")
			return
		}
		result, err := explorerService.LookupScoreRank(c.Request.Context(), req.Province, req.Year, subject, req.Score)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/rank-score", func(c *gin.Context) {
		var req model.RankScoreRequest
		if !bindJSON(c, &req) {
			return
		}
		req.Normalize()
		if err := req.Validate(); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		subject := normalizeLookupSubject(req.Year, req.Subject)
		if subject == "" {
			writeError(c, http.StatusBadRequest, "invalid subject")
			return
		}
		result, err := explorerService.LookupRankScore(c.Request.Context(), req.Province, req.Year, subject, req.Rank)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.GET("/api/colleges", func(c *gin.Context) {
		var req model.ExplorerCollegeListRequest
		if !bindQuery(c, &req) {
			return
		}
		req.Normalize()
		result, err := explorerService.ListColleges(c.Request.Context(), model.CollegeListFilter{
			Province: req.Province,
			Year:     req.Year,
			Subject:  req.Subject,
			Keyword:  req.Keyword,
			SortMode: strings.TrimSpace(req.SortMode),
			Page:     req.Page,
			Limit:    req.Limit,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/colleges", func(c *gin.Context) {
		var req model.ExplorerCollegeListRequest
		if !bindJSON(c, &req) {
			return
		}
		req.Normalize()
		result, err := explorerService.ListColleges(c.Request.Context(), model.CollegeListFilter{
			Province: req.Province,
			Year:     req.Year,
			Subject:  req.Subject,
			Keyword:  req.Keyword,
			SortMode: strings.TrimSpace(req.SortMode),
			Page:     req.Page,
			Limit:    req.Limit,
		})
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.GET("/api/colleges/:id", func(c *gin.Context) {
		collegeID, err := strconv.Atoi(c.Param("id"))
		if err != nil || collegeID <= 0 {
			writeError(c, http.StatusBadRequest, "invalid college id")
			return
		}
		var req model.CollegeDetailRequest
		if !bindQuery(c, &req) {
			return
		}
		req.Normalize()
		detail, err := explorerService.GetCollegeDetail(c.Request.Context(), collegeID, req.Province, req.Year, req.Subject)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, detail)
	})

	r.POST("/api/colleges/:id", func(c *gin.Context) {
		collegeID, err := strconv.Atoi(c.Param("id"))
		if err != nil || collegeID <= 0 {
			writeError(c, http.StatusBadRequest, "invalid college id")
			return
		}
		var req model.CollegeDetailRequest
		if !bindJSON(c, &req) {
			return
		}
		req.Normalize()
		detail, err := explorerService.GetCollegeDetail(c.Request.Context(), collegeID, req.Province, req.Year, req.Subject)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, detail)
	})

	return r
}

func buildPublicBaseURL(c *gin.Context, configuredBaseURL string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(configuredBaseURL), "/")
	if baseURL != "" {
		return baseURL
	}
	scheme := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	return strings.TrimRight(fmt.Sprintf("%s://%s", scheme, host), "/")
}

func saveAvatarFile(file multipart.File, header *multipart.FileHeader, uploadDir, publicBaseURL string) (string, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		return "", fmt.Errorf("unsupported avatar format")
	}
	randomPart := make([]byte, 16)
	if _, err := rand.Read(randomPart); err != nil {
		return "", fmt.Errorf("generate avatar filename failed: %w", err)
	}
	fileName := fmt.Sprintf("%d-%s%s", time.Now().Unix(), hex.EncodeToString(randomPart), ext)
	avatarDir := filepath.Join(uploadDir, "avatars")
	if err := os.MkdirAll(avatarDir, 0o755); err != nil {
		return "", fmt.Errorf("prepare avatar dir failed: %w", err)
	}
	filePath := filepath.Join(avatarDir, fileName)
	destination, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create avatar file failed: %w", err)
	}
	defer destination.Close()
	if _, err := io.Copy(destination, file); err != nil {
		return "", fmt.Errorf("save avatar file failed: %w", err)
	}
	return strings.TrimRight(publicBaseURL, "/") + "/uploads/avatars/" + fileName, nil
}

func normalizeLookupSubject(year int, subject string) string {
	subject = strings.TrimSpace(subject)
	if year > 0 && year <= 2023 {
		switch subject {
		case "历史", "文科", "历史类":
			return "文科"
		case "物理", "理科", "物理类":
			return "理科"
		}
	}
	switch subject {
	case "历史", "文科":
		return "历史类"
	case "物理", "理科":
		return "物理类"
	default:
		return subject
	}
}
