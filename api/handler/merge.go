package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
	"convertpdfgo/service" // ✅ sentinel errorlar uchun
)

// CreateMergeJob godoc
// @Summary      Create merge job
// @Tags         pdf-merge
// @Accept       json
// @Produce      json
// @Param        request body models.CreateMergeJobRequest true "merge job"
// @Success      201 {object} map[string]string
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/merge [post]
func (h Handler) CreateMergeJob(c *gin.Context) {
	var req models.CreateMergeJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, h.log, "invalid request body", http.StatusBadRequest, err.Error())
		return
	}

	// Optional auth: bor bo‘lsa user_id ni olamiz
	var userID *string
	if v, ok := c.Get("user_id"); ok {
		if s, ok := v.(string); ok && s != "" {
			userID = &s
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, err := h.services.Merge().Create(ctx, userID, req.InputFileIDs)
	if err != nil {
		handleResponse(c, h.log, "failed to create merge job", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "merge job created", http.StatusCreated, gin.H{"id": id})
}

// GetMergeJob godoc
// @Summary      Get merge job
// @Tags         pdf-merge
// @Accept       json
// @Produce      json
// @Param        id path string true "merge job ID"
// @Success      200 {object} models.MergeJob
// @Failure      404 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/merge/{id} [get]
func (h Handler) GetMergeJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		handleResponse(c, h.log, "missing job ID", http.StatusBadRequest, "id is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	job, err := h.services.Merge().GetByID(ctx, id)
	if err != nil {
		// Agar storage ErrNoRows ni to‘g‘ridan-to‘g‘ri qaytarsa, serviceda map qilmaganmiz.
		// Shuning uchun bu yerda ham umumiy 404 branch qo‘shamiz:
		if errors.Is(err, service.ErrJobNotFound) {
			handleResponse(c, h.log, "merge job not found", http.StatusNotFound, err.Error())
			return
		}
		handleResponse(c, h.log, "failed to get merge job", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "merge job fetched", http.StatusOK, job)
}

// ProcessMergeJob godoc
// @Summary         Process merge job (side-effect)
// @Description     Triggers processing for a merge job. Use POST (not GET).
// @Tags            pdf-merge
// @Param           id  path   string  true  "merge job ID"
// @Success         200 {object} models.Response     "Processed synchronously"
// @Failure         400 {object} models.Response     "Missing/invalid ID"
// @Failure         404 {object} models.Response     "Job not found"
// @Failure         409 {object} models.Response     "Job status not eligible for processing"
// @Failure         500 {object} models.Response
// @Router          /pdf/merge/{id}/process [post]
func (h Handler) ProcessMergeJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		handleResponse(c, h.log, "missing job ID", http.StatusBadRequest, "job ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	outputID, err := h.services.Merge().ProcessJob(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrJobNotFound):
			handleResponse(c, h.log, "merge job not found", http.StatusNotFound, err.Error())
			return
		case errors.Is(err, service.ErrJobInvalidState): // ✅ nomi to‘g‘rilandi
			handleResponse(c, h.log, "merge job invalid state", http.StatusConflict, err.Error())
			return
		case errors.Is(err, service.ErrJobInvalidInput):
			handleResponse(c, h.log, "invalid merge input", http.StatusBadRequest, err.Error())
			return
		default:
			handleResponse(c, h.log, "failed to process merge job", http.StatusInternalServerError, err.Error())
			return
		}
	}

	handleResponse(c, h.log, "merge job processed successfully", http.StatusOK, gin.H{
		"output_file_id": outputID,
	})
}
