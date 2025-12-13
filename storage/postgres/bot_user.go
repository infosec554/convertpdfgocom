package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type botUserRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewBotUserRepo(db *pgxpool.Pool, log logger.ILogger) storage.IBotUserStorage {
	return &botUserRepo{db: db, log: log}
}

func (r *botUserRepo) CreateOrUpdate(ctx context.Context, user models.BotUser) error {
	query := `
		INSERT INTO bot_users (telegram_id, username, first_name, coins, total_used, referred_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (telegram_id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name
	`

	_, err := r.db.Exec(ctx, query,
		user.TelegramID,
		user.Username,
		user.FirstName,
		user.Coins,
		user.TotalUsed,
		user.ReferredBy,
		user.CreatedAt,
	)
	if err != nil {
		r.log.Error("failed to create/update bot user", logger.Error(err))
		return err
	}
	return nil
}

func (r *botUserRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*models.BotUser, error) {
	query := `
		SELECT telegram_id, username, first_name, coins, total_used, referred_by,
		       COALESCE(language, 'uz') as language, premium_until, created_at
		FROM bot_users WHERE telegram_id = $1
	`

	var user models.BotUser
	var referredBy sql.NullInt64
	var username, firstName sql.NullString
	var premiumUntil sql.NullTime

	err := r.db.QueryRow(ctx, query, telegramID).Scan(
		&user.TelegramID,
		&username,
		&firstName,
		&user.Coins,
		&user.TotalUsed,
		&referredBy,
		&user.Language,
		&premiumUntil,
		&user.CreatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		r.log.Error("failed to get bot user", logger.Error(err))
		return nil, err
	}

	if username.Valid {
		user.Username = username.String
	}
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if referredBy.Valid {
		user.ReferredBy = &referredBy.Int64
	}
	if premiumUntil.Valid {
		user.PremiumUntil = &premiumUntil.Time
	}

	return &user, nil
}

func (r *botUserRepo) DeductCoin(ctx context.Context, telegramID int64) error {
	query := `
		UPDATE bot_users 
		SET coins = coins - 1, total_used = total_used + 1 
		WHERE telegram_id = $1 AND coins > 0
	`
	result, err := r.db.Exec(ctx, query, telegramID)
	if err != nil {
		r.log.Error("failed to deduct coin", logger.Error(err))
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("insufficient funds")
	}
	return nil
}

func (r *botUserRepo) AddCoins(ctx context.Context, telegramID int64, amount int) error {
	query := `UPDATE bot_users SET coins = coins + $1 WHERE telegram_id = $2`
	_, err := r.db.Exec(ctx, query, amount, telegramID)
	if err != nil {
		r.log.Error("failed to add coins", logger.Error(err))
		return err
	}
	return nil
}

func (r *botUserRepo) AddReferral(ctx context.Context, referrerID, referredID int64) error {
	query := `
		INSERT INTO referrals (referrer_id, referred_id, coins_given, created_at)
		VALUES ($1, $2, 10, $3)
		ON CONFLICT (referrer_id, referred_id) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, referrerID, referredID, time.Now())
	if err != nil {
		r.log.Error("failed to add referral", logger.Error(err))
		return err
	}
	return nil
}

func (r *botUserRepo) GetReferralCount(ctx context.Context, telegramID int64) (int, error) {
	query := `SELECT COUNT(*) FROM referrals WHERE referrer_id = $1`
	var count int
	err := r.db.QueryRow(ctx, query, telegramID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *botUserRepo) GetByUsername(ctx context.Context, username string) (*models.BotUser, error) {
	query := `
		SELECT telegram_id, username, first_name, coins, total_used, referred_by, created_at
		FROM bot_users WHERE LOWER(username) = LOWER($1)
	`

	var user models.BotUser
	var referredBy sql.NullInt64
	var uname, firstName sql.NullString

	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.TelegramID,
		&uname,
		&firstName,
		&user.Coins,
		&user.TotalUsed,
		&referredBy,
		&user.CreatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	if uname.Valid {
		user.Username = uname.String
	}
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if referredBy.Valid {
		user.ReferredBy = &referredBy.Int64
	}

	return &user, nil
}

func (r *botUserRepo) SetLanguage(ctx context.Context, telegramID int64, lang string) error {
	query := `UPDATE bot_users SET language = $1 WHERE telegram_id = $2`
	_, err := r.db.Exec(ctx, query, lang, telegramID)
	return err
}

func (r *botUserRepo) SetCurrentAction(ctx context.Context, telegramID int64, action string) error {
	query := `UPDATE bot_users SET current_action = $1 WHERE telegram_id = $2`
	_, err := r.db.Exec(ctx, query, action, telegramID)
	return err
}

func (r *botUserRepo) GetTopUsers(ctx context.Context, limit int) ([]models.BotUser, error) {
	query := `
		SELECT telegram_id, username, first_name, coins, total_used, referred_by, 
		       COALESCE(language, 'uz') as language, created_at
		FROM bot_users 
		ORDER BY total_used DESC 
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.BotUser
	for rows.Next() {
		var user models.BotUser
		var referredBy sql.NullInt64
		var username, firstName sql.NullString

		if err := rows.Scan(
			&user.TelegramID,
			&username,
			&firstName,
			&user.Coins,
			&user.TotalUsed,
			&referredBy,
			&user.Language,
			&user.CreatedAt,
		); err != nil {
			continue
		}

		if username.Valid {
			user.Username = username.String
		}
		if firstName.Valid {
			user.FirstName = firstName.String
		}
		if referredBy.Valid {
			user.ReferredBy = &referredBy.Int64
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *botUserRepo) AddDailyBonus(ctx context.Context, amount int) error {
	query := `UPDATE bot_users SET coins = coins + $1`
	_, err := r.db.Exec(ctx, query, amount)
	return err
}

func (r *botUserRepo) GetAllUsers(ctx context.Context) ([]models.BotUser, error) {
	query := `
		SELECT telegram_id, username, first_name, coins, total_used, referred_by,
		       COALESCE(language, 'uz') as language, created_at
		FROM bot_users
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.log.Error("failed to get all users", logger.Error(err))
		return nil, err
	}
	defer rows.Close()

	var users []models.BotUser
	for rows.Next() {
		var user models.BotUser
		var referredBy sql.NullInt64
		var username, firstName sql.NullString

		if err := rows.Scan(
			&user.TelegramID,
			&username,
			&firstName,
			&user.Coins,
			&user.TotalUsed,
			&referredBy,
			&user.Language,
			&user.CreatedAt,
		); err != nil {
			continue
		}

		if username.Valid {
			user.Username = username.String
		}
		if firstName.Valid {
			user.FirstName = firstName.String
		}
		if referredBy.Valid {
			user.ReferredBy = &referredBy.Int64
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *botUserRepo) SetPremium(ctx context.Context, telegramID int64, until time.Time) error {
	query := `UPDATE bot_users SET premium_until = $1 WHERE telegram_id = $2`
	_, err := r.db.Exec(ctx, query, until, telegramID)
	if err != nil {
		r.log.Error("failed to set premium", logger.Error(err))
		return err
	}
	return nil
}

func (r *botUserRepo) IncrementUsage(ctx context.Context, telegramID int64) error {
	query := `UPDATE bot_users SET total_used = total_used + 1 WHERE telegram_id = $1`
	_, err := r.db.Exec(ctx, query, telegramID)
	if err != nil {
		r.log.Error("failed to increment usage", logger.Error(err))
		return err
	}
	return nil
}
