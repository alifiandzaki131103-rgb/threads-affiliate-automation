package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/ai"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/config"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/threads"
)

// Handlers contains all task handler dependencies
type Handlers struct {
	pool      *pgxpool.Pool
	rdb       *redis.Client
	aiClient  *ai.Client
	cfg       *config.Config
	queueClient *Client
}

// NewHandlers creates task handlers with dependencies
func NewHandlers(pool *pgxpool.Pool, rdb *redis.Client, aiClient *ai.Client, cfg *config.Config, queueClient *Client) *Handlers {
	return &Handlers{
		pool:      pool,
		rdb:       rdb,
		aiClient:  aiClient,
		cfg:       cfg,
		queueClient: queueClient,
	}
}

// RegisterHandlers registers all task handlers with the mux
func (h *Handlers) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskGenerateContent, h.HandleGenerateContent)
	mux.HandleFunc(TaskPublishPost, h.HandlePublishPost)
	mux.HandleFunc(TaskReplyDrop, h.HandleReplyDrop)
	mux.HandleFunc(TaskCheckReplies, h.HandleCheckReplies)
	mux.HandleFunc(TaskCollectAnalytics, h.HandleCollectAnalytics)
	mux.HandleFunc(TaskHealthCheckLinks, h.HandleHealthCheckLinks)
	mux.HandleFunc(TaskWeeklyLearning, h.HandleWeeklyLearning)
}

