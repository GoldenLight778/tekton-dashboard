package crypto

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

// Claims represents the JWT payload claims for the platform
type Claims struct {
	UserID   string   `json:"uid"`
	Username string   `json:"name"`
	Role     string   `json:"role"`
	AuthMode string   `json:"auth_mode,omitempty"` // "password" or "k8s_token"
	K8sToken string   `json:"k8s_token,omitempty"`
	Groups   []string `json:"groups,omitempty"`
	jwt.RegisteredClaims
}

// HashPassword generates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword compares a bcrypt hashed password with its possible plaintext equivalent
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken creates a signed JWT token
func GenerateToken(userID, username, role, secret string, duration time.Duration) (string, error) {
	return GenerateExtendedToken(Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		AuthMode: "password",
	}, secret, duration)
}

// GenerateExtendedToken creates a signed JWT token with full K8s RBAC claims
func GenerateExtendedToken(claims Claims, secret string, duration time.Duration) (string, error) {
	if claims.AuthMode == "" {
		claims.AuthMode = "password"
	}
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "devops-platform",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenStr, nil
}

// ParseToken validates and parses a signed JWT token
func ParseToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
