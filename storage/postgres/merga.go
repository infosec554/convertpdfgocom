package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
	return &mergeRepo{
		db:  db,
		log: log,
	}
}

// Create merge job
func (m *mergeRepo) Create(ctx context.Context, job *models.MergeJob) error {
	const q = `
		INSERT INTO merge_jobs (id, user_id, output_file_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	var userID any
	if job.UserID != nil && *job.UserID != "" {
		userID = *job.UserID
	} else {
		userID = nil
	}

	var outputFileID any
	if job.OutputFileID != nil && *job.OutputFileID != "" {
		outputFileID = *job.OutputFileID
	} else {
		outputFileID = nil
	}

	_, err := m.db.Exec(ctx, q, job.ID, userID, outputFileID, job.Status, job.CreatedAt)
	if err != nil {
		m.log.Error("❌ failed to create merge job", logger.Error(err))
	}
	return err
}

// Get job by id (NULL-safe scan)
func (m *mergeRepo) GetByID(ctx context.Context, id string) (*models.MergeJob, error) {
	const q = `
		SELECT id, user_id, output_file_id, status, created_at
		FROM merge_jobs
		WHERE id = $1
	`

	var job models.MergeJob
	var userID pgtype.Text
	var outputFileID pgtype.Text

	err := m.db.QueryRow(ctx, q, id).Scan(
		&job.ID, &userID, &outputFileID, &job.Status, &job.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}
		m.log.Error("❌ failed to fetch merge job", logger.Error(err))
		return nil, err
	}

	if userID.Valid {
		u := userID.String
		job.UserID = &u
	}
	if outputFileID.Valid {
		o := outputFileID.String
		job.OutputFileID = &o
	}

	inputs, err := m.GetInputFiles(ctx, job.ID)
	if err != nil {
		m.log.Error("⚠️ failed to fetch input files for merge job", logger.Error(err))
	} else {
		job.InputFileIDs = inputs
	}

	return &job, nil
}

// Add input files
func (m *mergeRepo) AddInputFiles(ctx context.Context, jobID string, fileIDs []string) error {
	const q = `
		INSERT INTO merge_job_input_files (id, job_id, file_id)
		VALUES ($1, $2, $3)
	`
	for _, fileID := range fileIDs {
		if _, err := m.db.Exec(ctx, q, uuid.New().String(), jobID, fileID); err != nil {
			m.log.Error("❌ failed to add input file", logger.String("fileID", fileID), logger.Error(err))
			return err
		}
	}
	return nil
}

func (m *mergeRepo) GetInputFiles(ctx context.Context, jobID string) ([]string, error) {
	const q = `SELECT file_id FROM merge_job_input_files WHERE job_id = $1`
	rows, err := m.db.Query(ctx, q, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Update job (status/output_file_id)
func (m *mergeRepo) Update(ctx context.Context, job *models.MergeJob) error {
	const q = `
		UPDATE merge_jobs
		SET output_file_id = $1, status = $2
		WHERE id = $3
	`

	var outputFileID any
	if job.OutputFileID != nil && *job.OutputFileID != "" {
		outputFileID = *job.OutputFileID
	} else {
		outputFileID = nil
	}

	_, err := m.db.Exec(ctx, q, outputFileID, job.Status, job.ID)
	if err != nil {
		m.log.Error("❌ failed to update merge job", logger.Error(err))
	}
	return err
}

// 🔐 ATOMAR STATUS O‘TKAZISH (race'larni to‘xtatadi)
func (m *mergeRepo) TransitionStatus(ctx context.Context, id, fromStatus, toStatus string) (bool, error) {
	const q = `
		UPDATE merge_jobs
		SET status = $3
		WHERE id = $1 AND status = $2
	`
	tag, err := m.db.Exec(ctx, q, id, fromStatus, toStatus)
	if err != nil {
		m.log.Error("❌ failed to transition status", logger.Error(err))
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
