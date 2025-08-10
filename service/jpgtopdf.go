package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type JPGToPDFService interface {
	CreateJob(ctx context.Context, userID *string, inputFileIDs []string) (string, error)
	GetJobByID(ctx context.Context, id string) (*models.JPGToPDFJob, error)
}

type jpgToPDFService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewJPGToPDFService(stg storage.IStorage, log logger.ILogger) JPGToPDFService {
	return &jpgToPDFService{
		stg: stg,
		log: log,
	}
}

func (s *jpgToPDFService) CreateJob(ctx context.Context, userID *string, inputFileIDs []string) (string, error) {
	s.log.Info("JPGToPDFService.CreateJob called")

	if len(inputFileIDs) == 0 {
		return "", fmt.Errorf("no input files provided")
	}

	var inputPaths []string
	// Retrieve all file paths corresponding to input file IDs
	for _, fileID := range inputFileIDs {
		file, err := s.stg.File().GetByID(ctx, fileID)
		if err != nil {
			s.log.Error("file not found", logger.String("fileID", fileID), logger.Error(err))
			return "", fmt.Errorf("file not found: %s", fileID)
		}
		// Check if the file path exists before proceeding
		if _, err := os.Stat(file.FilePath); os.IsNotExist(err) {
			s.log.Error("Input file does not exist", logger.String("filePath", file.FilePath))
			return "", fmt.Errorf("input file does not exist: %s", file.FilePath)
		}
		inputPaths = append(inputPaths, file.FilePath)
	}

	// Generate a new job ID
	jobID := uuid.New().String()
	job := &models.JPGToPDFJob{
		ID:           jobID,
		UserID:       userID, // Pass nil for guest users
		InputFileIDs: inputFileIDs,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}

	// Create a job entry in DB
	if err := s.stg.JPGToPDF().Create(ctx, job); err != nil {
		s.log.Error("failed to create job entry in DB", logger.Error(err))
		return "", err
	}

	// Prepare output directory for PDF
	outputID := uuid.New().String()
	outputDir := filepath.Join("storage", "jpg_to_pdf")
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("failed to create output directory", logger.Error(err))
		return "", err
	}

	outputPath := filepath.Join(outputDir, outputID+".pdf")

	// Create a PDF from the JPG images
	pdf := gofpdf.New("P", "mm", "A4", "")
	for _, imgPath := range inputPaths {
		pdf.AddPage()
		// Ensure that each image is added to the PDF correctly
		pdf.ImageOptions(imgPath, 10, 10, 190, 0, false, gofpdf.ImageOptions{ImageType: "JPG", ReadDpi: true}, 0, "")
	}

	// Output the generated PDF to a file
	err := pdf.OutputFileAndClose(outputPath)
	if err != nil {
		s.log.Error("failed to generate PDF", logger.Error(err))
		return "", fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Save output PDF file metadata
	fi, err := os.Stat(outputPath)
	if err != nil {
		s.log.Error("cannot stat output PDF", logger.Error(err))
		return "", err
	}

	newFile := models.File{
		ID:         outputID,
		UserID:     userID, // Pass nil for guest users
		FileName:   filepath.Base(outputPath),
		FilePath:   outputPath,
		FileType:   "application/pdf",
		FileSize:   fi.Size(),
		UploadedAt: time.Now(),
	}

	// Save the file information to the storage
	if _, err := s.stg.File().Save(ctx, newFile); err != nil {
		s.log.Error("failed to save output file", logger.Error(err))
		return "", err
	}

	// Update job status and output file ID in DB
	job.OutputFileID = &outputID
	job.Status = "done"
	if err := s.stg.JPGToPDF().UpdateStatusAndOutput(ctx, job.ID, job.Status, *job.OutputFileID); err != nil {
		s.log.Error("failed to update job status in DB", logger.Error(err))
		return "", err
	}

	s.log.Info("JPG to PDF conversion job completed", logger.String("jobID", jobID))
	return jobID, nil
}

func (s *jpgToPDFService) GetJobByID(ctx context.Context, id string) (*models.JPGToPDFJob, error) {
	job, err := s.stg.JPGToPDF().GetByID(ctx, id)
	if err != nil {
		s.log.Error("failed to get job by ID", logger.Error(err))
		return nil, err
	}
	return job, nil
}
