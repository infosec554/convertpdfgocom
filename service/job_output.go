package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type FileDownload struct {
	FileID   string
	Path     string
	Name     string
	MimeType string
	Size     int64
}

type DownloadService interface {
	GetPrimary(ctx context.Context, jobType, jobID string) (*FileDownload, error)
}

type downloadService struct {
	stg storage.IStorage
	log logger.ILogger
}

func NewDownloadService(stg storage.IStorage, log logger.ILogger) DownloadService {
	return &downloadService{stg: stg, log: log}
}

// -------------------- PUBLIC --------------------

func (s *downloadService) GetPrimary(ctx context.Context, jobType, jobID string) (*FileDownload, error) {
	jt, err := normalizeType(jobType)
	if err != nil {
		return nil, err
	}
	s.log.Info("Download.GetPrimary", logger.String("type", jt), logger.String("id", jobID))

	// 1) Jobdan asosiy fileIDni topamiz (zip->single->multi[0])
	fileID, err := s.pickFileID(ctx, jt, jobID)
	if err != nil {
		return nil, err
	}

	// 2) File meta
	return s.fileMeta(ctx, fileID)
}

// -------------------- INTERNAL --------------------

func (s *downloadService) pickFileID(ctx context.Context, jt, id string) (string, error) {
	switch jt {

	// ===== Single-output odatiy ishlar =====
	case "merge":
		j, err := s.stg.Merge().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("merge: output not ready")
		}
		return *j.OutputFileID, nil

	case "compress":
		j, err := s.stg.Compress().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("compress: output not ready")
		}
		return *j.OutputFileID, nil

	case "remove-pages":
		j, err := s.stg.RemovePage().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("remove-pages: output not ready")
		}
		return *j.OutputFileID, nil

	case "extract":
		j, err := s.stg.ExtractPage().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			// Senda ba’zan extract ko‘p fayl yozishi mumkin — agar modelda Multi bo‘lsa, shu yerda o‘zgartirasan
			return "", errors.New("extract: output not ready")
		}
		return *j.OutputFileID, nil

	case "rotate":
		j, err := s.stg.Rotate().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("rotate: output not ready")
		}
		return *j.OutputFileID, nil

	case "crop":
		j, err := s.stg.Crop().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("crop: output not ready")
		}
		return *j.OutputFileID, nil

	case "add-page-numbers":
		j, err := s.stg.AddPageNumber().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("add-page-numbers: output not ready")
		}
		return *j.OutputFileID, nil

	case "unlock":
		j, err := s.stg.Unlock().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("unlock: output not ready")
		}
		return *j.OutputFileID, nil

	case "protect":
		j, err := s.stg.Protect().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("protect: output not ready")
		}
		return *j.OutputFileID, nil

	case "jpg-to-pdf":
		j, err := s.stg.JPGToPDF().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("jpg-to-pdf: output not ready")
		}
		return *j.OutputFileID, nil

	case "pdf-to-word":
		j, err := s.stg.PDFToWord().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("pdf-to-word: output not ready")
		}
		return *j.OutputFileID, nil

	case "word-to-pdf":
		j, err := s.stg.WordToPDF().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("word-to-pdf: output not ready")
		}
		return *j.OutputFileID, nil

	case "excel-to-pdf":
		j, err := s.stg.ExcelToPDF().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("excel-to-pdf: output not ready")
		}
		return *j.OutputFileID, nil

	case "ppt-to-pdf":
		j, err := s.stg.PowerPointToPDF().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("ppt-to-pdf: output not ready")
		}
		return *j.OutputFileID, nil

	case "add-watermark":
		j, err := s.stg.AddWatermark().GetByID(ctx, id)
		if err != nil || j == nil || j.OutputFileID == nil || *j.OutputFileID == "" {
			return "", errors.New("add-watermark: output not ready")
		}
		return *j.OutputFileID, nil

	// ===== Ko‘p-output/zip bo‘lishi mumkin bo‘lganlar =====
	case "pdf-to-jpg":
		j, err := s.stg.PDFToJPG().GetByID(ctx, id)
		if err != nil || j == nil {
			return "", errors.New("pdf-to-jpg: job not found")
		}
		// 1) ZIP bor-yo‘q
		if j.ZipFileID != nil && *j.ZipFileID != "" {
			return *j.ZipFileID, nil
		}
		// 2) Multi[0]
		if len(j.OutputFileIDs) > 0 {
			return j.OutputFileIDs[0], nil
		}
		return "", errors.New("pdf-to-jpg: outputs not ready")

	case "split":
		j, err := s.stg.Split().GetByID(ctx, id)
		if err != nil || j == nil {
			return "", errors.New("split: job not found")
		}
		// split ko‘p fayl bo‘lishi mumkin — birinchi faylni beramiz
		// modelinga mos ravishda: agar `output_file_ids` string[] bo'lsa, shu ishlaydi
		if len(j.OutputFileIDs) > 0 {
			return j.OutputFileIDs[0], nil
		}
		return "", errors.New("split: outputs not ready")
	}

	return "", fmt.Errorf("unsupported job type: %s", jt)
}

func (s *downloadService) fileMeta(ctx context.Context, fileID string) (*FileDownload, error) {
	f, err := s.stg.File().GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(f.FilePath)
	if err != nil {
		return nil, fmt.Errorf("file content missing on disk: %w", err)
	}
	name := f.FileName
	if name == "" {
		name = filepath.Base(f.FilePath)
	}
	return &FileDownload{
		FileID:   f.ID,
		Path:     f.FilePath,
		Name:     name,
		MimeType: guessMimeByExt(f.FileType, name),
		Size:     st.Size(),
	}, nil
}

func normalizeType(t string) (string, error) {
	t = strings.TrimSpace(strings.ToLower(t))
	switch t {
	case "merge", "split", "compress",
		"remove-pages", "extract", "rotate", "crop",
		"add-page-numbers", "unlock", "protect",
		"jpg-to-pdf", "pdf-to-jpg", "pdf-to-word",
		"word-to-pdf", "excel-to-pdf", "ppt-to-pdf",
		"add-watermark":
		return t, nil
	default:
		return "", fmt.Errorf("unsupported job type: %s", t)
	}
}

func guessMimeByExt(ext, name string) string {
	if ext == "" {
		ext = filepath.Ext(name)
	}
	ext = strings.ToLower(ext)
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	default:
		return "application/octet-stream"
	}
}
