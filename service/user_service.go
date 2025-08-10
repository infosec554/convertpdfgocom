package service

import (
	"context"
	"errors"
	"fmt"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/pkg/mailer"
	"convertpdfgo/pkg/security"
	"convertpdfgo/storage"
)

type UserService interface {
	Create(ctx context.Context, req models.SignupRequest) (string, error)
	GetForLoginByEmail(context.Context, string) (models.LoginUser, error)
	GetByID(ctx context.Context, id string) (*models.User, error)

	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
	GoogleAuth(ctx context.Context, email, name, googleID string) (string, error)
	GithubAuth(ctx context.Context, email, name, githubID string) (string, error)
	FacebookAuth(ctx context.Context, email, name, facebookID string) (string, error)
	SetRole(ctx context.Context, userID, role string) error
	PromoteToAdmin(ctx context.Context, userID string) error
	DemoteToUser(ctx context.Context, userID string) error
	UpdateAvatar(ctx context.Context, userID string, avatar string) error
	UpdateUserPreferences(ctx context.Context, userID, language string, notifications bool) error
	GetUserPreferences(ctx context.Context, userID string) (*models.UserPreferences, error)
	CreatePasswordResetToken(ctx context.Context, email string) (string, error)
	ValidatePasswordResetToken(ctx context.Context, token string) (string, error)
	ResetPassword(ctx context.Context, userID string, newPassword string) error
}

type userService struct {
	stg        storage.IUserStorage
	log        logger.ILogger
	mailerCore *mailer.Mailer
}

func NewUserService(stg storage.IStorage, log logger.ILogger, mailerCore *mailer.Mailer) UserService {
	return &userService{
		stg:        stg.User(),
		log:        log,
		mailerCore: mailerCore,
	}
}

func (s *userService) Create(ctx context.Context, req models.SignupRequest) (string, error) {
	s.log.Info("UserService.Create called", logger.String("email", req.Email))

	id, err := s.stg.Create(ctx, req)
	if err != nil {
		s.log.Error("failed to create user", logger.Error(err))
		return "", err
	}

	s.log.Info("user successfully created", logger.String("userID", id))
	return id, nil
}

func (s *userService) GetForLoginByEmail(ctx context.Context, email string) (models.LoginUser, error) {
	s.log.Info("UserService.GetForLoginByEmail called", logger.String("email", email))

	user, err := s.stg.GetForLoginByEmail(ctx, email)
	if err != nil {
		s.log.Error("failed to get user for login", logger.Error(err))
		return models.LoginUser{}, err
	}

	s.log.Info("user fetched for login", logger.String("userID", user.ID))
	return user, nil
}

func (s *userService) GetByID(ctx context.Context, id string) (*models.User, error) {
	s.log.Info("userService.GetByID called")
	return s.stg.GetByID(ctx, id)
}
func (s *userService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	hashedOldPassword, err := s.stg.GetPasswordByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := security.CompareHashAndPassword(hashedOldPassword, oldPassword); err != nil {
		return errors.New("old password is incorrect")
	}

	hashedNewPassword, err := security.HashPassword(newPassword)
	if err != nil {
		return errors.New("failed to hash new password")
	}

	return s.stg.UpdatePassword(ctx, userID, hashedNewPassword)
}

func (s *userService) GoogleAuth(ctx context.Context, email, name, googleID string) (string, error) {
	user, err := s.stg.GetForLoginByEmail(ctx, email)
	if err == nil {
		return user.ID, nil
	}
	req := models.SignupRequest{
		Name:     name,
		Email:    email,
		Password: "",
	}
	return s.stg.Create(ctx, req)
}

func (s *userService) GithubAuth(ctx context.Context, email, name, githubID string) (string, error) {
	user, err := s.stg.GetForLoginByEmail(ctx, email)
	if err == nil {
		return user.ID, nil // Login
	}
	req := models.SignupRequest{
		Name:     name,
		Email:    email,
		Password: "",
	}
	return s.stg.Create(ctx, req)
}

func (s *userService) FacebookAuth(ctx context.Context, email, name, facebookID string) (string, error) {
	user, err := s.stg.GetForLoginByEmail(ctx, email)
	if err == nil {
		return user.ID, nil // Login
	}
	req := models.SignupRequest{
		Name:     name,
		Email:    email,
		Password: "", // Facebook login uchun password kerakmas
		// FacebookID: facebookID, // Agar modeli va storage-da field bo‘lsa
	}
	return s.stg.Create(ctx, req)
}

