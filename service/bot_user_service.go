package service

import (
	"context"
	"time"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type BotUserService interface {
	EnsureUser(ctx context.Context, telegramID int64, username, firstName string, referrerID *int64) error
	CanUseService(ctx context.Context, telegramID int64) (bool, int, bool, error) // canUse, coins, isPremium, error
	UseService(ctx context.Context, telegramID int64) error
	AddCoins(ctx context.Context, telegramID int64, amount int) error
	AddCoinsByUsername(ctx context.Context, username string, amount int) (*models.BotUser, error)
	GetUserStats(ctx context.Context, telegramID int64) (*models.BotUser, int, error)
	ProcessReferral(ctx context.Context, referrerID, referredID int64) error
	SetLanguage(ctx context.Context, telegramID int64, lang string) error
	SetCurrentAction(ctx context.Context, telegramID int64, action string) error
	GetTopUsers(ctx context.Context, limit int) ([]models.BotUser, error)
	GetUser(ctx context.Context, telegramID int64) (*models.BotUser, error)
	GiveDailyBonus(ctx context.Context, amount int) error
	GetAllUsers(ctx context.Context) ([]models.BotUser, error)
	SetPremium(ctx context.Context, telegramID int64, months int) error
	IncrementUsage(ctx context.Context, telegramID int64) error
}

type botUserService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewBotUserService(stg storage.IStorage, log logger.ILogger) BotUserService {
	return &botUserService{stg: stg, log: log}
}

func (s *botUserService) EnsureUser(ctx context.Context, telegramID int64, username, firstName string, referrerID *int64) error {
	existing, err := s.stg.BotUser().GetByTelegramID(ctx, telegramID)
	if err != nil {
		return err
	}

	if existing != nil {
		// Faqat username va firstName yangilash
		existing.Username = username
		existing.FirstName = firstName
		return s.stg.BotUser().CreateOrUpdate(ctx, *existing)
	}

	// Yangi foydalanuvchi
	user := models.BotUser{
		TelegramID: telegramID,
		Username:   username,
		FirstName:  firstName,
		Coins:      100, // Boshlang'ich 30 ta tekin
		TotalUsed:  0,
		ReferredBy: referrerID,
		CreatedAt:  time.Now(),
	}

	if err := s.stg.BotUser().CreateOrUpdate(ctx, user); err != nil {
		return err
	}

	// Agar referal orqali kelgan bo'lsa
	if referrerID != nil && *referrerID != 0 {
		if err := s.ProcessReferral(ctx, *referrerID, telegramID); err != nil {
			s.log.Error("failed to process referral", logger.Error(err))
		}
	}

	return nil
}

func (s *botUserService) CanUseService(ctx context.Context, telegramID int64) (bool, int, bool, error) {
	user, err := s.stg.BotUser().GetByTelegramID(ctx, telegramID)
	if err != nil {
		return false, 0, false, err
	}
	if user == nil {
		return false, 0, false, nil
	}

	// Premium foydalanuvchi - cheksiz ishlatishi mumkin
	if user.IsPremium() {
		return true, user.Coins, true, nil
	}

	return user.Coins > 0, user.Coins, false, nil
}

func (s *botUserService) UseService(ctx context.Context, telegramID int64) error {
	// Premium foydalanuvchi bo'lsa coin ayirmaymiz
	user, err := s.stg.BotUser().GetByTelegramID(ctx, telegramID)
	if err != nil {
		return err
	}
	if user != nil && user.IsPremium() {
		// Faqat total_used ni oshiramiz
		return s.stg.BotUser().IncrementUsage(ctx, telegramID)
	}
	return s.stg.BotUser().DeductCoin(ctx, telegramID)
}

func (s *botUserService) AddCoins(ctx context.Context, telegramID int64, amount int) error {
	return s.stg.BotUser().AddCoins(ctx, telegramID, amount)
}

func (s *botUserService) AddCoinsByUsername(ctx context.Context, username string, amount int) (*models.BotUser, error) {
	user, err := s.stg.BotUser().GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	if err := s.stg.BotUser().AddCoins(ctx, user.TelegramID, amount); err != nil {
		return nil, err
	}

	// Yangilangan ma'lumotni olish
	return s.stg.BotUser().GetByTelegramID(ctx, user.TelegramID)
}

func (s *botUserService) GetUserStats(ctx context.Context, telegramID int64) (*models.BotUser, int, error) {
	user, err := s.stg.BotUser().GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, 0, err
	}
	if user == nil {
		return nil, 0, nil
	}

	referralCount, err := s.stg.BotUser().GetReferralCount(ctx, telegramID)
	if err != nil {
		referralCount = 0
	}

	return user, referralCount, nil
}

func (s *botUserService) ProcessReferral(ctx context.Context, referrerID, referredID int64) error {
	// Referal qo'shish
	if err := s.stg.BotUser().AddReferral(ctx, referrerID, referredID); err != nil {
		return err
	}

	// Referrer ga 5 coin qo'shish
	if err := s.stg.BotUser().AddCoins(ctx, referrerID, 5); err != nil {
		return err
	}

	s.log.Info("Referral processed",
		logger.Int64("referrer", referrerID),
		logger.Int64("referred", referredID))

	return nil
}

func (s *botUserService) SetLanguage(ctx context.Context, telegramID int64, lang string) error {
	return s.stg.BotUser().SetLanguage(ctx, telegramID, lang)
}

func (s *botUserService) SetCurrentAction(ctx context.Context, telegramID int64, action string) error {
	return s.stg.BotUser().SetCurrentAction(ctx, telegramID, action)
}

func (s *botUserService) GetTopUsers(ctx context.Context, limit int) ([]models.BotUser, error) {
	return s.stg.BotUser().GetTopUsers(ctx, limit)
}

func (s *botUserService) GetUser(ctx context.Context, telegramID int64) (*models.BotUser, error) {
	return s.stg.BotUser().GetByTelegramID(ctx, telegramID)
}

func (s *botUserService) GiveDailyBonus(ctx context.Context, amount int) error {
	return s.stg.BotUser().AddDailyBonus(ctx, amount)
}

func (s *botUserService) GetAllUsers(ctx context.Context) ([]models.BotUser, error) {
	return s.stg.BotUser().GetAllUsers(ctx)
}

func (s *botUserService) SetPremium(ctx context.Context, telegramID int64, months int) error {
	until := time.Now().AddDate(0, months, 0)
	return s.stg.BotUser().SetPremium(ctx, telegramID, until)
}

func (s *botUserService) IncrementUsage(ctx context.Context, telegramID int64) error {
	return s.stg.BotUser().IncrementUsage(ctx, telegramID)
}
