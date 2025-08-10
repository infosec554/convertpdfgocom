package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// CreatePDFToJPG godoc
// @Summary      Convert PDF to JPG
// @Description  Convert a PDF file's pages into JPG format
// @Tags         pdf-to-jpg
// @Accept       json
// @Produce      json
// @Param        request body models.PDFToJPGRequest true "PDF file ID"
// @Success      201 {object} map[string]string
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/pdf-to-jpg [post]
func (h *Handler) CreatePDFToJPG(c *gin.Context) {
	var req models.PDFToJPGRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, h.log, "invalid request body", http.StatusBadRequest, err.Error())
		return
	}

	// Guest foydalanuvchi uchun user_id bo'lishi mumkin, uni nil qilish
	var userID *string
	if uid := c.GetString("user_id"); uid != "" {
		userID = &uid // Agar foydalanuvchi tizimga kirgan bo'lsa, IDni oling
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// PDF to JPG conversion xizmatiga murojaat qilish
	jobID, err := h.services.PDFToJPG().Create(ctx, req, userID)
	if err != nil {
		handleResponse(c, h.log, "conversion failed", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "PDF to JPG conversion started", http.StatusCreated, gin.H{"job_id": jobID})
}

// GetPDFToJPG godoc
// @Summary      Get PDF to JPG conversion job
// @Description  Retrieve the status of the conversion job by Job ID
// @Tags         pdf-to-jpg
// @Produce      json
// @Param        id path string true "Job ID"
// @Success      200 {object} models.PDFToJPGJob
// @Failure      404 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/pdf-to-jpg/{id} [get]
func (h *Handler) GetPDFToJPG(c *gin.Context) {
	jobID := c.Param("id")

	// Ensure that job ID is provided
	if jobID == "" {
		handleResponse(c, h.log, "job ID is required", http.StatusBadRequest, "id required")
		return
	}

	// Set a timeout to fetch the job status
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Fetch the job details using the provided job ID
	job, err := h.services.PDFToJPG().GetByID(ctx, jobID)
	if err != nil {
		handleResponse(c, h.log, "job not found", http.StatusNotFound, err.Error())
		return
	}

	// Return the job details as the response
	handleResponse(c, h.log, "job fetched", http.StatusOK, job)
}
