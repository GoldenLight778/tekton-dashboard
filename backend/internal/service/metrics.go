package service

import (
	"context"

	"go.uber.org/zap"

	"devops-platform/internal/repository"
)

// MetricsService handles retrieval and computation of platform metrics
type MetricsService struct {
	metricsRepo repository.MetricsRepository
	logger      *zap.Logger
}

// NewMetricsService creates a new MetricsService
func NewMetricsService(metricsRepo repository.MetricsRepository, logger *zap.Logger) *MetricsService {
	return &MetricsService{
		metricsRepo: metricsRepo,
		logger:      logger,
	}
}

// GetDashboardMetrics returns high-level platform metrics
func (s *MetricsService) GetDashboardMetrics(ctx context.Context) (*repository.DashboardMetrics, error) {
	return s.metricsRepo.GetDashboardMetrics(ctx)
}
