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
	pkg "convertpdfgo/pkg/string_to_int"
	"convertpdfgo/storage"
)

type RemovePageService interface {
	Create(ctx context.Context, req models.RemovePagesRequest, userID *string) (string, error)
	GetByID(ctx context.Context, id string) (*models.RemoveJob, error)
}

type removePageService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewRemoveService(stg storage.IStorage, log logger.ILogger) RemovePageService {
	return &removePageService{
		stg: stg,
		log: log,
	}
}

func (s *removePageService) Create(ctx context.Context, req models.RemovePagesRequest, userID *string) (string, error) {
	s.log.Info("RemoveService.Create called")

	// 1. Kiruvchi faylni olish
	file, err := s.stg.File().GetByID(ctx, req.InputFileID)
	if err != nil {
		s.log.Error("input file not found", logger.Error(err))
		return "", err
	}

	// 2. Job modelini tayyorlash
	job := &models.RemoveJob{
		ID:            uuid.New().String(),
		UserID:        userID,
		InputFileID:   req.InputFileID,
		PagesToRemove: req.PagesToRemove,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	// 3. Job'ni DB ga yozish
	if err := s.stg.RemovePage().Create(ctx, job); err != nil {
		s.log.Error("failed to create remove job", logger.Error(err))
		return "", err
	}

	// 4. Output fayl uchun papkani yaratish
	outputDir := "storage/remove"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		s.log.Error("failed to create output dir", logger.Error(err))
		return "", err
	}

	inputPath := file.FilePath
	outputID := uuid.New().String()
	outputPath := filepath.Join(outputDir, outputID+".pdf")

	// 5. Sahifalar ro‘yxatini parse qilish
	pageList, err := parsePageList(req.PagesToRemove)
	if err != nil {
		s.log.Error("invalid page list", logger.Error(err))
		return "", err
	}

	pageStrs := pkg.IntSliceToStringSlice(pageList)

	// 6. PDF sahifalarni olib tashlash
	config := model.NewDefaultConfiguration()
	err = api.RemovePagesFile(inputPath, outputPath, pageStrs, config)
	if err != nil {
		s.log.Error("failed to remove pages", logger.Error(err))
		return "", err
	}

	// 7. Yaratilgan output faylni bazaga yozish
	info, err := os.Stat(outputPath)
	if err != nil {
		s.log.Error("cannot stat output file", logger.Error(err))
		return "", err
	}

	newFile := models.File{
		ID:         outputID,
		UserID:     userID,
		FileName:   filepath.Base(outputPath),
		FilePath:   outputPath,
		FileType:   "application/pdf",
		FileSize:   info.Size(),
		UploadedAt: time.Now(),
	}

	_, err = s.stg.File().Save(ctx, newFile)
	if err != nil {
		s.log.Error("failed to save output file", logger.Error(err))
		return "", err
	}

	// 8. Job ni yangilash (output_file_id, status)
	job.OutputFileID = &outputID
	job.Status = "done"

	if err := s.stg.RemovePage().Update(ctx, job); err != nil {
		s.log.Error("failed to update job", logger.Error(err))
		return "", err
	}

	s.log.Info("✅ remove pages completed", logger.String("jobID", job.ID))
	return job.ID, nil
}

func (s *removePageService) GetByID(ctx context.Context, id string) (*models.RemoveJob, error) {
	job, err := s.stg.RemovePage().GetByID(ctx, id)
	if err != nil {
		s.log.Error("failed to get remove job", logger.Error(err))
		return nil, err
	}
	return job, nil
}

// Sahifa raqamlarini ["1", "3-5"] dan []int formatga aylantiruvchi util funksiyasi
func parsePageList(input string) ([]int, error) {
	var pages []int
	tokens := strings.Split(input, ",")
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if strings.Contains(token, "-") {
			rangeParts := strings.Split(token, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range: %s", token)
			}
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 != nil || err2 != nil || start > end {
				return nil, fmt.Errorf("invalid range values: %s", token)
			}
			for i := start; i <= end; i++ {
				pages = append(pages, i)
			}
		} else {
			page, err := strconv.Atoi(token)
			if err != nil {
				return nil, fmt.Errorf("invalid page number: %s", token)
			}
			pages = append(pages, page)
		}
	}
	return pages, nil
}
