package models

import "time"

// WatermarkJob - PDF ga suv belgisi qo'shish ishi
type WatermarkJob struct {
	ID           string    `json:"id"`
	UserID       *string   `json:"user_id,omitempty"`
	InputFileID  string    `json:"input_file_id"`
	Text         string    `json:"text"`
	FontSize     int       `json:"font_size"`
	Position     string    `json:"position"` // center, diagonal
	Opacity      float64   `json:"opacity"`
	OutputFileID *string   `json:"output_file_id,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// WatermarkRequest - Watermark so'rovi
type WatermarkRequest struct {
	InputFileID string  `json:"input_file_id" binding:"required"`
	Text        string  `json:"text" binding:"required"`
	FontSize    int     `json:"font_size"`  // default: 48
	Position    string  `json:"position"`   // default: "diagonal"
	Opacity     float64 `json:"opacity"`    // default: 0.3
}
