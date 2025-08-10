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
	createzipfromfiles "convertpdfgo/pkg/createZipFromFiles"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type PDFToJPGService interface {
	Create(ctx context.Context, req models.PDFToJPGRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.PDFToJPGJob, error)
}

type pdfToJPGService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewPDFToJPGService(stg storage.IStorage, log logger.ILogger) PDFToJPGService {
	return &pdfToJPGService{
		stg: stg,
		log: log,
	}
}

func (s *pdfToJPGService) Create(ctx context.Context, req models.PDFToJPGRequest, userID *string) (string, error) {
	s.log.Info("PDFToJPGService.Create called")

	// PDF faylini saqlash
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("PDF file not found", logger.Error(err))
		return "", fmt.Errorf("failed to fetch PDF file: %w", err)
	}

	// Output faylni saqlash uchun papka yaratish
	jobID := uuid.NewString()
	outputDir := filepath.Join("storage/pdf_to_jpg", jobID)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("Failed to create output directory", logger.Error(err))
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Job yaratish
	job := &models.PDFToJPGJob{
		ID:            jobID,
		UserID:        userID, // UserIDni pointer qilib yuborish
		InputFileID:   req.InputFileID,
		OutputFileIDs: []string{},
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	// Jobni DBga qo'shish
	if err := s.stg.PDFToJPG().Create(ctx, job); err != nil {
		s.log.Error("Failed to create job entry in DB", logger.Error(err))
		return "", fmt.Errorf("failed to create job entry in database: %w", err)
	}

	// PDF -> JPGga aylantirish jarayonini boshlash
	outputPath := filepath.Join(outputDir, "page")
	cmd := exec.Command("pdftoppm", "-jpeg", file.FilePath, outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		s.log.Error("Failed to convert PDF to JPG", logger.Error(err))
		return "", fmt.Errorf("conversion failed: %w", err)
	}

	// JPG fayllarni yig'ish va saqlash
	var jpgPaths []string
	err = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".jpg" {
			jpgPaths = append(jpgPaths, path)

			fileID := uuid.NewString()
			newFile := models.File{
				ID:         fileID,
				UserID:     userID,
				FileName:   filepath.Base(path),
				FilePath:   path,
				FileType:   "image/jpeg",
				FileSize:   info.Size(),
				UploadedAt: time.Now(),
			}
			if _, err := s.stg.File().Save(ctx, newFile); err != nil {
				s.log.Error("Failed to save JPG file", logger.Error(err))
				return err
			}
			job.OutputFileIDs = append(job.OutputFileIDs, fileID)
		}
		return nil
	})
	if err != nil {
		s.log.Error("Error collecting JPG files", logger.Error(err))
		return "", fmt.Errorf("failed to collect JPG files: %w", err)
	}

	// Zip fayl yaratish
	zipPath := filepath.Join("storage/pdf_to_jpg", jobID+".zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		s.log.Error("Failed to create zip file", logger.Error(err))
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	if err := createzipfromfiles.CreateZipFromFiles(zipFile, jpgPaths); err != nil {
		s.log.Error("Failed to write zip", logger.Error(err))
		return "", fmt.Errorf("failed to create zip from JPG files: %w", err)
	}

	// Zip faylni saqlash
	info, err := os.Stat(zipPath)
	if err != nil {
		s.log.Error("Failed to stat zip file", logger.Error(err))
		return "", fmt.Errorf("failed to get stats of zip file: %w", err)
	}

	zipID := uuid.NewString()
	zipModel := models.File{
		ID:         zipID,
		UserID:     userID,
		FileName:   filepath.Base(zipPath),
		FilePath:   zipPath,
		FileType:   "application/zip",
		FileSize:   info.Size(),
		UploadedAt: time.Now(),
	}
	if _, err := s.stg.File().Save(ctx, zipModel); err != nil {
		s.log.Error("Failed to save zip file", logger.Error(err))
		return "", fmt.Errorf("failed to save zip file to storage: %w", err)
	}

	// Jobni yangilash
	job.ZipFileID = &zipID
	job.Status = "done"
	if err := s.stg.PDFToJPG().Update(ctx, job); err != nil {
		s.log.Error("Failed to update job status", logger.Error(err))
		return "", fmt.Errorf("failed to update job status: %w", err)
	}

	s.log.Info("PDF to JPG ZIP completed", logger.String("jobID", jobID))
	return jobID, nil
}

func (s *pdfToJPGService) GetByID(ctx context.Context, id string) (*models.PDFToJPGJob, error) {
	job, err := s.stg.PDFToJPG().GetByID(ctx, id)
	if err != nil {
		s.log.Error("Job not found", logger.Error(err))
		return nil, err
	}
	return job, nil
}
