package postgres

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type pdfToJPGRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewPDFToJPGRepo(db *pgxpool.Pool, log logger.ILogger) storage.IPDFToJPGStorage {
	return &pdfToJPGRepo{db: db, log: log}
}

func (r *pdfToJPGRepo) Create(ctx context.Context, job *models.PDFToJPGJob) error {
	query := `
		INSERT INTO pdf_to_jpg_jobs (id, user_id, input_file_id, output_file_ids, zip_file_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	var zipFileID sql.NullString
	if job.ZipFileID != nil {
		zipFileID = sql.NullString{String: *job.ZipFileID, Valid: true}
	}

	var userID interface{}
	if job.UserID != nil {
		userID = *job.UserID
	}

	_, err := r.db.Exec(ctx, query,
		job.ID,
		userID,
		job.InputFileID,
		job.OutputFileIDs,
		zipFileID,
		job.Status,
		job.CreatedAt,
	)
	if err != nil {
		r.log.Error("Failed to create pdf_to_jpg job", logger.Error(err))
		return err
	}
	return nil
}

func (r *pdfToJPGRepo) GetByID(ctx context.Context, id string) (*models.PDFToJPGJob, error) {
	query := `
		SELECT id, user_id, input_file_id, output_file_ids, zip_file_id, status, created_at
		FROM pdf_to_jpg_jobs WHERE id = $1
	`
	var job models.PDFToJPGJob
	var userID sql.NullString
	var zipFileID sql.NullString

	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&userID,
		&job.InputFileID,
		&job.OutputFileIDs,
		&zipFileID,
		&job.Status,
		&job.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if userID.Valid {
		job.UserID = &userID.String
	}
	if zipFileID.Valid {
		job.ZipFileID = &zipFileID.String
	}

	return &job, nil
}

func (r *pdfToJPGRepo) Update(ctx context.Context, job *models.PDFToJPGJob) error {
	query := `
		UPDATE pdf_to_jpg_jobs
		SET output_file_ids = $1, zip_file_id = $2, status = $3
		WHERE id = $4
	`
	var zipFileID sql.NullString
	if job.ZipFileID != nil {
		zipFileID = sql.NullString{String: *job.ZipFileID, Valid: true}
	}

	_, err := r.db.Exec(ctx, query, job.OutputFileIDs, zipFileID, job.Status, job.ID)
	return err
}
