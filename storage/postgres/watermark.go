package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
)

type watermarkRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewWatermarkRepo(db *pgxpool.Pool, log logger.ILogger) *watermarkRepo {
	return &watermarkRepo{db: db, log: log}
}

func (r *watermarkRepo) Create(ctx context.Context, job *models.WatermarkJob) error {
	query := `
		INSERT INTO add_watermark_jobs (id, user_id, input_file_id, text, font_size, position, opacity, output_file_id, status, created_at, font_name, fill_color, rotation, pages)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	var userID interface{}
	if job.UserID != nil && *job.UserID != "" {
		userID = *job.UserID
	}

	var outputFileID sql.NullString
	if job.OutputFileID != nil {
		outputFileID = sql.NullString{String: *job.OutputFileID, Valid: true}
	}

	_, err := r.db.Exec(ctx, query,
		job.ID,
		userID,
		job.InputFileID,
		job.Text,
		job.FontSize,
		job.Position,
		job.Opacity,
		outputFileID,
		job.Status,
		job.CreatedAt,
		"Helvetica", // font_name
		"gray",       // fill_color
		45,           // rotation
		"all",        // pages
	)
	if err != nil {
		r.log.Error("Failed to create watermark job", logger.Error(err))
		return err
	}
	return nil
}

func (r *watermarkRepo) GetByID(ctx context.Context, id string) (*models.WatermarkJob, error) {
	query := `
		SELECT id, user_id, input_file_id, text, font_size, position, opacity, output_file_id, status, created_at
		FROM add_watermark_jobs
		WHERE id = $1
	`

	var job models.WatermarkJob
	var userID sql.NullString
	var outputFileID sql.NullString

	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&userID,
		&job.InputFileID,
		&job.Text,
		&job.FontSize,
		&job.Position,
		&job.Opacity,
		&outputFileID,
		&job.Status,
		&job.CreatedAt,
	)
	if err != nil {
		r.log.Error("Failed to get watermark job by ID", logger.Error(err))
		return nil, err
	}

	if userID.Valid {
		job.UserID = &userID.String
	}
	if outputFileID.Valid {
		job.OutputFileID = &outputFileID.String
	}

	return &job, nil
}

func (r *watermarkRepo) Update(ctx context.Context, job *models.WatermarkJob) error {
	query := `
		UPDATE add_watermark_jobs
		SET output_file_id = $1, status = $2
		WHERE id = $3
	`

	var outputFileID sql.NullString
	if job.OutputFileID != nil {
		outputFileID = sql.NullString{String: *job.OutputFileID, Valid: true}
	}

	result, err := r.db.Exec(ctx, query, outputFileID, job.Status, job.ID)
	if err != nil {
		r.log.Error("Failed to update watermark job", logger.Error(err))
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no rows affected, job with id %s not found", job.ID)
	}

	return nil
}
