package models

import "time"

type AddWatermarkRequest struct {
	InputFileID string  `json:"input_file_id" binding:"required"`
	Text        string  `json:"text" binding:"required"`
	FontName    string  `json:"font_name" binding:"required"`              // e.g., Helvetica
	FontSize    int     `json:"font_size" binding:"required,min=8,max=72"` // e.g., 24
	Position    string  `json:"position" binding:"required"`               // e.g., "tl"
	Rotation    int     `json:"rotation" binding:"min=0,max=360"`          // e.g., 0
	Opacity     float64 `json:"opacity" binding:"gte=0,lte=1"`             // e.g., 0.6
	FillColor   string  `json:"fill_color" binding:"required"`             // e.g., "#FF0000"
	Pages       string  `json:"pages"`                                     // optional: "1-3"
}
type AddWatermarkJob struct {
	ID           string    `json:"id"`
	UserID       *string   `json:"user_id,omitempty"`
	InputFileID  string    `json:"input_file_id"`
	OutputFileID *string   `json:"output_file_id,omitempty"`
	Text         string    `json:"text"`
	FontName     string    `json:"font_name"`
	FontSize     int       `json:"font_size"`
	Position     string    `json:"position"`
	Rotation     int       `json:"rotation"`
	Opacity      float64   `json:"opacity"`
	FillColor    string    `json:"fill_color"`
	Pages        string    `json:"pages"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}
