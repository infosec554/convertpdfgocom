package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
)

type rotateRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewRotateRepo(db *pgxpool.Pool, log logger.ILogger) *rotateRepo {
	return &rotateRepo{db: db, log: log}
}

func (r *rotateRepo) Create(ctx context.Context, job *models.RotateJob) error {
	query := `
		INSERT INTO rotate_jobs (id, user_id, input_file_id, rotation_angle, pages, output_file_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
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
		job.Angle,
		job.Pages,
		outputFileID,
		job.Status,
		job.CreatedAt,
	)
	if err != nil {
		r.log.Error("Failed to create rotate job", logger.Error(err))
		return err
	}
	return nil
}

func (r *rotateRepo) GetByID(ctx context.Context, id string) (*models.RotateJob, error) {
	query := `
		SELECT id, user_id, input_file_id, rotation_angle, pages, output_file_id, status, created_at
		FROM rotate_jobs
		WHERE id = $1
	`

	var job models.RotateJob
	var userID sql.NullString
	var outputFileID sql.NullString

	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&userID,
		&job.InputFileID,
		&job.Angle,
		&job.Pages,
		&outputFileID,
		&job.Status,
		&job.CreatedAt,
	)
	if err != nil {
		r.log.Error("Failed to get rotate job by ID", logger.Error(err))
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

func (r *rotateRepo) Update(ctx context.Context, job *models.RotateJob) error {
	query := `
		UPDATE rotate_jobs
		SET output_file_id = $1, status = $2
		WHERE id = $3
	`

	var outputFileID sql.NullString
	if job.OutputFileID != nil {
		outputFileID = sql.NullString{String: *job.OutputFileID, Valid: true}
	}

	result, err := r.db.Exec(ctx, query, outputFileID, job.Status, job.ID)
	if err != nil {
		r.log.Error("Failed to update rotate job", logger.Error(err))
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no rows affected, job with id %s not found", job.ID)
	}

	return nil
}
