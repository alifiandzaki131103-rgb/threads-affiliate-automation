package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
)

type DashboardHandler struct {
	pool *pgxpool.Pool
}

func NewDashboardHandler(pool *pgxpool.Pool) *DashboardHandler {
	return &DashboardHandler{pool: pool}
}

func (h *DashboardHandler) GetStats(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	stats, err := repository.GetDashboardStats(c.Context(), h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch dashboard stats"})
	}

	return c.JSON(stats)
}
