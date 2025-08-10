package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// CreateRemovePagesJob godoc
// @Router       /pdf/remove-pages [post]
// @Summary      Create remove pages job
// @Tags         pdf-remove
// @Accept       json
// @Produce      json
// @Param        request body models.RemovePagesRequest true "remove pages job"
// @Success      201 {object} map[string]string
// @Failure      400 {object} models.Response
// @Failure      401 {object} models.Response
// @Failure      500 {object} models.Response
func (h Handler) CreateRemovePagesJob(c *gin.Context) {
	var req models.RemovePagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, h.log, "invalid request body", http.StatusBadRequest, err.Error())
		return
	}

	// ❗️ Majburiy emas qilamiz
	var userID *string
	if rawUserID := c.GetString("user_id"); rawUserID != "" {
		userID = &rawUserID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	jobID, err := h.services.RemovePage().Create(ctx, req, userID) // ❗️ pointer bo'lishi kerak
	if err != nil {
		handleResponse(c, h.log, "failed to create remove pages job", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "remove pages job created", http.StatusCreated, gin.H{"id": jobID})
}

// GetRemovePagesJob godoc
// @Router       /pdf/remove-pages/{id} [GET]
// @Summary      Get remove pages job by ID
// @Tags         pdf-remove
// @Accept       json
// @Produce      json
// @Param        id path string true "remove job ID"
// @Success      200 {object} models.RemoveJob
// @Failure      404 {object} models.Response
// @Failure      500 {object} models.Response
func (h Handler) GetRemovePagesJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		handleResponse(c, h.log, "missing id", http.StatusBadRequest, "id required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := h.services.RemovePage().GetByID(ctx, id)
	if err != nil {
		handleResponse(c, h.log, "failed to get remove job", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "remove job fetched", http.StatusOK, job)
}
