package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/domain/repository"
	jwtpkg "github.com/sungminna/upbit-trading-platform/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication and user management
type Service struct {
	userRepo       repository.UserRepository
	userAPIKeyRepo repository.UserAPIKeyRepository
	jwtManager     *jwtpkg.Manager
}

// NewService creates a new auth service
func NewService(
	userRepo repository.UserRepository,
	userAPIKeyRepo repository.UserAPIKeyRepository,
	jwtManager *jwtpkg.Manager,
) *Service {
	return &Service{
		userRepo:       userRepo,
		userAPIKeyRepo: userAPIKeyRepo,
		jwtManager:     jwtManager,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

// Register registers a new user
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// Check if user already exists
	existingUser, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := model.NewUser(req.Email, string(hashedPassword))
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate JWT token
	token, err := s.jwtManager.Generate(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Don't expose password hash in response
	user.Password = ""

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Generate JWT token
	token, err := s.jwtManager.Generate(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Don't expose password hash in response
	user.Password = ""

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

// GetUser retrieves user information
func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Don't expose password hash
	user.Password = ""

	return user, nil
}

// AddAPIKey adds a new Upbit API key for a user
func (s *Service) AddAPIKey(ctx context.Context, userID uuid.UUID, accessKey, secretKey, description string) (*model.UserAPIKey, error) {
	// Deactivate existing active API keys (only one active key per user)
	existingKeys, err := s.userAPIKeyRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing API keys: %w", err)
	}

	for _, key := range existingKeys {
		if key.IsActive {
			key.IsActive = false
			if err := s.userAPIKeyRepo.Update(ctx, key); err != nil {
				return nil, fmt.Errorf("failed to deactivate existing API key: %w", err)
			}
		}
	}

	// Create new API key
	apiKey := model.NewUserAPIKey(userID, accessKey, secretKey, description)
	if err := s.userAPIKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Don't expose secret key in response
	apiKey.SecretKey = ""

	return apiKey, nil
}

// GetActiveAPIKey retrieves the active API key for a user
func (s *Service) GetActiveAPIKey(ctx context.Context, userID uuid.UUID) (*model.UserAPIKey, error) {
	apiKey, err := s.userAPIKeyRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active API key: %w", err)
	}

	return apiKey, nil
}

// DeactivateAPIKey deactivates an API key
func (s *Service) DeactivateAPIKey(ctx context.Context, userID, apiKeyID uuid.UUID) error {
	apiKey, err := s.userAPIKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}

	if apiKey.UserID != userID {
		return fmt.Errorf("unauthorized: API key does not belong to user")
	}

	apiKey.IsActive = false
	if err := s.userAPIKeyRepo.Update(ctx, apiKey); err != nil {
		return fmt.Errorf("failed to deactivate API key: %w", err)
	}

	return nil
}
