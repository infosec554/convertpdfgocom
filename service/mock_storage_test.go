package service

import (
	"context"
	"convertpdfgo/api/models"
	"convertpdfgo/storage"
)

// MockStorage implements storage.IStorage
type MockStorage struct {
	FileImpl      *MockFileStorage
	WordToPDFImpl *MockWordToPDFStorage
	// Add others if needed
}

func (m *MockStorage) Close()                               {}
func (m *MockStorage) File() storage.IFileStorage           { return m.FileImpl }
func (m *MockStorage) WordToPDF() storage.IWordToPDFStorage { return m.WordToPDFImpl }

// Implement verify unused interfaces to satisfy compiler
func (m *MockStorage) Compress() storage.ICompressStorage               { return nil }
func (m *MockStorage) JPGToPDF() storage.IJPGToPDFStorage               { return nil }
func (m *MockStorage) ExcelToPDF() storage.IExcelToPDFStorage           { return nil }
func (m *MockStorage) PowerPointToPDF() storage.IPowerPointToPDFStorage { return nil }
func (m *MockStorage) PublicStats() storage.IPublicStatsStorage         { return nil }
func (m *MockStorage) BotUser() storage.IBotUserStorage                 { return nil }
func (m *MockStorage) Merge() storage.IMergeStorage                     { return nil }
func (m *MockStorage) Split() storage.ISplitStorage                     { return nil }
func (m *MockStorage) Rotate() storage.IRotateStorage                   { return nil }
func (m *MockStorage) Watermark() storage.IWatermarkStorage             { return nil }
func (m *MockStorage) Unlock() storage.IUnlockStorage                   { return nil }
func (m *MockStorage) PDFToJPG() storage.IPDFToJPGStorage               { return nil }
func (m *MockStorage) Protect() storage.IProtectStorage                 { return nil }

// MockFileStorage
type MockFileStorage struct {
	GetByIDFunc func(ctx context.Context, id string) (models.File, error)
	SaveFunc    func(ctx context.Context, file models.File) (string, error)
}

func (m *MockFileStorage) GetByID(ctx context.Context, id string) (models.File, error) {
	return m.GetByIDFunc(ctx, id)
}
func (m *MockFileStorage) Save(ctx context.Context, file models.File) (string, error) {
	return m.SaveFunc(ctx, file)
}

// Stubs for other methods
func (m *MockFileStorage) Delete(ctx context.Context, id string) error { return nil }
func (m *MockFileStorage) ListByUser(ctx context.Context, userID string) ([]models.File, error) {
	return nil, nil
}
func (m *MockFileStorage) GetOldFiles(ctx context.Context, olderThanDays int) ([]models.OldFile, error) {
	return nil, nil
}
func (m *MockFileStorage) DeleteByID(ctx context.Context, id string) error { return nil }
func (m *MockFileStorage) GetPendingDeletionFiles(ctx context.Context, expirationMinutes int) ([]models.File, error) {
	return nil, nil
}

// MockWordToPDFStorage
type MockWordToPDFStorage struct {
	CreateFunc  func(ctx context.Context, job *models.WordToPDFJob) error
	GetByIDFunc func(ctx context.Context, id string) (*models.WordToPDFJob, error)
}

func (m *MockWordToPDFStorage) Create(ctx context.Context, job *models.WordToPDFJob) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, job)
	}
	return nil
}
func (m *MockWordToPDFStorage) GetByID(ctx context.Context, id string) (*models.WordToPDFJob, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *MockWordToPDFStorage) Update(ctx context.Context, job *models.WordToPDFJob) error {
	return nil
}

// MockGotenbergClient
type MockGotenbergClient struct {
	WordToPDFFunc func(ctx context.Context, wordPath string) ([]byte, error)
}

func (m *MockGotenbergClient) WordToPDF(ctx context.Context, wordPath string) ([]byte, error) {
	if m.WordToPDFFunc != nil {
		return m.WordToPDFFunc(ctx, wordPath)
	}
	return []byte("dummy pdf content"), nil
}

// Stubs
func (m *MockGotenbergClient) PDFToWord(ctx context.Context, pdfPath string) ([]byte, error) {
	return nil, nil
}
func (m *MockGotenbergClient) ExcelToPDF(ctx context.Context, excelPath string) ([]byte, error) {
	return nil, nil
}
func (m *MockGotenbergClient) PowerPointToPDF(ctx context.Context, pptPath string) ([]byte, error) {
	return nil, nil
}
func (m *MockGotenbergClient) HTMLToPDF(ctx context.Context, htmlPath string) ([]byte, error) {
	return nil, nil
}
