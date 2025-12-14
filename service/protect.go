package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type ProtectService interface {
	Create(ctx context.Context, req models.ProtectPDFRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.ProtectPDFJob, error)
}

type protectService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewProtectService(stg storage.IStorage, log logger.ILogger) ProtectService {
	return &protectService{
		stg: stg,
		log: log,
	}
}

func (s *protectService) Create(ctx context.Context, req models.ProtectPDFRequest, userID *string) (string, error) {
	s.log.Info("ProtectService.Create called")

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
	job := &models.ProtectPDFJob{
		ID:          uuid.NewString(),
		UserID:      userID,
		InputFileID: req.InputFileID,
		Password:    req.Password,
		CreatedAt:   time.Now(),
	}

	if err := s.stg.Protect().Create(ctx, job); err != nil {
		s.log.Error("Failed to create protect job", logger.Error(err))
		return "", err
	}

	// 3. Output fayl yo'lini tayyorlash
	outputID := uuid.NewString()
	outputDir := "storage/protect"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("Failed to create output dir", logger.Error(err))
		return "", err
	}
	outputPath := filepath.Join(outputDir, outputID+".pdf")

	// 4. pdfcpu bilan parollash (Encrypt)
	conf := api.LoadConfiguration()
	conf.UserPW = req.Password
	conf.OwnerPW = req.Password
	conf.EncryptUsingAES = true
	conf.EncryptKeyLength = 256
	conf.Permissions = model.PermissionsAll
	
	if err := api.EncryptFile(file.FilePath, outputPath, conf); err != nil {
		s.log.Error("pdfcpu encrypt failed", logger.Error(err))
		return "", fmt.Errorf("encryption failed: %w", err)
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
		FileName:   fmt.Sprintf("protected_%s", filepath.Base(file.FileName)),
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

	if err := s.stg.Protect().Update(ctx, job); err != nil {
		s.log.Error("Failed to update protect job", logger.Error(err))
		return "", err
	}

	s.log.Info("Protect job completed", logger.String("jobID", job.ID))
	return job.ID, nil
}

func (s *protectService) GetByID(ctx context.Context, id string) (*models.ProtectPDFJob, error) {
	job, err := s.stg.Protect().GetByID(ctx, id)
	if err != nil {
		s.log.Error("Failed to get protect job", logger.Error(err))
		return nil, err
	}
	return job, nil
}
