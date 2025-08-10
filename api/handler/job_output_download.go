package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DownloadJobPrimary godoc
// @Summary      Download job result (universal)
// @Description  Merge/Compress/Rotate/... kabi single-outputlarda to'g'ridan-to'g'ri faylni, pdf-to-jpg kabi ko'p outputlarda ZIP yoki birinchi chiqishni qaytaradi.
// @Tags         jobs, download
// @Produce      application/octet-stream
// @Param        type path string true  "Job type (merge|split|compress|pdf-to-jpg|...)"
// @Param        id   path string true  "Job ID"
// @Success      200
// @Failure      404 {object} models.Response
// @Router       /jobs/{type}/{id}/download [get]
func (h Handler) DownloadJobPrimary(c *gin.Context) {
	jt := c.Param("type")
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	fd, err := h.services.Download().GetPrimary(ctx, jt, id)
	if err != nil {
		handleResponse(c, h.log, "download not available", http.StatusNotFound, err.Error())
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fd.Name))
	c.Header("Content-Type", fd.MimeType)
	c.File(fd.Path)
}
