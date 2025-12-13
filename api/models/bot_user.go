package models

import "time"

type BotUser struct {
	TelegramID    int64      `json:"telegram_id"`
	Username      string     `json:"username"`
	FirstName     string     `json:"first_name"`
	Coins         int        `json:"coins"`
	TotalUsed     int        `json:"total_used"`
	ReferredBy    *int64     `json:"referred_by"`
	Language      string     `json:"language"`
	CurrentAction string     `json:"current_action"`
	PremiumUntil  *time.Time `json:"premium_until"` // Premium tugash sanasi (NULL = premium yo'q)
	CreatedAt     time.Time  `json:"created_at"`
}

// IsPremium - foydalanuvchi premium ekanligini tekshirish
func (u *BotUser) IsPremium() bool {
	if u.PremiumUntil == nil {
		return false
	}
	return u.PremiumUntil.After(time.Now())
}

type Referral struct {
	ID         int       `json:"id"`
	ReferrerID int64     `json:"referrer_id"`
	ReferredID int64     `json:"referred_id"`
	CoinsGiven int       `json:"coins_given"`
	CreatedAt  time.Time `json:"created_at"`
}
