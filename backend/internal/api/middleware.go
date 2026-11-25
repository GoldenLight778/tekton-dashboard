package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"devops-platform/internal/crypto"
)

// AuthMiddleware extracts and validates Bearer JWT token from Authorization header.
// If allowDevFallback is true and Authorization header is missing, it falls back to the default admin user.
func AuthMiddleware(secret string, allowDevFallback ...bool) gin.HandlerFunc {
	isDev := len(allowDevFallback) > 0 && allowDevFallback[0]
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			if pToken := strings.TrimSpace(c.GetHeader("X-Platform-Token")); pToken != "" {
				if strings.HasPrefix(strings.ToLower(pToken), "bearer ") {
					authHeader = pToken
				} else {
					authHeader = "Bearer " + pToken
				}
			}
		}
		if authHeader == "" {
			if isDev {
				c.Set("claims", &crypto.Claims{
					UserID:   "admin",
					Username: "admin",
					Role:     "admin",
				})
				c.Set("userID", "admin")
				c.Set("username", "admin")
				c.Set("role", "admin")
				c.Next()
				return
			}
			Error(c, http.StatusUnauthorized, 40101, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			Error(c, http.StatusUnauthorized, 40102, "invalid authorization format, expected 'Bearer <token>'")
			c.Abort()
			return
		}

		claims, err := crypto.ParseToken(parts[1], secret)
		if err != nil {
			Error(c, http.StatusUnauthorized, 40103, "invalid or expired token")
			c.Abort()
			return
		}

		// Store claims and basic user identity in Gin context
		c.Set("claims", claims)
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RequireRole restricts endpoint access to specified roles
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			Error(c, http.StatusUnauthorized, 40100, "unauthorized")
			c.Abort()
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			Error(c, http.StatusForbidden, 40301, "forbidden")
			c.Abort()
			return
		}

		for _, r := range roles {
			if r == roleStr {
				c.Next()
				return
			}
		}

		Error(c, http.StatusForbidden, 40302, "insufficient permissions")
		c.Abort()
	}
}
