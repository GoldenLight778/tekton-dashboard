package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"devops-platform/internal/service"
)

// MetricsHandler handles HTTP endpoints for observability and metrics
type MetricsHandler struct {
	metricsService *service.MetricsService
}

// NewMetricsHandler creates a new MetricsHandler
func NewMetricsHandler(metricsService *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{metricsService: metricsService}
}

// Dashboard handles GET /api/platform/v1/metrics/dashboard
func (h *MetricsHandler) Dashboard(c *gin.Context) {
	metrics, err := h.metricsService.GetDashboardMetrics(c.Request.Context())
	if err != nil {
		Error(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}

	Success(c, metrics)
}
