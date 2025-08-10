package models

import "time"

type AdminFileFilter struct {
	UserID        *string    `form:"user_id"`        // ixtiyoriy
	IncludeGuests bool       `form:"include_guests"` // default false
	Q             *string    `form:"q"`              // file_name qidirish
	DateFrom      *time.Time `form:"from"`           // RFC3339
	DateTo        *time.Time `form:"to"`             // RFC3339
	Limit         int        `form:"limit"`          // default 20
	Offset        int        `form:"offset"`         // default 0
}

// Admin ro'yxati uchun user ma'lumoti ham kerak bo'lishi mumkin:
type FileRow struct {
	ID         string    `json:"id"`
	UserID     *string   `json:"user_id"`
	FileName   string    `json:"file_name"`
	FilePath   string    `json:"file_path"`
	FileType   string    `json:"file_type"`
	FileSize   int64     `json:"file_size"`
	UploadedAt time.Time `json:"uploaded_at"`
	UserEmail  *string   `json:"user_email,omitempty"`
}
