package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// AdminListJobs godoc
// @Summary      List jobs (all types)
// @Description  Merge, split, compress va boshqa joblar holati (filter + paginatsiya)
// @Tags         admin, jobs
// @Security     ApiKeyAuth
// @Produce      json
// @Param        type    query string false "Job type (merge|split|compress)"
// @Param        status  query string false "Status (pending|processing|done|failed)"
// @Param        user_id query string false "User ID"
// @Param        from    query string false "From (RFC3339, e.g. 2025-08-01T00:00:00Z)"
// @Param        to      query string false "To (RFC3339)"
// @Param        search  query string false "Job ID prefix"
// @Param        limit   query int    false "Limit (max 200)" default(50)
// @Param        offset  query int    false "Offset"          default(0)
// @Success      200 {array} models.JobSummary
// @Failure      400 {object} models.Response
// @Failure      500 {object} models.Response
// @Router       /admin/jobs [get]
func (h *Handler) AdminListJobs(c *gin.Context) {
	// parse filters
	f := models.AdminJobFilter{}

	if v := c.Query("type"); v != "" {
		f.Type = &v
	}
	if v := c.Query("status"); v != "" {
		f.Status = &v
	}
	if v := c.Query("user_id"); v != "" {
		f.UserID = &v
	}
	if v := c.Query("search"); v != "" {
		f.Search = &v
	}
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = &t
		} else {
			handleResponse(c, h.log, "invalid 'from' (use RFC3339)", http.StatusBadRequest, err.Error())
			return
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = &t
		} else {
			handleResponse(c, h.log, "invalid 'to' (use RFC3339)", http.StatusBadRequest, err.Error())
			return
		}
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	f.Limit = limit
	f.Offset = offset

	ctx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
	defer cancel()

	jobs, err := h.services.AdminJob().ListJobs(ctx, f)
	if err != nil {
		handleResponse(c, h.log, "failed to list jobs", http.StatusInternalServerError, err.Error())
		return
	}

	handleResponse(c, h.log, "jobs fetched", http.StatusOK, jobs)
}
