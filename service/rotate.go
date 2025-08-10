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

type RotateService interface {
	Create(ctx context.Context, req models.RotatePDFRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.RotateJob, error)
}

type rotateService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewRotateService(stg storage.IStorage, log logger.ILogger) RotateService {
	return &rotateService{stg: stg, log: log}
}

func (s *rotateService) Create(ctx context.Context, req models.RotatePDFRequest, userID *string) (string, error) {
	s.log.Info("RotateService.Create called")

	// 1. Faylni olish
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("input file not found", logger.Error(err))
		return "", fmt.Errorf("input file not found: %v", err)
	}

	// 2. Job yaratish (output_file_id keyinchalik belgilanadi)
	jobID := uuid.NewString()
	outputFileID := uuid.NewString()
	outputPath := filepath.Join("storage/rotate_files", outputFileID+".pdf")

	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return "", err
	}

	job := &models.RotateJob{
		ID:          jobID,
		UserID:      userID,
		InputFileID: req.InputFileID,
		Status:      "pending",
		Angle:       req.Angle,
		Pages:       req.Pages,
		CreatedAt:   time.Now(),
	}

	if err := s.stg.Rotate().Create(ctx, job); err != nil {
		return "", err
	}

	// 3. Faylni burish: pdfcpu CLI orqali
	args := []string{
		"rotate",
	}

	if req.Pages != "all" {
		args = append(args, "-pages", req.Pages)
	}

	args = append(args,
		file.FilePath,                // input file
		fmt.Sprintf("%d", req.Angle), // rotation angle
		outputPath,                   // output file
	)

	cmd := exec.Command("pdfcpu", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		s.log.Error("pdfcpu failed", logger.Error(err))
		return "", err
	}

	// 4. Yangi faylni saqlash
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

	// 5. Job holatini yangilash
	job.OutputFileID = &outputFileID
	job.Status = "done"

	if err := s.stg.Rotate().Update(ctx, job); err != nil {
		return "", err
	}

	s.log.Info("Rotate job completed", logger.String("jobID", jobID))
	return jobID, nil
}

func (s *rotateService) GetByID(ctx context.Context, id string) (*models.RotateJob, error) {
	return s.stg.Rotate().GetByID(ctx, id)
}
