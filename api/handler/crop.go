package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// CreateCropJob godoc
// @Summary      Crop PDF pages
// @Description  PDF sahifalarini belgilangan tomonlardan qirqadi (top, bottom, left, right)
// @Tags         PDF Edit
// @Accept       json
// @Produce      json
// @Param        request body models.CropPDFRequest true "Crop request"
// @Success      201 {object} map[string]string
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/crop [post]
func (h *Handler) CreateCropJob(c *gin.Context) {
	var req models.CropPDFRequest

	// JSONni binding qilish
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, h.log, "invalid request", http.StatusBadRequest, err.Error())
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

	// Crop job yaratish uchun kerakli kontekstni o'rnatish
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Crop jobni yaratish
	jobID, err := h.services.Crop().Create(ctx, req, userID)
	if err != nil {
		handleResponse(c, h.log, "crop job failed", http.StatusInternalServerError, err.Error())
		return
	}

	// Yaratilgan job haqida response
	handleResponse(c, h.log, "crop job created", http.StatusCreated, gin.H{"id": jobID})
}

// GetCropJob godoc
// @Summary      Get Crop Job info
// @Description  Crop PDF jarayonining natijasini olish
// @Tags         PDF Edit
// @Produce      json
// @Param        id path string true "Job ID"
// @Success      200 {object} models.CropPDFJob
// @Failure      404 {object} models.Response
// @Router       /pdf/crop/{id} [get]
func (h *Handler) GetCropJob(c *gin.Context) {
	jobID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := h.services.Crop().GetByID(ctx, jobID)
	if err != nil {
		handleResponse(c, h.log, "crop job not found", http.StatusNotFound, err.Error())
		return
	}

	handleResponse(c, h.log, "crop job found", http.StatusOK, job)
}
