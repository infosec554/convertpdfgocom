package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"convertpdfgo/api/models"
)

// POST /contact  (public)
func (h Handler) ContactCreate(c *gin.Context) {
	var req models.ContactCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, h.log, "invalid body", http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	id, err := h.services.Contact().Create(ctx, req)
	if err != nil {
		handleResponse(c, h.log, err.Error(), http.StatusBadRequest, nil)
		return
	}
	handleResponse(c, h.log, "ok", http.StatusCreated, models.ContactCreateResponse{ID: id})
}

// GET /admin/contacts?unread=true&limit=50&offset=0
func (h Handler) AdminListContacts(c *gin.Context) {
	unread := strings.EqualFold(c.DefaultQuery("unread", "false"), "true")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	msgs, err := h.services.Contact().List(ctx, unread, limit, offset) // yoki alohida ContactService.List
	if err != nil {
		handleResponse(c, h.log, "failed", http.StatusInternalServerError, err.Error())
		return
	}
	handleResponse(c, h.log, "ok", http.StatusOK, msgs)
}

// GET /admin/contacts/:id
func (h Handler) AdminGetContact(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	msg, err := h.services.Contact().GetByID(ctx, id)
	if err != nil {
		handleResponse(c, h.log, "not found", http.StatusNotFound, nil)
		return
	}
	handleResponse(c, h.log, "ok", http.StatusOK, msg)
}

// POST /admin/contacts/:id/read
func (h Handler) AdminMarkContactRead(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if err := h.services.Contact().MarkRead(ctx, id); err != nil {
		handleResponse(c, h.log, "failed", http.StatusInternalServerError, err.Error())
		return
	}
	handleResponse(c, h.log, "marked read", http.StatusOK, gin.H{"id": id})
}

// DELETE /admin/contacts/:id
func (h Handler) AdminDeleteContact(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if err := h.services.Contact().Delete(ctx, id); err != nil {
		handleResponse(c, h.log, "failed", http.StatusInternalServerError, err.Error())
		return
	}
	handleResponse(c, h.log, "deleted", http.StatusOK, gin.H{"id": id})
}
