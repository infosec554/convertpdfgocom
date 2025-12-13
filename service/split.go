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
	s.log.Info("SplitService.Create called", logger.String("inputFileID", req.InputFileID))

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
	job := &models.SplitJob{
		ID:            uuid.NewString(),
		UserID:        userID,
		InputFileID:   req.InputFileID,
		SplitRanges:   req.SplitRanges,
		OutputFileIDs: []string{},
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	if err := s.stg.Split().Create(ctx, job); err != nil {
		s.log.Error("Failed to create split job", logger.Error(err))
		return "", err
	}

	// 3. Output papkasini tayyorlash
	outputDir := filepath.Join("storage", "split", job.ID)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("Failed to create output dir", logger.Error(err))
		return "", err
	}

	// 4. PDF ni bo'lish (pdfcpu)
	// SplitRanges formati: "1-3,4-5,6" -> [1-3], [4-5], [6]
	ranges := strings.Split(req.SplitRanges, ",")
	var outputFileIDs []string

	for i, r := range ranges {
		r = strings.TrimSpace(r)
		// Har bir qism uchun alohida papka ochamiz (file nomini aniq bilish uchun)
		partDir := filepath.Join(outputDir, fmt.Sprintf("temp_%d", i))
		if err := os.MkdirAll(partDir, os.ModePerm); err != nil {
			s.log.Error("Failed to create part dir", logger.Error(err))
			continue
		}

		// pdfcpu ExtractPages - string formatda sahifalar kerak: "1-3" yoki "5"
		// Natija partDir ichiga tushadi
		if err := api.ExtractPagesFile(file.FilePath, partDir, []string{r}, nil); err != nil {
			s.log.Error("pdfcpu extract failed", logger.String("range", r), logger.Error(err))
			continue
		}

		// partDir ichidagi faylni topamiz
		files, err := os.ReadDir(partDir)
		if err != nil || len(files) == 0 {
			s.log.Error("No file generated in part dir", logger.String("range", r))
			continue
		}
		
		// Birinchi faylni olamiz (odatda bitta bo'ladi)
		generatedName := files[0].Name()
		generatedPath := filepath.Join(partDir, generatedName)
		
		outputPath := filepath.Join(outputDir, fmt.Sprintf("part_%d_%s", i+1, generatedName))

		// Faylni asosiy output papkaga ko'chirib, nomini chiroyli qilamiz
		if err := os.Rename(generatedPath, outputPath); err != nil {
			s.log.Error("Failed to rename/move output file", logger.Error(err))
			continue
		}
		
		// Temp papkani o'chiramiz
		os.RemoveAll(partDir)

		// 5. Output faylni saqlash
		fi, err := os.Stat(outputPath)
		if err != nil {
			s.log.Error("Cannot stat output file", logger.Error(err))
			continue
		}

		outputID := uuid.NewString()
		newFile := models.File{
			ID:         outputID,
			UserID:     userID,
			FileName:   fmt.Sprintf("split_%s", generatedName), // Original nomga yaqin saqlaymiz
			FilePath:   outputPath,
			FileType:   "application/pdf",
			FileSize:   fi.Size(),
			UploadedAt: time.Now(),
		}

		if _, err := s.stg.File().Save(ctx, newFile); err != nil {
			s.log.Error("Failed to save output file", logger.Error(err))
			continue
		}

		outputFileIDs = append(outputFileIDs, outputID)
	}

	// 6. Job ni update qilish
	job.OutputFileIDs = outputFileIDs
	job.Status = "done"

	if err := s.stg.Split().Update(ctx, job); err != nil {
		s.log.Error("Failed to update split job", logger.Error(err))
		return "", err
	}

	s.log.Info("Split job completed", logger.String("jobID", job.ID), logger.Int("outputCount", len(outputFileIDs)))
	return job.ID, nil
}

func (s *splitService) GetByID(ctx context.Context, id string) (*models.SplitJob, error) {
	job, err := s.stg.Split().GetByID(ctx, id)
	if err != nil {
		s.log.Error("Failed to get split job", logger.Error(err))
		return nil, err
	}
	return job, nil
}

