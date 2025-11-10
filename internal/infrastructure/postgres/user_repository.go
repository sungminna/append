package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sungminna/upbit-trading-platform/internal/domain/model"
	"github.com/sungminna/upbit-trading-platform/internal/domain/repository"
)

type userRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new PostgreSQL user repository
func NewUserRepository(pool *pgxpool.Pool) repository.UserRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.Password, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var user model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var user model.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET email = $2, password_hash = $3, updated_at = $4
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.Password, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

type userAPIKeyRepository struct {
	pool *pgxpool.Pool
}

// NewUserAPIKeyRepository creates a new PostgreSQL user API key repository
func NewUserAPIKeyRepository(pool *pgxpool.Pool) repository.UserAPIKeyRepository {
	return &userAPIKeyRepository{pool: pool}
}

func (r *userAPIKeyRepository) Create(ctx context.Context, apiKey *model.UserAPIKey) error {
	query := `
		INSERT INTO user_api_keys (id, user_id, access_key, secret_key, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		apiKey.ID, apiKey.UserID, apiKey.AccessKey, apiKey.SecretKey,
		apiKey.Description, apiKey.IsActive, apiKey.CreatedAt, apiKey.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user API key: %w", err)
	}
	return nil
}

func (r *userAPIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.UserAPIKey, error) {
	query := `
		SELECT id, user_id, access_key, secret_key, description, is_active, created_at, updated_at
		FROM user_api_keys
		WHERE id = $1
	`
	var apiKey model.UserAPIKey
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&apiKey.ID, &apiKey.UserID, &apiKey.AccessKey, &apiKey.SecretKey,
		&apiKey.Description, &apiKey.IsActive, &apiKey.CreatedAt, &apiKey.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user API key not found")
		}
		return nil, fmt.Errorf("failed to get user API key: %w", err)
	}
	return &apiKey, nil
}

func (r *userAPIKeyRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error) {
	query := `
		SELECT id, user_id, access_key, secret_key, description, is_active, created_at, updated_at
		FROM user_api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []*model.UserAPIKey
	for rows.Next() {
		var apiKey model.UserAPIKey
		err := rows.Scan(
			&apiKey.ID, &apiKey.UserID, &apiKey.AccessKey, &apiKey.SecretKey,
			&apiKey.Description, &apiKey.IsActive, &apiKey.CreatedAt, &apiKey.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user API key: %w", err)
		}
		apiKeys = append(apiKeys, &apiKey)
	}

	return apiKeys, nil
}

func (r *userAPIKeyRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*model.UserAPIKey, error) {
	query := `
		SELECT id, user_id, access_key, secret_key, description, is_active, created_at, updated_at
		FROM user_api_keys
		WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1
	`
	var apiKey model.UserAPIKey
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&apiKey.ID, &apiKey.UserID, &apiKey.AccessKey, &apiKey.SecretKey,
		&apiKey.Description, &apiKey.IsActive, &apiKey.CreatedAt, &apiKey.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no active user API key found")
		}
		return nil, fmt.Errorf("failed to get active user API key: %w", err)
	}
	return &apiKey, nil
}

func (r *userAPIKeyRepository) Update(ctx context.Context, apiKey *model.UserAPIKey) error {
	query := `
		UPDATE user_api_keys
		SET access_key = $2, secret_key = $3, description = $4, is_active = $5, updated_at = $6
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query,
		apiKey.ID, apiKey.AccessKey, apiKey.SecretKey, apiKey.Description,
		apiKey.IsActive, apiKey.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user API key: %w", err)
	}
	return nil
}

func (r *userAPIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_api_keys WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user API key: %w", err)
	}
	return nil
}
