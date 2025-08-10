package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type ExtractPageService interface {
	Create(ctx context.Context, req models.ExtractPagesRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.ExtractJob, error)
}

type extractPageService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewExtractService(stg storage.IStorage, log logger.ILogger) ExtractPageService {
	return &extractPageService{
		stg: stg,
		log: log,
	}
}
func (s *extractPageService) Create(ctx context.Context, req models.ExtractPagesRequest, userID *string) (string, error) {
	s.log.Info("ExtractService.Create called")

	// 1. Faylni olish
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("input file not found", logger.Error(err))
		return "", err
	}
	s.log.Info("input file path", logger.String("filePath", file.FilePath))

	// 2. PDF sahifa sonini olish
	pdfCtx, err := api.ReadContextFile(file.FilePath)
	if err != nil {
		s.log.Error("failed to read PDF context", logger.Error(err))
		return "", err
	}
	totalPages := pdfCtx.PageCount
	s.log.Info("PDF page count", logger.Int("pageCount", totalPages))

	// 3. Job yaratish
	jobID := uuid.NewString()
	job := &models.ExtractJob{
		ID:             jobID,
		UserID:         userID,
		InputFileID:    req.InputFileID,
		PagesToExtract: req.PageRanges,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}
	if err := s.stg.ExtractPage().Create(ctx, job); err != nil {
		s.log.Error("failed to create extract job", logger.Error(err))
		return "", err
	}

	// 4. Output papkani yaratish
	outputDir := filepath.Join("storage/extract", job.ID)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("failed to create output dir", logger.Error(err))
		return "", err
	}

	// 5. Sahifa raqamlarini ajratish
	var pages []string
	for _, r := range strings.Split(req.PageRanges, ",") {
		r = strings.TrimSpace(r)
		if strings.Contains(r, "-") {
			parts := strings.Split(r, "-")
			start, err1 := strconv.Atoi(parts[0])
			end, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil || start < 1 || end > totalPages || start > end {
				s.log.Error("invalid range", logger.String("range", r))
				continue
			}
			for i := start; i <= end; i++ {
				pages = append(pages, fmt.Sprintf("%d", i))
			}
		} else {
			page, err := strconv.Atoi(r)
			if err != nil || page < 1 || page > totalPages {
				s.log.Error("invalid page", logger.String("page", r))
				continue
			}
			pages = append(pages, fmt.Sprintf("%d", page))
		}
	}

	// 6. Agar sahifa topilmagan bo‘lsa
	if len(pages) == 0 {
		job.Status = "failed"
		_ = s.stg.ExtractPage().Update(ctx, job)
		s.log.Error("no valid pages were extracted", logger.String("jobID", job.ID))
		return job.ID, nil
	}

	// 7. Sahifalarni chiqarish
	err = api.ExtractPagesFile(file.FilePath, outputDir, pages, model.NewDefaultConfiguration())
	if err != nil {
		s.log.Error("pdfcpu extract failed", logger.Error(err))
		job.Status = "failed"
		_ = s.stg.ExtractPage().Update(ctx, job)
		return job.ID, nil
	}

	// 8. Fayllarni saqlash
	files, err := os.ReadDir(outputDir)
	if err != nil {
		s.log.Error("cannot read output dir", logger.Error(err))
		job.Status = "failed"
		_ = s.stg.ExtractPage().Update(ctx, job)
		return job.ID, nil
	}

	var firstOutputFileID *string
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".pdf") {
			continue
		}

		fullPath := filepath.Join(outputDir, f.Name())
		fi, err := os.Stat(fullPath)
		if err != nil {
			s.log.Error("cannot stat extracted file", logger.Error(err))
			continue
		}

		newFile := models.File{
			ID:         uuid.NewString(),
			UserID:     userID,
			FileName:   f.Name(),
			FilePath:   fullPath,
			FileType:   "application/pdf",
			FileSize:   fi.Size(),
			UploadedAt: time.Now(),
		}

		fileID, err := s.stg.File().Save(ctx, newFile)
		if err != nil {
			s.log.Error("failed to save extracted file", logger.Error(err))
			continue
		}

		if firstOutputFileID == nil {
			firstOutputFileID = &fileID
		}
	}

	// 9. Job holatini yangilash
	if firstOutputFileID != nil {
		job.Status = "done"
		job.OutputFileID = firstOutputFileID
	} else {
		job.Status = "failed"
	}
	_ = s.stg.ExtractPage().Update(ctx, job)

	s.log.Info("extract job finished", logger.String("jobID", job.ID), logger.String("status", job.Status))
	return job.ID, nil
}

func (s *extractPageService) GetByID(ctx context.Context, id string) (*models.ExtractJob, error) {
	job, err := s.stg.ExtractPage().GetByID(ctx, id)
	if err != nil {
		s.log.Error("failed to get extract job", logger.Error(err))
		return nil, err
	}
	return job, nil
}
