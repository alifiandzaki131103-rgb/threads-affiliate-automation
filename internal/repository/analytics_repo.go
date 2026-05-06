package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClicksByDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type ClicksByLink struct {
	LinkID      uuid.UUID `json:"link_id"`
	ProductName string    `json:"product_name"`
	ShortSlug   string    `json:"short_slug"`
	Clicks      int       `json:"clicks"`
}

type ClickAnalytics struct {
	ClicksByDay    []ClicksByDay  `json:"clicks_by_day"`
	ClicksByLink   []ClicksByLink `json:"clicks_by_link"`
	TotalClicks    int            `json:"total_clicks"`
	AvgDailyClicks float64       `json:"avg_daily_clicks"`
}

func GetClickAnalytics(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, since time.Time) (*ClickAnalytics, error) {
	analytics := &ClickAnalytics{}

	// Clicks by day
	rows, err := pool.Query(ctx, `
		SELECT DATE(cl.created_at) as date, COUNT(*) as count
		FROM click_logs cl
		JOIN affiliate_links al ON cl.link_id = al.id
		JOIN products p ON al.product_id = p.id
		WHERE p.user_id = $1 AND cl.created_at >= $2
		GROUP BY DATE(cl.created_at)
		ORDER BY date`, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cbd ClicksByDay
		var date time.Time
		if err := rows.Scan(&date, &cbd.Count); err != nil {
			return nil, err
		}
		cbd.Date = date.Format("2006-01-02")
		analytics.ClicksByDay = append(analytics.ClicksByDay, cbd)
	}
	if analytics.ClicksByDay == nil {
		analytics.ClicksByDay = []ClicksByDay{}
	}

	// Clicks by link (top 10)
	linkRows, err := pool.Query(ctx, `
		SELECT al.id, COALESCE(p.name, 'Unknown') as product_name, al.short_slug, COUNT(*) as clicks
		FROM click_logs cl
		JOIN affiliate_links al ON cl.link_id = al.id
		JOIN products p ON al.product_id = p.id
		WHERE p.user_id = $1 AND cl.created_at >= $2
		GROUP BY al.id, p.name, al.short_slug
		ORDER BY clicks DESC
		LIMIT 10`, userID, since)
	if err != nil {
		return nil, err
	}
	defer linkRows.Close()

	for linkRows.Next() {
		var cbl ClicksByLink
		if err := linkRows.Scan(&cbl.LinkID, &cbl.ProductName, &cbl.ShortSlug, &cbl.Clicks); err != nil {
			return nil, err
		}
		analytics.ClicksByLink = append(analytics.ClicksByLink, cbl)
	}
	if analytics.ClicksByLink == nil {
		analytics.ClicksByLink = []ClicksByLink{}
	}

	// Total clicks in period
	err = pool.QueryRow(ctx, `
		SELECT COALESCE(COUNT(*), 0)
		FROM click_logs cl
		JOIN affiliate_links al ON cl.link_id = al.id
		JOIN products p ON al.product_id = p.id
		WHERE p.user_id = $1 AND cl.created_at >= $2`, userID, since).Scan(&analytics.TotalClicks)
	if err != nil {
		return nil, err
	}

	// Calculate average daily clicks
	days := time.Since(since).Hours() / 24
	if days < 1 {
		days = 1
	}
	analytics.AvgDailyClicks = float64(analytics.TotalClicks) / days

	return analytics, nil
}
