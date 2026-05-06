package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
)

// CreateABTest inserts a new A/B test record.
func CreateABTest(ctx context.Context, pool *pgxpool.Pool, test *model.ABTest) error {
	return pool.QueryRow(ctx, `
		INSERT INTO ab_tests (link_id, variant_a_post_id, variant_b_post_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		test.LinkID, test.VariantAPostID, test.VariantBPostID, test.Status,
	).Scan(&test.ID, &test.CreatedAt)
}

// GetABTestsByUserID returns all A/B tests for a user (via their affiliate links).
func GetABTestsByUserID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]model.ABTest, error) {
	rows, err := pool.Query(ctx, `
		SELECT ab.id, ab.link_id, ab.variant_a_post_id, ab.variant_b_post_id,
		       ab.winner, ab.status, ab.created_at, ab.completed_at
		FROM ab_tests ab
		JOIN affiliate_links al ON ab.link_id = al.id
		JOIN products p ON al.product_id = p.id
		WHERE p.user_id = $1
		ORDER BY ab.created_at DESC
		LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tests []model.ABTest
	for rows.Next() {
		var t model.ABTest
		if err := rows.Scan(&t.ID, &t.LinkID, &t.VariantAPostID, &t.VariantBPostID,
			&t.Winner, &t.Status, &t.CreatedAt, &t.CompletedAt); err != nil {
			return nil, err
		}
		tests = append(tests, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tests, nil
}

// CreatePostReturningID inserts a new post and returns its ID.
func CreatePostReturningID(ctx context.Context, pool *pgxpool.Pool, accountID, linkID uuid.UUID, content, linkPlacement, persona, format, status string, scheduledAt time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO posts (id, account_id, link_id, content, link_placement, persona, format, scheduled_at, status, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id`,
		accountID, linkID, content, linkPlacement, persona, format, scheduledAt, status,
	).Scan(&id)
	return id, err
}
