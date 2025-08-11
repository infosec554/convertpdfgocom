package main

import (
	"context"
	"fmt"
	"os"

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

	// Postgresga ulanish
	pgStore, err := postgres.New(context.Background(), cfg, log, nil)
	if err != nil {
		log.Error("error while connecting to db", logger.Error(err))
		return
	}
	defer pgStore.Close()

	// Servislar
	mailService := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPSenderName)
	redisStore := redis.New(cfg)
	gotClient := gotenberg.New(cfg.GotenbergURL)

	services := service.New(pgStore, log, mailService, redisStore, gotClient, cfg.Google)

	// Railway uchun PORT ni olish
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // localda ishlatish uchun
	}

	// Serverni ishga tushirish
	server := api.New(services, log)
	log.Info("Service is running on", logger.String("host", "0.0.0.0"), logger.String("port", port))

	if err = server.Run(fmt.Sprintf("0.0.0.0:%s", port)); err != nil {
		panic(err)
	}
}
