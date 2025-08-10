package models

import "time"

type File struct {
	ID         string    `db:"id"`
	UserID     *string   `db:"user_id"` // NULL bo‘lishi mumkin
	FileName   string    `db:"file_name"`
	FilePath   string    `db:"file_path"`
	FileType   string    `db:"file_type"`
	FileSize   int64     `db:"file_size"`
	UploadedAt time.Time `db:"uploaded_at"`
}

type FileUploadRequest struct {
	UserID   string
	FileName string
	FileData []byte
}
type OldFile struct {
	ID       string `json:"id"`
	FilePath string `json:"file_path"`
}
