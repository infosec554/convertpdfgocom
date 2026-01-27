package service

import (
	"context"
	"convertpdfgo/api/models"
	"errors"
	"testing"
	"time"
)

// MockBotUserStorage implements storage.IBotUserStorage for testing
type MockBotUserStorage struct {
	users           map[int64]*models.BotUser
	GetAllUsersFunc func(ctx context.Context) ([]models.BotUser, error)
}

func NewMockBotUserStorage() *MockBotUserStorage {
	return &MockBotUserStorage{
		users: make(map[int64]*models.BotUser),
	}
}

func (m *MockBotUserStorage) CreateOrUpdate(ctx context.Context, user models.BotUser) error {
	m.users[user.TelegramID] = &user
	return nil
}

func (m *MockBotUserStorage) GetByTelegramID(ctx context.Context, telegramID int64) (*models.BotUser, error) {
	if user, ok := m.users[telegramID]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *MockBotUserStorage) AddDailyBonus(ctx context.Context, amount int) error {
	for _, user := range m.users {
		user.Coins += amount
	}
	return nil
}

func (m *MockBotUserStorage) GetByUsername(ctx context.Context, username string) (*models.BotUser, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *MockBotUserStorage) DeductCoin(ctx context.Context, telegramID int64) error {
	if user, ok := m.users[telegramID]; ok {
		if user.Coins > 0 {
			user.Coins--
			user.TotalUsed++
			return nil
		}
		return errors.New("insufficient funds")
	}
	return errors.New("user not found")
}

func (m *MockBotUserStorage) AddCoins(ctx context.Context, telegramID int64, amount int) error {
	if user, ok := m.users[telegramID]; ok {
		user.Coins += amount
		return nil
	}
	return errors.New("user not found")
}

func (m *MockBotUserStorage) AddReferral(ctx context.Context, referrerID, referredID int64) error {
	return nil
}

func (m *MockBotUserStorage) GetReferralCount(ctx context.Context, telegramID int64) (int, error) {
	return 0, nil
}

func (m *MockBotUserStorage) SetLanguage(ctx context.Context, telegramID int64, lang string) error {
	if user, ok := m.users[telegramID]; ok {
		user.Language = lang
		return nil
	}
	return errors.New("user not found")
}

func (m *MockBotUserStorage) SetCurrentAction(ctx context.Context, telegramID int64, action string) error {
	return nil
}

func (m *MockBotUserStorage) GetTopUsers(ctx context.Context, limit int) ([]models.BotUser, error) {
	var users []models.BotUser
	for _, u := range m.users {
		users = append(users, *u)
	}
	return users, nil
}

func (m *MockBotUserStorage) GetAllUsers(ctx context.Context) ([]models.BotUser, error) {
	if m.GetAllUsersFunc != nil {
		return m.GetAllUsersFunc(ctx)
	}
	var users []models.BotUser
	for _, u := range m.users {
		users = append(users, *u)
	}
	return users, nil
}

func (m *MockBotUserStorage) SetPremium(ctx context.Context, telegramID int64, until time.Time) error {
	return nil
}

func (m *MockBotUserStorage) IncrementUsage(ctx context.Context, telegramID int64) error {
	if user, ok := m.users[telegramID]; ok {
		user.TotalUsed++
		return nil
	}
	return errors.New("user not found")
}

// Tests

func TestBotUserService_CoinDeduction(t *testing.T) {
	t.Run("DeductCoinSuccess", func(t *testing.T) {
		storage := NewMockBotUserStorage()
		storage.users[123] = &models.BotUser{
			TelegramID: 123,
			Coins:      10,
			TotalUsed:  0,
		}

		err := storage.DeductCoin(context.Background(), 123)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		user := storage.users[123]
		if user.Coins != 9 {
			t.Errorf("expected coins 9, got %d", user.Coins)
		}
		if user.TotalUsed != 1 {
			t.Errorf("expected total_used 1, got %d", user.TotalUsed)
		}
	})

	t.Run("DeductCoinInsufficientFunds", func(t *testing.T) {
		storage := NewMockBotUserStorage()
		storage.users[123] = &models.BotUser{
			TelegramID: 123,
			Coins:      0,
		}

		err := storage.DeductCoin(context.Background(), 123)
		if err == nil {
			t.Error("expected error for insufficient funds")
		}
	})
}

func TestBotUserService_AddCoins(t *testing.T) {
	t.Run("AddCoinsSuccess", func(t *testing.T) {
		storage := NewMockBotUserStorage()
		storage.users[123] = &models.BotUser{
			TelegramID: 123,
			Coins:      5,
		}

		err := storage.AddCoins(context.Background(), 123, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		user := storage.users[123]
		if user.Coins != 15 {
			t.Errorf("expected coins 15, got %d", user.Coins)
		}
	})
}

func TestBotUserService_DailyBonus(t *testing.T) {
	t.Run("DailyBonusToAllUsers", func(t *testing.T) {
		storage := NewMockBotUserStorage()
		storage.users[1] = &models.BotUser{TelegramID: 1, Coins: 5}
		storage.users[2] = &models.BotUser{TelegramID: 2, Coins: 10}
		storage.users[3] = &models.BotUser{TelegramID: 3, Coins: 0}

		err := storage.AddDailyBonus(context.Background(), 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check all users got bonus
		if storage.users[1].Coins != 15 {
			t.Errorf("user 1: expected 15 coins, got %d", storage.users[1].Coins)
		}
		if storage.users[2].Coins != 20 {
			t.Errorf("user 2: expected 20 coins, got %d", storage.users[2].Coins)
		}
		if storage.users[3].Coins != 10 {
			t.Errorf("user 3: expected 10 coins, got %d", storage.users[3].Coins)
		}
	})
}

func TestBotUserService_GetAllUsers(t *testing.T) {
	t.Run("GetAllUsersSuccess", func(t *testing.T) {
		storage := NewMockBotUserStorage()
		storage.users[1] = &models.BotUser{TelegramID: 1, Username: "user1"}
		storage.users[2] = &models.BotUser{TelegramID: 2, Username: "user2"}

		users, err := storage.GetAllUsers(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(users) != 2 {
			t.Errorf("expected 2 users, got %d", len(users))
		}
	})
}

func TestBotUserService_SetLanguage(t *testing.T) {
	t.Run("SetLanguageSuccess", func(t *testing.T) {
		storage := NewMockBotUserStorage()
		storage.users[123] = &models.BotUser{TelegramID: 123, Language: "uz"}

		err := storage.SetLanguage(context.Background(), 123, "ru")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if storage.users[123].Language != "ru" {
			t.Errorf("expected language 'ru', got '%s'", storage.users[123].Language)
		}
	})
}
