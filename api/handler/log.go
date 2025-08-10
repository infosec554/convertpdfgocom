package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// GetLogsByJobID godoc
// @Summary Get logs for a specific job
// @Description Get logs by job ID (admin)
// @Tags admin, logs
// @Accept json
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {array} models.Log
// @Failure 400 {object} models.Response
// @Failure 404 {object} models.Response
// @Failure 500 {object} models.Response
// @Security ApiKeyAuth
// @Router /admin/logs/{id} [get]   // ✅ to‘g‘ri path
func (h Handler) GetLogsByJobID(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		handleResponse(c, h.log, "missing job ID", http.StatusBadRequest, gin.H{})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	logs, err := h.services.Log().GetLogsByJobID(ctx, jobID)
	if err != nil {
		handleResponse(c, h.log, "failed to fetch logs for job "+jobID, http.StatusInternalServerError, gin.H{})
		return
	}

	// Logs bo‘sh bo‘lsa 200 + bo‘sh ro‘yxat qaytarish variant
	if len(logs) == 0 {
		handleResponse(c, h.log, "no logs found for job "+jobID, http.StatusOK, []models.Log{})
		return
	}

	handleResponse(c, h.log, "logs retrieved successfully", http.StatusOK, logs)
}
