package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// CreateExtractJob godoc
// @Router       /pdf/extract [POST]
// @Summary      Create extract job
// @Tags         pdf-extract
// @Accept       json
// @Produce      json
// @Param        request body models.ExtractPagesRequest true "extract request body"
// @Success      201 {object} map[string]string
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
func (h *Handler) CreateExtractJob(c *gin.Context) {
	var req models.ExtractPagesRequest

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

	jobID, err := h.services.ExtractPage().Create(ctx, req, userID)
	if err != nil {
		handleResponse(c, h.log, "failed to create extract job", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "extract job created", http.StatusCreated, gin.H{"id": jobID})
}

// GetExtractJob godoc
// @Router       /pdf/extract/{id} [GET]
// @Summary      Get extract job by ID
// @Tags         pdf-extract
// @Accept       json
// @Produce      json
// @Param        id path string true "extract job ID"
// @Success      200 {object} models.ExtractJob
// @Failure      404 {object} models.Response
// @Failure      500 {object} models.Response
func (h *Handler) GetExtractJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		handleResponse(c, h.log, "missing id", http.StatusBadRequest, "id required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := h.services.ExtractPage().GetByID(ctx, id)
	if err != nil {
		handleResponse(c, h.log, "failed to get extract job", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "extract job fetched", http.StatusOK, job)
}
