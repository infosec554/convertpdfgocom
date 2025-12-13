package models

type MergeJob struct {
	ID           string   `json:"id"`
	UserID       int64    `json:"user_id"`
	InputFileIDs []string `json:"input_file_ids"`
	OutputFileID *string  `json:"output_file_id"`
	Status       string   `json:"status"`
}
