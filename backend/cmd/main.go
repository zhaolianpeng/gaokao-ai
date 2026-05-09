package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"

	"gaokao-ai/backend/api"
	"gaokao-ai/backend/config"
	"gaokao-ai/backend/logging"
	"gaokao-ai/backend/repository"
	"gaokao-ai/backend/service"
)

func main() {
	cfg := config.Load()
	if _, err := logging.Setup(cfg.LogDir); err != nil {
		log.Fatalf("init logger failed: %v", err)
	}
	gin.SetMode(cfg.GinMode)

	db, err := repository.NewDB(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		log.Fatalf("connect database failed: %v", err)
	}
	defer db.Close()

	collegeRepo := repository.NewCollegeRepository(db)
	expertRepo := repository.NewExpertRepository(db)
	authRepo := repository.NewAuthRepository(db)
	adminRepo := repository.NewAdminRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	feedbackRepo, err := repository.NewFeedbackRepository(db, cfg.DBDriver)
	if err != nil {
		log.Fatalf("init feedback repository failed: %v", err)
	}
	cacheStore, err := service.NewResultCache(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.RedisTTL)
	if err != nil {
		log.Fatalf("connect redis failed: %v", err)
	}
	defer cacheStore.Close()

	recommendService := service.NewRecommendService(collegeRepo, cacheStore)
	aiService := service.NewAIService(cfg.DeepSeekAPIKey, cfg.DeepSeekBase, cfg.DeepSeekTimeout)
	explorerService := service.NewExplorerService(expertRepo, cacheStore)
	authService := service.NewAuthService(cfg.WeChatAppID, cfg.WeChatAppSecret, authRepo)
	taskService := service.NewTaskService(taskRepo, aiService)
	feedbackService := service.NewFeedbackService(feedbackRepo)
	adminService := service.NewAdminService(adminRepo)
	if err := adminService.EnsureBootstrap(context.Background()); err != nil {
		log.Fatalf("init admin bootstrap failed: %v", err)
	}
	var payService *service.PayService
	payService, err = service.NewPayService(cfg.WeChatAppID, cfg.WeChatPayMchID, cfg.WeChatPayCertSerial, cfg.WeChatPayPrivateKeyPath, cfg.WeChatPayNotifyURL, authRepo, adminRepo)
	if err != nil {
		log.Printf("init pay service failed: %v", err)
	}

	router := api.NewRouter(recommendService, aiService, explorerService, authService, payService, taskService, feedbackService, adminService, cfg.TrustedProxies, cfg.UploadDir, cfg.PublicBaseURL, cfg.LogBodyLimitBytes)

	if err := router.Run(cfg.ServerAddr); err != nil {
		log.Fatalf("run server failed: %v", err)
	}
}
