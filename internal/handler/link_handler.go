package handler

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/shortener"
)

type LinkHandler struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
}

func NewLinkHandler(pool *pgxpool.Pool, rdb *redis.Client) *LinkHandler {
	return &LinkHandler{pool: pool, rdb: rdb}
}

func detectPlatform(url string) string {
	lower := strings.ToLower(url)
	if strings.Contains(lower, "shopee") {
		return "shopee"
	}
	if strings.Contains(lower, "tiktok") {
		return "tiktok"
	}
	return "unknown"
}

func (h *LinkHandler) AddLink(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req model.AddLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.URL == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "url is required"})
	}

	platform := detectPlatform(req.URL)

	// Create product
	productName := "Unknown Product"
	if req.ProductName != nil && *req.ProductName != "" {
		productName = *req.ProductName
	}

	var price float64
	if req.Price != nil {
		price = *req.Price
	}

	category := ""
	if req.Category != nil {
		category = *req.Category
	}

	product := &model.Product{
		UserID:   userID,
		Name:     productName,
		Price:    price,
		Category: category,
		Platform: platform,
		Status:   "active",
	}

	if err := repository.CreateProduct(c.Context(), h.pool, product); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create product"})
	}

	// Generate short slug
	slug := shortener.GenerateSlug(6)

	// Create affiliate link
	link := &model.AffiliateLink{
		ProductID:   product.ID,
		OriginalURL: req.URL,
		ShortSlug:   slug,
		Platform:    platform,
		Status:      "active",
		ClickCount:  0,
	}

	if err := repository.CreateLink(c.Context(), h.pool, link); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create link"})
	}

	// Register in Redis for fast redirect
	if err := shortener.RegisterLink(c.Context(), h.rdb, slug, req.URL, link.ID.String()); err != nil {
		// Non-fatal: link still works via DB lookup
		_ = err
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id":           link.ID,
		"product_id":   product.ID,
		"original_url": link.OriginalURL,
		"short_slug":   link.ShortSlug,
		"platform":     link.Platform,
		"status":       link.Status,
	})
}

func (h *LinkHandler) BulkAddLinks(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req model.BulkAddLinksRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if len(req.URLs) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "urls array is required"})
	}

	results := make([]fiber.Map, 0, len(req.URLs))

	for _, url := range req.URLs {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		platform := detectPlatform(url)
		slug := shortener.GenerateSlug(6)

		product := &model.Product{
			UserID:   userID,
			Name:     "Unknown Product",
			Platform: platform,
			Status:   "needs_info",
		}
		_ = repository.CreateProduct(c.Context(), h.pool, product)

		link := &model.AffiliateLink{
			ProductID:   product.ID,
			OriginalURL: url,
			ShortSlug:   slug,
			Platform:    platform,
			Status:      "active",
		}
		_ = repository.CreateLink(c.Context(), h.pool, link)
		_ = shortener.RegisterLink(c.Context(), h.rdb, slug, url, link.ID.String())

		results = append(results, fiber.Map{
			"id":           link.ID,
			"original_url": url,
			"short_slug":   slug,
			"platform":     platform,
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"count":   len(results),
		"results": results,
	})
}

func (h *LinkHandler) ListLinks(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	links, err := repository.GetLinksByUserID(c.Context(), h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch links"})
	}

	return c.JSON(fiber.Map{
		"count": len(links),
		"links": links,
	})
}

func (h *LinkHandler) DeleteLink(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	linkID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid link id"})
	}

	// TODO: verify ownership before delete
	_ = linkID
	_ = userID

	return c.SendStatus(http.StatusNoContent)
}