func (s *userService) SetRole(ctx context.Context, userID, role string) error {
	if role != "admin" && role != "user" {
		return fmt.Errorf("invalid role: %s", role)
	}
	return s.stg.UpdateRole(ctx, userID, role)
}

func (s *userService) PromoteToAdmin(ctx context.Context, userID string) error {
	return s.SetRole(ctx, userID, "admin")
}

func (s *userService) DemoteToUser(ctx context.Context, userID string) error {
	return s.SetRole(ctx, userID, "user")
}

func (s *userService) UpdateAvatar(ctx context.Context, userID string, avatar string) error {
	err := s.stg.UpdateAvatar(ctx, userID, avatar)
	if err != nil {
		s.log.Error("failed to update user avatar", logger.Error(err))
		return err
	}
	return nil
}
func (s *userService) UpdateUserPreferences(ctx context.Context, userID, language string, notifications bool) error {
	return s.stg.UpdateUserPreferences(ctx, userID, language, notifications)
}
func (s *userService) GetUserPreferences(ctx context.Context, userID string) (*models.UserPreferences, error) {
	return s.stg.GetUserPreferences(ctx, userID)
}

func (s *userService) CreatePasswordResetToken(ctx context.Context, email string) (string, error) {
	// Foydalanuvchini email orqali topish
	user, err := s.stg.GetForLoginByEmail(ctx, email)
	if err != nil {
		return "", errors.New("user not found")
	}

	// Token yaratish
	token, err := s.stg.CreatePasswordResetToken(ctx, user.ID)
	if err != nil {
		return "", err
	}

	subject := "Password Reset Request"
	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Password Reset Request</title>
		<style>
			body {
				font-family: Arial, sans-serif;
				background-color: #f4f4f4;
				margin: 0;
				padding: 0;
			}

			.container {
				width: 100%;
				max-width: 600px;
				margin: 0 auto;
				background-color: #ffffff;
				padding: 30px;
				border-radius: 8px;
				box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
			}

			h2 {
				color: #333;
				font-size: 24px;
				margin-bottom: 15px;
			}

			p {
				color: #666;
				font-size: 16px;
				line-height: 1.6;
			}

			.button {
				display: inline-block;
				background-color: #007bff;
				color: white;
				padding: 12px 20px;
				font-size: 16px;
				text-decoration: none;
				border-radius: 4px;
				font-weight: bold;
				margin-top: 20px;
				text-align: center;
			}

			.footer {
				margin-top: 30px;
				font-size: 14px;
				text-align: center;
				color: #aaa;
			}

			.footer a {
				color: #007bff;
				text-decoration: none;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<h2>Password Reset Request</h2>
			<p>Hi,</p>
			<p>You requested to reset your password. Please click the button below to reset your password:</p>
			<p>
				<a href="https://yourapp.com/reset-password?token=%s" class="button">Reset Password</a>
			</p>
			<p>If you didn't request this change, please ignore this email.</p>

			<div class="footer">
				<p>Best regards,</p>
				<p>YourApp Team</p>
			</div>
		</div>
	</body>
	</html>
`, token)

	// Email yuborish
	err = s.mailerCore.Send(email, subject, body)
	if err != nil {
		return "", errors.New("failed to send reset token")
	}

	return token, nil

	// Email yuborish
	err = s.mailerCore.Send(email, subject, body)
	if err != nil {
		return "", errors.New("failed to send reset token")
	}

	return token, nil
}

// Parolni tiklash uchun tokenni tasdiqlash
func (s *userService) ValidatePasswordResetToken(ctx context.Context, token string) (string, error) {
	userID, err := s.stg.ValidatePasswordResetToken(ctx, token)
	if err != nil {
		return "", err
	}
	return userID, nil
}

// Yangi parolni yangilash
func (s *userService) ResetPassword(ctx context.Context, userID string, newPassword string) error {
	// Yangi parolni hash qilish
	hashedPassword, err := security.HashPassword(newPassword)
	if err != nil {
		return errors.New("failed to hash new password")
	}

	// Parolni yangilash
	err = s.stg.UpdatePassword(ctx, userID, hashedPassword)
	if err != nil {
		return err
	}

	return nil
}
