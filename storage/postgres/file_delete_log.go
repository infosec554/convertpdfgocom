package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type fileDeletionLogRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

func NewFileDeletionLogRepo(db *pgxpool.Pool, log logger.ILogger) storage.IFileDeletionLogStorage {
	return &fileDeletionLogRepo{db: db, log: log}
}

func (r *fileDeletionLogRepo) LogDeletion(ctx context.Context, log models.FileDeletionLog) error {
	query := `
		INSERT INTO files_deletion_logs
		    (id, file_id, user_id, deleted_by, deleted_at, reason)
		VALUES
		    ($1, $2, $3, $4, $5, $6)
	`

	if log.ID == "" {
		log.ID = GenerateUUID() // sizning UUID yaratish funksiyangiz
	}

	_, err := r.db.Exec(ctx, query,
		log.ID, log.FileID, log.UserID, log.DeletedBy, log.DeletedAt, log.Reason)
	if err != nil {
		r.log.Error("failed to log file deletion", logger.Error(err))
	}
	return err
}

func (r *fileDeletionLogRepo) GetDeletionLogs(ctx context.Context, limit, offset int) ([]models.FileDeletionLog, error) {
	query := `
		SELECT
		    id,
		    file_id,
		    user_id,
		    deleted_by,
		    deleted_at,
		    reason
		FROM
		    files_deletion_logs
		ORDER BY
		    deleted_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		r.log.Error("failed to query deleted logs", logger.Error(err))
		return nil, err
	}
	defer rows.Close()

	var logs []models.FileDeletionLog
	for rows.Next() {
		var log models.FileDeletionLog
		err := rows.Scan(
			&log.ID,
			&log.FileID,
			&log.UserID,
			&log.DeletedBy,
			&log.DeletedAt,
			&log.Reason,
		)
		if err != nil {
			r.log.Error("failed to scan deleted log row", logger.Error(err))
			continue
		}
		logs = append(logs, log)
	}

	if rows.Err() != nil {
		r.log.Error("error iterating deleted logs rows", logger.Error(rows.Err()))
		return nil, rows.Err()
	}

	return logs, nil
}
func GenerateUUID() string {
	return uuid.NewString()
}
