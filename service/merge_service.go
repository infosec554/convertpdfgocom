package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type MergeService interface {
	Create(ctx context.Context, job models.MergeJob) (string, error)
	Get(ctx context.Context, id string) (*models.MergeJob, error)
	MergeFiles(ctx context.Context, jobID string) (*models.File, error)
}

type mergeService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewMergeService(stg storage.IStorage, log logger.ILogger) MergeService {
	return &mergeService{stg: stg, log: log}
}

func (s *mergeService) Create(ctx context.Context, job models.MergeJob) (string, error) {
	return s.stg.Merge().Create(ctx, job)
}

func (s *mergeService) Get(ctx context.Context, id string) (*models.MergeJob, error) {
	return s.stg.Merge().Get(ctx, id)
}

func (s *mergeService) MergeFiles(ctx context.Context, jobID string) (*models.File, error) {
	job, err := s.stg.Merge().Get(ctx, jobID)
	if err != nil {
		return nil, err
	}

	// 1. Fayllarni yig'ish
	var inputPaths []string
	for _, fileID := range job.InputFileIDs {
		file, err := s.stg.File().GetByID(ctx, fileID)
		if err != nil {
			return nil, fmt.Errorf("failed to get file %s: %w", fileID, err)
		}
		inputPaths = append(inputPaths, file.FilePath)
	}

	// 2. Output path yaratish
	outputDir := filepath.Join("storage", "merged_files")
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return nil, err
	}
	outputFileName := uuid.NewString() + "_merged.pdf"
	outputFilePath := filepath.Join(outputDir, outputFileName)

	// 3. pdfcpu orqali birlashtirish
	conf := model.NewDefaultConfiguration()
	// conf.ValidationMode = model.ValidationNone // Removed to avoid dependency issues

	if err := api.MergeCreateFile(inputPaths, outputFilePath, false, conf); err != nil {
		s.log.Error("pdfcpu merge failed", logger.Error(err))
		return nil, err
	}

	// 4. Output faylni DB ga saqlash
	outputFile := models.File{
		ID:       uuid.NewString(),
		FileName: "merged.pdf",
		FilePath: outputFilePath,
		FileType: "application/pdf",
		FileSize: 0,
	}

	fileInfo, err := os.Stat(outputFilePath)
	if err == nil {
		outputFile.FileSize = fileInfo.Size()
	}

	fileID, err := s.stg.File().Save(ctx, outputFile)
	if err != nil {
		return nil, err
	}
	outputFile.ID = fileID

	// 5. Job ni update qilish
	job.OutputFileID = &fileID
	job.Status = "completed"
	if err := s.stg.Merge().Update(ctx, *job); err != nil {
		return nil, err
	}

	return &outputFile, nil
}
