package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)



type jobDownloadStorage struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewJobDownloadStorage(db *pgxpool.Pool, log logger.ILogger) storage.JobDownloadStorage {
	return &jobDownloadStorage{db: db, log: log}
}

// Ruxsat berilgan job turlari va ularning jadval nomlari
var allowedJobTables = map[string]string{
	"merge":            "merge_jobs",
	"split":            "split_jobs",
	"remove-pages":     "remove_pages_jobs",
	"extract":          "extract_jobs",
	"compress":         "compress_jobs",
	"jpg-to-pdf":       "jpg_to_pdf_jobs",
	"pdf-to-jpg":       "pdf_to_jpg_jobs",
	"pdf-to-word":      "pdf_to_word_jobs",
	"word-to-pdf":      "word_to_pdf_jobs",
	"excel-to-pdf":     "excel_to_pdf_jobs",
	"ppt-to-pdf":       "powerpoint_to_pdf_jobs",
	"rotate":           "rotate_jobs",
	"crop":             "crop_jobs",
	"add-page-numbers": "add_page_numbers_jobs",
	"unlock":           "unlock_jobs",
	"protect":          "protect_jobs",
}

// Universal function: job_type bo‘yicha fayllarni olish
func (s *jobDownloadStorage) GetJobFiles(ctx context.Context, jobType, jobID string) (models.JobFilesResult, error) {
	table, ok := allowedJobTables[jobType]
	if !ok {
		return models.JobFilesResult{}, errors.New("invalid job type")
	}

	var result models.JobFilesResult

	// Barcha joblar uchun umumiy qoidalar:
	// - Agar bitta chiqish bo‘lsa: output_file_id
	// - Agar ko‘p chiqish bo‘lsa: output_file_ids ARRAY + zip_file_id (bo‘lishi mumkin)
	query := fmt.Sprintf(`
		SELECT user_id,
			   COALESCE(output_file_id, '') AS single_output,
			   COALESCE(output_file_ids, '{}') AS multi_outputs,
			   COALESCE(zip_file_id, '') AS zip_output
		FROM %s
		WHERE id = $1
	`, table)

	var singleOutputID string
	var outputFileIDs []string
	var zipFileID string

	err := s.db.QueryRow(ctx, query, jobID).Scan(
		&result.UserID,
		&singleOutputID,
		&outputFileIDs,
		&zipFileID,
	)
	if err != nil {
		s.log.Error("failed to fetch job files", logger.Error(err))
		return models.JobFilesResult{}, err
	}

	if singleOutputID != "" {
		result.SingleOutputID = &singleOutputID
	}
	if len(outputFileIDs) > 0 {
		result.MultiOutputIDs = outputFileIDs
	}
	if zipFileID != "" {
		result.ZipFileID = &zipFileID
	}

	return result, nil
}
