package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
)

func CreateAccount(ctx context.Context, pool *pgxpool.Pool, account *model.ThreadsAccount) error {
	account.ID = uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO threads_accounts (id, user_id, threads_user_id, access_token, persona, niche, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		account.ID, account.UserID, account.ThreadsUserID, account.AccessToken,
		account.Persona, account.Niche, account.Status,
	)
	return err
}

func GetAccountsByUserID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]model.ThreadsAccount, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, user_id, threads_user_id, persona, niche, status, created_at
		FROM threads_accounts WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.ThreadsAccount
	for rows.Next() {
		var a model.ThreadsAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.ThreadsUserID, &a.Persona, &a.Niche, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return accounts, nil
}

func GetAccountByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*model.ThreadsAccount, error) {
	var a model.ThreadsAccount
	err := pool.QueryRow(ctx, `
		SELECT id, user_id, threads_user_id, access_token, persona, niche, status, created_at
		FROM threads_accounts WHERE id = $1`, id).Scan(
		&a.ID, &a.UserID, &a.ThreadsUserID, &a.AccessToken, &a.Persona, &a.Niche, &a.Status, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func GetAccountByIDForUser(ctx context.Context, pool *pgxpool.Pool, id, userID uuid.UUID) (*model.ThreadsAccount, error) {
	var a model.ThreadsAccount
	err := pool.QueryRow(ctx, `
		SELECT id, user_id, threads_user_id, access_token, persona, niche, status, created_at
		FROM threads_accounts WHERE id = $1 AND user_id = $2`, id, userID).Scan(
		&a.ID, &a.UserID, &a.ThreadsUserID, &a.AccessToken, &a.Persona, &a.Niche, &a.Status, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func UpdateAccount(ctx context.Context, pool *pgxpool.Pool, id, userID uuid.UUID, persona, niche, status string) error {
	result, err := pool.Exec(ctx, `
		UPDATE threads_accounts SET persona = $1, niche = $2, status = $3 WHERE id = $4 AND user_id = $5`,
		persona, niche, status, id, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func UpdateAccountAutoMode(ctx context.Context, pool *pgxpool.Pool, id, userID uuid.UUID, autoMode bool) error {
	var query string
	if autoMode {
		query = `UPDATE threads_accounts SET auto_mode = true, auto_mode_enabled_at = NOW() WHERE id = $1 AND user_id = $2`
	} else {
		query = `UPDATE threads_accounts SET auto_mode = false, auto_mode_enabled_at = NULL WHERE id = $1 AND user_id = $2`
	}
	result, err := pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func DeleteAccount(ctx context.Context, pool *pgxpool.Pool, id, userID uuid.UUID) error {
	result, err := pool.Exec(ctx, `DELETE FROM threads_accounts WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func IsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
