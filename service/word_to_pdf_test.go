package service

import (
	"context"
	"convertpdfgo/api/models"
	"convertpdfgo/pkg/logger"
	"errors"
	"os"
	"testing"
)

func TestWordToPDFService_Create(t *testing.T) {
	// Setup Mocks
	mockFileStorage := &MockFileStorage{}
	mockWordStorage := &MockWordToPDFStorage{}
	mockStorage := &MockStorage{
		FileImpl:      mockFileStorage,
		WordToPDFImpl: mockWordStorage,
	}
	mockGotClient := &MockGotenbergClient{}
	testLogger := logger.New("test") // Assuming simple constructor

	service := NewWordToPDFService(mockStorage, testLogger, mockGotClient)

	// Create dummy input file
	dummyInputFile := "test_input.docx"
	os.WriteFile(dummyInputFile, []byte("dummy word content"), 0644)
	defer os.Remove(dummyInputFile)

	// Test Case 1: Success
	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		userID := "user123"
		req := models.WordToPDFRequest{InputFileID: "file123"}

		// Mock expectations
		mockFileStorage.GetByIDFunc = func(ctx context.Context, id string) (models.File, error) {
			if id == "file123" {
				return models.File{
					ID:       "file123",
					FilePath: dummyInputFile,
				}, nil
			}
			return models.File{}, errors.New("not found")
		}

		mockGotClient.WordToPDFFunc = func(ctx context.Context, wordPath string) ([]byte, error) {
			if wordPath != dummyInputFile {
				t.Errorf("expected word path %s, got %s", dummyInputFile, wordPath)
			}
			return []byte("%PDF-1.4..."), nil
		}

		mockFileStorage.SaveFunc = func(ctx context.Context, file models.File) (string, error) {
			if file.FileType != "application/pdf" {
				t.Errorf("expected file type application/pdf, got %s", file.FileType)
			}
			return "new_file_id", nil
		}

		mockWordStorage.CreateFunc = func(ctx context.Context, job *models.WordToPDFJob) error {
			if job.Status != "done" {
				t.Errorf("expected status done, got %s", job.Status)
			}
			return nil
		}

		// Execute
		jobID, err := service.Create(ctx, req, &userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if jobID == "" {
			t.Error("expected jobID, got empty")
		}
		
		// Cleanup created file (file path is generated with UUID inside service)
		// Since we don't know the exact UUID, we might have to clean the whole folder or ignore.
		// Detailed cleanup is skipped for now, but in real world we should tracking it.
	})

	// Test Case 2: Input File Not Found
	t.Run("InputFileNotFound", func(t *testing.T) {
		ctx := context.Background()
		req := models.WordToPDFRequest{InputFileID: "missing_file"}

		mockFileStorage.GetByIDFunc = func(ctx context.Context, id string) (models.File, error) {
			return models.File{}, errors.New("db error")
		}

		_, err := service.Create(ctx, req, nil)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