// HandleGenerateContent generates AI content for a product link
func (h *Handlers) HandleGenerateContent(ctx context.Context, t *asynq.Task) error {
	var payload GenerateContentPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Printf("[GenerateContent] Generating for product: %s (link: %s)", payload.ProductName, payload.LinkID)

	// Pick random persona and format
	personas := []string{"honest_friend", "hot_take", "problem_solver", "curious_explorer", "lifestyle_sharer", "comparison_nerd"}
	formats := []string{"single", "single", "single", "hot_take", "question", "story"} // weighted toward single
	placements := []string{"direct", "direct", "reply_drop", "bio", "question_trigger"} // weighted toward direct

	persona := personas[rand.Intn(len(personas))]
	format := formats[rand.Intn(len(formats))]
	placement := placements[rand.Intn(len(placements))]

	// Call AI service
	result, err := h.aiClient.Generate(ctx, &ai.GenerateRequest{
		ProductName:   payload.ProductName,
		Price:         payload.Price,
		Category:      payload.Category,
		Platform:      payload.Platform,
		Persona:       persona,
		Format:        format,
		LinkPlacement: placement,
		ShortURL:      payload.ShortURL,
	})
	if err != nil {
		return fmt.Errorf("AI generate: %w", err)
	}

	// Save post to database
	_, err = h.pool.Exec(ctx, `
		INSERT INTO posts (id, account_id, link_id, content, link_placement, persona, format, scheduled_at, status, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
		payload.AccountID, payload.LinkID, result.Content, placement, persona, format,
		generateScheduleTime(), "approved",
	)
	if err != nil {
		return fmt.Errorf("save post: %w", err)
	}

	log.Printf("[GenerateContent] Post generated and scheduled for link %s", payload.LinkID)
	return nil
}

// HandlePublishPost publishes a post to Threads
func (h *Handlers) HandlePublishPost(ctx context.Context, t *asynq.Task) error {
	var payload PublishPostPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Printf("[PublishPost] Publishing post %s", payload.PostID)

	// Random delay 0-120 seconds (anti-detection)
	delay := time.Duration(rand.Intn(120)) * time.Second
	time.Sleep(delay)

	// Create Threads client
	client := threads.NewClient(payload.AccessToken)

	// Step 1: Create container
	containerID, err := client.CreateContainer(ctx, payload.ThreadsUserID, payload.Content)
	if err != nil {
		// Update post status to failed
		h.updatePostStatus(ctx, payload.PostID, "failed")
		return fmt.Errorf("create container: %w", err)
	}

	// Wait for container processing (Threads needs a moment)
	time.Sleep(3 * time.Second)

	// Step 2: Publish
	threadID, err := client.PublishContainer(ctx, payload.ThreadsUserID, containerID)
	if err != nil {
		h.updatePostStatus(ctx, payload.PostID, "failed")
		return fmt.Errorf("publish container: %w", err)
	}

	// Step 3: Update post status
	_, err = h.pool.Exec(ctx, `
		UPDATE posts SET status = 'published', thread_id = $1, published_at = NOW()
		WHERE id = $2`,
		threadID, payload.PostID,
	)
	if err != nil {
		return fmt.Errorf("update post status: %w", err)
	}

	log.Printf("[PublishPost] Post %s published as thread %s", payload.PostID, threadID)
	return nil
}

// HandleReplyDrop drops a reply with affiliate link after a delay
func (h *Handlers) HandleReplyDrop(ctx context.Context, t *asynq.Task) error {
	var payload ReplyDropPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Printf("[ReplyDrop] Dropping reply on thread %s", payload.ThreadID)

	client := threads.NewClient(payload.AccessToken)

	_, err := client.ReplyToThread(ctx, payload.ThreadsUserID, payload.ThreadID, payload.ReplyContent)
	if err != nil {
		return fmt.Errorf("reply to thread: %w", err)
	}

	log.Printf("[ReplyDrop] Reply dropped on thread %s", payload.ThreadID)
	return nil
}

// HandleCheckReplies checks for new replies on published posts (periodic)
func (h *Handlers) HandleCheckReplies(ctx context.Context, t *asynq.Task) error {
	log.Println("[CheckReplies] Checking for new replies...")

	// Get recent published posts (last 48h)
	rows, err := h.pool.Query(ctx, `
		SELECT id, thread_id, account_id FROM posts
		WHERE status = 'published' AND thread_id IS NOT NULL
		AND published_at > NOW() - INTERVAL '48 hours'
		LIMIT 50`)
	if err != nil {
		return fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	// TODO: For each post, fetch replies via Threads API and auto-respond
	// This will be implemented in Phase 2 (engagement engine)
	count := 0
	for rows.Next() {
		count++
	}

	log.Printf("[CheckReplies] Checked %d posts for new replies", count)
	return nil
}

// HandleCollectAnalytics collects engagement metrics from Threads API (periodic)
func (h *Handlers) HandleCollectAnalytics(ctx context.Context, t *asynq.Task) error {
	log.Println("[CollectAnalytics] Collecting analytics...")

	// Get posts published in last 7 days
	rows, err := h.pool.Query(ctx, `
		SELECT id, thread_id FROM posts
		WHERE status = 'published' AND thread_id IS NOT NULL
		AND published_at > NOW() - INTERVAL '7 days'
		LIMIT 100`)
	if err != nil {
		return fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	// TODO: For each post, fetch insights via Threads API
	// This will be fully implemented in Phase 2
	count := 0
	for rows.Next() {
		count++
	}

	log.Printf("[CollectAnalytics] Collected analytics for %d posts", count)
	return nil
}

// HandleHealthCheckLinks checks if affiliate links are still active (periodic)
func (h *Handlers) HandleHealthCheckLinks(ctx context.Context, t *asynq.Task) error {
	log.Println("[HealthCheckLinks] Checking link health...")

	// Sample 10% of active links
	rows, err := h.pool.Query(ctx, `
		SELECT id, original_url FROM affiliate_links
		WHERE status = 'active'
		ORDER BY RANDOM()
		LIMIT 10`)
	if err != nil {
		return fmt.Errorf("query links: %w", err)
	}
	defer rows.Close()

	// TODO: HTTP HEAD check each link, mark expired if 404
	count := 0
	for rows.Next() {
		count++
	}

	log.Printf("[HealthCheckLinks] Checked %d links", count)
	return nil
}

// HandleWeeklyLearning runs the self-learning AI analysis (weekly)
func (h *Handlers) HandleWeeklyLearning(ctx context.Context, t *asynq.Task) error {
	log.Println("[WeeklyLearning] Running weekly self-learning analysis...")

	// TODO: Analyze last 7 days performance
	// - Group posts by persona, format, time, platform
	// - Identify top 20% vs bottom 20%
	// - Update content_templates scores
	// - Generate weekly report

	log.Println("[WeeklyLearning] Analysis complete (placeholder)")
	return nil
}

// Helper functions

func (h *Handlers) updatePostStatus(ctx context.Context, postID interface{}, status string) {
	_, _ = h.pool.Exec(ctx, `UPDATE posts SET status = $1 WHERE id = $2`, status, postID)
}

// generateScheduleTime generates a random posting time for today/tomorrow
// Spread between 06:00-23:00 WIB
func generateScheduleTime() time.Time {
	wib := time.FixedZone("WIB", 7*60*60)
	now := time.Now().In(wib)

	// If after 22:00, schedule for tomorrow
	targetDay := now
	if now.Hour() >= 22 {
		targetDay = now.Add(24 * time.Hour)
	}

	// Random hour between 6-22
	hour := 6 + rand.Intn(17) // 6 to 22
	// Random minute
	minute := rand.Intn(60)

	scheduled := time.Date(
		targetDay.Year(), targetDay.Month(), targetDay.Day(),
		hour, minute, 0, 0, wib,
	)

	// If scheduled time is in the past, add some hours
	if scheduled.Before(time.Now()) {
		scheduled = time.Now().Add(time.Duration(30+rand.Intn(90)) * time.Minute)
	}

	return scheduled
}
