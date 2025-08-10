package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type pdfToWordService struct {
	stg storage.IStorage
	log logger.ILogger
}

type PDFToWordService interface {
	Create(ctx context.Context, req models.PDFToWordRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.PDFToWordJob, error)
}

func NewPDFToWordService(stg storage.IStorage, log logger.ILogger) PDFToWordService {
	return &pdfToWordService{
		stg: stg,
		log: log,
	}
}

func (s *pdfToWordService) Create(ctx context.Context, req models.PDFToWordRequest, userID *string) (string, error) {
	s.log.Info("PDFToWordService.Create called")

	// 1. Faylni olish
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("input file not found", logger.Error(err))
		return "", fmt.Errorf("input file not found: %v", err)
	}

	jobID := uuid.NewString()
	outputFileID := uuid.NewString()
	outputPath := filepath.Join("storage/pdf_to_word", outputFileID+".docx")

	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		s.log.Error("failed to create output directory", logger.Error(err))
		return "", err
	}

	// 2. Gotenberg orqali Word'ga konvertatsiya
	cmd := exec.Command(
		"curl", "-X", "POST",
		"-F", fmt.Sprintf("files=@%s", file.FilePath),
		"-F", "convert=pdf", // Gotenberg convert flag
		"-o", outputPath,
		"http://localhost:3000/forms/libreoffice/convert",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		s.log.Error("gotenberg conversion failed", logger.Error(err))
		return "", err
	}

	fi, err := os.Stat(outputPath)
	if err != nil {
		s.log.Error("cannot stat output file", logger.Error(err))
		return "", err
	}

	newFile := models.File{
		ID:         outputFileID,
		UserID:     userID,
		FileName:   filepath.Base(outputPath),
		FilePath:   outputPath,
		FileType:   "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		FileSize:   fi.Size(),
		UploadedAt: time.Now(),
	}
	if _, err := s.stg.File().Save(ctx, newFile); err != nil {
		s.log.Error("failed to save output file", logger.Error(err))
		return "", err
	}

	job := &models.PDFToWordJob{
		ID:           jobID,
		UserID:       userID,
		InputFileID:  req.InputFileID,
		OutputFileID: &outputFileID,
		Status:       "done",
		CreatedAt:    time.Now(),
	}
	if err := s.stg.PDFToWord().Create(ctx, job); err != nil {
		s.log.Error("failed to create pdf to word job", logger.Error(err))
		return "", err
	}

	s.log.Info("PDFToWord job completed", logger.String("jobID", jobID))
	return jobID, nil
}

func (s *pdfToWordService) GetByID(ctx context.Context, id string) (*models.PDFToWordJob, error) {
	job, err := s.stg.PDFToWord().GetByID(ctx, id)
	if err != nil {
		s.log.Error("pdf to word job not found", logger.Error(err))
		return nil, err
	}
	return job, nil
}
