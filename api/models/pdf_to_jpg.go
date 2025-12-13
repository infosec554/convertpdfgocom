package models

import "time"

// PDFToJPGJob - PDF ni JPG ga aylantirish ishi
type PDFToJPGJob struct {
	ID            string    `json:"id"`
	UserID        *string   `json:"user_id,omitempty"`
	InputFileID   string    `json:"input_file_id"`
	OutputFileIDs []string  `json:"output_file_ids,omitempty"`
	ZipFileID     *string   `json:"zip_file_id,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// PDFToJPGRequest - PDF to JPG so'rovi
type PDFToJPGRequest struct {
	InputFileID string `json:"input_file_id" binding:"required"`
	Quality     int    `json:"quality"` // default: 90
}
