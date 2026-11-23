package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops-platform/internal/repository"
	"devops-platform/internal/service"
)

// PipelineHandler handles HTTP routes for pipelines
type PipelineHandler struct {
	pipelineService *service.PipelineService
}

// NewPipelineHandler creates a new PipelineHandler
func NewPipelineHandler(pipelineService *service.PipelineService) *PipelineHandler {
	return &PipelineHandler{pipelineService: pipelineService}
}

type CreatePipelineRequest struct {
	Name       string `json:"name" binding:"required"`
	DSLContent string `json:"dsl_content"`
}

type UpdatePipelineRequest struct {
	Name       string `json:"name"`
	DSLContent string `json:"dsl_content"`
}

// Create handles POST /api/platform/v1/projects/:id/pipelines
func (h *PipelineHandler) Create(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		projectID = c.Param("projectId")
	}
	var req CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, 40001, "invalid request: name is required")
		return
	}

	userID := c.GetString("userID")
	p, err := h.pipelineService.CreatePipeline(c.Request.Context(), userID, c.ClientIP(), projectID, req.Name, req.DSLContent)
	if err != nil {
		if errors.Is(err, repository.ErrProjectNotFound) {
			Error(c, http.StatusNotFound, 40401, "project not found")
			return
		}
		if errors.Is(err, repository.ErrPipelineAlreadyExists) {
			Error(c, http.StatusConflict, 40901, "pipeline name already exists in this project")
			return
		}
		if errors.Is(err, service.ErrInvalidPipelineName) {
			Error(c, http.StatusBadRequest, 40002, err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, p)
}

// Get handles GET /api/platform/v1/pipelines/:id
func (h *PipelineHandler) Get(c *gin.Context) {
	id := c.Param("id")
	p, err := h.pipelineService.GetPipeline(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrPipelineNotFound) {
			Error(c, http.StatusNotFound, 40401, "pipeline not found")
			return
		}
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, p)
}

// ListByProject handles GET /api/platform/v1/projects/:id/pipelines
func (h *PipelineHandler) ListByProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		projectID = c.Param("projectId")
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := h.pipelineService.ListPipelinesByProject(c.Request.Context(), projectID, page, pageSize)
	if err != nil {
		if errors.Is(err, repository.ErrProjectNotFound) {
			Error(c, http.StatusNotFound, 40401, "project not found")
			return
		}
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

// Update handles PUT /api/platform/v1/pipelines/:id
func (h *PipelineHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req UpdatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, 40001, "invalid request")
		return
	}

	userID := c.GetString("userID")
	p, err := h.pipelineService.UpdatePipeline(c.Request.Context(), userID, c.ClientIP(), id, req.Name, req.DSLContent)
	if err != nil {
		if errors.Is(err, repository.ErrPipelineNotFound) {
			Error(c, http.StatusNotFound, 40401, "pipeline not found")
			return
		}
		if errors.Is(err, service.ErrInvalidPipelineName) {
			Error(c, http.StatusBadRequest, 40002, err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, p)
}

// Delete handles DELETE /api/platform/v1/pipelines/:id
func (h *PipelineHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userID")

	if err := h.pipelineService.DeletePipeline(c.Request.Context(), userID, c.ClientIP(), id); err != nil {
		if errors.Is(err, repository.ErrPipelineNotFound) {
			Error(c, http.StatusNotFound, 40401, "pipeline not found")
			return
		}
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, gin.H{"deleted": true})
}
