package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops-platform/internal/repository"
	"devops-platform/internal/service"
)

// ProjectHandler handles HTTP routes for projects
type ProjectHandler struct {
	projectService *service.ProjectService
}

// NewProjectHandler creates a new ProjectHandler
func NewProjectHandler(projectService *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
}

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// Create handles POST /api/platform/v1/projects
func (h *ProjectHandler) Create(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, 40001, "invalid request: name is required")
		return
	}

	userID := c.GetString("userID")
	p, err := h.projectService.CreateProject(c.Request.Context(), userID, c.ClientIP(), req.Name, req.Description)
	if err != nil {
		if errors.Is(err, repository.ErrProjectAlreadyExists) {
			Error(c, http.StatusConflict, 40901, "project name already exists")
			return
		}
		if errors.Is(err, service.ErrInvalidProjectName) {
			Error(c, http.StatusBadRequest, 40002, err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, p)
}

// Get handles GET /api/platform/v1/projects/:id
func (h *ProjectHandler) Get(c *gin.Context) {
	id := c.Param("id")
	p, err := h.projectService.GetProject(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrProjectNotFound) {
			Error(c, http.StatusNotFound, 40401, "project not found")
			return
		}
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, p)
}

// List handles GET /api/platform/v1/projects
func (h *ProjectHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := h.projectService.ListProjects(c.Request.Context(), page, pageSize)
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

// Update handles PUT /api/platform/v1/projects/:id
func (h *ProjectHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, 40001, "invalid request: name is required")
		return
	}

	userID := c.GetString("userID")
	p, err := h.projectService.UpdateProject(c.Request.Context(), userID, c.ClientIP(), id, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, repository.ErrProjectNotFound) {
			Error(c, http.StatusNotFound, 40401, "project not found")
			return
		}
		if errors.Is(err, service.ErrInvalidProjectName) {
			Error(c, http.StatusBadRequest, 40002, err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, p)
}

// Delete handles DELETE /api/platform/v1/projects/:id
func (h *ProjectHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userID")

	if err := h.projectService.DeleteProject(c.Request.Context(), userID, c.ClientIP(), id); err != nil {
		if errors.Is(err, repository.ErrProjectNotFound) {
			Error(c, http.StatusNotFound, 40401, "project not found")
			return
		}
		Error(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}

	Success(c, gin.H{"deleted": true})
}
