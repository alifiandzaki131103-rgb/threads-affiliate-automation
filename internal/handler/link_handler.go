package handler

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/queue"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/shortener"
)

type LinkHandler struct {
	pool        *pgxpool.Pool
	rdb         *redis.Client
	aiAPIURL    string
	queueClient *queue.Client
}

func NewLinkHandler(pool *pgxpool.Pool, rdb *redis.Client, aiAPIURL string, queueClient ...*queue.Client) *LinkHandler {
	h := &LinkHandler{pool: pool, rdb: rdb, aiAPIURL: aiAPIURL}
	if len(queueClient) > 0 {
		h.queueClient = queueClient[0]
	}
	return h
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

	// Auto-generate content if user has auto_mode enabled
	if h.queueClient != nil {
		go h.autoGenerateForLink(context.Background(), userID, link.ID, productName, price, category, platform, slug)
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

func (h *LinkHandler) CSVUpload(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Parse multipart form file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to open file"})
	}
	defer file.Close()

	// Read CSV
	reader := csv.NewReader(file)

	// Read header row
	header, err := reader.Read()
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "failed to read CSV header"})
	}

	// Map header columns to indices
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.TrimSpace(strings.ToLower(col))] = i
	}

	if _, ok := colIndex["url"]; !ok {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "CSV must have a 'url' column"})
	}

	type csvResult struct {
		ID          string  `json:"id"`
		ProductName string  `json:"product_name"`
		OriginalURL string  `json:"original_url"`
		ShortSlug   string  `json:"short_slug"`
		Platform    string  `json:"platform"`
		Price       float64 `json:"price,omitempty"`
		Category    string  `json:"category,omitempty"`
	}

	var (
		total   int
		success int
		failed  int
		links   []csvResult
	)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			failed++
			total++
			continue
		}

		total++
		if total > 100 {
			break
		}

		// Extract URL
		url := strings.TrimSpace(record[colIndex["url"]])
		if url == "" {
			failed++
			continue
		}

		// Extract optional fields
		productName := ""
		if idx, ok := colIndex["product_name"]; ok && idx < len(record) {
			productName = strings.TrimSpace(record[idx])
		}

		category := ""
		if idx, ok := colIndex["category"]; ok && idx < len(record) {
			category = strings.TrimSpace(record[idx])
		}

		var price float64
		if idx, ok := colIndex["price"]; ok && idx < len(record) {
			if p, err := strconv.ParseFloat(strings.TrimSpace(record[idx]), 64); err == nil {
				price = p
			}
		}

		platform := detectPlatform(url)

		// Use provided name or default
		if productName == "" {
			productName = "Unknown Product"
		}

		// Create product
		product := &model.Product{
			UserID:   userID,
			Name:     productName,
			Price:    price,
			Category: category,
			Platform: platform,
			Status:   "active",
		}

		if err := repository.CreateProduct(c.Context(), h.pool, product); err != nil {
			failed++
			continue
		}

		// Generate short slug
		slug := shortener.GenerateSlug(6)

		// Create affiliate link
		link := &model.AffiliateLink{
			ProductID:   product.ID,
			OriginalURL: url,
			ShortSlug:   slug,
			Platform:    platform,
			Status:      "active",
			ClickCount:  0,
		}

		if err := repository.CreateLink(c.Context(), h.pool, link); err != nil {
			failed++
			continue
		}

		// Register in Redis for fast redirect
		_ = shortener.RegisterLink(c.Context(), h.rdb, slug, url, link.ID.String())

		success++
		links = append(links, csvResult{
			ID:          link.ID.String(),
			ProductName: productName,
			OriginalURL: url,
			ShortSlug:   slug,
			Platform:    platform,
			Price:       price,
			Category:    category,
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"total":   total,
		"success": success,
		"failed":  failed,
		"links":   links,
	})
}

// autoGenerateForLink checks if the user has an account with auto_mode enabled,
// and if so, enqueues a content generation task for the new link.
func (h *LinkHandler) autoGenerateForLink(ctx context.Context, userID, linkID uuid.UUID, productName string, price float64, category, platform, slug string) {
	// Find user's first account with auto_mode enabled
	var accountID uuid.UUID
	err := h.pool.QueryRow(ctx, `
		SELECT id FROM threads_accounts
		WHERE user_id = $1 AND auto_mode = true AND status = 'active'
		LIMIT 1`, userID).Scan(&accountID)
	if err != nil {
		// No auto_mode account found, skip silently
		return
	}

	shortURL := "https://affiliate.billingku.online/s/" + slug

	payload := &queue.GenerateContentPayload{
		LinkID:      linkID,
		ProductName: productName,
		Price:       price,
		Category:    category,
		Platform:    platform,
		ShortURL:    shortURL,
		UserID:      userID,
		AccountID:   accountID,
	}

	task, err := queue.NewGenerateContentTask(payload)
	if err != nil {
		log.Printf("[AutoGenerate] Failed to create task for link %s: %v", linkID, err)
		return
	}

	_, err = h.queueClient.Enqueue(task)
	if err != nil {
		log.Printf("[AutoGenerate] Failed to enqueue task for link %s: %v", linkID, err)
		return
	}

	log.Printf("[AutoGenerate] Content generation queued for link %s (auto_mode)", linkID)
}
