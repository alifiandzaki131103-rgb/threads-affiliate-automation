package repository

import (
	"context"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateLink inserts a new affiliate link and sets the generated ID on the model.
func CreateLink(ctx context.Context, pool *pgxpool.Pool, link *model.AffiliateLink) error {
	return pool.QueryRow(ctx, `
		INSERT INTO affiliate_links (product_id, original_url, short_slug, platform, status, click_count)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, link.ProductID, link.OriginalURL, link.ShortSlug, link.Platform, link.Status, link.ClickCount).Scan(&link.ID)
}

// GetLinksByUserID returns affiliate links for all products owned by a user.
func GetLinksByUserID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]model.AffiliateLink, error) {
	rows, err := pool.Query(ctx, `
		SELECT al.id, al.product_id, al.original_url, al.short_slug, al.platform, al.status, al.click_count, al.created_at
		FROM affiliate_links al
		JOIN products p ON p.id = al.product_id
		WHERE p.user_id = $1
		ORDER BY al.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := make([]model.AffiliateLink, 0)
	for rows.Next() {
		var link model.AffiliateLink
		if err := rows.Scan(
			&link.ID,
			&link.ProductID,
			&link.OriginalURL,
			&link.ShortSlug,
			&link.Platform,
			&link.Status,
			&link.ClickCount,
			&link.CreatedAt,
		); err != nil {
			return nil, err
		}

		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

// GetLinkBySlug returns an affiliate link by short slug.
func GetLinkBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (*model.AffiliateLink, error) {
	link := &model.AffiliateLink{}
	err := pool.QueryRow(ctx, `
		SELECT id, product_id, original_url, short_slug, platform, status, click_count, created_at
		FROM affiliate_links
		WHERE short_slug = $1
	`, slug).Scan(
		&link.ID,
		&link.ProductID,
		&link.OriginalURL,
		&link.ShortSlug,
		&link.Platform,
		&link.Status,
		&link.ClickCount,
		&link.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return link, nil
}

// IncrementClickCount increments an affiliate link's click counter.
func IncrementClickCount(ctx context.Context, pool *pgxpool.Pool, linkID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE affiliate_links
		SET click_count = click_count + 1
		WHERE id = $1
	`, linkID)
	return err
}

// CreateProduct inserts a new product and sets the generated ID on the model.
func CreateProduct(ctx context.Context, pool *pgxpool.Pool, product *model.Product) error {
	return pool.QueryRow(ctx, `
		INSERT INTO products (user_id, name, price, category, platform, image_url, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, product.UserID, product.Name, product.Price, product.Category, product.Platform, product.ImageURL, product.Status).Scan(&product.ID)
}
