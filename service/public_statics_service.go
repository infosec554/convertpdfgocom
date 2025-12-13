package service

import (
	"context"
	"fmt"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type PublicStatsService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewPublicStatsService(stg storage.IStorage, log logger.ILogger) *PublicStatsService {
	return &PublicStatsService{
		stg: stg,
		log: log,
	}
}

func formatNumber(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.3fM", float64(n)/1_000_000)
	} else if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func (s PublicStatsService) GetStats(ctx context.Context) (models.PublicStats, error) {
	return s.stg.PublicStats().GetPublicStats(ctx)
}

func (s PublicStatsService) GetDisplayStats(ctx context.Context) (models.PublicStatsDisplay, error) {
	raw, err := s.stg.PublicStats().GetPublicStats(ctx)
	if err != nil {
		return models.PublicStatsDisplay{}, err
	}

	return models.PublicStatsDisplay{
		TotalUsers: formatNumber(raw.TotalUsers),
		ToolsCount: raw.ToolsCount,
		TotalUsage: formatNumber(raw.TotalUsage),
	}, nil
}
