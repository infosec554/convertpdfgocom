package service

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/gotenberg"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type PowerPointToPDFService interface {
	Create(ctx context.Context, req models.PowerPointToPDFRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.PowerPointToPDFJob, error)
}

type powerPointToPDFService struct {
	stg       storage.IStorage
	log       logger.ILogger
	gotClient gotenberg.Client
}

func NewPowerPointToPDFService(stg storage.IStorage, log logger.ILogger, gotClient gotenberg.Client) PowerPointToPDFService {
	return &powerPointToPDFService{
		stg:       stg,
		log:       log,
		gotClient: gotClient,
	}
}

func (s *powerPointToPDFService) Create(ctx context.Context, req models.PowerPointToPDFRequest, userID *string) (string, error) {
	s.log.Info("PowerPointToPDFService.Create called")

	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("input file not found", logger.Error(err))
		return "", err
	}

	resultBytes, err := s.gotClient.PowerPointToPDF(ctx, file.FilePath)
	if err != nil {
		s.log.Error("Gotenberg conversion failed", logger.Error(err))
		return "", err
	}

	jobID := uuid.NewString()
	outputFileID := uuid.NewString()
	outputPath := filepath.Join("storage/powerpoint_to_pdf", outputFileID+".pdf")

	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return "", err
	}
	if err := os.WriteFile(outputPath, resultBytes, 0644); err != nil {
		return "", err
	}

	fi, _ := os.Stat(outputPath)
	newFile := models.File{
		ID:         outputFileID,
		UserID:     userID,
		FileName:   filepath.Base(outputPath),
		FilePath:   outputPath,
		FileType:   "application/pdf",
		FileSize:   fi.Size(),
		UploadedAt: time.Now(),
	}
	if _, err := s.stg.File().Save(ctx, newFile); err != nil {
		s.log.Error("failed to save output file", logger.Error(err))
		return "", err
	}

	job := &models.PowerPointToPDFJob{
		ID:           jobID,
		UserID:       userID,
		InputFileID:  req.InputFileID,
		OutputFileID: &outputFileID,
		Status:       "done",
		CreatedAt:    time.Now(),
	}
	if err := s.stg.PowerPointToPDF().Create(ctx, job); err != nil {
		s.log.Error("failed to save job", logger.Error(err))
		return "", err
	}

	return jobID, nil
}

func (s *powerPointToPDFService) GetByID(ctx context.Context, id string) (*models.PowerPointToPDFJob, error) {
	job, err := s.stg.PowerPointToPDF().GetByID(ctx, id)
	if err != nil {
		s.log.Error("PowerPointToPDF job not found", logger.Error(err))
		return nil, err
	}
	return job, nil
}
