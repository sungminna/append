package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
)

// UserRepository defines methods for user data access
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// UserAPIKeyRepository defines methods for user API key data access
type UserAPIKeyRepository interface {
	Create(ctx context.Context, apiKey *model.UserAPIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.UserAPIKey, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*model.UserAPIKey, error)
	Update(ctx context.Context, apiKey *model.UserAPIKey) error
	Delete(ctx context.Context, id uuid.UUID) error
}
