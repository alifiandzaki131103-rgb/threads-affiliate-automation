package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostPerformance represents a post's performance data for learning analysis
type PostPerformance struct {
	UserID    uuid.UUID
	AccountID uuid.UUID
	PostID    uuid.UUID
	Persona   string
	Format    string
	HourWIB   int
	Clicks    int
	Views     int
}

// PersonaWeight represents a persona weight record
type PersonaWeight struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Persona       string
	Weight        float64
	TotalPosts    int
	TotalClicks   int
	AvgEngagement float64
	LastUpdatedAt time.Time
}

// FormatWeight represents a format weight record
type FormatWeight struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Format        string
	Weight        float64
	TotalPosts    int
	TotalClicks   int
	AvgEngagement float64
	LastUpdatedAt time.Time
}

// TimeWeight represents a time weight record
type TimeWeight struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	HourWIB       int
	Weight        float64
	TotalPosts    int
	TotalClicks   int
	LastUpdatedAt time.Time
}

// GetPostPerformanceLast7Days returns all published posts from the last 7 days
// joined with their affiliate link click counts, grouped by user.
func GetPostPerformanceLast7Days(ctx context.Context, pool *pgxpool.Pool) ([]PostPerformance, error) {
	rows, err := pool.Query(ctx, `
		SELECT ta.user_id, p.account_id, p.id, p.persona, p.format,
		       EXTRACT(HOUR FROM (p.published_at AT TIME ZONE 'Asia/Jakarta'))::int AS hour_wib,
		       COALESCE(al.click_count, 0) AS clicks,
		       COALESCE(pa.views, 0) AS views
		FROM posts p
		JOIN threads_accounts ta ON p.account_id = ta.id
		LEFT JOIN affiliate_links al ON p.link_id = al.id
		LEFT JOIN post_analytics pa ON pa.post_id = p.id
		WHERE p.status = 'published'
		  AND p.published_at > NOW() - INTERVAL '7 days'
		  AND p.published_at IS NOT NULL
		ORDER BY ta.user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PostPerformance
	for rows.Next() {
		var pp PostPerformance
		if err := rows.Scan(&pp.UserID, &pp.AccountID, &pp.PostID, &pp.Persona, &pp.Format, &pp.HourWIB, &pp.Clicks, &pp.Views); err != nil {
			return nil, err
		}
		results = append(results, pp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// UpsertPersonaWeight inserts or updates a persona weight for a user.
func UpsertPersonaWeight(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, persona string, weight float64, totalPosts, totalClicks int, avgEngagement float64) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO persona_weights (id, user_id, persona, weight, total_posts, total_clicks, avg_engagement, last_updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, persona) DO UPDATE SET
			weight = EXCLUDED.weight,
			total_posts = EXCLUDED.total_posts,
			total_clicks = EXCLUDED.total_clicks,
			avg_engagement = EXCLUDED.avg_engagement,
			last_updated_at = NOW()`,
		userID, persona, weight, totalPosts, totalClicks, avgEngagement)
	return err
}

// UpsertFormatWeight inserts or updates a format weight for a user.
func UpsertFormatWeight(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, format string, weight float64, totalPosts, totalClicks int, avgEngagement float64) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO format_weights (id, user_id, format, weight, total_posts, total_clicks, avg_engagement, last_updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, format) DO UPDATE SET
			weight = EXCLUDED.weight,
			total_posts = EXCLUDED.total_posts,
			total_clicks = EXCLUDED.total_clicks,
			avg_engagement = EXCLUDED.avg_engagement,
			last_updated_at = NOW()`,
		userID, format, weight, totalPosts, totalClicks, avgEngagement)
	return err
}

// UpsertTimeWeight inserts or updates a time weight for a user.
func UpsertTimeWeight(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, hourWIB int, weight float64, totalPosts, totalClicks int) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO time_weights (id, user_id, hour_wib, weight, total_posts, total_clicks, last_updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id, hour_wib) DO UPDATE SET
			weight = EXCLUDED.weight,
			total_posts = EXCLUDED.total_posts,
			total_clicks = EXCLUDED.total_clicks,
			last_updated_at = NOW()`,
		userID, hourWIB, weight, totalPosts, totalClicks)
	return err
}

// InsertWeeklyReport inserts a new weekly report for a user.
func InsertWeeklyReport(ctx context.Context, pool *pgxpool.Pool, report *WeeklyReport) error {
	recommendationsJSON, err := json.Marshal(report.Recommendations)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO weekly_reports (id, user_id, week_start, week_end, total_posts, total_clicks, total_views,
		                            best_persona, best_format, best_hour, top_post_id, recommendations, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())`,
		report.UserID, report.WeekStart, report.WeekEnd, report.TotalPosts, report.TotalClicks,
		report.TotalViews, report.BestPersona, report.BestFormat, report.BestHour,
		report.TopPostID, recommendationsJSON)
	return err
}

// GetTopPostForUser returns the post ID with the most clicks in the last 7 days for a user.
func GetTopPostForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*uuid.UUID, error) {
	var postID uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT p.id
		FROM posts p
		JOIN threads_accounts ta ON p.account_id = ta.id
		LEFT JOIN affiliate_links al ON p.link_id = al.id
		WHERE ta.user_id = $1
		  AND p.status = 'published'
		  AND p.published_at > NOW() - INTERVAL '7 days'
		ORDER BY COALESCE(al.click_count, 0) DESC
		LIMIT 1`, userID).Scan(&postID)
	if err != nil {
		return nil, err
	}
	return &postID, nil
}
