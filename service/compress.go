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

type CompressService interface {
	Create(ctx context.Context, req models.CompressRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.CompressJob, error)
}

type compressService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewCompressService(stg storage.IStorage, log logger.ILogger) CompressService {
	return &compressService{
		stg: stg,
		log: log,
	}
}
func (s *compressService) Create(ctx context.Context, req models.CompressRequest, userID *string) (string, error) {
	s.log.Info("CompressService.Create called")

	// 1. Kiruvchi faylni olish
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("Input file not found", logger.String("fileID", req.InputFileID), logger.Error(err))
		return "", err
	}

	// Ensure the file exists before proceeding
	if _, err := os.Stat(file.FilePath); os.IsNotExist(err) {
		s.log.Error("Input file does not exist", logger.String("filePath", file.FilePath))
		return "", fmt.Errorf("input file does not exist: %s", file.FilePath)
	}

	// 2. Job yaratish
	job := &models.CompressJob{
		ID:               uuid.NewString(),
		UserID:           userID, // Correctly pass the userID (nil for guest users)
		InputFileID:      req.InputFileID,
		CompressionLevel: req.Compression, // Correctly pass compression level
		Status:           "pending",
		CreatedAt:        time.Now(),
	}

	// Handle output_file_id as nil initially
	job.OutputFileID = nil

	if err := s.stg.Compress().Create(ctx, job); err != nil {
		s.log.Error("Failed to create compress job", logger.Error(err))
		return "", err
	}

	// 3. Output fayl yo‘lini tayyorlash
	outputID := uuid.NewString()
	outputDir := "storage/compress"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("Failed to create output dir", logger.Error(err))
		return "", err
	}
	outputPath := filepath.Join(outputDir, outputID+".pdf")

	// 4. Pdfcpu konfiguratsiyasi
	conf := model.NewDefaultConfiguration()
	conf.Cmd = model.OPTIMIZE

	// 5. Faylni siqish (strukturaviy optimallashtirish)
	if err := api.OptimizeFile(file.FilePath, outputPath, conf); err != nil {
		s.log.Error("pdfcpu optimize failed", logger.Error(err))
		return "", err
	}

	// 6. Natijaviy faylni sistemaga saqlash
	fi, err := os.Stat(outputPath)
	if err != nil {
		s.log.Error("Cannot stat output file", logger.Error(err))
		return "", err
	}

	newFile := models.File{
		ID:         outputID,
		UserID:     userID,
		FileName:   filepath.Base(outputPath),
		FilePath:   outputPath,
		FileType:   "application/pdf",
		FileSize:   fi.Size(),
		UploadedAt: time.Now(),
	}

	if _, err := s.stg.File().Save(ctx, newFile); err != nil {
		s.log.Error("Failed to save output file", logger.Error(err))
		return "", err
	}

	// 7. Jobni update qilish
	job.OutputFileID = &outputID
	job.Status = "done"

	if err := s.stg.Compress().Update(ctx, job); err != nil {
		s.log.Error("Failed to update compress job", logger.Error(err))
		return "", err
	}

	s.log.Info("Compress job completed", logger.String("jobID", job.ID))
	return job.ID, nil
}

func (s *compressService) GetByID(ctx context.Context, id string) (*models.CompressJob, error) {
	job, err := s.stg.Compress().GetByID(ctx, id)
	if err != nil {
		s.log.Error("Failed to get compress job", logger.Error(err))
		return nil, err
	}
	return job, nil
}
