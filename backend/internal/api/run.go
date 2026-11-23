package api

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops-platform/internal/repository"
	"devops-platform/internal/service"
)

// RunHandler handles HTTP requests for pipeline runs and logs
type RunHandler struct {
	runService *service.RunService
}

// NewRunHandler creates a new RunHandler
func NewRunHandler(runService *service.RunService) *RunHandler {
	return &RunHandler{runService: runService}
}

// Trigger handles POST /api/platform/v1/pipelines/:id/runs
func (h *RunHandler) Trigger(c *gin.Context) {
	pipelineID := c.Param("id")
	userID := c.GetString("userID")
	username := c.GetString("username")
	if username == "" {
		username = "manual"
	}

	run, err := h.runService.TriggerRun(c.Request.Context(), userID, c.ClientIP(), pipelineID, username)
	if err != nil {
		if errors.Is(err, repository.ErrPipelineNotFound) {
			Error(c, http.StatusNotFound, 40401, "pipeline not found")
			return
		}
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, run)
}

// Get handles GET /api/platform/v1/runs/:id
func (h *RunHandler) Get(c *gin.Context) {
	id := c.Param("id")
	run, err := h.runService.GetRun(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrRunNotFound) {
			Error(c, http.StatusNotFound, 40401, "pipeline run not found")
			return
		}
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, run)
}

// ListByPipeline handles GET /api/platform/v1/pipelines/:id/runs
func (h *RunHandler) ListByPipeline(c *gin.Context) {
	pipelineID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := h.runService.ListRunsByPipeline(c.Request.Context(), pipelineID, page, pageSize)
	if err != nil {
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, gin.H{
		"items":    list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// ListByProject handles GET /api/platform/v1/projects/:id/runs
func (h *RunHandler) ListByProject(c *gin.Context) {
	projectID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := h.runService.ListRunsByProject(c.Request.Context(), projectID, page, pageSize)
	if err != nil {
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, gin.H{
		"items":    list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// Cancel handles POST /api/platform/v1/runs/:id/cancel
func (h *RunHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userID")

	if err := h.runService.CancelRun(c.Request.Context(), userID, c.ClientIP(), id); err != nil {
		if errors.Is(err, repository.ErrRunNotFound) {
			Error(c, http.StatusNotFound, 40401, "pipeline run not found")
			return
		}
		Error(c, http.StatusBadRequest, 40010, err.Error())
		return
	}

	Success(c, gin.H{"cancelled": true})
}

// LogsSSE handles GET /api/platform/v1/runs/:id/logs/sse using Server-Sent Events
func (h *RunHandler) LogsSSE(c *gin.Context) {
	id := c.Param("id")

	logChan, err := h.runService.StreamLogs(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrRunNotFound) {
			Error(c, http.StatusNotFound, 40401, "pipeline run not found")
			return
		}
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case msg, ok := <-logChan:
			if !ok {
				return false
			}
			c.SSEvent("message", msg)
			return true
		}
	})
}
