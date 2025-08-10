package handler

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	// bu path loyihangizga qarab bo'lishi mumkin

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"convertpdfgo/api/models"
	"convertpdfgo/pkg/faylchek"
)

// UploadFile godoc
// @Summary      Upload file
// @Description  Guest ham, ro‘yxatdan o‘tgan user ham yuklay oladi. (optional auth)
// @Tags         file
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "Upload file"
// @Success      201  {object}  map[string]string
// @Failure      400  {object}  models.Response
// @Failure      500  {object}  models.Response
// @Router       /file [post]
func (h Handler) UploadFile(c *gin.Context) {
	// Auth optional: user_id bo'lishi shart emas
	var ptrUserID *string
	if uid, exists := c.Get("user_id"); exists {
		if strID, ok := uid.(string); ok && strID != "" {
			ptrUserID = &strID
		}
	}

	// Faylni olish
	fileHeader, err := c.FormFile("file")
	if err != nil {
		handleResponse(c, h.log, "file is required", http.StatusBadRequest, err.Error())
		return
	}

	// Fayl parametrlari
	fileName := fileHeader.Filename
	fileType := filepath.Ext(fileName)
	fileSize := fileHeader.Size
	fileID := uuid.NewString()
	savePath := fmt.Sprintf("uploads/%s%s", fileID, fileType)

	// Fayl kengaytmasini tekshirish
	if faylchek.IsBlacklistedExtension(fileType) {
		handleResponse(c, h.log, "❌ Fayl turi xavfli va yuklash taqiqlangan", http.StatusBadRequest, nil)
		return
	}

	if !faylchek.IsAllowedExtension(fileType) {
		handleResponse(c, h.log, "❌ Bu turdagi fayllarni yuklash ruxsat etilmagan", http.StatusBadRequest, nil)
		return
	}
	// 🔒 Fayl hajmi cheklovi
	const guestMaxSize = 20 * 1024 * 1024      // 30 MB
	const registeredMaxSize = 30 * 1024 * 1024 // 50 MB

	if ptrUserID == nil && fileSize > guestMaxSize {
		handleResponse(c, h.log, "Guests can upload files up to 30MB only", http.StatusBadRequest, nil)
		return
	}
	if ptrUserID != nil && fileSize > registeredMaxSize {
		handleResponse(c, h.log, "Registered users can upload files up to 50MB only", http.StatusBadRequest, nil)
		return
	}

	// Faylni saqlash
	if err := c.SaveUploadedFile(fileHeader, savePath); err != nil {
		handleResponse(c, h.log, "failed to save file", http.StatusInternalServerError, err.Error())
		return
	}

	// Bazaga yozish uchun model
	file := models.File{
		ID:         fileID,
		UserID:     ptrUserID, // <-- pointer bo'lishi kerak
		FileName:   fileName,
		FilePath:   savePath,
		FileType:   fileType,
		FileSize:   fileSize,
		UploadedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id, err := h.services.File().Upload(ctx, file)
	if err != nil {
		handleResponse(c, h.log, "upload failed", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "file uploaded", http.StatusCreated, gin.H{"id": id})
}

// GetFile godoc
// @Summary      Get file by ID
// @Tags         file
// @Security     ApiKeyAuth
// @Param        id path string true "File ID"
// @Produce      json
// @Success      200  {object}  models.File
// @Failure      404  {object}  models.Response
// @Router       /file/{id} [get]
func (h Handler) GetFile(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	file, err := h.services.File().Get(ctx, id)
	if err != nil {
		handleResponse(c, h.log, "file not found", http.StatusNotFound, err.Error())
		return
	}

	handleResponse(c, h.log, "file found", http.StatusOK, file)
}

// DeleteFile godoc
// @Summary      Delete file by ID
// @Tags         file
// @Security     ApiKeyAuth
// @Param        id path string true "File ID"
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      500  {object}  models.Response
// @Router       /file/{id} [delete]
func (h Handler) DeleteFile(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.services.File().Delete(ctx, id)
	if err != nil {
		handleResponse(c, h.log, "delete failed", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "file deleted", http.StatusOK, gin.H{"id": id})
}

// ListUserFiles godoc
// @Summary      List all user's files
// @Tags         file
// @Security     ApiKeyAuth
// @Produce      json
// @Param        limit query int false "Limit" default(20)
// @Param        page  query int false "Page"  default(1)
// @Success      200  {array}  models.File
// @Failure      500  {object}  models.Response
// @Router       /file [get]
func (h Handler) ListUserFiles(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		handleResponse(c, h.log, "missing user_id in context", http.StatusUnauthorized, nil)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	files, err := h.services.File().List(ctx, userID)
	if err != nil {
		handleResponse(c, h.log, "list failed", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "user files", http.StatusOK, files)
}

// CleanupOldFiles godoc
// @Summary      Cleanup old files
// @Description  Admin-only endpoint to delete files older than N days
// @Tags         admin, files
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /admin/files/cleanup [post]
func (h *Handler) CleanupOldFiles(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const olderThanDays = 7

	count, err := h.services.File().CleanupOldFiles(ctx, olderThanDays)
	if err != nil {
		handleResponse(c, h.log, "failed to cleanup old files", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "old files cleaned up", http.StatusOK, gin.H{"deleted_files": count})
}

// AdminListPendingDeletionFiles godoc
// @Summary      List files pending deletion
// @Description  List files that are older than expirationMinutes minutes (pending deletion)
// @Tags         admin, files
// @Security     ApiKeyAuth
// @Produce      json
// @Param        expirationMinutes query int false "Expiration time in minutes" default(5)
// @Success      200 {array} models.File
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /admin/files/pending-deletion [get]
func (h *Handler) AdminListPendingDeletionFiles(c *gin.Context) {
	expMinutesStr := c.DefaultQuery("expirationMinutes", "5")
	expMinutes, err := strconv.Atoi(expMinutesStr)
	if err != nil || expMinutes <= 0 {
		handleResponse(c, h.log, "invalid expirationMinutes param", http.StatusBadRequest, nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	files, err := h.services.File().ListPendingDeletionFiles(ctx, expMinutes)
	if err != nil {
		handleResponse(c, h.log, "failed to fetch pending deletion files", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "pending deletion files retrieved", http.StatusOK, files)
}

// AdminListFiles godoc
// @Summary      Admin: list files
// @Description  Foydalanuvchi bo'yicha, qidiruv, sana va guest filterlari bilan fayllar ro'yxati
// @Tags         admin, files
// @Security     ApiKeyAuth
// @Produce      json
// @Param        user_id         query string  false "Filter by user_id"
// @Param        include_guests  query bool    false "Include files with NULL user_id" default(false)
// @Param        q               query string  false "Search in file_name (ILIKE)"
// @Param        from            query string  false "From date (RFC3339)"
// @Param        to              query string  false "To date (RFC3339)"
// @Param        limit           query int     false "Limit"  default(20)
// @Param        offset          query int     false "Offset" default(0)
// @Success      200  {array}    models.FileRow
// @Failure      400  {object}   models.Response
// @Failure      500  {object}   models.Response
// @Router       /admin/files [get]
func (h Handler) AdminListFiles(c *gin.Context) {
	var f models.AdminFileFilter

	if uid := c.Query("user_id"); uid != "" {
		f.UserID = &uid
	}
	f.IncludeGuests = c.DefaultQuery("include_guests", "false") == "true"

	if q := c.Query("q"); q != "" {
		f.Q = &q
	}

	if s := c.Query("from"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			f.DateFrom = &t
		} else {
			handleResponse(c, h.log, "invalid 'from' date (RFC3339)", http.StatusBadRequest, err.Error())
			return
		}
	}
	if s := c.Query("to"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			f.DateTo = &t
		} else {
			handleResponse(c, h.log, "invalid 'to' date (RFC3339)", http.StatusBadRequest, err.Error())
			return
		}
	}

	if limStr := c.DefaultQuery("limit", "20"); limStr != "" {
		if lim, err := strconv.Atoi(limStr); err == nil {
			f.Limit = lim
		}
	}
	if offStr := c.DefaultQuery("offset", "0"); offStr != "" {
		if off, err := strconv.Atoi(offStr); err == nil {
			f.Offset = off
		}
	}

	rows, err := h.services.File().AdminListFiles(c.Request.Context(), f)
	if err != nil {
		handleResponse(c, h.log, "failed to list files", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "ok", http.StatusOK, rows)
}
