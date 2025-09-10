package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cast"
)

type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type Config struct {
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	ServiceName string
	LoggerLevel string

	RedisHost      string
	RedisPort      string
	RedisPassword  string
	RedisTTL       time.Duration // ✅ YANGI QO‘SHILDI
	RedisDB        int           // ✅ Qo‘shildi
	SMTPHost       string
	SMTPPort       string
	SMTPUser       string
	SMTPPass       string
	SMTPSenderName string
	JWTSecretKey   string // ✅ YANGI QO‘SHILDI

	GotenbergURL     string
	AsynqConcurrency int

	BotToken string
	BotDebug bool

	Google OAuthProviderConfig
	Github   OAuthProviderConfig
	Facebook OAuthProviderConfig
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		fmt.Println("error!!!", err)
	}

	cfg := Config{}

	cfg.PostgresHost = cast.ToString(getOrReturnDefault("POSTGRES_HOST", "localhost"))
	cfg.PostgresPort = cast.ToString(getOrReturnDefault("POSTGRES_PORT", "5432"))
	cfg.PostgresUser = cast.ToString(getOrReturnDefault("POSTGRES_USER", "postgres"))
	cfg.PostgresPassword = cast.ToString(getOrReturnDefault("POSTGRES_PASSWORD", "1234"))
	cfg.PostgresDB = cast.ToString(getOrReturnDefault("POSTGRES_DB", "convertpdfgo"))

	cfg.ServiceName = cast.ToString(getOrReturnDefault("SERVICE_NAME", "convertpdfgo"))
	cfg.LoggerLevel = cast.ToString(getOrReturnDefault("LOGGER_LEVEL", "debug"))

	cfg.JWTSecretKey = cast.ToString(getOrReturnDefault("JWT_SECRET_KEY", "default_secret_key"))

	cfg.RedisHost = cast.ToString(getOrReturnDefault("REDIS_HOST", "localhost"))
	cfg.RedisPort = cast.ToString(getOrReturnDefault("REDIS_PORT", "6379"))
	cfg.RedisPassword = cast.ToString(getOrReturnDefault("REDIS_PASSWORD", "1234"))
	cfg.RedisTTL = cast.ToDuration(getOrReturnDefault("REDIS_TTL", "5m"))

	// 🔧 SMTP configlar qo‘shildi:
	cfg.SMTPHost = cast.ToString(getOrReturnDefault("SMTP_HOST", "smtp.gmail.com"))
	cfg.SMTPPort = cast.ToString(getOrReturnDefault("SMTP_PORT", "587"))
	cfg.SMTPUser = cast.ToString(getOrReturnDefault("SMTP_USER", "infosec557@gmail.com"))
	cfg.SMTPPass = cast.ToString(getOrReturnDefault("SMTP_PASS", "wwvn ehzs qsvs ojcf"))
	cfg.SMTPSenderName = cast.ToString(getOrReturnDefault("SMTP_SENDER_NAME", "pdfninja"))
	cfg.GotenbergURL = cast.ToString(getOrReturnDefault("GOTENBERG_URL", "http://localhost:3000"))

	// ✅ Telegram bot config
	cfg.BotToken = cast.ToString(getOrReturnDefault("BOT_TOKEN", ""))
	cfg.BotDebug = cast.ToBool(getOrReturnDefault("BOT_DEBUG", false))

	cfg.Google = OAuthProviderConfig{
		ClientID:     cast.ToString(getOrReturnDefault("GOOGLE_CLIENT_ID", "")),
		ClientSecret: cast.ToString(getOrReturnDefault("GOOGLE_CLIENT_SECRET", "")),
		RedirectURL:  cast.ToString(getOrReturnDefault("GOOGLE_REDIRECT_URL", "")),
	}
	cfg.Github = OAuthProviderConfig{
		ClientID:     cast.ToString(getOrReturnDefault("GITHUB_CLIENT_ID", "")),
		ClientSecret: cast.ToString(getOrReturnDefault("GITHUB_CLIENT_SECRET", "")),
		RedirectURL:  cast.ToString(getOrReturnDefault("GITHUB_REDIRECT_URL", "")),
	}
	cfg.Facebook = OAuthProviderConfig{
		ClientID:     cast.ToString(getOrReturnDefault("FACEBOOK_CLIENT_ID", "")),
		ClientSecret: cast.ToString(getOrReturnDefault("FACEBOOK_CLIENT_SECRET", "")),
		RedirectURL:  cast.ToString(getOrReturnDefault("FACEBOOK_REDIRECT_URL", "")),
	}

	return cfg
}

func getOrReturnDefault(key string, defaultValue interface{}) interface{} {
	value := os.Getenv(key)
	if value != "" {
		return value
	}

	return defaultValue
}
