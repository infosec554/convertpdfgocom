package service

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type AddWatermarkService interface {
	Create(ctx context.Context, req models.AddWatermarkRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.AddWatermarkJob, error)
}

type addWatermarkService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewAddWatermarkService(stg storage.IStorage, log logger.ILogger) AddWatermarkService {
	return &addWatermarkService{stg: stg, log: log}
}

func (s *addWatermarkService) Create(ctx context.Context, req models.AddWatermarkRequest, userID *string) (string, error) {
	s.log.Info("AddWatermarkService.Create called")

	// 1. Get input file
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("input file not found", logger.Error(err))
		return "", err
	}

	// 2. Prepare output
	jobID := uuid.New().String()
	outputID := uuid.New().String()
	outputDir := "storage/watermark"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("failed to create output dir", logger.Error(err))
		return "", err
	}
	outputPath := filepath.Join(outputDir, outputID+".pdf")

	// 3. Prepare watermark config
	config := "fontname:Helvetica, scale:0.7 rel, pos:tl, rot:0, op:.6, fillcolor:#FF0000"
	wm, err := pdfcpu.ParseTextWatermarkDetails(req.Text, config, true, types.POINTS)
	if err != nil {
		s.log.Error("failed to parse watermark", logger.Error(err))
		return "", err
	}

	// 4. Pages
	var pages []string
	if req.Pages != "" {
		pages = append(pages, req.Pages)
	}

	// 5. Add watermark
	if err := api.AddWatermarksFile(file.FilePath, outputPath, pages, wm, nil); err != nil {
		s.log.Error("failed to apply watermark", logger.Error(err))
		return "", err
	}

	// 6. Save output file
	info, err := os.Stat(outputPath)
	if err != nil {
		s.log.Error("cannot stat output file", logger.Error(err))
		return "", err
	}

	outputFile := models.File{
		ID:         outputID,
		UserID:     userID,
		FileName:   filepath.Base(outputPath),
		FilePath:   outputPath,
		FileType:   "application/pdf",
		FileSize:   info.Size(),
		UploadedAt: time.Now(),
	}

	if _, err := s.stg.File().Save(ctx, outputFile); err != nil {
		s.log.Error("failed to save output file", logger.Error(err))
		return "", err
	}

	// 7. Create job
	job := &models.AddWatermarkJob{
		ID:           jobID,
		UserID:       userID,
		InputFileID:  req.InputFileID,
		OutputFileID: &outputID,
		Text:         req.Text,
		Pages:        req.Pages,
		Status:       "done",
		CreatedAt:    time.Now(),
	}

	if err := s.stg.AddWatermark().Create(ctx, job); err != nil {
		s.log.Error("failed to create watermark job", logger.Error(err))
		return "", err
	}

	s.log.Info("✅ Watermark successfully added", logger.String("job_id", jobID))
	return jobID, nil
}

func (s *addWatermarkService) GetByID(ctx context.Context, id string) (*models.AddWatermarkJob, error) {
	return s.stg.AddWatermark().GetByID(ctx, id)
}
