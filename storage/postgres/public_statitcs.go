package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type publicStatsRepo struct {
	db  *pgxpool.Pool
	log logger.ILogger
}

// Ixtiyoriy: constructor interfeys qaytarsa ham bo'ladi
func NewPublicStatsRepo(db *pgxpool.Pool, log logger.ILogger) storage.IPublicStatsStorage {
	return &publicStatsRepo{db: db, log: log}
}

// MUHIM: nomi IPublicStatsStorage dagi bilan BIR xil bo'lsin
func (r *publicStatsRepo) GetPublicStats(ctx context.Context) (models.PublicStats, error) {
	var stats models.PublicStats

	query := `
SELECT
    (SELECT COUNT(*) FROM users) AS total_users,
    (SELECT COUNT(DISTINCT tool_name) FROM tools) AS tools_count,
    (
        (SELECT COUNT(*) FROM merge_jobs) +
        (SELECT COUNT(*) FROM split_jobs) +
        (SELECT COUNT(*) FROM remove_pages_jobs) +
        (SELECT COUNT(*) FROM compress_jobs) +
        (SELECT COUNT(*) FROM extract_pages_jobs) +
        (SELECT COUNT(*) FROM jpg_to_pdf_jobs) +
        (SELECT COUNT(*) FROM pdf_to_jpg_jobs) +
        (SELECT COUNT(*) FROM pdf_to_word_jobs) +
        (SELECT COUNT(*) FROM rotate_jobs) +
        (SELECT COUNT(*) FROM add_page_number_jobs) +
        (SELECT COUNT(*) FROM crop_pdf_jobs) +
        (SELECT COUNT(*) FROM protect_jobs) +
        (SELECT COUNT(*) FROM unlock_jobs)
    ) AS total_usage

`

	err := r.db.QueryRow(ctx, query).Scan(
		&stats.TotalUsers,
		&stats.ToolsCount,
		&stats.TotalUsage,
	)
	if err != nil {
		r.log.Error("failed to get public stats", logger.Error(err))
		return stats, err
	}
	return stats, nil
}
