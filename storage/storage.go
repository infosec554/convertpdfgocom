package storage

import (
	"context"
	"time"

	"convertpdfgo/api/models"
)

type IStorage interface {
	Close()
	File() IFileStorage
	Compress() ICompressStorage
	JPGToPDF() IJPGToPDFStorage
	WordToPDF() IWordToPDFStorage
	ExcelToPDF() IExcelToPDFStorage
	PowerPointToPDF() IPowerPointToPDFStorage
	PublicStats() IPublicStatsStorage
	BotUser() IBotUserStorage
	Merge() IMergeStorage
	Split() ISplitStorage
	Rotate() IRotateStorage
	Watermark() IWatermarkStorage
	Unlock() IUnlockStorage
	PDFToJPG() IPDFToJPGStorage // Yangi
	Protect() IProtectStorage   // Protect PDF
}

type IMergeStorage interface {
	Create(ctx context.Context, job models.MergeJob) (string, error)
	Get(ctx context.Context, id string) (*models.MergeJob, error)
	Update(ctx context.Context, job models.MergeJob) error
} // + method to postgres implementation

type IBotUserStorage interface {
	CreateOrUpdate(ctx context.Context, user models.BotUser) error
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.BotUser, error)
	AddDailyBonus(ctx context.Context, amount int) error
	GetByUsername(ctx context.Context, username string) (*models.BotUser, error)
	DeductCoin(ctx context.Context, telegramID int64) error
	AddCoins(ctx context.Context, telegramID int64, amount int) error
	AddReferral(ctx context.Context, referrerID, referredID int64) error
	GetReferralCount(ctx context.Context, telegramID int64) (int, error)
	SetLanguage(ctx context.Context, telegramID int64, lang string) error
	SetCurrentAction(ctx context.Context, telegramID int64, action string) error
	GetTopUsers(ctx context.Context, limit int) ([]models.BotUser, error)
	GetAllUsers(ctx context.Context) ([]models.BotUser, error)
	SetPremium(ctx context.Context, telegramID int64, until time.Time) error
	IncrementUsage(ctx context.Context, telegramID int64) error
}

type IFileStorage interface {
	Save(ctx context.Context, file models.File) (string, error)
	GetByID(ctx context.Context, id string) (models.File, error)
	Delete(ctx context.Context, id string) error
	ListByUser(ctx context.Context, userID string) ([]models.File, error)
	GetOldFiles(ctx context.Context, olderThanDays int) ([]models.OldFile, error)
	DeleteByID(ctx context.Context, id string) error
	GetPendingDeletionFiles(ctx context.Context, expirationMinutes int) ([]models.File, error)
}

type ICompressStorage interface {
	Create(ctx context.Context, job *models.CompressJob) error
	Update(ctx context.Context, job *models.CompressJob) error
	GetByID(ctx context.Context, id string) (*models.CompressJob, error)
}

type IJPGToPDFStorage interface {
	Create(ctx context.Context, job *models.JPGToPDFJob) error
	GetByID(ctx context.Context, id string) (*models.JPGToPDFJob, error)
	UpdateStatusAndOutput(ctx context.Context, id, status, outputFileID string) error
}

type IWordToPDFStorage interface {
	Create(ctx context.Context, job *models.WordToPDFJob) error
	GetByID(ctx context.Context, id string) (*models.WordToPDFJob, error)
	Update(ctx context.Context, job *models.WordToPDFJob) error
}

type IExcelToPDFStorage interface {
	Create(ctx context.Context, job *models.ExcelToPDFJob) error
	GetByID(ctx context.Context, id string) (*models.ExcelToPDFJob, error)
	Update(ctx context.Context, job *models.ExcelToPDFJob) error
}

type IPowerPointToPDFStorage interface {
	Create(ctx context.Context, job *models.PowerPointToPDFJob) error
	GetByID(ctx context.Context, id string) (*models.PowerPointToPDFJob, error)
	Update(ctx context.Context, job *models.PowerPointToPDFJob) error
}

type IPublicStatsStorage interface {
	GetPublicStats(ctx context.Context) (models.PublicStats, error)
}

// ISplitStorage - PDF Split job uchun storage interface
type ISplitStorage interface {
	Create(ctx context.Context, job *models.SplitJob) error
	GetByID(ctx context.Context, id string) (*models.SplitJob, error)
	Update(ctx context.Context, job *models.SplitJob) error
}

// IRotateStorage - PDF Rotate job uchun storage interface
type IRotateStorage interface {
	Create(ctx context.Context, job *models.RotateJob) error
	GetByID(ctx context.Context, id string) (*models.RotateJob, error)
	Update(ctx context.Context, job *models.RotateJob) error
}

// IWatermarkStorage - PDF Watermark job uchun storage interface
type IWatermarkStorage interface {
	Create(ctx context.Context, job *models.WatermarkJob) error
	GetByID(ctx context.Context, id string) (*models.WatermarkJob, error)
	Update(ctx context.Context, job *models.WatermarkJob) error
}

// IUnlockStorage - PDF Unlock job uchun storage interface
type IUnlockStorage interface {
	Create(ctx context.Context, job *models.UnlockJob) error
	GetByID(ctx context.Context, id string) (*models.UnlockJob, error)
	Update(ctx context.Context, job *models.UnlockJob) error
}

// IPDFToJPGStorage - PDF to JPG job uchun storage interface
type IPDFToJPGStorage interface {
	Create(ctx context.Context, job *models.PDFToJPGJob) error
	GetByID(ctx context.Context, id string) (*models.PDFToJPGJob, error)
	Update(ctx context.Context, job *models.PDFToJPGJob) error
}

// IProtectStorage - Protect PDF job uchun storage interface
type IProtectStorage interface {
	Create(ctx context.Context, job *models.ProtectPDFJob) error
	GetByID(ctx context.Context, id string) (*models.ProtectPDFJob, error)
	Update(ctx context.Context, job *models.ProtectPDFJob) error
}
