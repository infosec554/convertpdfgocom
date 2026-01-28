package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
)

type unlockRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewUnlockRepo(db *pgxpool.Pool, log logger.ILogger) *unlockRepo {
	return &unlockRepo{db: db, log: log}
}

func (r *unlockRepo) Create(ctx context.Context, job *models.UnlockJob) error {
	query := `
		INSERT INTO unlock_jobs (id, user_id, input_file_id, output_file_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
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
		outputFileID,
		job.Status,
		job.CreatedAt,
	)
	if err != nil {
		r.log.Error("Failed to create unlock job", logger.Error(err))
		return err
	}
	return nil
}

func (r *unlockRepo) GetByID(ctx context.Context, id string) (*models.UnlockJob, error) {
	query := `
		SELECT id, user_id, input_file_id, output_file_id, status, created_at
		FROM unlock_jobs
		WHERE id = $1
	`

	var job models.UnlockJob
	var userID sql.NullString
	var outputFileID sql.NullString

	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&userID,
		&job.InputFileID,
		&outputFileID,
		&job.Status,
		&job.CreatedAt,
	)
	if err != nil {
		r.log.Error("Failed to get unlock job by ID", logger.Error(err))
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

func (r *unlockRepo) Update(ctx context.Context, job *models.UnlockJob) error {
	query := `
		UPDATE unlock_jobs
		SET output_file_id = $1, status = $2
		WHERE id = $3
	`

	var outputFileID sql.NullString
	if job.OutputFileID != nil {
		outputFileID = sql.NullString{String: *job.OutputFileID, Valid: true}
	}

	result, err := r.db.Exec(ctx, query, outputFileID, job.Status, job.ID)
	if err != nil {
		r.log.Error("Failed to update unlock job", logger.Error(err))
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no rows affected, job with id %s not found", job.ID)
	}

	return nil
}
