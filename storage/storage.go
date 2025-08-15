package storage

import (
	"context"
	"time"

	"convertpdfgo/api/models"
)

type IStorage interface {
	User() IUserStorage

	Redis() IRedisStorage
	Close()

	Merge() IMergeStorage
	File() IFileStorage
	Split() ISplitStorage
	RemovePage() IRemovePageStorage
	ExtractPage() IExtractPageStorage

	Compress() ICompressStorage
	PDFToJPG() IPDFToJPGStorage
	Rotate() IRotateStorage
	AddPageNumber() IAddPageNumberStorage
	Crop() ICropPDFStorage
	Unlock() IUnlockPDFStorage
	Protect() IProtectStorage
	Stat() IStatStorage
	Log() ILogService
	JPGToPDF() IJPGToPDFStorage

	SharedLink() ISharedLinkStorage

	PDFToWord() IPDFToWordStorage
	WordToPDF() IWordToPDFStorage
	ExcelToPDF() IExcelToPDFStorage
	PowerPointToPDF() IPowerPointToPDFStorage
	AddWatermark() IAddWatermarkStorage
	FileDeletionLog() IFileDeletionLogStorage

	AdminJob() IAdminJobStorage

	JobDownload() JobDownloadStorage
	Contact() IContactStorage

	PublicStats() IPublicStatsStorage
}

type IUserStorage interface {
	Create(ctx context.Context, req models.SignupRequest) (string, error)

	GetForLoginByEmail(ctx context.Context, email string) (models.LoginUser, error)

	GetByID(ctx context.Context, id string) (*models.User, error)

	UpdatePassword(ctx context.Context, userID, newPassword string) error

	GetPasswordByID(ctx context.Context, userID string) (string, error)

	UpdateRole(ctx context.Context, userID, role string) error

	UpdateAvatar(ctx context.Context, userID string, avatar string) error

	UpdateUserPreferences(ctx context.Context, userID, language string, notifications bool) error

	GetUserPreferences(ctx context.Context, userID string) (*models.UserPreferences, error)

	CreatePasswordResetToken(ctx context.Context, userID string) (string, error)

	ValidatePasswordResetToken(ctx context.Context, token string) (string, error)
}

