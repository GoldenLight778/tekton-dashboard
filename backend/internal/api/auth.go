package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"devops-platform/internal/repository"
	"devops-platform/internal/service"
)

// AuthHandler handles HTTP requests related to user authentication
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// LoginRequest is the request body for login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles POST /api/platform/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, 40001, "invalid request: username and password required")
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			Error(c, http.StatusUnauthorized, 40110, "incorrect username or password")
			return
		}
		Error(c, http.StatusInternalServerError, 50000, "internal server error during login")
		return
	}

	Success(c, result)
}

// Me handles GET /api/platform/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		Error(c, http.StatusUnauthorized, 40100, "unauthorized")
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			Error(c, http.StatusNotFound, 40401, "user not found")
			return
		}
		Error(c, http.StatusInternalServerError, 50000, "failed to retrieve user profile")
		return
	}

	Success(c, user)
}
