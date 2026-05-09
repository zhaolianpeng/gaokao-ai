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
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Year == 0 {
			req.Year = 2025
		}
		if req.Province == "" {
			req.Province = "黑龙江"
		}

		resp, err := recommendService.Recommend(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	})

	r.POST("/api/analyze", func(c *gin.Context) {
		var req model.AIAnalyzeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		report, err := aiService.Analyze(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"report": report})
	})

	r.POST("/api/analyze-task", func(c *gin.Context) {
		var req model.AIAnalyzeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if taskService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "task service unavailable"})
			return
		}
		taskID, status, err := taskService.SubmitAnalyzeTask(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"taskId": taskID, "status": status, "title": "AI 志愿分析报告"})
	})

	r.GET("/api/analyze/task", func(c *gin.Context) {
		if taskService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "task service unavailable"})
			return
		}
		result, err := taskService.GetTaskStatus(c.Request.Context(), c.Query("taskId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/analyze/task", func(c *gin.Context) {
		if taskService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "task service unavailable"})
			return
		}
		var req struct {
			TaskID string `json:"taskId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := taskService.GetTaskStatus(c.Request.Context(), req.TaskID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/auth/wx-login", func(c *gin.Context) {
		var req model.WechatLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if authService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "auth service unavailable"})
			return
		}
		user, err := authService.Login(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, user)
	})

	r.POST("/api/auth/wx-profile", func(c *gin.Context) {
		var req model.WechatProfileUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if authService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "auth service unavailable"})
			return
		}
		user, err := authService.UpdateProfile(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": user})
	})

	r.POST("/api/auth/wx-avatar", func(c *gin.Context) {
		if authService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "auth service unavailable"})
			return
		}
		userID := strings.TrimSpace(c.PostForm("userId"))
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "userId required"})
			return
		}
		file, header, err := c.Request.FormFile("avatar")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "avatar file required"})
			return
		}
		defer file.Close()
		if header.Size <= 0 || header.Size > maxAvatarUploadSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "avatar file too large"})
			return
		}
		user, err := authService.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		avatarURL, err := saveAvatarFile(file, header, uploadDir, buildPublicBaseURL(c, publicBaseURL))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updatedUser, err := authService.UpdateProfile(c.Request.Context(), model.WechatProfileUpdateRequest{
			UserID:    user.ID,
			Phone:     user.Phone,
			Nickname:  user.Nickname,
			AvatarURL: avatarURL,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": updatedUser, "avatarUrl": avatarURL})
	})

	r.POST("/api/vip/pay", func(c *gin.Context) {
		var req model.WechatPayRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if payService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "pay service unavailable"})
			return
		}
		result, err := payService.CreatePayment(c.Request.Context(), req)
		if err != nil {
			statusCode := http.StatusBadRequest
			if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "missing") {
				statusCode = http.StatusInternalServerError
			}
			c.JSON(statusCode, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/vip/pay/confirm", func(c *gin.Context) {
		var req model.WechatPayConfirmRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if payService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "pay service unavailable"})
			return
		}
		c.JSON(http.StatusOK, payService.ConfirmPayment(c.Request.Context(), req))
	})

	r.POST("/api/vip/pay/notify", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "成功"})
	})

	r.POST("/api/about/feedback", func(c *gin.Context) {
		var req model.FeedbackSubmitRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if feedbackService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback service unavailable"})
			return
		}
		result, err := feedbackService.Submit(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/agent-recommend", func(c *gin.Context) {
		var req model.AgentRecommendRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if taskService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "task service unavailable"})
			return
		}
		taskID, status, err := taskService.SubmitAgentRecommend(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"taskId": taskID, "status": status, "title": "AI 智能体报考建议"})
	})

	r.GET("/api/agent-recommend/task", func(c *gin.Context) {
		if taskService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "task service unavailable"})
			return
		}
		result, err := taskService.GetTaskStatus(c.Request.Context(), c.Query("taskId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/agent-recommend/task", func(c *gin.Context) {
		if taskService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "task service unavailable"})
			return
		}
		var req struct {
			TaskID string `json:"taskId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := taskService.GetTaskStatus(c.Request.Context(), req.TaskID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.GET("/api/dashboard/overview", func(c *gin.Context) {
		year, _ := strconv.Atoi(c.DefaultQuery("year", "2025"))
		overview, err := explorerService.GetDashboardOverview(c.Request.Context(), c.Query("province"), year, c.Query("subject"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, overview)
	})

	r.GET("/api/province-lines", func(c *gin.Context) {
		year, _ := strconv.Atoi(c.DefaultQuery("year", "2025"))
		province := c.DefaultQuery("province", "黑龙江")
		subject := normalizeLookupSubject(year, c.Query("subject"))
		items, err := explorerService.GetProvinceScoreLines(c.Request.Context(), province, year, subject)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	})

	r.POST("/api/province-lines", func(c *gin.Context) {
		var req struct {
			Province string `json:"province"`
			Year     int    `json:"year"`
			Subject  string `json:"subject"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Year == 0 {
			req.Year = 2025
		}
		province := strings.TrimSpace(req.Province)
		if province == "" {
			province = "黑龙江"
		}
		subject := normalizeLookupSubject(req.Year, req.Subject)
		items, err := explorerService.GetProvinceScoreLines(c.Request.Context(), province, req.Year, subject)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	})

	r.GET("/api/score-rank", func(c *gin.Context) {
		year, _ := strconv.Atoi(c.DefaultQuery("year", "2025"))
		score, _ := strconv.Atoi(c.DefaultQuery("score", "0"))
		if score <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid score"})
			return
		}
		province := c.DefaultQuery("province", "黑龙江")
		subject := normalizeLookupSubject(year, c.Query("subject"))
		if subject == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject"})
			return
		}
		result, err := explorerService.LookupScoreRank(c.Request.Context(), province, year, subject, score)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/score-rank", func(c *gin.Context) {
		var req struct {
			Province string `json:"province"`
			Year     int    `json:"year"`
			Subject  string `json:"subject"`
			Score    int    `json:"score"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Year == 0 {
			req.Year = 2025
		}
		if req.Score <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid score"})
			return
		}
		province := strings.TrimSpace(req.Province)
		if province == "" {
			province = "黑龙江"
		}
		subject := normalizeLookupSubject(req.Year, req.Subject)
		if subject == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject"})
			return
		}
		result, err := explorerService.LookupScoreRank(c.Request.Context(), province, req.Year, subject, req.Score)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/rank-score", func(c *gin.Context) {
		var req struct {
			Province string `json:"province"`
			Year     int    `json:"year"`
			Subject  string `json:"subject"`
			Rank     int    `json:"rank"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Year == 0 {
			req.Year = 2025
		}
		if req.Rank <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rank"})
			return
		}
		province := strings.TrimSpace(req.Province)
		if province == "" {
			province = "黑龙江"
		}
		subject := normalizeLookupSubject(req.Year, req.Subject)
		if subject == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject"})
			return
		}
		result, err := explorerService.LookupRankScore(c.Request.Context(), province, req.Year, subject, req.Rank)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.GET("/api/colleges", func(c *gin.Context) {
		year, _ := strconv.Atoi(c.DefaultQuery("year", "2025"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		sortMode := strings.TrimSpace(c.DefaultQuery("sortMode", "tier"))
		result, err := explorerService.ListColleges(c.Request.Context(), model.CollegeListFilter{
			Province: c.Query("province"),
			Year:     year,
			Subject:  c.Query("subject"),
			Keyword:  c.Query("keyword"),
			SortMode: sortMode,
			Page:     page,
			Limit:    limit,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/colleges", func(c *gin.Context) {
		var req struct {
			Province string `json:"province"`
			Year     int    `json:"year"`
			Subject  string `json:"subject"`
			Keyword  string `json:"keyword"`
			SortMode string `json:"sortMode"`
			Page     int    `json:"page"`
			Limit    int    `json:"limit"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Year == 0 {
			req.Year = 2025
		}
		if req.Page <= 0 {
			req.Page = 1
		}
		if req.Limit <= 0 {
			req.Limit = 20
		}
		if strings.TrimSpace(req.SortMode) == "" {
			req.SortMode = "tier"
		}
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.GET("/api/colleges/:id", func(c *gin.Context) {
		collegeID, err := strconv.Atoi(c.Param("id"))
		if err != nil || collegeID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid college id"})
			return
		}
		year, _ := strconv.Atoi(c.DefaultQuery("year", "2025"))
		detail, err := explorerService.GetCollegeDetail(c.Request.Context(), collegeID, c.Query("province"), year, c.Query("subject"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, detail)
	})

	r.POST("/api/colleges/:id", func(c *gin.Context) {
		collegeID, err := strconv.Atoi(c.Param("id"))
		if err != nil || collegeID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid college id"})
			return
		}
		var req struct {
			Province string `json:"province"`
			Year     int    `json:"year"`
			Subject  string `json:"subject"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Year == 0 {
			req.Year = 2025
		}
		detail, err := explorerService.GetCollegeDetail(c.Request.Context(), collegeID, req.Province, req.Year, req.Subject)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
