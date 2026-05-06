package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/ai"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/config"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/database"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/handler"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/middleware"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/queue"
)

func main() {
	// Load config
	cfg := config.Load()

	// Connect to database
	pool, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Run migrations
	if err := database.RunMigrations(pool, "migrations"); err != nil {
		log.Printf("Warning: migration error: %v", err)
	}

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Setup Fiber
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "threads-affiliate-api"})
	})

	// Handlers
	authHandler := handler.NewAuthHandler(pool, cfg)
	linkHandler := handler.NewLinkHandler(pool, rdb)

	aiClient := ai.NewClient(cfg.AI.APIURL)
	queueClient := queue.NewClient(fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port))
	accountHandler := handler.NewAccountHandler(pool)
	postHandler := handler.NewPostHandler(pool, rdb, aiClient, queueClient)

	// Public routes
	api := app.Group("/api")
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// Protected routes
	protected := api.Group("", middleware.AuthRequired(cfg.JWT.Secret))
	protected.Post("/links", linkHandler.AddLink)
	protected.Post("/links/bulk", linkHandler.BulkAddLinks)
	protected.Get("/links", linkHandler.ListLinks)
	protected.Delete("/links/:id", linkHandler.DeleteLink)

	protected.Post("/accounts", accountHandler.CreateAccount)
	protected.Get("/accounts", accountHandler.ListAccounts)
	protected.Put("/accounts/:id", accountHandler.UpdateAccount)
	protected.Delete("/accounts/:id", accountHandler.DeleteAccount)

	protected.Get("/posts", postHandler.ListPosts)
	protected.Post("/posts/generate", postHandler.GenerateContent)
	protected.Post("/posts/:id/approve", postHandler.ApprovePost)
	protected.Post("/posts/:id/publish", postHandler.PublishNow)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nShutting down...")
		_ = app.Shutdown()
	}()

	// Start server
	addr := fmt.Sprintf(":%s", cfg.App.Port)
	fmt.Printf("🚀 Threads Affiliate API starting on %s (env: %s)\n", addr, cfg.App.Env)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{"error": err.Error()})
}
