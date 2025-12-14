package service

import (
	"convertpdfgo/pkg/gotenberg"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type IServiceManager interface {
	File() FileService
	Compress() CompressService
	JPGToPDF() JPGToPDFService
	WordToPDF() WordToPDFService
	ExcelToPDF() ExcelToPDFService
	PowerPointToPDF() PowerPointToPDFService
	PublicStats() *PublicStatsService
	BotUser() BotUserService
	Merge() MergeService
	Split() SplitService
	Rotate() RotateService
	Watermark() WatermarkService
	Unlock() UnlockService
	PDFToJPG() PDFToJPGService
	Protect() ProtectService
}

type serviceManager struct {
	file            FileService
	compress        CompressService
	wordToPDF       WordToPDFService
	excelToPDF      ExcelToPDFService
	powerPointToPDF PowerPointToPDFService
	jpgToPDF        JPGToPDFService
	publicStats     *PublicStatsService
	botUser         BotUserService
	merge           MergeService
	split           SplitService
	rotate          RotateService
	watermark       WatermarkService
	unlock          UnlockService
	pdfToJPG        PDFToJPGService
	protect         ProtectService
}

func New(stg storage.IStorage, log logger.ILogger, gotClient gotenberg.Client) IServiceManager {
	return &serviceManager{
		file:            NewFileService(stg, log),
		compress:        NewCompressService(stg, log),
		wordToPDF:       NewWordToPDFService(stg, log, gotClient),
		excelToPDF:      NewExcelToPDFService(stg, log, gotClient),
		powerPointToPDF: NewPowerPointToPDFService(stg, log, gotClient),
		jpgToPDF:        NewJPGToPDFService(stg, log),
		publicStats:     NewPublicStatsService(stg, log),
		botUser:         NewBotUserService(stg, log),
		merge:           NewMergeService(stg, log),
		split:           NewSplitService(stg, log),
		rotate:          NewRotateService(stg, log),
		watermark:       NewWatermarkService(stg, log),
		unlock:          NewUnlockService(stg, log),
		pdfToJPG:        NewPDFToJPGService(stg, log),
		protect:         NewProtectService(stg, log),
	}
}

func (s *serviceManager) File() FileService {
	return s.file
}

func (s *serviceManager) Compress() CompressService {
	return s.compress
}

func (s *serviceManager) JPGToPDF() JPGToPDFService {
	return s.jpgToPDF
}

func (s *serviceManager) WordToPDF() WordToPDFService {
	return s.wordToPDF
}

func (s *serviceManager) ExcelToPDF() ExcelToPDFService {
	return s.excelToPDF
}

func (s *serviceManager) PowerPointToPDF() PowerPointToPDFService {
	return s.powerPointToPDF
}

func (s *serviceManager) PublicStats() *PublicStatsService {
	return s.publicStats
}

func (s *serviceManager) BotUser() BotUserService {
	return s.botUser
}

func (s *serviceManager) Merge() MergeService {
	return s.merge
}

func (s *serviceManager) Split() SplitService {
	return s.split
}

func (s *serviceManager) Rotate() RotateService {
	return s.rotate
}

func (s *serviceManager) Watermark() WatermarkService {
	return s.watermark
}

func (s *serviceManager) Unlock() UnlockService {
	return s.unlock
}

func (s *serviceManager) PDFToJPG() PDFToJPGService {
	return s.pdfToJPG
}

func (s *serviceManager) Protect() ProtectService {
	return s.protect
}
