package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
)

// ReportHandler handles weekly report and insights endpoints
type ReportHandler struct {
	pool *pgxpool.Pool
}

// NewReportHandler creates a new ReportHandler
func NewReportHandler(pool *pgxpool.Pool) *ReportHandler {
	return &ReportHandler{pool: pool}
}

// ListReports returns paginated weekly reports for the authenticated user
// GET /api/reports?page=1&limit=10
func (h *ReportHandler) ListReports(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	offset := (page - 1) * limit

	reports, total, err := repository.GetWeeklyReports(c.Context(), h.pool, userID, limit, offset)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch reports"})
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return c.JSON(fiber.Map{
		"reports": reports,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetLatestReport returns the most recent weekly report
// GET /api/reports/latest
func (h *ReportHandler) GetLatestReport(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	report, err := repository.GetLatestWeeklyReport(c.Context(), h.pool, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "no reports found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch report"})
	}

	return c.JSON(report)
}

// GetReportByID returns a specific weekly report by ID
// GET /api/reports/:id
func (h *ReportHandler) GetReportByID(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	reportID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid report ID"})
	}

	report, err := repository.GetWeeklyReportByID(c.Context(), h.pool, userID, reportID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "report not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch report"})
	}

	return c.JSON(report)
}

// GetInsights returns real-time insights for the current week
// GET /api/insights
func (h *ReportHandler) GetInsights(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	ctx := c.Context()

	// Get current week stats
	currentWeek, err := repository.GetCurrentWeekStats(ctx, h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch current week stats"})
	}

	// Get persona performance
	personaPerf, err := repository.GetPersonaPerformance(ctx, h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch persona performance"})
	}
	if personaPerf == nil {
		personaPerf = []repository.PersonaPerformance{}
	}

	// Get format performance
	formatPerf, err := repository.GetFormatPerformance(ctx, h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch format performance"})
	}
	if formatPerf == nil {
		formatPerf = []repository.FormatPerformance{}
	}

	// Get best posting hours
	bestHours, err := repository.GetBestPostingHours(ctx, h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch posting hours"})
	}
	if bestHours == nil {
		bestHours = []repository.HourPerformance{}
	}

	// Generate recommendations
	recommendations := generateRecommendations(personaPerf, formatPerf, bestHours, currentWeek)

	return c.JSON(fiber.Map{
		"current_week":       currentWeek,
		"persona_performance": personaPerf,
		"format_performance":  formatPerf,
		"best_posting_hours":  bestHours,
		"recommendations":     recommendations,
	})
}

// generateRecommendations creates actionable recommendations based on performance data
func generateRecommendations(
	personas []repository.PersonaPerformance,
	formats []repository.FormatPerformance,
	hours []repository.HourPerformance,
	weekStats *repository.CurrentWeekStats,
) []string {
	var recs []string

	// Recommend best persona
	if len(personas) > 0 {
		best := personas[0]
		if best.Weight > 1.0 && best.Posts > 0 {
			recs = append(recs, fmt.Sprintf("Use %s persona more - it has %.1fx higher engagement", best.Persona, best.Weight))
		}
	}

	// Recommend best format
	if len(formats) > 0 {
		best := formats[0]
		if best.Weight > 1.0 && best.Posts > 0 {
			recs = append(recs, fmt.Sprintf("Try more %s format posts - performing %.1fx above average", best.Format, best.Weight))
		}
	}

	// Recommend best posting time
	if len(hours) > 0 {
		best := hours[0]
		if best.Weight > 1.0 {
			recs = append(recs, fmt.Sprintf("Post more around %d:00 WIB - your best performing hour", best.Hour))
		}
	}

	// Recommend posting frequency
	if weekStats.PostsPublished < 5 {
		recs = append(recs, "Increase posting frequency - aim for at least 7 posts per week")
	} else if weekStats.PostsPublished >= 14 {
		recs = append(recs, "Good posting volume! Focus on quality over quantity")
	}

	// Recommend based on engagement
	if weekStats.PostsPublished > 0 && weekStats.AvgClicksPerPost < 10 {
		recs = append(recs, "Try different CTAs and link placements to improve click-through rate")
	}

	// Avoid worst performers
	if len(personas) > 1 {
		worst := personas[len(personas)-1]
		if worst.Weight < 0.5 && worst.Posts > 2 {
			recs = append(recs, fmt.Sprintf("Consider reducing %s persona usage - underperforming", worst.Persona))
		}
	}

	if len(recs) == 0 {
		recs = append(recs, "Keep posting consistently to build up performance data")
	}

	return recs
}
