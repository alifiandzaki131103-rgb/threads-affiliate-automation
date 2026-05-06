package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
)

func GetPostsByUserID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]model.Post, error) {
	rows, err := pool.Query(ctx, `
		SELECT p.id, p.account_id, p.link_id, p.content, p.link_placement, p.persona, p.format,
		       p.scheduled_at, p.published_at, p.thread_id, p.status, p.created_at
		FROM posts p
		JOIN threads_accounts ta ON p.account_id = ta.id
		WHERE ta.user_id = $1
		ORDER BY p.created_at DESC
		LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.AccountID, &p.LinkID, &p.Content, &p.LinkPlacement,
			&p.Persona, &p.Format, &p.ScheduledAt, &p.PublishedAt, &p.ThreadID, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

func UpdatePostStatus(ctx context.Context, pool *pgxpool.Pool, postID, userID uuid.UUID, status string) error {
	result, err := pool.Exec(ctx, `
		UPDATE posts SET status = $1
		WHERE id = $2
		AND account_id IN (SELECT id FROM threads_accounts WHERE user_id = $3)`, status, postID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func GetPostByID(ctx context.Context, pool *pgxpool.Pool, postID uuid.UUID) (*model.Post, error) {
	var p model.Post
	err := pool.QueryRow(ctx, `
		SELECT id, account_id, link_id, content, link_placement, persona, format,
		       scheduled_at, published_at, thread_id, status, created_at
		FROM posts WHERE id = $1`, postID).Scan(
		&p.ID, &p.AccountID, &p.LinkID, &p.Content, &p.LinkPlacement,
		&p.Persona, &p.Format, &p.ScheduledAt, &p.PublishedAt, &p.ThreadID, &p.Status, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetPostByIDForUser(ctx context.Context, pool *pgxpool.Pool, postID, userID uuid.UUID) (*model.Post, error) {
	var p model.Post
	err := pool.QueryRow(ctx, `
		SELECT p.id, p.account_id, p.link_id, p.content, p.link_placement, p.persona, p.format,
		       p.scheduled_at, p.published_at, p.thread_id, p.status, p.created_at
		FROM posts p
		JOIN threads_accounts ta ON p.account_id = ta.id
		WHERE p.id = $1 AND ta.user_id = $2`, postID, userID).Scan(
		&p.ID, &p.AccountID, &p.LinkID, &p.Content, &p.LinkPlacement,
		&p.Persona, &p.Format, &p.ScheduledAt, &p.PublishedAt, &p.ThreadID, &p.Status, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetScheduledPostsDue(ctx context.Context, pool *pgxpool.Pool) ([]model.Post, error) {
	rows, err := pool.Query(ctx, `
		SELECT p.id, p.account_id, p.content, p.scheduled_at, p.status
		FROM posts p
		WHERE p.status = 'approved' AND p.scheduled_at <= $1
		ORDER BY p.scheduled_at ASC
		LIMIT 25`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.AccountID, &p.Content, &p.ScheduledAt, &p.Status); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}
