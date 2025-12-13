package models

import "time"

// UnlockJob - PDF parolini olib tashlash ishi
type UnlockJob struct {
	ID           string    `json:"id"`
	UserID       *string   `json:"user_id,omitempty"`
	InputFileID  string    `json:"input_file_id"`
	Password     string    `json:"password,omitempty"` // agar fayl parolli bo'lsa
	OutputFileID *string   `json:"output_file_id,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// UnlockRequest - Unlock so'rovi
type UnlockRequest struct {
	InputFileID string `json:"input_file_id" binding:"required"`
	Password    string `json:"password"` // optional
}
