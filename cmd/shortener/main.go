package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/redis/go-redis/v9"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/config"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/shortener"
)

func main() {
	cfg := config.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	app := fiber.New(fiber.Config{
		AppName: "threads-affiliate-shortener",
	})

	app.Use(logger.New())

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "shortener"})
	})

	// Redirect endpoint
	app.Get("/go/:slug", func(c *fiber.Ctx) error {
		slug := c.Params("slug")

		originalURL, linkID, err := shortener.Resolve(c.Context(), rdb, slug)
		if err != nil {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "link not found"})
		}

		// Track click asynchronously
		ip := c.IP()
		hashedIP := hashIP(ip)
		userAgent := c.Get("User-Agent")
		referrer := c.Get("Referer")

		go func() {
			_ = shortener.TrackClick(c.Context(), rdb, linkID, hashedIP, userAgent, referrer)
		}()

		// 301 redirect
		return c.Redirect(originalURL, http.StatusMovedPermanently)
	})

	addr := fmt.Sprintf(":%s", cfg.Shortener.Port)
	fmt.Printf("🔗 URL Shortener starting on %s\n", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("Failed to start shortener: %v", err)
	}
}

func hashIP(ip string) string {
	h := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(h[:])
}
