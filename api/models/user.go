package models

import (
	"time"
)

type User struct {
	ID            string    `json:"id"`               // Foydalanuvchi ID
	Name          string    `json:"name"`             // Foydalanuvchi ismi
	Email         string    `json:"email"`            // Foydalanuvchi emaili
	Status        string    `json:"status"`           // Foydalanuvchi holati (active, blocked)
	Role          string    `json:"role"`             // Foydalanuvchi roli (admin, user)
	Avatar        *string   `json:"avatar,omitempty"` // Foydalanuvchi profil rasmi URL (ixtiyoriy)
	Language      string    `json:"language"`         // Foydalanuvchi tanlagan til (default: 'en')
	Notifications bool      `json:"notifications"`    // Foydalanuvchi bildirishnomalarni olishni xohlayaptimi
	CreatedAt     time.Time `json:"created_at"`       // Ro‘yxatdan o‘tgan vaqt
	UpdatedAt     time.Time `json:"updated_at"`       // Profil yangilangan vaqti
}
type SignupRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginUser struct { // Faqat login uchun DBdan o‘qiladigan struct
	ID       string
	Password string
	Status   string
	Role     string
}

type LoginResponse struct {
	ID           string `json:"id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Role         string `json:"role"`
}

type SignupResponse struct {
	ID string `json:"id"` // Foydalanuvchi ID si
}

type UserPreferences struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Language      string    `json:"language"`
	Notifications bool      `json:"notifications"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
type UpdateAvatarRequest struct {
	Avatar string `json:"avatar" binding:"required"` // Yangi avatar URL
}

// Parolni tiklash uchun foydalanuvchi emaili
type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"` // Foydalanuvchi emaili
}

// Parolni tiklash uchun token va yangi parol
type ResetPasswordRequest struct {
	Token          string `json:"token" binding:"required"`           // Token
	NewPassword    string `json:"new_password" binding:"required"`    // Yangi parol
	RepeatPassword string `json:"repeat_password" binding:"required"` // Takrorlangan parol

}
