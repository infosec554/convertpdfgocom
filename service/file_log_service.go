package service

import (
	"context"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type FileDeletionLogService interface {
	LogDeletion(ctx context.Context, log models.FileDeletionLog) error
	GetDeletionLogs(ctx context.Context, limit, offset int) ([]models.FileDeletionLog, error)
}

type fileDeletionLogService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewFileDeletionLogService(stg storage.IStorage, log logger.ILogger) FileDeletionLogService {
	return &fileDeletionLogService{
		stg: stg,
		log: log,
	}
}

func (s *fileDeletionLogService) LogDeletion(ctx context.Context, log models.FileDeletionLog) error {
	s.log.Info("FileDeletionLogService.LogDeletion called", logger.String("file_id", log.FileID))

	if err := s.stg.FileDeletionLog().LogDeletion(ctx, log); err != nil {
		s.log.Error("failed to log file deletion", logger.Error(err))
		return err
	}
	return nil
}

func (s *fileDeletionLogService) GetDeletionLogs(ctx context.Context, limit, offset int) ([]models.FileDeletionLog, error) {
	s.log.Info("FileDeletionLogService.GetDeletionLogs called", logger.Int("limit", limit), logger.Int("offset", offset))

	logs, err := s.stg.FileDeletionLog().GetDeletionLogs(ctx, limit, offset)
	if err != nil {
		s.log.Error("failed to get deletion logs", logger.Error(err))
		return nil, err
	}
	return logs, nil
}
