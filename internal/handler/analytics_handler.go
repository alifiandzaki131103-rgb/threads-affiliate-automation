package handler

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
)

type AnalyticsHandler struct {
	pool *pgxpool.Pool
}

func NewAnalyticsHandler(pool *pgxpool.Pool) *AnalyticsHandler {
	return &AnalyticsHandler{pool: pool}
}

func (h *AnalyticsHandler) GetClickAnalytics(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Get period from query (default: 7d)
	period := c.Query("period", "7d")
	days := 7
	switch period {
	case "1d":
		days = 1
	case "7d":
		days = 7
	case "30d":
		days = 30
	}

	since := time.Now().AddDate(0, 0, -days)

	stats, err := repository.GetClickAnalytics(c.Context(), h.pool, userID, since)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch analytics"})
	}

	return c.JSON(stats)
}
