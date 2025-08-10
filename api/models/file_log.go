package models

import "time"

type FileDeletionLog struct {
	ID        string    `json:"id"`
	FileID    string    `json:"file_id"`
	UserID    *string   `json:"user_id,omitempty"`
	DeletedBy string    `json:"deleted_by"`
	DeletedAt time.Time `json:"deleted_at"`
	Reason    *string   `json:"reason,omitempty"`
}
