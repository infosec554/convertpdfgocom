package models

import "time"

type JobSummary struct {
	ID           string     `json:"id"`
	JobType      string     `json:"job_type"`
	Status       string     `json:"status"`
	UserID       *string    `json:"user_id,omitempty"`
	OutputFileID *string    `json:"output_file_id,omitempty"` // split uchun NULL bo'lishi mumkin
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

type AdminJobFilter struct {
	Type   *string    `json:"type,omitempty"`   // merge|split|compress
	Status *string    `json:"status,omitempty"` // pending|processing|done|failed
	UserID *string    `json:"user_id,omitempty"`
	From   *time.Time `json:"from,omitempty"`   // created_at >= from
	To     *time.Time `json:"to,omitempty"`     // created_at <= to
	Search *string    `json:"search,omitempty"` // id prefix exact/like
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}
