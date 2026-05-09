package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddr              string
	LogDir                  string
	LogBodyLimitBytes       int
	PublicBaseURL           string
	UploadDir               string
	DBDriver                string
	DBDSN                   string
	RedisAddr               string
	RedisPassword           string
	RedisDB                 int
	RedisTTL                time.Duration
	WeChatAppID             string
	WeChatAppSecret         string
	WeChatPayMchID          string
	WeChatPayAPIv3Key       string
	WeChatPayCertSerial     string
	WeChatPayPrivateKeyPath string
	WeChatPayNotifyURL      string
	PostgresDSN             string
	DeepSeekAPIKey          string
	DeepSeekBase            string
	DeepSeekTimeout         time.Duration
	GinMode                 string
	TrustedProxies          []string
}

func Load() Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Overload(".env.local")

	cfg := Config{
		ServerAddr:              getEnv("SERVER_ADDR", ":8080"),
		LogDir:                  getEnv("LOG_DIR", "/home/ubuntu/system_logs"),
		LogBodyLimitBytes:       getEnvIntAllowZero("LOG_BODY_LIMIT_BYTES", 1048576),
		PublicBaseURL:           strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")),
		UploadDir:               getEnv("UPLOAD_DIR", "uploads"),
		DBDriver:                getEnv("DB_DRIVER", "mysql"),
		DBDSN:                   getEnv("DB_DSN", getEnv("MYSQL_DSN", "gaokao_app:GaokaoApi_2026_Auth@tcp(127.0.0.1:3306)/gaokao?charset=utf8mb4&parseTime=true&loc=Local")),
		RedisAddr:               strings.TrimSpace(os.Getenv("REDIS_ADDR")),
		RedisPassword:           os.Getenv("REDIS_PASSWORD"),
		RedisDB:                 getEnvIntAllowZero("REDIS_DB", 0),
		RedisTTL:                time.Duration(getEnvInt("REDIS_TTL_SECONDS", 21600)) * time.Second,
		WeChatAppID:             strings.TrimSpace(os.Getenv("WECHAT_APP_ID")),
		WeChatAppSecret:         strings.TrimSpace(os.Getenv("WECHAT_APP_SECRET")),
		WeChatPayMchID:          strings.TrimSpace(os.Getenv("WECHAT_PAY_MCH_ID")),
		WeChatPayAPIv3Key:       strings.TrimSpace(os.Getenv("WECHAT_PAY_API_V3_KEY")),
		WeChatPayCertSerial:     strings.TrimSpace(os.Getenv("WECHAT_PAY_CERT_SERIAL")),
		WeChatPayPrivateKeyPath: strings.TrimSpace(os.Getenv("WECHAT_PAY_PRIVATE_KEY_PATH")),
		WeChatPayNotifyURL:      strings.TrimSpace(os.Getenv("WECHAT_PAY_NOTIFY_URL")),
		PostgresDSN:             getEnv("POSTGRES_DSN", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=gaokao sslmode=disable"),
		DeepSeekAPIKey:          os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekBase:            getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		DeepSeekTimeout:         time.Duration(getEnvInt("DEEPSEEK_TIMEOUT_SECONDS", 120)) * time.Second,
		GinMode:                 getEnv("GIN_MODE", "release"),
		TrustedProxies:          getEnvList("TRUSTED_PROXIES", []string{"127.0.0.1", "::1"}),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func getEnvIntAllowZero(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvList(key string, fallback []string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
