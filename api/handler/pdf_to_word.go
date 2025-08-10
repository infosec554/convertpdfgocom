package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// CreatePDFToWordJob godoc
// @Summary      Convert PDF to Word
// @Description  PDF faylni Word formatiga o‘tkazish
// @Tags         PDF to Word
// @Accept       json
// @Produce      json
// @Param        request body models.PDFToWordRequest true "PDF to Word request"
// @Success      201 {object} map[string]string
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/pdf-to-word [post]
func (h *Handler) CreatePDFToWordJob(c *gin.Context) {
	var req models.PDFToWordRequest

	// So‘rovni tekshirish
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, h.log, "invalid request", http.StatusBadRequest, err.Error())
		return
	}

	if len(req.InputFileID) == 0 {
		handleResponse(c, h.log, "no input file", http.StatusBadRequest, "input_file_id required")
		return
	}

	var userID *string
	if uid := c.GetString("user_id"); uid != "" {
		userID = &uid
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	jobID, err := h.services.PDFToWord().Create(ctx, req, userID)
	if err != nil {
		handleResponse(c, h.log, "conversion failed", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "pdf to word job created", http.StatusCreated, gin.H{"id": jobID})
}

// GetPDFToWordJob godoc
// @Summary      Get PDF to Word Job
// @Description  PDF to Word ish holatini ko‘rish
// @Tags         PDF to Word
// @Produce      json
// @Param        id path string true "Job ID"
// @Success      200 {object} models.PDFToWordJob
// @Failure      404 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/pdf-to-word/{id} [get]
func (h *Handler) GetPDFToWordJob(c *gin.Context) {
	jobID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := h.services.PDFToWord().GetByID(ctx, jobID)
	if err != nil {
		handleResponse(c, h.log, "job not found", http.StatusNotFound, err.Error())
		return
	}

	handleResponse(c, h.log, "job found", http.StatusOK, job)
}
