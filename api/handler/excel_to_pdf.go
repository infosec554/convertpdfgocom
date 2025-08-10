package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// CreateExcelToPDF godoc
// @Summary      Excel → PDF konvertatsiya qilish
// @Description  Excel faylni PDF formatga o‘tkazish
// @Tags         Excel to PDF
// @Accept       json
// @Produce      json
// @Param        request body models.ExcelToPDFRequest true "Excel fayl IDsi"
// @Success      201 {object} models.Response{data=string} "Yaratilgan job ID"
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/excel-to-pdf [post]
func (h *Handler) CreateExcelToPDF(c *gin.Context) {
	var req models.ExcelToPDFRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, h.log, "invalid request", http.StatusBadRequest, err.Error())
		return
	}
	if len(req.InputFileID) == 0 {
		handleResponse(c, h.log, "no input files", http.StatusBadRequest, "input_file_ids required")
		return
	}
	// Handle guest user (if user_id is empty)
	var userID *string
	if uid := c.GetString("user_id"); uid != "" {
		userID = &uid
	} else {
		// For guest user, we pass nil
		userID = nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	jobID, err := h.services.ExcelToPDF().Create(ctx, req, userID)
	if err != nil {
		handleResponse(c, h.log, "failed to convert Excel to PDF", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "excel to pdf job created", http.StatusCreated, jobID)
}

// GetExcelToPDFJob godoc
// @Summary      Excel→PDF Job holatini olish
// @Description  Job statusini ko‘rish
// @Tags         Excel to PDF
// @Produce      json
// @Param        id path string true "Job ID"
// @Success      200 {object} models.ExcelToPDFJob
// @Failure      404 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/excel-to-pdf/{id} [get]
func (h *Handler) GetExcelToPDFJob(c *gin.Context) {
	jobID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := h.services.ExcelToPDF().GetByID(ctx, jobID)
	if err != nil {
		handleResponse(c, h.log, "job not found", http.StatusNotFound, err.Error())
		return
	}

	handleResponse(c, h.log, "job found", http.StatusOK, job)
}
