package repository

import (
	"context"
	"fmt"

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

// GetLinksWithProductByUserID returns affiliate links with product info for all products owned by a user.
func GetLinksWithProductByUserID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]model.LinkWithProduct, error) {
	rows, err := pool.Query(ctx, `
		SELECT al.id, al.product_id, al.original_url, al.short_slug, al.platform, al.status, al.click_count,
		       COALESCE(p.name, 'Unknown'), COALESCE(p.price, 0)::float8, COALESCE(p.category, ''), al.created_at
		FROM affiliate_links al
		JOIN products p ON al.product_id = p.id
		WHERE p.user_id = $1
		ORDER BY al.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []model.LinkWithProduct
	for rows.Next() {
		var l model.LinkWithProduct
		if err := rows.Scan(&l.ID, &l.ProductID, &l.OriginalURL, &l.ShortSlug, &l.Platform,
			&l.Status, &l.ClickCount, &l.ProductName, &l.Price, &l.Category, &l.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, l)
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

// DeleteLinkByUser deletes an affiliate link owned by the given user.
func DeleteLinkByUser(ctx context.Context, pool *pgxpool.Pool, linkID, userID uuid.UUID) error {
	result, err := pool.Exec(ctx, `
		DELETE FROM affiliate_links
		WHERE id = $1 AND product_id IN (SELECT id FROM products WHERE user_id = $2)
	`, linkID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("link not found or not owned by user")
	}
	return nil
}

// GetAllActiveLinks returns all affiliate links with status 'active'.
func GetAllActiveLinks(ctx context.Context, pool *pgxpool.Pool) ([]model.AffiliateLink, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, original_url FROM affiliate_links WHERE status = 'active'
	`)
	if err != nil {
		return nil, fmt.Errorf("query active links: %w", err)
	}
	defer rows.Close()

	var links []model.AffiliateLink
	for rows.Next() {
		var link model.AffiliateLink
		if err := rows.Scan(&link.ID, &link.OriginalURL); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return links, nil
}

// UpdateLinkHealth updates the health_status and last_checked_at for a link.
func UpdateLinkHealth(ctx context.Context, pool *pgxpool.Pool, linkID uuid.UUID, healthStatus string) error {
	_, err := pool.Exec(ctx, `
		UPDATE affiliate_links SET health_status = $1, last_checked_at = NOW() WHERE id = $2
	`, healthStatus, linkID)
	return err
}
