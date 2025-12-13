package models

import "time"

// RotateJob - PDF aylantirish ishi
type RotateJob struct {
	ID           string    `json:"id"`
	UserID       *string   `json:"user_id,omitempty"`
	InputFileID  string    `json:"input_file_id"`
	Angle        int       `json:"angle"`         // 90, 180, 270
	Pages        string    `json:"pages"`         // "all", "1-3", "1,3,5"
	OutputFileID *string   `json:"output_file_id,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// RotateRequest - Rotate so'rovi
type RotateRequest struct {
	InputFileID string `json:"input_file_id" binding:"required"`
	Angle       int    `json:"angle" binding:"required"` // 90, 180, 270
	Pages       string `json:"pages"`                    // default: "all"
}
