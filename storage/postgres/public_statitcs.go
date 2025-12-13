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
    (SELECT COUNT(*) FROM bot_users) AS total_users,
    10 AS tools_count,
    (SELECT COALESCE(SUM(total_used), 0) FROM bot_users) AS total_usage
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
