package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardStats struct {
	TotalLinks     int        `json:"total_links"`
	TotalClicks    int        `json:"total_clicks"`
	TotalPosts     int        `json:"total_posts"`
	PublishedPosts int        `json:"published_posts"`
	PendingPosts   int        `json:"pending_posts"`
	TopLinks       []TopLink  `json:"top_links"`
}

type TopLink struct {
	ID          uuid.UUID `json:"id"`
	ProductName string    `json:"product_name"`
	ShortSlug   string    `json:"short_slug"`
	Platform    string    `json:"platform"`
	ClickCount  int       `json:"click_count"`
}

func GetDashboardStats(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Total links & clicks
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(al.click_count), 0)
		FROM affiliate_links al
		JOIN products p ON al.product_id = p.id
		WHERE p.user_id = $1`, userID).Scan(&stats.TotalLinks, &stats.TotalClicks)
	if err != nil {
		return nil, err
	}

	// Total posts, published, pending
	err = pool.QueryRow(ctx, `
		SELECT 
			COALESCE(COUNT(*), 0),
			COALESCE(SUM(CASE WHEN p.status = 'published' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN p.status IN ('draft', 'pending_review', 'approved') THEN 1 ELSE 0 END), 0)
		FROM posts p
		JOIN threads_accounts ta ON p.account_id = ta.id
		WHERE ta.user_id = $1`, userID).Scan(&stats.TotalPosts, &stats.PublishedPosts, &stats.PendingPosts)
	if err != nil {
		return nil, err
	}

	// Top 5 links by clicks
	rows, err := pool.Query(ctx, `
		SELECT al.id, COALESCE(pr.name, 'Unknown'), al.short_slug, al.platform, al.click_count
		FROM affiliate_links al
		JOIN products pr ON al.product_id = pr.id
		WHERE pr.user_id = $1
		ORDER BY al.click_count DESC
		LIMIT 5`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tl TopLink
		if err := rows.Scan(&tl.ID, &tl.ProductName, &tl.ShortSlug, &tl.Platform, &tl.ClickCount); err != nil {
			return nil, err
		}
		stats.TopLinks = append(stats.TopLinks, tl)
	}

	if stats.TopLinks == nil {
		stats.TopLinks = []TopLink{}
	}

	return stats, nil
}
