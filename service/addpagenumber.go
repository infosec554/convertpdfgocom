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

type AddPageNumberService interface {
	Create(ctx context.Context, req models.AddPageNumbersRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.AddPageNumberJob, error)
}

type addPageNumberService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewAddPageNumberService(stg storage.IStorage, log logger.ILogger) AddPageNumberService {
	return &addPageNumberService{stg: stg, log: log}
}

func (s *addPageNumberService) Create(ctx context.Context, req models.AddPageNumbersRequest, userID *string) (string, error) {
	s.log.Info("AddPageNumberService.Create called")

	// 1. Input faylni tekshirish
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("input file not found", logger.Error(err))
		return "", fmt.Errorf("input file not found: %v", err)
	}

	// 2. Job ID va output path yaratish
	jobID := uuid.NewString()
	outputFileID := uuid.NewString()
	outputPath := filepath.Join("storage/add_page_numbers", outputFileID+".pdf")

	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return "", err
	}

	// 3. Job ma'lumotlarini saqlash
	job := &models.AddPageNumberJob{
		ID:          jobID,
		UserID:      userID,
		InputFileID: req.InputFileID,
		Status:      "pending",
		FirstNumber: req.FirstNumber,
		PageRange:   req.PageRange,
		Position:    req.Position,
		Color:       req.Color,
		FontSize:    req.FontSize,
		CreatedAt:   time.Now(),
	}

	if err := s.stg.AddPageNumber().Create(ctx, job); err != nil {
		return "", err
	}

	// 4. CLI buyrug‘ini to‘g‘ri formatda tuzish
	args := []string{
		"stamp", "add",
		"-mode", "text",
		"-pages", req.PageRange,
		"--",
		"Page %p of %P", // matn formati
		fmt.Sprintf(
			"scale:1.0 abs, pos:%s, rot:0, fillcolor:%s, fontname:Helvetica, points:%d",
			req.Position,
			req.Color,
			req.FontSize,
		),
		file.FilePath,
		outputPath,
	}

	// 5. Komandani ishga tushirish
	cmd := exec.Command("pdfcpu", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		s.log.Error("pdfcpu failed", logger.Error(err))
		return "", err
	}

	// 6. Natijaviy faylni tekshirish va DBga saqlash
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

	// 7. Job holatini yangilash
	job.OutputFileID = &outputFileID
	job.Status = "done"

	if err := s.stg.AddPageNumber().Update(ctx, job); err != nil {
		return "", err
	}

	s.log.Info("Add page number job completed", logger.String("jobID", jobID))
	return jobID, nil
}

func (s *addPageNumberService) GetByID(ctx context.Context, id string) (*models.AddPageNumberJob, error) {
	return s.stg.AddPageNumber().GetByID(ctx, id)
}
