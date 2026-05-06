package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/ai"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/config"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/database"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/queue"
)

func main() {
	cfg := config.Load()

	// Connect to database
	pool, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Connect to Redis
	redisAddr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// AI client
	aiClient := ai.NewClient(cfg.AI.APIURL)

	// Queue client (for enqueuing follow-up tasks)
	queueClient := queue.NewClient(redisAddr)
	defer queueClient.Close()

	// Create task handlers
	handlers := queue.NewHandlers(pool, rdb, aiClient, cfg, queueClient)

	// Setup Asynq server (worker)
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr, Password: cfg.Redis.Password, DB: cfg.Redis.DB},
		asynq.Config{
			Concurrency: 5,
			Queues: map[string]int{
				"critical": 6, // publishing tasks
				"default":  3, // content generation
				"low":      1, // analytics, health checks
			},
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				return time.Duration(n*30) * time.Second // 30s, 60s, 90s...
			},
		},
	)

	// Register handlers
	mux := asynq.NewServeMux()
	handlers.RegisterHandlers(mux)

	// Start scheduler (periodic tasks)
	scheduler, err := queue.NewScheduler(redisAddr)
	if err != nil {
		log.Fatalf("Failed to create scheduler: %v", err)
	}

	if err := scheduler.RegisterPeriodicTasks(); err != nil {
		log.Fatalf("Failed to register periodic tasks: %v", err)
	}

	// Start scheduler in background
	go func() {
		if err := scheduler.Start(); err != nil {
			log.Printf("Scheduler error: %v", err)
		}
	}()

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nShutting down worker...")
		srv.Shutdown()
		scheduler.Shutdown()
	}()

	// Start worker
	fmt.Printf("🔄 Asynq Worker starting (concurrency: 5, redis: %s)\n", redisAddr)
	fmt.Println("   Queues: critical(6), default(3), low(1)")
	fmt.Println("   Periodic: check_replies(10min), analytics(4h), health_check(1h), weekly_learning(Mon 02:00)")

	if err := srv.Run(mux); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}
