package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type mergeRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewMergeRepo(db *pgxpool.Pool, log logger.ILogger) storage.IMergeStorage {
	return &mergeRepo{db: db, log: log}
}

func (r *mergeRepo) Create(ctx context.Context, job models.MergeJob) (string, error) {
	id := uuid.NewString()
	query := `
		INSERT INTO merge_jobs (id, user_id, input_file_ids, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query, id, job.UserID, job.InputFileIDs, "pending", time.Now())
	if err != nil {
		r.log.Error("failed to create merge job", logger.Error(err))
		return "", err
	}
	return id, nil
}

func (r *mergeRepo) Get(ctx context.Context, id string) (*models.MergeJob, error) {
	var job models.MergeJob
	query := `SELECT id, user_id, input_file_ids, output_file_id, status FROM merge_jobs WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&job.UserID,
		&job.InputFileIDs,
		&job.OutputFileID,
		&job.Status,
	)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *mergeRepo) Update(ctx context.Context, job models.MergeJob) error {
	query := `
		UPDATE merge_jobs 
		SET output_file_id = $1, status = $2 
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, job.OutputFileID, job.Status, job.ID)
	return err
}
