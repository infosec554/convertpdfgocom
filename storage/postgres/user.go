package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/jwt"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type userRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewUserRepo(db *pgxpool.Pool, log logger.ILogger) storage.IUserStorage {
	return &userRepo{
		db:  db,
		log: log,
	}
}
func (r *userRepo) Create(ctx context.Context, req models.SignupRequest) (string, error) {
	id := uuid.New().String()
	query := `
		INSERT INTO users (id, name, email, password, status, role)
		VALUES ($1, $2, $3, $4, 'active', 'user')
	`
	_, err := r.db.Exec(ctx, query, id, req.Name, req.Email, req.Password)
	if err != nil {
		r.log.Error("error inserting user", logger.Error(err))
		return "", err
	}
	return id, nil
}
func (r *userRepo) GetForLoginByEmail(ctx context.Context, email string) (models.LoginUser, error) {
	var user models.LoginUser
	query := `
		SELECT id, password, status, role
		FROM users
		WHERE email = $1 AND status = 'active'
	`
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Password,
		&user.Status,
		&user.Role,
	)
	if err != nil {
		r.log.Error("failed to get user by email", logger.Error(err))
		return models.LoginUser{}, err
	}
	return user, nil
}
func (r *userRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, name, email, status, role, created_at
		FROM users
		WHERE id = $1 AND status = 'active'
	`
	var user models.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Status,
		&user.Role,
		&user.CreatedAt,
	)
	if err != nil {
		r.log.Error("failed to get user by ID", logger.Error(err))
		return nil, err
	}
	return &user, nil
}

// postgres/userRepo.go

func (r *userRepo) UpdatePassword(ctx context.Context, userID, newPassword string) error {
	query := `UPDATE users SET password = $1 WHERE id = $2 AND status = 'active'`
	_, err := r.db.Exec(ctx, query, newPassword, userID)
	if err != nil {
		r.log.Error("failed to update user password", logger.Error(err))
		return err
	}
	return nil
}

func (r *userRepo) GetPasswordByID(ctx context.Context, userID string) (string, error) {
	var hashedPassword string
	query := `SELECT password FROM users WHERE id = $1 AND status = 'active'`
	err := r.db.QueryRow(ctx, query, userID).Scan(&hashedPassword)
	if err != nil {
		r.log.Error("failed to get user password", logger.Error(err))
		return "", err
	}
	return hashedPassword, nil
}
func (r *userRepo) UpdateRole(ctx context.Context, userID, role string) error {
	query := `UPDATE users SET role = $1 WHERE id = $2 AND status = 'active'`
	_, err := r.db.Exec(ctx, query, role, userID)
	if err != nil {
		r.log.Error("failed to update user role", logger.Error(err))
		return err
	}
	return nil
}

func (r *userRepo) UpdateAvatar(ctx context.Context, userID string, avatar string) error {
	query := `UPDATE users SET avatar = $1, updated_at = NOW() WHERE id = $2 AND status = 'active'`
	_, err := r.db.Exec(ctx, query, avatar, userID)
	if err != nil {
		r.log.Error("failed to update user avatar", logger.Error(err))
		return err
	}
	return nil
}
func (r *userRepo) UpdateUserPreferences(ctx context.Context, userID, language string, notifications bool) error {
	query := `
        UPDATE user_preferences
        SET language = $1, notifications = $2, updated_at = NOW()
        WHERE user_id = $3
    `
	_, err := r.db.Exec(ctx, query, language, notifications, userID)
	return err
}

func (r *userRepo) GetUserPreferences(ctx context.Context, userID string) (*models.UserPreferences, error) {
	// Foydalanuvchi sozlamalarini olish uchun SQL so‘rovi
	query := `
		SELECT id, user_id, language, notifications, created_at, updated_at
		FROM user_preferences
		WHERE user_id = $1
	`
	var prefs models.UserPreferences
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&prefs.ID,
		&prefs.UserID,
		&prefs.Language,
		&prefs.Notifications,
		&prefs.CreatedAt,
		&prefs.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &prefs, nil
}

// Token yaratish va foydalanuvchiga yuborish
func (r *userRepo) CreatePasswordResetToken(ctx context.Context, userID string) (string, error) {
	// Token yaratish uchun JWT yordamida user_id va role ni yuboramiz
	token, err := jwt.GenerateAccessToken(userID, "user") // Yoki kerakli role'ni yuborish
	if err != nil {
		r.log.Error("failed to create password reset token", logger.Error(err))
		return "", err
	}

	// Tokenni saqlash (agar kerak bo'lsa)
	// Agar saqlashni istasangiz, saqlang. Bu tokenni database yoki Redisga saqlash mumkin.
	// Saqlash jarayonini faqat kerakli bo'lganda amalga oshirish kerak.

	// Masalan, tokenni saqlash (bu joy optional)
	query := `INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`
	_, err = r.db.Exec(ctx, query, userID, token, time.Now().Add(1*time.Hour)) // Tokenning amal qilish muddati
	if err != nil {
		r.log.Error("failed to save password reset token", logger.Error(err))
		return "", err
	}

	return token, nil
}

// Tokenni tasdiqlash (amal qilish vaqti bilan tekshirish)
func (r *userRepo) ValidatePasswordResetToken(ctx context.Context, token string) (string, error) {
	var userID string
	query := `
        SELECT user_id FROM password_reset_tokens
        WHERE token = $1 AND expires_at > NOW()
    `
	err := r.db.QueryRow(ctx, query, token).Scan(&userID)
	if err != nil {
		r.log.Error("failed to validate password reset token", logger.Error(err))
		return "", err
	}
	return userID, nil
}
