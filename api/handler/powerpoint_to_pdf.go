package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// CreatePowerPointToPDF godoc
// @Summary      Convert PowerPoint to PDF
// @Description  PowerPoint faylni PDF formatga o‘tkazish
// @Tags         PowerPoint to PDF
// @Accept       json
// @Produce      json
// @Param        data body models.PowerPointToPDFRequest true "PowerPoint to PDF request"
// @Success      201 {object} models.Response{data=string} "job_id qaytadi"
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/ppt-to-pdf [post]
func (h *Handler) CreatePowerPointToPDF(c *gin.Context) {
	var req models.PowerPointToPDFRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, h.log, "invalid request body", http.StatusBadRequest, err.Error())
		return
	}

	// Check if inputFileIDs are provided
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

	jobID, err := h.services.PowerPointToPDF().Create(ctx, req, userID)
	if err != nil {
		handleResponse(c, h.log, "conversion failed", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "conversion started", http.StatusCreated, jobID)
}

// GetPowerPointToPDFJob godoc
// @Summary      Get PowerPoint→PDF Job Status
// @Description  Konvertatsiya holatini olish
// @Tags         PowerPoint to PDF
// @Produce      json
// @Param        id path string true "Job ID"
// @Success      200 {object} models.PowerPointToPDFJob
// @Failure      404 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/ppt-to-pdf/{id} [get]
func (h *Handler) GetPowerPointToPDFJob(c *gin.Context) {
	jobID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := h.services.PowerPointToPDF().GetByID(ctx, jobID)
	if err != nil {
		handleResponse(c, h.log, "job not found", http.StatusNotFound, err.Error())
		return
	}

	handleResponse(c, h.log, "job found", http.StatusOK, job)
}
