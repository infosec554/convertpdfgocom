package bot

import (
	"context"
	"fmt"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"

	"convertpdfgo/pkg/logger"
	"convertpdfgo/service"
)

type CronService struct {
	cron     *cron.Cron
	services service.IServiceManager
	log      logger.ILogger
	botAPI   *tgbotapi.BotAPI
}

func NewCronService(services service.IServiceManager, log logger.ILogger) *CronService {
	return &CronService{
		cron:     cron.New(),
		services: services,
		log:      log,
	}
}

func (c *CronService) SetBotAPI(bot *tgbotapi.BotAPI) {
	c.botAPI = bot
}

func (c *CronService) Start() {
	c.cron.AddFunc("*/5 * * * *", c.cleanupFiles)

	c.cron.AddFunc("0 * * * *", c.logStats)

	c.cron.AddFunc("0 0 * * 1", c.weeklyBonus)

	c.cron.Start()
	c.log.Info("Cron service started")
}

func (c *CronService) Stop() {
	c.cron.Stop()
	c.log.Info("Cron service stopped")
}

func (c *CronService) cleanupFiles() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	c.log.Info("Starting file cleanup job for files older than 5 minutes")

	files, err := c.services.File().ListPendingDeletionFiles(ctx, 5)
	if err != nil {
		c.log.Error("failed to get old files", logger.Error(err))
		return
	}

	c.log.Info("Found files to delete", logger.Int("count", len(files)))

	deletedCount := 0
	for _, file := range files {
		if err := os.Remove(file.FilePath); err != nil && !os.IsNotExist(err) {
			c.log.Error("failed to delete file from disk",
				logger.String("file_id", file.ID),
				logger.String("path", file.FilePath),
				logger.Error(err))
			continue
		}

		if err := c.services.File().Delete(ctx, file.ID); err != nil {
			c.log.Error("failed to delete file from db",
				logger.String("file_id", file.ID),
				logger.Error(err))
			continue
		}

		deletedCount++
	}

	c.log.Info("File cleanup completed",
		logger.Int("deleted", deletedCount),
		logger.Int("total", len(files)))
}

func (c *CronService) logStats() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stats, err := c.services.PublicStats().GetStats(ctx)
	if err != nil {
		c.log.Error("failed to get stats", logger.Error(err))
		return
	}

	c.log.Info("Hourly stats",
		logger.Int64("total_users", stats.TotalUsers),
		logger.Int64("total_jobs", stats.TotalUsage))
}

func (c *CronService) CleanupNow(ctx context.Context) (int, error) {
	files, err := c.services.File().ListPendingDeletionFiles(ctx, 0) // 0 minutes means all files
	if err != nil {
		return 0, err
	}

	deletedCount := 0
	for _, file := range files {
		if err := os.Remove(file.FilePath); err != nil && !os.IsNotExist(err) {
			continue
		}

		if err := c.services.File().Delete(ctx, file.ID); err != nil {
			continue
		}

		deletedCount++
	}

	return deletedCount, nil
}

func (c *CronService) weeklyBonus() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	amount := 10

	users, err := c.services.BotUser().GetAllUsers(ctx)
	if err != nil {
		c.log.Error("failed to get all users for daily bonus", logger.Error(err))
		return
	}

	if err := c.services.BotUser().GiveDailyBonus(ctx, amount); err != nil {
		c.log.Error("failed to give daily bonus", logger.Error(err))
		return
	}

	c.log.Info("Weekly bonus given", logger.Int("amount", amount), logger.Int("users_count", len(users)))

	if c.botAPI == nil {
		c.log.Warning("Bot API not set, skipping notifications")
		return
	}

	successCount := 0
	failCount := 0

	for _, user := range users {
		lang := user.Language
		if lang == "" {
			lang = "uz"
		}

		newBalance := user.Coins + amount

		text := fmt.Sprintf(T(lang, "daily_bonus"), amount, newBalance)

		msg := tgbotapi.NewMessage(user.TelegramID, text)
		msg.ParseMode = "Markdown"

		_, err := c.botAPI.Send(msg)
		if err != nil {
			failCount++
			continue
		}

		successCount++

		time.Sleep(50 * time.Millisecond)
	}

	c.log.Info("Weekly bonus notifications sent",
		logger.Int("success", successCount),
		logger.Int("failed", failCount),
		logger.Int("total", len(users)))
}
