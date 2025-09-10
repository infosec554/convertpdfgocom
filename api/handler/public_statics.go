package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetPublicStats godoc
// @Router       /public-stats [get]
// @Summary      Get public statistics
// @Tags         public-stats
// @Accept       json
// @Produce      json
// @Success      200 {object} models.PublicStatsDisplay
// @Failure      500 {object} models.Response
func (h Handler) GetPublicStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := h.services.PublicStats().GetDisplayStats(ctx)
	if err != nil {
		handleResponse(c, h.log, "failed to get public stats", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "public stats fetched", http.StatusOK, stats)
}
