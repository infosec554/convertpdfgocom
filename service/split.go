package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type SplitService interface {
	Create(ctx context.Context, req models.CreateSplitJobRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.SplitJob, error)
}

type splitService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewSplitService(stg storage.IStorage, log logger.ILogger) SplitService {
	return &splitService{
		stg: stg,
		log: log,
	}
}

func (s *splitService) Create(ctx context.Context, req models.CreateSplitJobRequest, userID *string) (string, error) {
	s.log.Info("SplitService.Create called")
	s.log.Info("Received InputFileID", logger.String("input_file_id", req.InputFileID))
	s.log.Info("Received SplitRanges", logger.String("split_ranges", req.SplitRanges))
	if userID != nil {
		s.log.Info("UserID", logger.String("user_id", *userID))
	} else {
		s.log.Info("UserID", logger.String("user_id", "guest"))
	}

	// 1. Faylni bazadan olish
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("❌ input file not found", logger.Error(err))
		return "", err
	}
	s.log.Info("✅ Input file found", logger.String("file_path", file.FilePath))

	// 2. SplitJob struct
	job := &models.SplitJob{
		ID:            uuid.New().String(),
		UserID:        userID,
		InputFileID:   req.InputFileID,
		SplitRanges:   req.SplitRanges,
		OutputFileIDs: []string{},
		Status:        "pending",
		CreatedAt:     time.Now(),
	}
	s.log.Info("✅ Split job created", logger.String("job_id", job.ID))

	// 3. Jobni bazaga yozish
	if err := s.stg.Split().Create(ctx, job); err != nil {
		s.log.Error("❌ failed to create split job", logger.Error(err))
		return "", err
	}
	s.log.Info("✅ Split job saved to DB")

	// 4. Split qilish
	inputPath := file.FilePath
	outputDir := fmt.Sprintf("storage/split/%s", job.ID)

	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("❌ failed to create output dir", logger.Error(err))
		return "", err
	}
	s.log.Info("✅ Output directory created", logger.String("output_dir", outputDir))

	// 5. Split PDF into separate files
	config := model.NewDefaultConfiguration()
	span := 1
	if strings.TrimSpace(req.SplitRanges) != "" {
		if n, err := fmt.Sscanf(req.SplitRanges, "%d", &span); err != nil || n != 1 || span < 1 {
			s.log.Error("❌ invalid split range span", logger.String("value", req.SplitRanges), logger.Error(err))
			return "", fmt.Errorf("invalid split range span: %s", req.SplitRanges)
		}
	}
	s.log.Info("✅ Starting PDF split...", logger.Int("span", span))

	if err := api.SplitFile(inputPath, outputDir, span, config); err != nil {
		s.log.Error("❌ failed to split PDF", logger.Error(err))
		return "", err
	}
	s.log.Info("✅ PDF split completed")

	// 6. Output fayllarni o‘qish va DBga saqlash
	files, err := os.ReadDir(outputDir)
	if err != nil {
		s.log.Error("❌ failed to read output dir", logger.Error(err))
		return "", err
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".pdf") {
			continue
		}

		fullPath := filepath.Join(outputDir, f.Name())
		fi, err := os.Stat(fullPath)
		if err != nil {
			s.log.Error("❌ failed to stat split file", logger.String("path", fullPath), logger.Error(err))
			continue
		}

		newFile := models.File{
			ID:         uuid.New().String(),
			UserID:     userID,
			FileName:   f.Name(),
			FilePath:   fullPath,
			FileType:   "application/pdf",
			FileSize:   fi.Size(),
			UploadedAt: time.Now(),
		}

		fileID, err := s.stg.File().Save(ctx, newFile)
		if err != nil {
			s.log.Error("❌ failed to save split file", logger.String("file", f.Name()), logger.Error(err))
			continue
		}
		s.log.Info("✅ Output file saved", logger.String("file_id", fileID), logger.String("name", f.Name()))

		job.OutputFileIDs = append(job.OutputFileIDs, fileID)
	}

	// 7. Jobni yangilash
	if err := s.stg.Split().UpdateOutputFiles(ctx, job.ID, job.OutputFileIDs); err != nil {
		s.log.Error("❌ failed to update split job", logger.Error(err))
		return "", err
	}
	s.log.Info("✅ Split job updated with output files", logger.Int("total_files", len(job.OutputFileIDs)))

	return job.ID, nil
}

func (s *splitService) GetByID(ctx context.Context, id string) (*models.SplitJob, error) {
	job, err := s.stg.Split().GetByID(ctx, id)
	if err != nil {
		s.log.Error("failed to get split job", logger.Error(err))
		return nil, err
	}
	return job, nil
}
