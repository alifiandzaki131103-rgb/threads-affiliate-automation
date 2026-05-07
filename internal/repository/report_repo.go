package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WeeklyReport represents a weekly learning/insight report
type WeeklyReport struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	WeekStart       time.Time
	WeekEnd         time.Time
	TotalPosts      int
	TotalClicks     int
	TotalViews      int
	BestPersona     string
	BestFormat      string
	BestHour        int
	TopPostID       *uuid.UUID
	Recommendations []string
	CreatedAt       time.Time
}

// PersonaPerformance represents persona performance data
type PersonaPerformance struct {
	Persona string  `json:"persona"`
	Weight  float64 `json:"weight"`
	Posts   int     `json:"posts"`
	Clicks  int     `json:"clicks"`
}

// FormatPerformance represents format performance data
type FormatPerformance struct {
	Format string  `json:"format"`
	Weight float64 `json:"weight"`
	Posts  int     `json:"posts"`
	Clicks int     `json:"clicks"`
}

// HourPerformance represents posting hour performance data
type HourPerformance struct {
	Hour   int     `json:"hour"`
	Weight float64 `json:"weight"`
	Clicks int     `json:"clicks"`
}

// CurrentWeekStats represents stats for the current week
type CurrentWeekStats struct {
	PostsPublished   int     `json:"posts_published"`
	TotalClicks      int     `json:"total_clicks"`
	AvgClicksPerPost float64 `json:"avg_clicks_per_post"`
}

// WeeklyReportResponse is the JSON-friendly version of WeeklyReport for API responses
type WeeklyReportResponse struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	WeekStart       time.Time  `json:"week_start"`
	WeekEnd         time.Time  `json:"week_end"`
	TotalPosts      int        `json:"total_posts"`
	TotalClicks     int        `json:"total_clicks"`
	TotalViews      int        `json:"total_views"`
	BestPersona     string     `json:"best_persona"`
	BestFormat      string     `json:"best_format"`
	BestHour        int        `json:"best_hour"`
	TopPostID       *uuid.UUID `json:"top_post_id"`
	Recommendations []string   `json:"recommendations"`
	CreatedAt       time.Time  `json:"created_at"`
}

