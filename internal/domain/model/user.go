package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a platform user
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password_hash"` // Never expose password in JSON
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UserAPIKey represents Upbit API credentials for a user
type UserAPIKey struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	AccessKey   string    `json:"access_key" db:"access_key"`
	SecretKey   string    `json:"-" db:"secret_key"` // Never expose secret in JSON
	Description string    `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// NewUser creates a new user with generated UUID
func NewUser(email, passwordHash string) *User {
	now := time.Now()
	return &User{
		ID:        uuid.New(),
		Email:     email,
		Password:  passwordHash,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewUserAPIKey creates a new API key for a user
func NewUserAPIKey(userID uuid.UUID, accessKey, secretKey, description string) *UserAPIKey {
	now := time.Now()
	return &UserAPIKey{
		ID:          uuid.New(),
		UserID:      userID,
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		Description: description,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
