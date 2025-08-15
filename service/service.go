package service

import (
	// tgbotapi import qilinmoqda

	"convertpdfgo/config"
	"convertpdfgo/pkg/gotenberg"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/pkg/mailer"
	"convertpdfgo/storage"
)

type IServiceManager interface {
	User() UserService
	Mailer() MailerService
	Redis() RedisService
	Stat() StatsService
	Log() LogService
	File() FileService

	Merge() MergeService
	Split() SplitService
	RemovePage() RemovePageService
	ExtractPage() ExtractPageService
	Compress() CompressService
	PDFToJPG() PDFToJPGService
	Rotate() RotateService
	AddPageNumber() AddPageNumberService
	Crop() CropPDFService
	Unlock() UnlockService
	Protect() ProtectPDFService
	JPGToPDF() JPGToPDFService
	SharedLink() SharedLinkService
	PDFToWord() PDFToWordService
	WordToPDF() WordToPDFService
	ExcelToPDF() ExcelToPDFService
	PowerPointToPDF() PowerPointToPDFService
	AddWatermark() AddWatermarkService

	Google() GoogleService
	Github() GithubService
	Facebook() FacebookService

	FileDeletionLog() FileDeletionLogService
	AdminJob() AdminJobService
	Download() DownloadService
	Contact() ContactService
	PublicStats() PublicStatsService
}

type service struct {
	userService          UserService
	mailer               MailerService
	mergeService         MergeService
	fileService          FileService
	splitService         SplitService
	removepageService    RemovePageService
	extractPageService   ExtractPageService
	compressService      CompressService
	pdfToJPGService      PDFToJPGService
	rotateSrvice         RotateService
	addPageNumberService AddPageNumberService
	cropPDFService       CropPDFService
	unlockService        UnlockService
	protectPDFService    ProtectPDFService
	statsService         StatsService
	logService           LogService
	jPGToPDF             JPGToPDFService
	sharedLinkService    SharedLinkService
	pdfToWordService     PDFToWordService
	wordToPDFService     WordToPDFService
	excelToPDF           ExcelToPDFService
	powerPointToPDF      PowerPointToPDFService
	addWatermark         AddWatermarkService
	redisService         RedisService
	googleService        GoogleService
	githubService        GithubService
	facebookService      FacebookService
	fileDeletionLog      FileDeletionLogService
	adminJobService      AdminJobService
	downloadService      DownloadService
	contact              ContactService
	publicStatsService   PublicStatsService
}

func New(storage storage.IStorage, log logger.ILogger, mailerCore *mailer.Mailer, redis storage.IRedisStorage, gotClient gotenberg.Client, googleCfg config.OAuthProviderConfig) IServiceManager {
	return &service{
		userService:          NewUserService(storage, log, mailerCore),
		mailer:               NewMailerService(mailerCore),
		mergeService:         NewMergeService(storage, log),
		fileService:          NewFileService(storage, log),
		splitService:         NewSplitService(storage, log),
		removepageService:    NewRemoveService(storage, log),
		extractPageService:   NewExtractService(storage, log),
		compressService:      NewCompressService(storage, log),
		pdfToJPGService:      NewPDFToJPGService(storage, log),
		rotateSrvice:         NewRotateService(storage, log),
		addPageNumberService: NewAddPageNumberService(storage, log),
		cropPDFService:       NewCropPDFService(storage, log),
		unlockService:        NewUnlockService(storage, log),
		protectPDFService:    NewProtectPDFService(storage, log),
		statsService:         NewStatsService(storage, log),
		logService:           NewLogService(storage, log),
		jPGToPDF:             NewJPGToPDFService(storage, log),
		sharedLinkService:    NewSharedLinkService(storage, log),
		pdfToWordService:     NewPDFToWordService(storage, log),
		wordToPDFService:     NewWordToPDFService(storage, log, gotClient),
		excelToPDF:           NewExcelToPDFService(storage, log, gotClient),
		powerPointToPDF:      NewPowerPointToPDFService(storage, log, gotClient),
		addWatermark:         NewAddWatermarkService(storage, log),
		redisService:         NewRedisService(redis, log),
		googleService:        NewGoogleService(GoogleOAuthConfig(googleCfg)), // <-- config ni uzatish!
		githubService:        NewGithubService(GithubOAuthConfig(googleCfg)),
		facebookService:      NewFacebookService(FacebookOAuthConfig(googleCfg)),
		fileDeletionLog:      NewFileDeletionLogService(storage, log),
		adminJobService:      NewAdminJobService(storage, log),
		downloadService:      NewDownloadService(storage, log),
		contact:              NewContactService(storage, log),
		publicStatsService:   *NewPublicStatsService(storage, log),
	}
}

func (s *service) User() UserService {
	return s.userService
}

func (s *service) Mailer() MailerService {
	return s.mailer
}

func (s *service) Merge() MergeService {
	return s.mergeService
}

func (s *service) File() FileService {
	return s.fileService
}

func (s *service) Split() SplitService {
	return s.splitService
}

func (s *service) RemovePage() RemovePageService {
	return s.removepageService
}

func (s *service) ExtractPage() ExtractPageService {
	return s.extractPageService
}

func (s *service) Compress() CompressService {
	return s.compressService
}

func (s *service) PDFToJPG() PDFToJPGService {
	return s.pdfToJPGService
}

func (s *service) Rotate() RotateService {
	return s.rotateSrvice
}

func (s *service) AddPageNumber() AddPageNumberService {
	return s.addPageNumberService
}

func (s *service) Crop() CropPDFService {
	return s.cropPDFService
}

func (s *service) Unlock() UnlockService {
	return s.unlockService
}

func (s *service) Protect() ProtectPDFService {
	return s.protectPDFService
}

func (s *service) Stat() StatsService {
	return s.statsService
}

func (s *service) Log() LogService {
	return s.logService
}

func (s *service) JPGToPDF() JPGToPDFService {
	return s.jPGToPDF
}

func (s *service) SharedLink() SharedLinkService {
	return s.sharedLinkService
}

func (s *service) PDFToWord() PDFToWordService {
	return s.pdfToWordService
}

func (s *service) WordToPDF() WordToPDFService {
	return s.wordToPDFService
}

func (s *service) ExcelToPDF() ExcelToPDFService {
	return s.excelToPDF
}

func (s *service) PowerPointToPDF() PowerPointToPDFService {
	return s.powerPointToPDF
}

func (s *service) AddWatermark() AddWatermarkService {
	return s.addWatermark
}
func (s *service) Redis() RedisService {
	return s.redisService
}

func (s *service) Google() GoogleService {
	return s.googleService
}

func (s *service) Github() GithubService {
	return s.githubService
}

func (s *service) Facebook() FacebookService {
	return s.facebookService
}

func (s *service) FileDeletionLog() FileDeletionLogService {
	return s.fileDeletionLog
}

func (s *service) AdminJob() AdminJobService {
	return s.adminJobService
}
func (s *service) Download() DownloadService {
	return s.downloadService
}
func (s *service) Contact() ContactService {
	return s.contact
}

func (s *service) PublicStats() PublicStatsService {
	return s.publicStatsService
}
