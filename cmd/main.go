package main

import (
	"context"

	"convertpdfgo/api"
	"convertpdfgo/config"
	"convertpdfgo/pkg/gotenberg" // gotenberg package'ni import qilish
	"convertpdfgo/pkg/logger"
	"convertpdfgo/pkg/mailer"
	"convertpdfgo/service"
	"convertpdfgo/storage/postgres"
	"convertpdfgo/storage/redis"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.ServiceName)
	pgStore, err := postgres.New(context.Background(), cfg, log, nil)
	if err != nil {
		log.Error("error while connecting to db", logger.Error(err))
		return
	}
	defer pgStore.Close()

	mailService := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPSenderName)
	redisStore := redis.New(cfg)
	gotClient := gotenberg.New(cfg.GotenbergURL)

	
	services := service.New(pgStore, log, mailService, redisStore, gotClient, cfg.Google)

	server := api.New(services, log)
	log.Info("Service is running on", logger.Int("port", 8080))
	if err = server.Run("localhost:8080"); err != nil {
		panic(err)
	}
}
