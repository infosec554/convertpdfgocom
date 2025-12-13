package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"convertpdfgo/service"
)

type ValidationHandler struct {
	services    service.IServiceManager
	adminUserID int64
}

func New(services service.IServiceManager, adminUserID string) *ValidationHandler {
	id, _ := strconv.ParseInt(adminUserID, 10, 64)
	return &ValidationHandler{
		services:    services,
		adminUserID: id,
	}
}

func (h *ValidationHandler) GetStatsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "stats.html", gin.H{
		"admin_id": h.adminUserID,
	})
}

func (h *ValidationHandler) GetStatsAPI(c *gin.Context) {
	userIDStr := c.Query("user_id")

	// 1. Personal Stats
	if userIDStr != "" {
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err == nil && userID > 0 {
			user, refCount, err := h.services.BotUser().GetUserStats(c.Request.Context(), userID)
			if err == nil && user != nil {
				c.JSON(http.StatusOK, gin.H{
					"type":           "user",
					"coins":          user.Coins,
					"total_used":     user.TotalUsed,
					"referral_count": refCount,
				})
				return
			}
		}
	}

	// 2. Global Stats (Default)
	stats, err := h.services.PublicStats().GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Type 'global' qo'shamiz
	response := gin.H{
		"type":        "global",
		"total_users": stats.TotalUsers,
		"tools_count": stats.ToolsCount,
		"total_usage": stats.TotalUsage,
		// ... boshqa fieldlar
	}
	c.JSON(http.StatusOK, response)
}
