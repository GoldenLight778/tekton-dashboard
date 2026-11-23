package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"devops-platform/internal/repository"
	"devops-platform/internal/service"
)

// CredentialHandler handles credential endpoints
type CredentialHandler struct {
	credService *service.CredentialService
}

// NewCredentialHandler creates a new CredentialHandler
func NewCredentialHandler(credService *service.CredentialService) *CredentialHandler {
	return &CredentialHandler{credService: credService}
}

type CreateCredentialRequest struct {
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type"` // git_token, ssh_key, kubeconfig, secret_text
	Secret string `json:"secret" binding:"required"`
}

// Create handles POST /api/platform/v1/projects/:id/credentials
func (h *CredentialHandler) Create(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		projectID = c.Param("projectId")
	}

	var req CreateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, 40001, "invalid request: name and secret are required")
		return
	}

	if req.Type == "" {
		req.Type = "secret_text"
	}

	userID := c.GetString("userID")
	cred, err := h.credService.CreateCredential(c.Request.Context(), userID, c.ClientIP(), projectID, req.Name, req.Type, req.Secret)
	if err != nil {
		if errors.Is(err, repository.ErrProjectNotFound) {
			Error(c, http.StatusNotFound, 40401, "project not found")
			return
		}
		if errors.Is(err, repository.ErrCredentialAlreadyExists) {
			Error(c, http.StatusConflict, 40901, "credential name already exists in this project")
			return
		}
		if errors.Is(err, service.ErrInvalidCredentialName) ||
			errors.Is(err, service.ErrInvalidCredentialType) ||
			errors.Is(err, service.ErrEmptySecret) {
			Error(c, http.StatusBadRequest, 40002, err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}

	Success(c, cred)
}

// ListByProject handles GET /api/platform/v1/projects/:id/credentials
func (h *CredentialHandler) ListByProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		projectID = c.Param("projectId")
	}

	creds, err := h.credService.ListByProject(c.Request.Context(), projectID)
	if err != nil {
		if errors.Is(err, repository.ErrProjectNotFound) {
			Error(c, http.StatusNotFound, 40401, "project not found")
			return
		}
		Error(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}

	Success(c, creds)
}

// Delete handles DELETE /api/platform/v1/credentials/:id
func (h *CredentialHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userID")

	if err := h.credService.DeleteCredential(c.Request.Context(), userID, c.ClientIP(), id); err != nil {
		if errors.Is(err, repository.ErrCredentialNotFound) {
			Error(c, http.StatusNotFound, 40401, "credential not found")
			return
		}
		Error(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}

	Success(c, gin.H{"deleted": true})
}
