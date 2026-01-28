package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
)

type protectRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewProtectRepo(db *pgxpool.Pool, log logger.ILogger) *protectRepo {
	return &protectRepo{db: db, log: log}
}

func (r *protectRepo) Create(ctx context.Context, job *models.ProtectPDFJob) error {
	query := `
		INSERT INTO security_jobs (id, user_id, input_file_id, type, password, output_file_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
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
		"protect", // Hardcoded type
		job.Password,
		outputFileID,
		job.CreatedAt,
	)
	if err != nil {
		r.log.Error("Failed to create protect job", logger.Error(err))
		return err
	}
	return nil
}

func (r *protectRepo) GetByID(ctx context.Context, id string) (*models.ProtectPDFJob, error) {
	query := `
		SELECT id, user_id, input_file_id, password, output_file_id, created_at
		FROM security_jobs
		WHERE id = $1 AND type = 'protect'
	`

	var job models.ProtectPDFJob
	var userID sql.NullString
	var outputFileID sql.NullString

	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&userID,
		&job.InputFileID,
		&job.Password,
		&outputFileID,
		&job.CreatedAt,
	)
	if err != nil {
		r.log.Error("Failed to get protect job by ID", logger.Error(err))
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

func (r *protectRepo) Update(ctx context.Context, job *models.ProtectPDFJob) error {
	query := `
		UPDATE security_jobs
		SET output_file_id = $1
		WHERE id = $2
	`

	var outputFileID sql.NullString
	if job.OutputFileID != nil {
		outputFileID = sql.NullString{String: *job.OutputFileID, Valid: true}
	}

	result, err := r.db.Exec(ctx, query, outputFileID, job.ID)
	if err != nil {
		r.log.Error("Failed to update protect job", logger.Error(err))
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no rows affected, job with id %s not found", job.ID)
	}

	return nil
}
