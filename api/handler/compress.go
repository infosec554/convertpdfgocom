package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// CreateCompressJob godoc
// @Router       /pdf/compress [POST]
// @Summary      Create compress job
// @Tags         pdf-compress
// @Accept       json
// @Produce      json
// @Param        request body models.CompressRequest true "compress job"
// @Success      201 {object} map[string]string
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
func (h *Handler) CreateCompressJob(c *gin.Context) {
	var req models.CompressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, h.log, "invalid request", http.StatusBadRequest, err.Error())
		return
	}

	// Validate compression level
	if req.Compression != models.Low && req.Compression != models.Medium && req.Compression != models.High {
		handleResponse(c, h.log, "invalid compression level", http.StatusBadRequest, "compression must be low, medium, or high")
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

	// Set timeout for processing
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Create the compression job
	jobID, err := h.services.Compress().Create(ctx, req, userID)
	if err != nil {
		handleResponse(c, h.log, "failed to create compress job", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "compression job created", http.StatusCreated, gin.H{"id": jobID})
}

// GetCompressJob godoc
// @Router       /pdf/compress/{id} [GET]
// @Summary      Get compress job by ID
// @Tags         pdf-compress
// @Accept       json
// @Produce      json
// @Param        id path string true "compress job ID"
// @Success      200 {object} models.CompressJob
// @Failure      404 {object} models.Response
// @Failure      500 {object} models.Response
func (h *Handler) GetCompressJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		handleResponse(c, h.log, "missing job id", http.StatusBadRequest, "id required")
		return
	}

	// Set timeout to fetch the job details
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get job by ID
	job, err := h.services.Compress().GetByID(ctx, id)
	if err != nil {
		handleResponse(c, h.log, "failed to get compress job", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "compress job fetched", http.StatusOK, job)
}
