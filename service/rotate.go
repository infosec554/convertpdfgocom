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

type RotateService interface {
	Create(ctx context.Context, req models.RotateRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.RotateJob, error)
}

type rotateService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewRotateService(stg storage.IStorage, log logger.ILogger) RotateService {
	return &rotateService{
		stg: stg,
		log: log,
	}
}

func (s *rotateService) Create(ctx context.Context, req models.RotateRequest, userID *string) (string, error) {
	s.log.Info("RotateService.Create called", logger.Int("angle", req.Angle))

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

	// Pages default qiymat
	pages := req.Pages
	if pages == "" {
		pages = "all"
	}

	// Angleni tekshirish (90, 180, 270)
	if req.Angle != 90 && req.Angle != 180 && req.Angle != 270 {
		return "", fmt.Errorf("invalid angle: %d (must be 90, 180 or 270)", req.Angle)
	}

	// 2. Job yaratish
	job := &models.RotateJob{
		ID:          uuid.NewString(),
		UserID:      userID,
		InputFileID: req.InputFileID,
		Angle:       req.Angle,
		Pages:       pages,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := s.stg.Rotate().Create(ctx, job); err != nil {
		s.log.Error("Failed to create rotate job", logger.Error(err))
		return "", err
	}

	// 3. Output fayl yo'lini tayyorlash
	outputID := uuid.NewString()
	outputDir := "storage/rotate"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("Failed to create output dir", logger.Error(err))
		return "", err
	}
	outputPath := filepath.Join(outputDir, outputID+".pdf")

	// 4. Faylni nusxalash (original faylni o'zgartirmaslik uchun)
	inputBytes, err := os.ReadFile(file.FilePath)
	if err != nil {
		s.log.Error("Failed to read input file", logger.Error(err))
		return "", err
	}
	if err := os.WriteFile(outputPath, inputBytes, 0644); err != nil {
		s.log.Error("Failed to write temp file", logger.Error(err))
		return "", err
	}

	// 5. pdfcpu bilan aylantirish
	// pdfcpu Rotate: angle va pages
	var selectedPages []string
	if pages != "all" {
		selectedPages = []string{pages}
	}

	if err := api.RotateFile(outputPath, "", req.Angle, selectedPages, nil); err != nil {
		s.log.Error("pdfcpu rotate failed", logger.Error(err))
		os.Remove(outputPath)
		return "", fmt.Errorf("rotate failed: %w", err)
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
		FileName:   fmt.Sprintf("rotated_%d_%s", req.Angle, filepath.Base(file.FileName)),
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

	if err := s.stg.Rotate().Update(ctx, job); err != nil {
		s.log.Error("Failed to update rotate job", logger.Error(err))
		return "", err
	}

	s.log.Info("Rotate job completed", logger.String("jobID", job.ID), logger.Int("angle", req.Angle))
	return job.ID, nil
}

func (s *rotateService) GetByID(ctx context.Context, id string) (*models.RotateJob, error) {
	job, err := s.stg.Rotate().GetByID(ctx, id)
	if err != nil {
		s.log.Error("Failed to get rotate job", logger.Error(err))
		return nil, err
	}
	return job, nil
}
