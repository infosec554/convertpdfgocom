package postgres

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type addWatermarkRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewAddWatermarkRepo(db *pgxpool.Pool, log logger.ILogger) storage.IAddWatermarkStorage {
	return &addWatermarkRepo{db: db, log: log}
}

func (r *addWatermarkRepo) Create(ctx context.Context, job *models.AddWatermarkJob) error {
	query := `
		INSERT INTO add_watermark_jobs (
			id, user_id, input_file_id, output_file_id, text,
			font_name, font_size, position, rotation, opacity,
			fill_color, pages, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5,
		        $6, $7, $8, $9, $10,
		        $11, $12, $13, $14)
	`
	var userID interface{}
	if job.UserID != nil && *job.UserID != "" {
		userID = *job.UserID
	} else {
		userID = nil // NULL for guest users
	}
	var outputFileID sql.NullString
	if job.OutputFileID != nil {
		outputFileID = sql.NullString{String: *job.OutputFileID, Valid: true}
	} else {
		outputFileID = sql.NullString{Valid: false} // NULL value in DB
	}
	_, err := r.db.Exec(ctx, query,
		job.ID, userID, job.InputFileID, outputFileID, job.Text,
		job.FontName, job.FontSize, job.Position, job.Rotation, job.Opacity,
		job.FillColor, job.Pages, job.Status, job.CreatedAt,
	)

	return err
}

func (r *addWatermarkRepo) GetByID(ctx context.Context, id string) (*models.AddWatermarkJob, error) {
	query := `
		SELECT id, user_id, input_file_id, output_file_id, text,
		       font_name, font_size, position, rotation, opacity,
		       fill_color, pages, status, created_at
		FROM add_watermark_jobs
		WHERE id = $1
	`

	row := r.db.QueryRow(ctx, query, id)

	var job models.AddWatermarkJob
	err := row.Scan(
		&job.ID, &job.UserID, &job.InputFileID, &job.OutputFileID, &job.Text,
		&job.FontName, &job.FontSize, &job.Position, &job.Rotation, &job.Opacity,
		&job.FillColor, &job.Pages, &job.Status, &job.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (r *addWatermarkRepo) Update(ctx context.Context, job *models.AddWatermarkJob) error {
	query := `
		UPDATE add_watermark_jobs
		SET output_file_id = $1,
		    status = $2
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, job.OutputFileID, job.Status, job.ID)
	return err
}