// GetWeeklyReports returns paginated weekly reports for a user (newest first)
func GetWeeklyReports(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, limit, offset int) ([]WeeklyReportResponse, int, error) {
	// Get total count
	var total int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM weekly_reports WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := pool.Query(ctx, `
		SELECT id, user_id, week_start, week_end, total_posts, total_clicks, total_views,
		       COALESCE(best_persona, ''), COALESCE(best_format, ''), COALESCE(best_hour, 0),
		       top_post_id, recommendations, created_at
		FROM weekly_reports
		WHERE user_id = $1
		ORDER BY week_start DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []WeeklyReportResponse
	for rows.Next() {
		var r WeeklyReportResponse
		var recommendations []byte
		if err := rows.Scan(
			&r.ID, &r.UserID, &r.WeekStart, &r.WeekEnd,
			&r.TotalPosts, &r.TotalClicks, &r.TotalViews,
			&r.BestPersona, &r.BestFormat, &r.BestHour,
			&r.TopPostID, &recommendations, &r.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		r.Recommendations = parseRecommendations(recommendations)
		reports = append(reports, r)
	}

	return reports, total, nil
}

// GetLatestWeeklyReport returns the most recent weekly report for a user
func GetLatestWeeklyReport(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*WeeklyReportResponse, error) {
	var r WeeklyReportResponse
	var recommendations []byte
	err := pool.QueryRow(ctx, `
		SELECT id, user_id, week_start, week_end, total_posts, total_clicks, total_views,
		       COALESCE(best_persona, ''), COALESCE(best_format, ''), COALESCE(best_hour, 0),
		       top_post_id, recommendations, created_at
		FROM weekly_reports
		WHERE user_id = $1
		ORDER BY week_start DESC
		LIMIT 1`, userID,
	).Scan(
		&r.ID, &r.UserID, &r.WeekStart, &r.WeekEnd,
		&r.TotalPosts, &r.TotalClicks, &r.TotalViews,
		&r.BestPersona, &r.BestFormat, &r.BestHour,
		&r.TopPostID, &recommendations, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.Recommendations = parseRecommendations(recommendations)
	return &r, nil
}

// GetWeeklyReportByID returns a specific weekly report by ID (scoped to user)
func GetWeeklyReportByID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, reportID uuid.UUID) (*WeeklyReportResponse, error) {
	var r WeeklyReportResponse
	var recommendations []byte
	err := pool.QueryRow(ctx, `
		SELECT id, user_id, week_start, week_end, total_posts, total_clicks, total_views,
		       COALESCE(best_persona, ''), COALESCE(best_format, ''), COALESCE(best_hour, 0),
		       top_post_id, recommendations, created_at
		FROM weekly_reports
		WHERE id = $1 AND user_id = $2`, reportID, userID,
	).Scan(
		&r.ID, &r.UserID, &r.WeekStart, &r.WeekEnd,
		&r.TotalPosts, &r.TotalClicks, &r.TotalViews,
		&r.BestPersona, &r.BestFormat, &r.BestHour,
		&r.TopPostID, &recommendations, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.Recommendations = parseRecommendations(recommendations)
	return &r, nil
}

// GetCurrentWeekStats returns real-time stats for the current week
func GetCurrentWeekStats(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*CurrentWeekStats, error) {
	stats := &CurrentWeekStats{}

	// Get start of current week (Monday)
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -(weekday - 1))
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, time.UTC)

	err := pool.QueryRow(ctx, `
		SELECT 
			COALESCE(COUNT(*), 0),
			COALESCE(SUM(al.click_count), 0)
		FROM posts p
		JOIN threads_accounts ta ON p.account_id = ta.id
		LEFT JOIN affiliate_links al ON p.link_id = al.id
		WHERE ta.user_id = $1 
		  AND p.status = 'published' 
		  AND p.published_at >= $2`, userID, weekStart,
	).Scan(&stats.PostsPublished, &stats.TotalClicks)
	if err != nil {
		return nil, err
	}

	if stats.PostsPublished > 0 {
		stats.AvgClicksPerPost = float64(stats.TotalClicks) / float64(stats.PostsPublished)
	}

	return stats, nil
}

// GetPersonaPerformance returns persona performance data for a user
func GetPersonaPerformance(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]PersonaPerformance, error) {
	rows, err := pool.Query(ctx, `
		SELECT persona, weight, total_posts, total_clicks
		FROM persona_weights
		WHERE user_id = $1
		ORDER BY weight DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PersonaPerformance
	for rows.Next() {
		var p PersonaPerformance
		if err := rows.Scan(&p.Persona, &p.Weight, &p.Posts, &p.Clicks); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, nil
}

// GetFormatPerformance returns format performance data for a user
func GetFormatPerformance(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]FormatPerformance, error) {
	rows, err := pool.Query(ctx, `
		SELECT format, weight, total_posts, total_clicks
		FROM format_weights
		WHERE user_id = $1
		ORDER BY weight DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []FormatPerformance
	for rows.Next() {
		var f FormatPerformance
		if err := rows.Scan(&f.Format, &f.Weight, &f.Posts, &f.Clicks); err != nil {
			return nil, err
		}
		results = append(results, f)
	}
	return results, nil
}

// GetBestPostingHours returns posting hour performance data for a user
func GetBestPostingHours(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]HourPerformance, error) {
	rows, err := pool.Query(ctx, `
		SELECT hour_wib, weight, total_clicks
		FROM time_weights
		WHERE user_id = $1
		ORDER BY weight DESC
		LIMIT 10`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []HourPerformance
	for rows.Next() {
		var h HourPerformance
		if err := rows.Scan(&h.Hour, &h.Weight, &h.Clicks); err != nil {
			return nil, err
		}
		results = append(results, h)
	}
	return results, nil
}

// parseRecommendations parses a JSONB byte slice into a string slice
func parseRecommendations(data []byte) []string {
	if len(data) == 0 {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal(data, &result); err != nil {
		return []string{}
	}
	if result == nil {
		return []string{}
	}
	return result
}
