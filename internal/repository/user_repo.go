package repository

import (
	"context"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateUser inserts a new user and sets the generated ID on the model.
func CreateUser(ctx context.Context, pool *pgxpool.Pool, user *model.User) error {
	return pool.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash, plan)
		VALUES (gen_random_uuid(), $1, $2, $3)
		RETURNING id
	`, user.Email, user.PasswordHash, user.Plan).Scan(&user.ID)
}

// GetUserByEmail returns a user by email address.
func GetUserByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (*model.User, error) {
	user := &model.User{}
	err := pool.QueryRow(ctx, `
		SELECT id, email, password_hash, plan, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Plan,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID returns a user by ID.
func GetUserByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	err := pool.QueryRow(ctx, `
		SELECT id, email, password_hash, plan, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Plan,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}
