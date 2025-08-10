package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// AdminDeletedFilesLogs godoc
// @Summary      List deleted files audit logs
// @Description  O‘chirilgan fayllar bo‘yicha audit loglar (admin-only, paginatsiya bilan)
// @Tags         admin, files
// @Security     ApiKeyAuth
// @Produce      json
// @Param        limit   query    int   false  "Limit"  default(50)
// @Param        offset  query    int   false  "Offset" default(0)
// @Success      200 {array}  models.FileDeletionLog
// @Failure      400 {object}  models.Response
// @Failure      500 {object}  models.Response
// @Router       /v1/admin/files/deleted-logs [get]
func (h *Handler) AdminDeletedFilesLogs(c *gin.Context) {
	// Parse pagination
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 200 {
		handleResponse(c, h.log, "invalid 'limit' param", http.StatusBadRequest, nil)
		return
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		handleResponse(c, h.log, "invalid 'offset' param", http.StatusBadRequest, nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	logs, svcErr := h.services.FileDeletionLog().GetDeletionLogs(ctx, limit, offset)
	if svcErr != nil {
		handleResponse(c, h.log, "failed to fetch deleted files logs", http.StatusInternalServerError, svcErr.Error())
		return
	}

	// Bo'sh bo'lsa ham 200 qaytaramiz — frontendga qulay
	if len(logs) == 0 {
		handleResponse(c, h.log, "no deleted file logs found", http.StatusOK, []models.FileDeletionLog{})
		return
	}

	handleResponse(c, h.log, "deleted files logs fetched", http.StatusOK, logs)
}