type IRedisStorage interface {
	SetX(ctx context.Context, key string, value interface{}, duration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error // ⬅️ YANGI

}

type IFileStorage interface {
	Save(ctx context.Context, file models.File) (string, error)
	GetByID(ctx context.Context, id string) (models.File, error)
	Delete(ctx context.Context, id string) error
	ListByUser(ctx context.Context, userID string) ([]models.File, error)

	GetOldFiles(ctx context.Context, olderThanDays int) ([]models.OldFile, error)
	DeleteByID(ctx context.Context, id string) error

	GetPendingDeletionFiles(ctx context.Context, expirationMinutes int) ([]models.File, error)

	AdminListFiles(ctx context.Context, f models.AdminFileFilter) ([]models.FileRow, error)
	AdminCountFiles(ctx context.Context, f models.AdminFileFilter) (int64, error)
}
type IMergeStorage interface {
	Create(ctx context.Context, job *models.MergeJob) error
	GetByID(ctx context.Context, id string) (*models.MergeJob, error)
	AddInputFiles(ctx context.Context, jobID string, fileIDs []string) error
	GetInputFiles(ctx context.Context, jobID string) ([]string, error)
	Update(ctx context.Context, job *models.MergeJob) error

	TransitionStatus(ctx context.Context, id, fromStatus, toStatus string) (bool, error)
}

type ISplitStorage interface {
	Create(ctx context.Context, job *models.SplitJob) error
	GetByID(ctx context.Context, id string) (*models.SplitJob, error)
	UpdateOutputFiles(ctx context.Context, jobID string, outputIDs []string) error
}

type IRemovePageStorage interface {
	Create(ctx context.Context, job *models.RemoveJob) error
	Update(ctx context.Context, job *models.RemoveJob) error
	GetByID(ctx context.Context, id string) (*models.RemoveJob, error)
}

type IExtractPageStorage interface {
	Create(ctx context.Context, job *models.ExtractJob) error
	Update(ctx context.Context, job *models.ExtractJob) error
	GetByID(ctx context.Context, id string) (*models.ExtractJob, error)
}

type IOrganizeStorage interface {
	Create(ctx context.Context, job *models.OrganizeJob) error
	Update(ctx context.Context, job *models.OrganizeJob) error
	GetByID(ctx context.Context, id string) (*models.OrganizeJob, error)
}

type ICompressStorage interface {
	Create(ctx context.Context, job *models.CompressJob) error
	Update(ctx context.Context, job *models.CompressJob) error
	GetByID(ctx context.Context, id string) (*models.CompressJob, error)
}

type IPDFToJPGStorage interface {
	Create(ctx context.Context, job *models.PDFToJPGJob) error
	Update(ctx context.Context, job *models.PDFToJPGJob) error
	GetByID(ctx context.Context, id string) (*models.PDFToJPGJob, error)
}

type IRotateStorage interface {
	Create(ctx context.Context, job *models.RotateJob) error
	GetByID(ctx context.Context, id string) (*models.RotateJob, error)
	Update(ctx context.Context, job *models.RotateJob) error
}

type IAddPageNumberStorage interface {
	Create(ctx context.Context, job *models.AddPageNumberJob) error
	GetByID(ctx context.Context, id string) (*models.AddPageNumberJob, error)
	Update(ctx context.Context, job *models.AddPageNumberJob) error
}

type ICropPDFStorage interface {
	Create(ctx context.Context, job *models.CropPDFJob) error
	GetByID(ctx context.Context, id string) (*models.CropPDFJob, error)
	Update(ctx context.Context, job *models.CropPDFJob) error
}

type IUnlockPDFStorage interface {
	Create(ctx context.Context, job *models.UnlockPDFJob) error
	GetByID(ctx context.Context, id string) (*models.UnlockPDFJob, error)
	Update(ctx context.Context, job *models.UnlockPDFJob) error
}
type IProtectStorage interface {
	Create(ctx context.Context, job *models.ProtectPDFJob) error
	GetByID(ctx context.Context, id string) (*models.ProtectPDFJob, error)
	Update(ctx context.Context, job *models.ProtectPDFJob) error
}

type IStatStorage interface {
	GetUserStats(ctx context.Context, userID string) (models.UserStats, error)
}

type ILogService interface {
	GetLogsByJobID(ctx context.Context, jobID string) ([]models.Log, error)
}
type IJPGToPDFStorage interface {
	Create(ctx context.Context, job *models.JPGToPDFJob) error
	GetByID(ctx context.Context, id string) (*models.JPGToPDFJob, error)
	UpdateStatusAndOutput(ctx context.Context, id, status, outputFileID string) error
}

type ISharedLinkStorage interface {
	Create(ctx context.Context, req *models.SharedLink) error
	GetByToken(ctx context.Context, token string) (*models.SharedLink, error)
}

type IPDFToWordStorage interface {
	Create(ctx context.Context, job *models.PDFToWordJob) error
	GetByID(ctx context.Context, id string) (*models.PDFToWordJob, error)
	Update(ctx context.Context, job *models.PDFToWordJob) error
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

type IAddWatermarkStorage interface {
	Create(ctx context.Context, job *models.AddWatermarkJob) error
	GetByID(ctx context.Context, id string) (*models.AddWatermarkJob, error)
	Update(ctx context.Context, job *models.AddWatermarkJob) error
}

type IFileDeletionLogStorage interface {
	LogDeletion(ctx context.Context, log models.FileDeletionLog) error
	GetDeletionLogs(ctx context.Context, limit, offset int) ([]models.FileDeletionLog, error)
}
type IAdminJobStorage interface {
	ListJobs(ctx context.Context, f models.AdminJobFilter) ([]models.JobSummary, error)
}

type JobDownloadStorage interface {
	GetJobFiles(ctx context.Context, jobType, jobID string) (models.JobFilesResult, error)
}
type IContactStorage interface {
	Create(ctx context.Context, req models.ContactCreateRequest, id string) error
	GetByID(ctx context.Context, id string) (*models.ContactMessage, error)
	List(ctx context.Context, onlyUnread bool, limit, offset int) ([]models.ContactMessage, error)
	MarkRead(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

type IPublicStatsStorage interface {
	GetPublicStats(ctx context.Context) (models.PublicStats, error)
}
