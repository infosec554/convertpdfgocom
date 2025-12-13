package service

import (
	"context"
	"os"
	"time"

	"github.com/google/uuid"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type FileService interface {
	Upload(ctx context.Context, req models.File) (string, error)
	Save(ctx context.Context, req models.File) (string, error)
	Get(ctx context.Context, id string) (models.File, error)
	GetByID(ctx context.Context, id string) (models.File, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID string) ([]models.File, error)
	GetOldFiles(ctx context.Context, olderThanDays int) ([]models.OldFile, error)
	CleanupOldFiles(ctx context.Context, olderThanDays int) (int, error)
	ListPendingDeletionFiles(ctx context.Context, expirationMinutes int) ([]models.File, error)
}

type fileService struct {
	stg storage.IFileStorage
	log logger.ILogger
}

func NewFileService(stg storage.IStorage, log logger.ILogger) FileService {
	return &fileService{
		stg: stg.File(),
		log: log,
	}
}

func (s *fileService) Upload(ctx context.Context, req models.File) (string, error) {
	s.log.Info("FileService.Upload called", logger.String("file_name", req.FileName))

	req.ID = uuid.NewString()
	req.UploadedAt = time.Now()

	id, err := s.stg.Save(ctx, req)
	if err != nil {
		s.log.Error("failed to save file", logger.Error(err))
		return "", err
	}
	return id, nil
}

func (s *fileService) Save(ctx context.Context, req models.File) (string, error) {
	return s.stg.Save(ctx, req)
}

func (s *fileService) Get(ctx context.Context, id string) (models.File, error) {
	return s.stg.GetByID(ctx, id)
}

func (s *fileService) GetByID(ctx context.Context, id string) (models.File, error) {
	return s.stg.GetByID(ctx, id)
}

func (s *fileService) GetOldFiles(ctx context.Context, olderThanDays int) ([]models.OldFile, error) {
	return s.stg.GetOldFiles(ctx, olderThanDays)
}

func (s *fileService) Delete(ctx context.Context, id string) error {
	return s.stg.Delete(ctx, id)
}

func (s *fileService) List(ctx context.Context, userID string) ([]models.File, error) {
	return s.stg.ListByUser(ctx, userID)
}

func (s *fileService) CleanupOldFiles(ctx context.Context, olderThanDays int) (int, error) {
	s.log.Info("Cleaning up old files...", logger.Int("older_than_days", olderThanDays))

	oldFiles, err := s.stg.GetOldFiles(ctx, olderThanDays)
	if err != nil {
		s.log.Error("failed to get old files", logger.Error(err))
		return 0, err
	}

	count := 0
	for _, file := range oldFiles {
		select {
		case <-ctx.Done():
			s.log.Info("CleanupOldFiles canceled by context")
			return count, ctx.Err()
		default:
		}

		if err := os.Remove(file.FilePath); err != nil {
			s.log.Error("failed to delete file from disk", logger.Error(err), logger.String("path", file.FilePath))
			continue
		}
		if err := s.stg.DeleteByID(ctx, file.ID); err != nil {
			s.log.Error("failed to delete file from db", logger.Error(err), logger.String("id", file.ID))
			continue
		}
		count++
	}

	s.log.Info("CleanupOldFiles finished", logger.Int("files_deleted", count))
	return count, nil
}

func (s *fileService) ListPendingDeletionFiles(ctx context.Context, expirationMinutes int) ([]models.File, error) {
	s.log.Info("FileService.ListPendingDeletionFiles called", logger.Int("expiration_minutes", expirationMinutes))
	return s.stg.GetPendingDeletionFiles(ctx, expirationMinutes)
}
