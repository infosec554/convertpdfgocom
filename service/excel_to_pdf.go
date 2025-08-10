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

type ExcelToPDFService interface {
	Create(ctx context.Context, req models.ExcelToPDFRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.ExcelToPDFJob, error)
}

type excelToPDFService struct {
	stg       storage.IStorage
	log       logger.ILogger
	gotClient gotenberg.Client
}

func NewExcelToPDFService(stg storage.IStorage, log logger.ILogger, gotClient gotenberg.Client) ExcelToPDFService {
	return &excelToPDFService{
		stg:       stg,
		log:       log,
		gotClient: gotClient,
	}
}

func (s *excelToPDFService) Create(ctx context.Context, req models.ExcelToPDFRequest, userID *string) (string, error) {
	s.log.Info("ExcelToPDFService.Create called")

	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("input file not found", logger.Error(err))
		return "", err
	}

	// 📥 Excelni PDFga aylantirish
	resultBytes, err := s.gotClient.WordToPDF(ctx, file.FilePath)
	if err != nil {
		s.log.Error("Gotenberg conversion failed", logger.Error(err))
		return "", err
	}

	jobID := uuid.NewString()
	outputFileID := uuid.NewString()
	outputPath := filepath.Join("storage/excel_to_pdf", outputFileID+".pdf")

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

	job := &models.ExcelToPDFJob{
		ID:           jobID,
		UserID:       userID,
		InputFileID:  req.InputFileID,
		OutputFileID: &outputFileID,
		Status:       "done",
		CreatedAt:    time.Now(),
	}
	if err := s.stg.ExcelToPDF().Create(ctx, job); err != nil {
		s.log.Error("failed to save job", logger.Error(err))
		return "", err
	}

	return jobID, nil
}

func (s *excelToPDFService) GetByID(ctx context.Context, id string) (*models.ExcelToPDFJob, error) {
	job, err := s.stg.ExcelToPDF().GetByID(ctx, id)
	if err != nil {
		s.log.Error("excelToPDF job not found", logger.Error(err))
		return nil, err
	}
	return job, nil
}
