package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/api/middleware"
	"github.com/sungminna/upbit-trading-platform/internal/service/auth"
)

// UserHandler handles user-related endpoints
type UserHandler struct {
	authService *auth.Service
}

// NewUserHandler creates a new user handler
func NewUserHandler(authService *auth.Service) *UserHandler {
	return &UserHandler{
		authService: authService,
	}
}

// Register handles user registration
// POST /api/v1/auth/register
func (h *UserHandler) Register(c *gin.Context) {
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login handles user login
// POST /api/v1/auth/login
func (h *UserHandler) Login(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetMe retrieves current user information
// GET /api/v1/users/me
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// AddAPIKeyRequest represents a request to add an API key
type AddAPIKeyRequest struct {
	AccessKey   string `json:"access_key" binding:"required"`
	SecretKey   string `json:"secret_key" binding:"required"`
	Description string `json:"description"`
}

// AddAPIKey adds a new Upbit API key for the user
// POST /api/v1/users/api-keys
func (h *UserHandler) AddAPIKey(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req AddAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey, err := h.authService.AddAPIKey(c.Request.Context(), userID, req.AccessKey, req.SecretKey, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, apiKey)
}

// GetActiveAPIKey retrieves the active API key for the user
// GET /api/v1/users/api-keys/active
func (h *UserHandler) GetActiveAPIKey(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	apiKey, err := h.authService.GetActiveAPIKey(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Don't expose secret key
	apiKey.SecretKey = ""

	c.JSON(http.StatusOK, apiKey)
}

// DeactivateAPIKey deactivates an API key
// DELETE /api/v1/users/api-keys/:id
func (h *UserHandler) DeactivateAPIKey(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	apiKeyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid API key ID"})
		return
	}

	if err := h.authService.DeactivateAPIKey(c.Request.Context(), userID, apiKeyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deactivated"})
}
