package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	jwtpkg "github.com/sungminna/upbit-trading-platform/pkg/jwt"
)

const (
	userIDKey = "user_id"
	emailKey  = "email"
)

// AuthMiddleware creates authentication middleware
func AuthMiddleware(jwtManager *jwtpkg.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := jwtManager.Verify(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(userIDKey, claims.UserID)
		c.Set(emailKey, claims.Email)

		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (uuid.UUID, error) {
	value, exists := c.Get(userIDKey)
	if !exists {
		return uuid.Nil, ErrUserNotFound
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrInvalidUserID
	}

	return userID, nil
}

// GetEmail extracts email from context
func GetEmail(c *gin.Context) (string, error) {
	value, exists := c.Get(emailKey)
	if !exists {
		return "", ErrUserNotFound
	}

	email, ok := value.(string)
	if !ok {
		return "", ErrInvalidEmail
	}

	return email, nil
}

var (
	ErrUserNotFound  = &AuthError{message: "user not found in context"}
	ErrInvalidUserID = &AuthError{message: "invalid user ID in context"}
	ErrInvalidEmail  = &AuthError{message: "invalid email in context"}
)

// AuthError represents an authentication error
type AuthError struct {
	message string
}

func (e *AuthError) Error() string {
	return e.message
}
