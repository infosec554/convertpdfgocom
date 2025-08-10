package service

import (
	"context"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type AdminJobService interface {
	ListJobs(ctx context.Context, f models.AdminJobFilter) ([]models.JobSummary, error)
}

type adminJobService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewAdminJobService(stg storage.IStorage, log logger.ILogger) AdminJobService {
	return &adminJobService{stg: stg, log: log}
}

func (s *adminJobService) ListJobs(ctx context.Context, f models.AdminJobFilter) ([]models.JobSummary, error) {
	s.log.Info("AdminJobService.ListJobs", logger.Any("filter", f))
	return s.stg.AdminJob().ListJobs(ctx, f)
}
