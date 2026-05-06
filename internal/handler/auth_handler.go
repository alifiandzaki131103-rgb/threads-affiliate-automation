package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/auth"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/config"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
)

type AuthHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewAuthHandler(pool *pgxpool.Pool, cfg *config.Config) *AuthHandler {
	return &AuthHandler{pool: pool, cfg: cfg}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "email and password required"})
	}

	if len(req.Password) < 6 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "password must be at least 6 characters"})
	}

	// Check if user exists
	existing, _ := repository.GetUserByEmail(c.Context(), h.pool, req.Email)
	if existing != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "email already registered"})
	}

	// Hash password
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}

	// Create user
	user := &model.User{
		Email:        req.Email,
		PasswordHash: hash,
		Plan:         "trial",
	}

	if err := repository.CreateUser(c.Context(), h.pool, user); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create user"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id":    user.ID,
		"email": user.Email,
		"plan":  user.Plan,
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "email and password required"})
	}

	// Get user
	user, err := repository.GetUserByEmail(c.Context(), h.pool, req.Email)
	if err != nil || user == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	// Check password
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	// Generate tokens
	accessToken, refreshToken, err := auth.GenerateTokens(user.ID, user.Email, h.cfg)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate tokens"})
	}

	return c.JSON(model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: model.UserResponse{
			ID:    user.ID,
			Email: user.Email,
			Plan:  user.Plan,
		},
	})
}

// Helper to get userID from context (set by auth middleware)
func GetUserID(c *fiber.Ctx) uuid.UUID {
	id, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}
