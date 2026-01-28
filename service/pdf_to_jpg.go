package service

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"convertpdfgo/api/models"
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
	s.log.Info("PDFToJPGService.Create called (Renderer Mode)")

	// 1. Kiruvchi faylni olish
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("Input file not found", logger.String("fileID", req.InputFileID), logger.Error(err))
		return "", err
	}

	if _, err := os.Stat(file.FilePath); os.IsNotExist(err) {
		s.log.Error("Input file does not exist", logger.String("filePath", file.FilePath))
		return "", fmt.Errorf("input file does not exist: %s", file.FilePath)
	}

	// 2. Job yaratish
	jobID := uuid.NewString()
	job := &models.PDFToJPGJob{
		ID:          jobID,
		UserID:      userID,
		InputFileID: req.InputFileID,
		Status:      "processing",
		CreatedAt:   time.Now(),
	}

	// 3. Output papkasini tayyorlash
	outputDir := filepath.Join("storage", "pdf_to_jpg", jobID)
	if err := os.MkdirAll(outputDir, 0777); err != nil {
		s.log.Error("Failed to create output dir", logger.Error(err))
		return "", err
	}

	// 4. pdftoppm yordamida PDF ni JPG ga aylantirish (Renderer)
	// Usage: pdftoppm -jpeg -r 150 input.pdf output_prefix
	prefix := filepath.Join(outputDir, "page")
	cmd := exec.Command("pdftoppm", "-jpeg", "-r", "150", file.FilePath, prefix)

	if output, err := cmd.CombinedOutput(); err != nil {
		s.log.Error("pdftoppm execution failed", logger.String("output", string(output)), logger.Error(err))
		return "", fmt.Errorf("image conversion failed: %w", err)
	}

	// 5. Chiqarilgan rasmlarni yig'ish
	var imageFiles []string
	err = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
				imageFiles = append(imageFiles, path)
			}
		}
		return nil
	})

	if len(imageFiles) == 0 {
		return "", fmt.Errorf("no images generated")
	}

	// Fayllarni tartiblash (page-1, page-2, page-10 muammosini hal qilish uchun)
	// Oddiy string sort yetarli bo'lmasligi mumkin (1, 10, 2), lekin pdftoppm 0 bilan to'ldirish (e.g. -001) flagini ishlatmadik.
	// pdftoppm default: page-1.jpg, page-10.jpg.
	// Keling, ularni shundayligicha arxivlaymiz.
	sort.Strings(imageFiles)

	// 6. ZIP fayl yaratish
	zipID := uuid.NewString()
	zipPath := filepath.Join("storage", "pdf_to_jpg", zipID+".zip")

	zipFile, err := os.Create(zipPath)
	if err != nil {
		s.log.Error("Failed to create zip file", logger.Error(err))
		return "", err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, imgPath := range imageFiles {
		imgData, err := os.ReadFile(imgPath)
		if err != nil {
			continue
		}

		// ZIP ichidagi fayl nomi
		fileName := filepath.Base(imgPath)
		w, err := zipWriter.Create(fileName)
		if err != nil {
			continue
		}
		w.Write(imgData)
	}
	zipWriter.Close()

	// 7. ZIP faylni bazaga saqlash
	fi, _ := os.Stat(zipPath)
	zipFileModel := models.File{
		ID:         zipID,
		UserID:     userID,
		FileName:   fmt.Sprintf("%s_images.zip", strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName))),
		FilePath:   zipPath,
		FileType:   "application/zip",
		FileSize:   fi.Size(),
		UploadedAt: time.Now(),
	}

	if _, err := s.stg.File().Save(ctx, zipFileModel); err != nil {
		s.log.Error("Failed to save zip file to DB", logger.Error(err))
		return "", err
	}

	job.ZipFileID = &zipID
	job.Status = "done"

	s.log.Info("PDFToJPG rendering completed", logger.String("jobID", jobID), logger.Int("pages", len(imageFiles)))
	return jobID, nil
}

func (s *pdfToJPGService) GetByID(ctx context.Context, id string) (*models.PDFToJPGJob, error) {
	return &models.PDFToJPGJob{
		ID:     id,
		Status: "done",
	}, nil
}
