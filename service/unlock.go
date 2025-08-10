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

type UnlockService interface {
	Create(ctx context.Context, req models.UnlockPDFRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.UnlockPDFJob, error)
}

type unlockService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewUnlockService(stg storage.IStorage, log logger.ILogger) UnlockService {
	return &unlockService{stg: stg, log: log}
}

func (s *unlockService) Create(ctx context.Context, req models.UnlockPDFRequest, userID *string) (string, error) {
	s.log.Info("UnlockService.Create called")

	// 1. Kiruvchi faylni olish
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("input file not found", logger.Error(err))
		return "", fmt.Errorf("input file not found: %v", err)
	}

	// 2. Yangi fayl nomi va joylashuv
	jobID := uuid.NewString()
	outputFileID := uuid.NewString()
	outputPath := filepath.Join("storage/unlock_pdf", outputFileID+".pdf")

	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		s.log.Error("failed to create output directory", logger.Error(err))
		return "", err
	}

	// 3. `pdfcpu decrypt` bajarish (parol bilan)
	args := []string{
		"decrypt",
		"-upw", req.Password,
		file.FilePath,
		outputPath,
	}
	cmd := exec.Command("pdfcpu", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		s.log.Error("pdfcpu decrypt failed", logger.Error(err))
		return "", err
	}

	// 4. Output fayl mavjudligini tekshirish
	fi, err := os.Stat(outputPath)
	if err != nil {
		s.log.Error("cannot stat output file", logger.Error(err))
		return "", err
	}

	// 5. Faylni DBga saqlash (avval files jadvaliga)
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

	// 6. Job obyektini yaratish (keyin unlock_jobs jadvaliga)
	job := &models.UnlockPDFJob{
		ID:           jobID,
		UserID:       userID,
		InputFileID:  req.InputFileID,
		OutputFileID: &outputFileID,
		Status:       "done",
		CreatedAt:    time.Now(),
	}
	if err := s.stg.Unlock().Create(ctx, job); err != nil {
		s.log.Error("failed to create unlock job", logger.Error(err))
		return "", err
	}

	s.log.Info("Unlock job completed", logger.String("jobID", jobID))
	return jobID, nil
}

func (s *unlockService) GetByID(ctx context.Context, id string) (*models.UnlockPDFJob, error) {
	job, err := s.stg.Unlock().GetByID(ctx, id)
	if err != nil {
		s.log.Error("unlock job not found", logger.Error(err))
		return nil, err
	}
	return job, nil
}
