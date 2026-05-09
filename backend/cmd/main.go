package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"gaokao-ai/backend/api"
	"gaokao-ai/backend/config"
	"gaokao-ai/backend/repository"
	"gaokao-ai/backend/service"
)

func main() {
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	db, err := repository.NewDB(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		log.Fatalf("connect database failed: %v", err)
	}
	defer db.Close()

	collegeRepo := repository.NewCollegeRepository(db)
	expertRepo := repository.NewExpertRepository(db)
	authRepo := repository.NewAuthRepository(db)
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
	var payService *service.PayService
	payService, err = service.NewPayService(cfg.WeChatAppID, cfg.WeChatPayMchID, cfg.WeChatPayCertSerial, cfg.WeChatPayPrivateKeyPath, cfg.WeChatPayNotifyURL, authRepo)
	if err != nil {
		log.Printf("init pay service failed: %v", err)
	}

	router := api.NewRouter(recommendService, aiService, explorerService, authService, payService, taskService, feedbackService, cfg.TrustedProxies, cfg.UploadDir, cfg.PublicBaseURL)

	if err := router.Run(cfg.ServerAddr); err != nil {
		log.Fatalf("run server failed: %v", err)
	}
}
