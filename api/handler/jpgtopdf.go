package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// CreateJPGToPDF godoc
// @Summary      Convert JPG to PDF
// @Description  Convert a JPG to PDF
// @Tags         jpg-to-pdf
// @Accept       json
// @Produce      json
// @Param        request body models.CreateJPGToPDFRequest true "JPG to PDF request"
// @Success      201 {object} map[string]string
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/jpg-to-pdf [post]
func (h Handler) CreateJPGToPDF(c *gin.Context) {
	var req models.CreateJPGToPDFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, h.log, "invalid request", http.StatusBadRequest, err.Error())
		return
	}

	// Check if inputFileIDs are provided
	if len(req.InputFileIDs) == 0 {
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

	// Create the JPG to PDF job
	jobID, err := h.services.JPGToPDF().CreateJob(ctx, userID, req.InputFileIDs)
	if err != nil {
		handleResponse(c, h.log, "failed to create jpg to pdf job", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "jpg to pdf job created", http.StatusCreated, gin.H{"id": jobID})
}

// GetJPGToPDFJob godoc
// @Router       /pdf/jpg-to-pdf/{id} [GET]
// @Summary      Get JPG to PDF job
// @Description  JPG to PDF konversiya ishining holatini olish
// @Tags         jpg-to-pdf
// @Produce      json
// @Param        id path string true "Job ID"
// @Success      200 {object} models.JPGToPDFJob
// @Failure      404 {object} models.Response
// @Failure      500 {object} models.Response
func (h Handler) GetJPGToPDFJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		handleResponse(c, h.log, "missing job id", http.StatusBadRequest, "id is required")
		return
	}

	// Set timeout to fetch the job details
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get job by ID
	job, err := h.services.JPGToPDF().GetJobByID(ctx, id)
	if err != nil {
		handleResponse(c, h.log, "jpg to pdf job not found", http.StatusNotFound, err.Error())
		return
	}

	// Return job details
	handleResponse(c, h.log, "jpg to pdf job fetched", http.StatusOK, job)
}
