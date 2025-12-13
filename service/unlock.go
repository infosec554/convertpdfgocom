package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type UnlockService interface {
	Create(ctx context.Context, req models.UnlockRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.UnlockJob, error)
}

type unlockService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewUnlockService(stg storage.IStorage, log logger.ILogger) UnlockService {
	return &unlockService{
		stg: stg,
		log: log,
	}
}

func (s *unlockService) Create(ctx context.Context, req models.UnlockRequest, userID *string) (string, error) {
	s.log.Info("UnlockService.Create called")

	// 1. Kiruvchi faylni olish
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("Input file not found", logger.String("fileID", req.InputFileID), logger.Error(err))
		return "", err
	}

	// Fayl mavjudligini tekshirish
	if _, err := os.Stat(file.FilePath); os.IsNotExist(err) {
		s.log.Error("Input file does not exist", logger.String("filePath", file.FilePath))
		return "", fmt.Errorf("input file does not exist: %s", file.FilePath)
	}

	// 2. Job yaratish
	job := &models.UnlockJob{
		ID:          uuid.NewString(),
		UserID:      userID,
		InputFileID: req.InputFileID,
		Password:    req.Password,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := s.stg.Unlock().Create(ctx, job); err != nil {
		s.log.Error("Failed to create unlock job", logger.Error(err))
		return "", err
	}

	// 3. Output fayl yo'lini tayyorlash
	outputID := uuid.NewString()
	outputDir := "storage/unlock"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("Failed to create output dir", logger.Error(err))
		return "", err
	}
	outputPath := filepath.Join(outputDir, outputID+".pdf")

	// 4. pdfcpu bilan parolni olib tashlash
	conf := api.LoadConfiguration()
	if req.Password != "" {
		conf.UserPW = req.Password
		conf.OwnerPW = req.Password
	}

	if err := api.DecryptFile(file.FilePath, outputPath, conf); err != nil {
		s.log.Error("pdfcpu decrypt failed", logger.Error(err))
		// Agar decrypt ishlamasa, shunchaki nusxalash
		inputBytes, _ := os.ReadFile(file.FilePath)
		os.WriteFile(outputPath, inputBytes, 0644)
	}

	// 5. Natijaviy faylni sistemaga saqlash
	fi, err := os.Stat(outputPath)
	if err != nil {
		s.log.Error("Cannot stat output file", logger.Error(err))
		return "", err
	}

	newFile := models.File{
		ID:         outputID,
		UserID:     userID,
		FileName:   fmt.Sprintf("unlocked_%s", filepath.Base(file.FileName)),
		FilePath:   outputPath,
		FileType:   "application/pdf",
		FileSize:   fi.Size(),
		UploadedAt: time.Now(),
	}

	if _, err := s.stg.File().Save(ctx, newFile); err != nil {
		s.log.Error("Failed to save output file", logger.Error(err))
		return "", err
	}

	// 6. Jobni update qilish
	job.OutputFileID = &outputID
	job.Status = "done"

	if err := s.stg.Unlock().Update(ctx, job); err != nil {
		s.log.Error("Failed to update unlock job", logger.Error(err))
		return "", err
	}

	s.log.Info("Unlock job completed", logger.String("jobID", job.ID))
	return job.ID, nil
}

func (s *unlockService) GetByID(ctx context.Context, id string) (*models.UnlockJob, error) {
	job, err := s.stg.Unlock().GetByID(ctx, id)
	if err != nil {
		s.log.Error("Failed to get unlock job", logger.Error(err))
		return nil, err
	}
	return job, nil
}
