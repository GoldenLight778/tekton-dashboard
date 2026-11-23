package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"devops-platform/internal/repository"
	"devops-platform/internal/service"
	"devops-platform/internal/webhook"
)

// WebhookHandler processes external Git webhooks
type WebhookHandler struct {
	pipelineRepo repository.PipelineRepository
	runService   *service.RunService
}

// NewWebhookHandler creates a new WebhookHandler
func NewWebhookHandler(pipelineRepo repository.PipelineRepository, runService *service.RunService) *WebhookHandler {
	return &WebhookHandler{
		pipelineRepo: pipelineRepo,
		runService:   runService,
	}
}

// Handle processes POST /api/platform/v1/webhooks/:id
func (h *WebhookHandler) Handle(c *gin.Context) {
	projectID := c.Param("id")

	// Read body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		Error(c, http.StatusBadRequest, 40001, "failed to read webhook payload")
		return
	}

	headers := make(map[string]string)
	for k := range c.Request.Header {
		headers[k] = c.Request.Header.Get(k)
	}

	event, _ := webhook.ParseEvent(headers, bodyBytes)

	// Fetch pipelines for this project
	pipelines, total, err := h.pipelineRepo.ListByProject(c.Request.Context(), projectID, 1, 10)
	if err != nil || total == 0 {
		Error(c, http.StatusNotFound, 40401, "no pipelines found in project to trigger")
		return
	}

	// Trigger the primary pipeline
	targetPipeline := pipelines[0]
	triggerSource := fmt.Sprintf("webhook:%s:%s", event.Platform, event.Branch)

	run, err := h.runService.TriggerRun(c.Request.Context(), "system:webhook", c.ClientIP(), targetPipeline.ID, triggerSource)
	if err != nil {
		Error(c, http.StatusInternalServerError, 50000, fmt.Sprintf("failed to trigger pipeline: %v", err))
		return
	}

	Success(c, gin.H{
		"triggered":     true,
		"pipeline_id":   targetPipeline.ID,
		"pipeline_name": targetPipeline.Name,
		"run_id":        run.ID,
		"branch":        event.Branch,
		"commit":        event.Commit,
	})
}
