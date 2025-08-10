package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// CreateProtectJob godoc
// @Summary      Protect PDF with password
// @Description  PDF fayliga parol qo‘shadi (foydalanuvchi tomonidan berilgan parol asosida)
// @Tags         PDF Security
// @Accept       json
// @Produce      json
// @Param        request body models.ProtectPDFRequest true "Protect request"
// @Success      201 {object} map[string]string
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /pdf/protect [post]
func (h *Handler) CreateProtectJob(c *gin.Context) {
	var req models.ProtectPDFRequest

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	jobID, err := h.services.Protect().Create(ctx, req, userID)
	if err != nil {
		handleResponse(c, h.log, "failed to protect PDF", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "PDF protected successfully", http.StatusCreated, gin.H{"id": jobID})
}

// GetProtectJob godoc
// @Summary      Get protected PDF job by ID
// @Description  Parollangan PDF faylga oid ish holatini qaytaradi
// @Tags         PDF Security
// @Produce      json
// @Param        id path string true "Job ID"
// @Success      200 {object} models.ProtectPDFJob
// @Failure      404 {object} models.Response
// @Router       /pdf/protect/{id} [get]
func (h *Handler) GetProtectJob(c *gin.Context) {
	jobID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := h.services.Protect().GetByID(ctx, jobID)
	if err != nil {
		handleResponse(c, h.log, "protect job not found", http.StatusNotFound, err.Error())
		return
	}

	handleResponse(c, h.log, "protect job found", http.StatusOK, job)
}
