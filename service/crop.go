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

type CropPDFService interface {
	Create(ctx context.Context, req models.CropPDFRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.CropPDFJob, error)
}

type cropPDFService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewCropPDFService(stg storage.IStorage, log logger.ILogger) CropPDFService {
	return &cropPDFService{stg: stg, log: log}
}

func (s *cropPDFService) Create(ctx context.Context, req models.CropPDFRequest, userID *string) (string, error) {
	s.log.Info("CropPDFService.Create called")

	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("Input file not found", logger.Error(err))
		return "", fmt.Errorf("input file not found: %v", err)
	}

	jobID := uuid.NewString()
	outputFileID := uuid.NewString()
	outputPath := filepath.Join("storage/crop_pdfs", outputFileID+".pdf")

	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return "", err
	}

	job := &models.CropPDFJob{
		ID:          jobID,
		UserID:      userID,
		InputFileID: req.InputFileID,
		Top:         req.Top,
		Bottom:      req.Bottom,
		Left:        req.Left,
		Right:       req.Right,
		Box:         req.Box,
		Pages:       req.Pages,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := s.stg.Crop().Create(ctx, job); err != nil {
		return "", err
	}

	// ✅ To‘g‘rilangan crop description
	cropDesc := fmt.Sprintf("%d %d %d %d", req.Top, req.Right, req.Bottom, req.Left)

	args := []string{"crop"}

	if req.Pages != "" {
		args = append(args, "-pages", req.Pages)
	}

	args = append(args, "--", cropDesc, file.FilePath, outputPath)

	cmd := exec.Command("pdfcpu", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		s.log.Error("pdfcpu crop failed", logger.Error(err))
		return "", err
	}

	fi, err := os.Stat(outputPath)
	if err != nil {
		return "", err
	}

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
		return "", err
	}

	job.OutputFileID = &outputFileID
	job.Status = "done"

	if err := s.stg.Crop().Update(ctx, job); err != nil {
		return "", err
	}

	s.log.Info("Crop job completed", logger.String("jobID", jobID))
	return jobID, nil
}

func (s *cropPDFService) GetByID(ctx context.Context, id string) (*models.CropPDFJob, error) {
	return s.stg.Crop().GetByID(ctx, id)
}
