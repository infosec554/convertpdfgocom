package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type WatermarkService interface {
	Create(ctx context.Context, req models.WatermarkRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.WatermarkJob, error)
}

type watermarkService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewWatermarkService(stg storage.IStorage, log logger.ILogger) WatermarkService {
	return &watermarkService{
		stg: stg,
		log: log,
	}
}

func (s *watermarkService) Create(ctx context.Context, req models.WatermarkRequest, userID *string) (string, error) {
	s.log.Info("WatermarkService.Create called", logger.String("text", req.Text))

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

	// Default qiymatlar
	fontSize := req.FontSize
	if fontSize <= 0 {
		fontSize = 48
	}
	position := req.Position
	if position == "" {
		position = "diagonal"
	}
	opacity := req.Opacity
	if opacity <= 0 {
		opacity = 0.3
	}

	// 2. Job yaratish
	job := &models.WatermarkJob{
		ID:          uuid.NewString(),
		UserID:      userID,
		InputFileID: req.InputFileID,
		Text:        req.Text,
		FontSize:    fontSize,
		Position:    position,
		Opacity:     opacity,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := s.stg.Watermark().Create(ctx, job); err != nil {
		s.log.Error("Failed to create watermark job", logger.Error(err))
		return "", err
	}

	// 3. Output fayl yo'lini tayyorlash
	outputID := uuid.NewString()
	outputDir := "storage/watermark"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("Failed to create output dir", logger.Error(err))
		return "", err
	}
	outputPath := filepath.Join(outputDir, outputID+".pdf")

	// 4. Faylni nusxalash
	inputBytes, err := os.ReadFile(file.FilePath)
	if err != nil {
		s.log.Error("Failed to read input file", logger.Error(err))
		return "", err
	}
	if err := os.WriteFile(outputPath, inputBytes, 0644); err != nil {
		s.log.Error("Failed to write temp file", logger.Error(err))
		return "", err
	}

	// 5. pdfcpu bilan watermark qo'shish
	// Watermark description: "text, font:Helvetica, points:48, color:gray, opacity:0.3, rotation:45, diagonal:1"
	rotation := 45
	if position == "center" {
		rotation = 0
	}
	
	wmDesc := fmt.Sprintf("%s, font:Helvetica, points:%d, color:gray, opacity:%.1f, rotation:%d, scale:1.0 abs, position:c",
		req.Text, fontSize, opacity, rotation)

	wm, err := api.TextWatermark(wmDesc, "", true, false, types.POINTS)
	if err != nil {
		s.log.Error("Failed to create watermark", logger.Error(err))
		os.Remove(outputPath)
		return "", fmt.Errorf("watermark create failed: %w", err)
	}

	if err := api.AddWatermarksFile(outputPath, "", nil, wm, nil); err != nil {
		s.log.Error("pdfcpu watermark failed", logger.Error(err))
		os.Remove(outputPath)
		return "", fmt.Errorf("watermark failed: %w", err)
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
		FileName:   fmt.Sprintf("watermark_%s", filepath.Base(file.FileName)),
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

	if err := s.stg.Watermark().Update(ctx, job); err != nil {
		s.log.Error("Failed to update watermark job", logger.Error(err))
		return "", err
	}

	s.log.Info("Watermark job completed", logger.String("jobID", job.ID))
	return job.ID, nil
}

func (s *watermarkService) GetByID(ctx context.Context, id string) (*models.WatermarkJob, error) {
	job, err := s.stg.Watermark().GetByID(ctx, id)
	if err != nil {
		s.log.Error("Failed to get watermark job", logger.Error(err))
		return nil, err
	}
	return job, nil
}
