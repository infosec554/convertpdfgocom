package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// AddTextWatermark godoc
// @Summary      Add text watermark to PDF
// @Description  Add a watermark text to specific pages of a PDF file
// @Tags         pdf-watermark
// @Accept       json
// @Produce      json
// @Param        request body models.AddWatermarkRequest true "Watermark request body"
// @Success      200 {object} map[string]string
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/watermark [post]
func (h Handler) AddTextWatermark(c *gin.Context) {
	var req models.AddWatermarkRequest
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	jobID, err := h.services.AddWatermark().Create(ctx, req, userID)
	if err != nil {
		handleResponse(c, h.log, "failed to create watermark job", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "watermark job created", http.StatusOK, gin.H{
		"id": jobID,
	})
}

// GetWatermarkJob godoc
// @Summary      Get watermark job by ID
// @Description  Retrieve watermark job details and status
// @Tags         pdf-watermark
// @Accept       json
// @Produce      json
// @Param        id path string true "Watermark Job ID"
// @Success      200 {object} models.AddWatermarkJob
// @Failure      400 {object} models.Response
// @Failure      404 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/watermark/{id} [get]
func (h Handler) GetWatermarkJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		handleResponse(c, h.log, "missing id", http.StatusBadRequest, "id param is required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	job, err := h.services.AddWatermark().GetByID(ctx, id)
	if err != nil {
		handleResponse(c, h.log, "failed to get watermark job", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "watermark job retrieved", http.StatusOK, job)
}
