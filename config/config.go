package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cast"
)

type Config struct {
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	ServiceName string
	LoggerLevel string

	GotenbergURL string

	BotToken        string
	AdminUserID     string
	RequiredChannel string // @channelname yoki channel ID
	AppHost         string
	AppPort         string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		fmt.Println("error loading .env file:", err)
	}

	cfg := &Config{}

	cfg.AppHost = cast.ToString(getOrReturnDefault("APP_HOST", "localhost"))
	cfg.AppPort = cast.ToString(getOrReturnDefault("APP_PORT", ":8080"))

	cfg.PostgresHost = cast.ToString(getOrReturnDefault("POSTGRES_HOST", "localhost"))
	cfg.PostgresPort = cast.ToString(getOrReturnDefault("POSTGRES_PORT", "5432"))
	cfg.PostgresUser = cast.ToString(getOrReturnDefault("POSTGRES_USER", "postgres"))
	cfg.PostgresPassword = cast.ToString(getOrReturnDefault("POSTGRES_PASSWORD", "1234"))
	cfg.PostgresDB = cast.ToString(getOrReturnDefault("POSTGRES_DB", "convertpdfgo"))

	cfg.ServiceName = cast.ToString(getOrReturnDefault("SERVICE_NAME", "convertpdfgo"))
	cfg.LoggerLevel = cast.ToString(getOrReturnDefault("LOGGER_LEVEL", "debug"))

	cfg.GotenbergURL = cast.ToString(getOrReturnDefault("GOTENBERG_URL", "http://localhost:3000"))

	cfg.BotToken = cast.ToString(getOrReturnDefault("BOT_TOKEN", "7605533369:AAFzCwG_gJtKpNQBG-iM9-h0PbRd9uqhDYw"))
	cfg.AdminUserID = cast.ToString(getOrReturnDefault("ADMIN_USER_ID", "7697210313"))
	cfg.RequiredChannel = cast.ToString(getOrReturnDefault("REQUIRED_CHANNEL", "@convertpdfgo"))

	return cfg
}

func getOrReturnDefault(key string, defaultValue any) any {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}
