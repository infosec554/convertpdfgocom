package models

type JobFilesResult struct {
	UserID         *string  `json:"user_id"`
	SingleOutputID *string  `json:"single_output_id,omitempty"`
	MultiOutputIDs []string `json:"multi_output_ids,omitempty"`
	ZipFileID      *string  `json:"zip_file_id,omitempty"`
}
