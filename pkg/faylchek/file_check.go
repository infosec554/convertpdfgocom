package faylchek

import "strings"

// Ruxsat etilgan kengaytmalar ro‘yxati (faqat shu turlar yuklanadi)
var AllowedExtensions = map[string]bool{
	".pdf":  true,
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".docx": true,
	".xlsx": true,
	".txt":  true,
	".pttx": true,
	".pptx":      true,
}

// Bloklangan kengaytmalar ro‘yxati (aniq rad qilinadi, xavfli fayllar)
var BlacklistedExtensions = map[string]bool{
	".exe":  true,
	".bat":  true,
	".sh":   true,
	".js":   true,
	".php":  true,
	".html": true,
}

// Fayl kengaytmasi ruxsat etilganmi?
func IsAllowedExtension(ext string) bool {
	ext = strings.ToLower(ext) // .PDF → .pdf
	return AllowedExtensions[ext]
}

// Fayl xavfli (bloklangan) turga tegishlimi?
func IsBlacklistedExtension(ext string) bool {
	ext = strings.ToLower(ext)
	return BlacklistedExtensions[ext]
}
