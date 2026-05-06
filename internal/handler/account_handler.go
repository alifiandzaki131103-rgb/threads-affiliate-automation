package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
)

type AccountHandler struct {
	pool *pgxpool.Pool
}

func NewAccountHandler(pool *pgxpool.Pool) *AccountHandler {
	return &AccountHandler{pool: pool}
}

func (h *AccountHandler) CreateAccount(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req struct {
		ThreadsUserID string `json:"threads_user_id"`
		AccessToken   string `json:"access_token"`
		Persona       string `json:"persona"`
		Niche         string `json:"niche"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.ThreadsUserID == "" || req.AccessToken == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "threads_user_id and access_token are required"})
	}

	account := &model.ThreadsAccount{
		UserID:        userID,
		ThreadsUserID: req.ThreadsUserID,
		AccessToken:   req.AccessToken,
		Persona:       req.Persona,
		Niche:         req.Niche,
		Status:        "active",
	}

	if err := repository.CreateAccount(c.Context(), h.pool, account); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create account"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id":              account.ID,
		"threads_user_id": account.ThreadsUserID,
		"persona":         account.Persona,
		"niche":           account.Niche,
		"status":          account.Status,
	})
}

func (h *AccountHandler) ListAccounts(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	accounts, err := repository.GetAccountsByUserID(c.Context(), h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch accounts"})
	}

	return c.JSON(fiber.Map{"count": len(accounts), "accounts": accounts})
}

func (h *AccountHandler) UpdateAccount(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	accountID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid account id"})
	}

	var req struct {
		Persona string `json:"persona"`
		Niche   string `json:"niche"`
		Status  string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Status == "" {
		req.Status = "active"
	}

	if err := repository.UpdateAccount(c.Context(), h.pool, accountID, userID, req.Persona, req.Niche, req.Status); err != nil {
		if repository.IsNoRows(err) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "account not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update account"})
	}

	return c.JSON(fiber.Map{"message": "account updated"})
}

func (h *AccountHandler) DeleteAccount(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	accountID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid account id"})
	}

	if err := repository.DeleteAccount(c.Context(), h.pool, accountID, userID); err != nil {
		if repository.IsNoRows(err) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "account not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete account"})
	}

	return c.SendStatus(http.StatusNoContent)
}
