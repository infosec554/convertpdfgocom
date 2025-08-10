package service

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/gotenberg" // ⬅️ Gotenberg client paketini import qilasiz
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type WordToPDFService interface {
	Create(ctx context.Context, req models.WordToPDFRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.WordToPDFJob, error)
}

type wordToPDFService struct {
	stg       storage.IStorage
	log       logger.ILogger
	gotClient gotenberg.Client // ⬅️ Gotenberg interfeysi bilan
}

func NewWordToPDFService(stg storage.IStorage, log logger.ILogger, gotClient gotenberg.Client) WordToPDFService {
	return &wordToPDFService{
		stg:       stg,
		log:       log,
		gotClient: gotClient,
	}
}

func (s *wordToPDFService) Create(ctx context.Context, req models.WordToPDFRequest, userID *string) (string, error) {
	s.log.Info("WordToPDFService.Create called")

	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("input file not found", logger.Error(err))
		return "", err
	}

	// 🔥 Gotenbergdan PDF faylni olish
	resultBytes, err := s.gotClient.WordToPDF(ctx, file.FilePath)
	if err != nil {
		s.log.Error("Gotenberg conversion failed", logger.Error(err))
		return "", err
	}

	jobID := uuid.NewString()
	outputFileID := uuid.NewString()
	outputPath := filepath.Join("storage/word_to_pdf", outputFileID+".pdf")

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

	job := &models.WordToPDFJob{
		ID:           jobID,
		UserID:       userID,
		InputFileID:  req.InputFileID,
		OutputFileID: &outputFileID,
		Status:       "done",
		CreatedAt:    time.Now(),
	}
	if err := s.stg.WordToPDF().Create(ctx, job); err != nil {
		s.log.Error("failed to save job", logger.Error(err))
		return "", err
	}

	return jobID, nil
}

func (s *wordToPDFService) GetByID(ctx context.Context, id string) (*models.WordToPDFJob, error) {
	job, err := s.stg.WordToPDF().GetByID(ctx, id)
	if err != nil {
		s.log.Error("wordToPDF job not found", logger.Error(err))
		return nil, err
	}
	return job, nil
}
