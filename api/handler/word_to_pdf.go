package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// CreateWordToPDF godoc
// @Summary      Convert Word to PDF
// @Description  Word hujjatni PDF formatga o‘tkazish
// @Tags         Word to PDF
// @Accept       json
// @Produce      json
// @Param        request body models.WordToPDFRequest true "Word to PDF so‘rovi"
// @Success      201 {object} map[string]string
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/word-to-pdf [post]
func (h *Handler) CreateWordToPDF(c *gin.Context) {
	var req models.WordToPDFRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, h.log, "invalid request", http.StatusBadRequest, err.Error())
		return
	}

	if len(req.InputFileID) == 0 {
		handleResponse(c, h.log, "input_file_id required", http.StatusBadRequest, nil)
		return
	}

	// Foydalanuvchi ID sini olish
	var userID *string
	if uid := c.GetString("user_id"); uid != "" {
		userID = &uid
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	jobID, err := h.services.WordToPDF().Create(ctx, req, userID)
	if err != nil {
		handleResponse(c, h.log, "conversion failed", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "conversion started", http.StatusCreated, gin.H{"id": jobID})
}

// GetWordToPDFJob godoc
// @Summary      Get Word→PDF Job Status
// @Description  Konvertatsiya holatini olish
// @Tags         Word to PDF
// @Produce      json
// @Param        id path string true "Job ID"
// @Success      200 {object} models.WordToPDFJob
// @Failure      404 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/word-to-pdf/{id} [get]
func (h *Handler) GetWordToPDFJob(c *gin.Context) {
	jobID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := h.services.WordToPDF().GetByID(ctx, jobID)
	if err != nil {
		handleResponse(c, h.log, "job not found", http.StatusNotFound, err.Error())
		return
	}

	handleResponse(c, h.log, "job found", http.StatusOK, job)
}
