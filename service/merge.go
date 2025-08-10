package service

import (
	"context"
	"errors"
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

// Sentinel errors (handler map qilishi uchun)
var (
	ErrJobNotFound     = errors.New("job not found")
	ErrJobInvalidInput = errors.New("need at least two files to merge")
	ErrJobInvalidState = errors.New("job status not eligible for processing")
)

type MergeService interface {
	Create(ctx context.Context, userID *string, inputFileIDs []string) (string, error)
	GetByID(ctx context.Context, id string) (*models.MergeJob, error)
	ProcessJob(ctx context.Context, jobID string) (string, error)
}

type mergeService struct {
	stg storage.IMergeStorage
	fs  storage.IFileStorage
	log logger.ILogger
}

func NewMergeService(stg storage.IStorage, log logger.ILogger) MergeService {
	return &mergeService{
		stg: stg.Merge(),
		fs:  stg.File(),
		log: log,
	}
}

func (s *mergeService) Create(ctx context.Context, userID *string, inputFileIDs []string) (string, error) {
	if userID == nil {
		s.log.Info("📥 MergeService.Create by guest")
	} else {
		s.log.Info("📥 MergeService.Create", logger.String("userID", *userID))
	}

	job := &models.MergeJob{
		ID:        uuid.New().String(),
		UserID:    userID,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	if err := s.stg.Create(ctx, job); err != nil {
		s.log.Error("❌ failed to create merge job", logger.Error(err))
		return "", err
	}
	if err := s.stg.AddInputFiles(ctx, job.ID, inputFileIDs); err != nil {
		s.log.Error("❌ failed to add input files", logger.Error(err))
		return "", err
	}

	s.log.Info("✅ merge job created", logger.String("jobID", job.ID))
	return job.ID, nil
}

func (s *mergeService) GetByID(ctx context.Context, id string) (*models.MergeJob, error) {
	s.log.Info("📥 MergeService.GetByID", logger.String("jobID", id))

	job, err := s.stg.GetByID(ctx, id)
	if err != nil {
		s.log.Error("❌ failed to get merge job", logger.Error(err))
		return nil, err
	}
	return job, nil
}

func (s *mergeService) ProcessJob(ctx context.Context, jobID string) (string, error) {
	s.log.Info("📥 MergeService.ProcessJob", logger.String("jobID", jobID))

	// Job ni olish
	job, err := s.stg.GetByID(ctx, jobID)
	if err != nil || job == nil {
		s.log.Error("❌ job not found", logger.Error(err), logger.String("jobID", jobID))
		return "", ErrJobNotFound
	}

	// Status guard + race prevention: faqat pending -> processing ga ruxsat
	ok, err := s.stg.TransitionStatus(ctx, jobID, "pending", "processing")
	if err != nil {
		s.log.Error("❌ status transition error", logger.Error(err))
		return "", fmt.Errorf("status transition failed: %w", err)
	}
	if !ok {
		s.log.Info("⚠️ job not in pending state", logger.String("status", job.Status))
		return "", ErrJobInvalidState
	}

	// Input check
	if len(job.InputFileIDs) < 2 {
		s.log.Error("❌ not enough input files", logger.Any("input_ids", job.InputFileIDs))
		job.Status = "failed"
		_ = s.stg.Update(ctx, job)
		return "", ErrJobInvalidInput
	}

	// Output folder
	if err := os.MkdirAll("storage/merge", 0o755); err != nil {
		s.log.Error("❌ failed to create merge dir", logger.Error(err))
		job.Status = "failed"
		_ = s.stg.Update(ctx, job)
		return "", fmt.Errorf("mkdir: %w", err)
	}

	// Input pathlarni yig‘ish
	inputPaths := make([]string, 0, len(job.InputFileIDs))
	for _, fileID := range job.InputFileIDs {
		file, err := s.fs.GetByID(ctx, fileID)
		if err != nil {
			s.log.Error("❌ failed to get input file", logger.String("fileID", fileID), logger.Error(err))
			job.Status = "failed"
			_ = s.stg.Update(ctx, job)
			return "", fmt.Errorf("get file: %w", err)
		}
		if _, err := os.Stat(file.FilePath); os.IsNotExist(err) {
			s.log.Error("❌ input file not found", logger.String("filePath", file.FilePath))
			job.Status = "failed"
			_ = s.stg.Update(ctx, job)
			return "", fmt.Errorf("input file does not exist: %s", file.FilePath)
		}
		inputPaths = append(inputPaths, file.FilePath)
	}

	s.log.Info("📄 All input files ready", logger.Any("paths", inputPaths))

	// Merge
	outputID := uuid.New().String()
	outputPath := filepath.Join("storage/merge", outputID+".pdf")

	conf := model.NewDefaultConfiguration()
	if err := api.MergeCreateFile(inputPaths, outputPath, false, conf); err != nil {
		s.log.Error("❌ pdf merge failed", logger.Error(err))
		job.Status = "failed"
		_ = s.stg.Update(ctx, job)
		return "", fmt.Errorf("merge failed: %w", err)
	}
	s.log.Info("✅ PDF successfully merged", logger.String("outputPath", outputPath))

	// Output file metadata
	info, err := os.Stat(outputPath)
	if err != nil {
		s.log.Error("❌ failed to stat merged file", logger.Error(err))
		job.Status = "failed"
		_ = s.stg.Update(ctx, job)
		return "", fmt.Errorf("output file stat failed: %w", err)
	}

	outputFile := models.File{
		ID:         outputID,
		UserID:     job.UserID,
		FileName:   filepath.Base(outputPath),
		FilePath:   outputPath,
		FileType:   "application/pdf",
		FileSize:   info.Size(),
		UploadedAt: time.Now(),
	}
	if _, err = s.fs.Save(ctx, outputFile); err != nil {
		s.log.Error("❌ failed to save merged file", logger.Error(err))
		job.Status = "failed"
		_ = s.stg.Update(ctx, job)
		return "", fmt.Errorf("save output: %w", err)
	}

	// Done
	job.OutputFileID = &outputID
	job.Status = "done"
	// ixtiyoriy: job.FinishedAt = time.Now()
	if err := s.stg.Update(ctx, job); err != nil {
		s.log.Error("❌ failed to update merge job", logger.Error(err))
		return "", fmt.Errorf("update job: %w", err)
	}

	s.log.Info("🎉 Merge job done", logger.String("outputID", outputID))
	return outputID, nil
}
