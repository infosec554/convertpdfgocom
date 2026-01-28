package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
)

type splitRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewSplitRepo(db *pgxpool.Pool, log logger.ILogger) *splitRepo {
	return &splitRepo{db: db, log: log}
}

func (r *splitRepo) Create(ctx context.Context, job *models.SplitJob) error {
	query := `
		INSERT INTO split_jobs (id, user_id, input_file_id, split_ranges, output_file_ids, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var userID = handleUserID(job.UserID)

	_, err := r.db.Exec(ctx, query,
		job.ID,
		userID,
		job.InputFileID,
		job.SplitRanges,
		pq.Array(job.OutputFileIDs),
		job.Status,
		job.CreatedAt,
	)
	if err != nil {
		r.log.Error("Failed to create split job", logger.Error(err))
		return err
	}
	return nil
}

func (r *splitRepo) GetByID(ctx context.Context, id string) (*models.SplitJob, error) {
	query := `
		SELECT id, user_id, input_file_id, split_ranges, output_file_ids, status, created_at
		FROM split_jobs
		WHERE id = $1
	`

	var job models.SplitJob
	var userID sql.NullString
	var outputFileIDs []string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&userID,
		&job.InputFileID,
		&job.SplitRanges,
		&outputFileIDs,
		&job.Status,
		&job.CreatedAt,
	)
	if err != nil {
		r.log.Error("Failed to get split job by ID", logger.Error(err))
		return nil, err
	}

	if userID.Valid {
		job.UserID = &userID.String
	}
	job.OutputFileIDs = outputFileIDs

	return &job, nil
}

func (r *splitRepo) Update(ctx context.Context, job *models.SplitJob) error {
	query := `
		UPDATE split_jobs
		SET output_file_ids = $1, status = $2
		WHERE id = $3
	`

	result, err := r.db.Exec(ctx, query, pq.Array(job.OutputFileIDs), job.Status, job.ID)
	if err != nil {
		r.log.Error("Failed to update split job", logger.Error(err))
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no rows affected, job with id %s not found", job.ID)
	}

	return nil
}
