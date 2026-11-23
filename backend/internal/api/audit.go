package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops-platform/internal/repository"
)

// AuditHandler handles HTTP requests for audit logs
type AuditHandler struct {
	auditRepo repository.AuditRepository
}

// NewAuditHandler creates a new AuditHandler
func NewAuditHandler(auditRepo repository.AuditRepository) *AuditHandler {
	return &AuditHandler{auditRepo: auditRepo}
}

// List handles GET /api/platform/v1/audit-logs
func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	logs, total, err := h.auditRepo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, gin.H{
		"items":    logs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
