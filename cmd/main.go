package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"convertpdfgo/api"
	"convertpdfgo/bot"
	"convertpdfgo/config"
	"convertpdfgo/pkg/gotenberg"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/service"
	"convertpdfgo/storage/postgres"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.ServiceName)

	pgStore, err := postgres.New(context.Background(), cfg, log)
	if err != nil {
		log.Error("error while connecting to db", logger.Error(err))
		return
	}
	defer pgStore.Close()

	gotClient := gotenberg.New(cfg.GotenbergURL)
	services := service.New(pgStore, log, gotClient)

	// Cron Service
	cronService := bot.NewCronService(services, log)
	cronService.Start()
	defer cronService.Stop()

	// HTTP Server
	server := api.New(cfg, log, services)
	go func() {
		if err := server.Run(); err != nil {
			log.Error("failed to start http server", logger.Error(err))
		}
	}()
	log.Info("HTTP server started", logger.String("port", cfg.AppPort))

	if cfg.BotToken == "" {
		log.Error("BOT_TOKEN is not set")
		return
	}

	telegramBot, err := bot.New(cfg.BotToken, services, log, cfg.AdminUserID, cfg.RequiredChannel)
	if err != nil {
		log.Error("failed to create telegram bot", logger.Error(err))
		return
	}

	// CronService'ga Bot API ni ulash (notification yuborish uchun)
	cronService.SetBotAPI(telegramBot.GetAPI())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go telegramBot.Start(ctx)
	log.Info("Telegram bot started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down...")
}
