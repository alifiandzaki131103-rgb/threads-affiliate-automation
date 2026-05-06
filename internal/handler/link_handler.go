package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/shortener"
)

type LinkHandler struct {
	pool     *pgxpool.Pool
	rdb      *redis.Client
	aiAPIURL string
}

func NewLinkHandler(pool *pgxpool.Pool, rdb *redis.Client, aiAPIURL string) *LinkHandler {
	return &LinkHandler{pool: pool, rdb: rdb, aiAPIURL: aiAPIURL}
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

// resolveProductURL calls AI service to extract product info from URL
func (h *LinkHandler) resolveProductURL(url string) (name string, price float64, category string) {
	name = "Unknown Product"
	price = 0
	category = ""

	if h.aiAPIURL == "" {
		return
	}

	type resolveReq struct {
		URL string `json:"url"`
	}
	type resolveResp struct {
		ProductName string  `json:"product_name"`
		Price       float64 `json:"price"`
		Category    string  `json:"category"`
		Resolved    bool    `json:"resolved"`
	}

	body, _ := json.Marshal(resolveReq{URL: url})
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(h.aiAPIURL+"/resolve-url", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[ResolveURL] Failed to call AI service: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	var result resolveResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	if result.Resolved && result.ProductName != "" {
		name = result.ProductName
		price = result.Price
		category = result.Category
		log.Printf("[ResolveURL] Detected: %s (%.0f) [%s]", name, price, category)
	}

	return
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

	// Try auto-detect product info if not provided
	productName := "Unknown Product"
	var price float64
	category := ""

	if req.ProductName != nil && *req.ProductName != "" {
		productName = *req.ProductName
	}
	if req.Price != nil {
		price = *req.Price
	}
	if req.Category != nil {
		category = *req.Category
	}

	// Auto-resolve if product name not provided
	if productName == "Unknown Product" {
		resolvedName, resolvedPrice, resolvedCategory := h.resolveProductURL(req.URL)
		productName = resolvedName
		if price == 0 {
			price = resolvedPrice
		}
		if category == "" {
			category = resolvedCategory
		}
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
		"product_name": productName,
		"price":        price,
		"category":     category,
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

		// Auto-resolve product info
		productName, price, category := h.resolveProductURL(url)

		status := "active"
		if productName == "Unknown Product" {
			status = "needs_info"
		}

		product := &model.Product{
			UserID:   userID,
			Name:     productName,
			Price:    price,
			Category: category,
			Platform: platform,
			Status:   status,
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
			"product_name": productName,
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

	if err := repository.DeleteLinkByUser(c.Context(), h.pool, linkID, userID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("failed to delete link: %v", err)})
	}

	return c.SendStatus(http.StatusNoContent)
}
