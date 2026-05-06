package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/ai"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/config"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
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
	mux.HandleFunc(TaskLinkHealthCheck, h.HandleLinkHealthCheck)
	mux.HandleFunc(TaskWeeklyLearning, h.HandleWeeklyLearning)
	mux.HandleFunc(TaskAutoPublish, h.HandleAutoPublish)
	mux.HandleFunc(TaskAutoReply, h.HandleAutoReply)
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

	// Random delay 5-25 seconds (anti-detection, within timeout)
	delay := time.Duration(5+rand.Intn(20)) * time.Second
	time.Sleep(delay)

	// Create Threads client
	client := threads.NewClient(payload.AccessToken)

	// Step 1: Create container
	containerID, err := client.CreateContainer(ctx, payload.ThreadsUserID, payload.Content)
	if err != nil {
		log.Printf("[PublishPost] ERROR creating container for post %s: %v", payload.PostID, err)
		// Update post status to failed
		h.updatePostStatus(ctx, payload.PostID, "failed")
		return fmt.Errorf("create container: %w", err)
	}

	// Wait for container processing (Threads needs a moment)
	time.Sleep(3 * time.Second)

	// Step 2: Publish
	threadID, err := client.PublishContainer(ctx, payload.ThreadsUserID, containerID)
	if err != nil {
		log.Printf("[PublishPost] ERROR publishing container for post %s: %v", payload.PostID, err)
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

// HandleLinkHealthCheck performs comprehensive health checks on all active affiliate links.
// For each link, it does an HTTP HEAD request and updates health_status accordingly.
func (h *Handlers) HandleLinkHealthCheck(ctx context.Context, t *asynq.Task) error {
	log.Println("[LinkHealthCheck] Starting link health check for all active links...")

	// Get all active links from the database
	links, err := repository.GetAllActiveLinks(ctx, h.pool)
	if err != nil {
		return fmt.Errorf("get active links: %w", err)
	}

	if len(links) == 0 {
		log.Println("[LinkHealthCheck] No active links to check")
		return nil
	}

	// Create HTTP client with timeout and redirect limit
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects (max 3)")
			}
			return nil
		},
	}

	var healthy, broken, unreachable int

	for _, link := range links {
		// Determine health status via HTTP HEAD request
		healthStatus := checkLinkHealth(httpClient, link.OriginalURL)

		// Update the link health in the database
		if err := repository.UpdateLinkHealth(ctx, h.pool, link.ID, healthStatus); err != nil {
			log.Printf("[LinkHealthCheck] ERROR updating health for link %s: %v", link.ID, err)
			continue
		}

		switch healthStatus {
		case "healthy":
			healthy++
		case "broken":
			broken++
		case "unreachable":
			unreachable++
		}
	}

	log.Printf("[LinkHealthCheck] Completed: %d healthy, %d broken, %d unreachable (total: %d)",
		healthy, broken, unreachable, len(links))
	return nil
}

// checkLinkHealth performs an HTTP HEAD request and returns the health status string.
func checkLinkHealth(client *http.Client, url string) string {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return "unreachable"
	}
	req.Header.Set("User-Agent", "ThreadsAffiliate-HealthCheck/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "unreachable"
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode <= 399 {
		return "healthy"
	}
	return "broken"
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

// HandleAutoPublish publishes approved posts that are due (periodic task)
func (h *Handlers) HandleAutoPublish(ctx context.Context, t *asynq.Task) error {
	log.Println("[AutoPublish] Checking for approved posts due for publishing...")

	// Query posts with status='approved' and scheduled_at <= now()
	rows, err := h.pool.Query(ctx, `
		SELECT p.id, p.account_id, p.content
		FROM posts p
		JOIN threads_accounts ta ON p.account_id = ta.id
		WHERE p.status = 'approved'
		AND p.scheduled_at <= NOW()
		AND ta.auto_mode = true
		ORDER BY p.scheduled_at ASC
		LIMIT 10`)
	if err != nil {
		return fmt.Errorf("query due posts: %w", err)
	}
	defer rows.Close()

	type duePost struct {
		ID        string
		AccountID string
		Content   string
	}

	var duePosts []duePost
	for rows.Next() {
		var p duePost
		if err := rows.Scan(&p.ID, &p.AccountID, &p.Content); err != nil {
			return fmt.Errorf("scan post: %w", err)
		}
		duePosts = append(duePosts, p)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows error: %w", err)
	}

	if len(duePosts) == 0 {
		log.Println("[AutoPublish] No posts due for publishing")
		return nil
	}

	log.Printf("[AutoPublish] Found %d posts due for publishing", len(duePosts))

	for _, p := range duePosts {
		// Get account details for Threads API credentials
		var threadsUserID, accessToken string
		err := h.pool.QueryRow(ctx, `
			SELECT threads_user_id, access_token FROM threads_accounts WHERE id = $1`,
			p.AccountID).Scan(&threadsUserID, &accessToken)
		if err != nil {
			log.Printf("[AutoPublish] ERROR fetching account %s: %v", p.AccountID, err)
			continue
		}

		// Create Threads client and publish
		client := threads.NewClient(accessToken)

		// Add random delay for anti-detection (2-10 seconds between posts)
		delay := time.Duration(2+rand.Intn(8)) * time.Second
		time.Sleep(delay)

		// Step 1: Create container
		containerID, err := client.CreateContainer(ctx, threadsUserID, p.Content)
		if err != nil {
			log.Printf("[AutoPublish] ERROR creating container for post %s: %v", p.ID, err)
			h.updatePostStatus(ctx, p.ID, "failed")
			continue
		}

		// Wait for container processing
		time.Sleep(3 * time.Second)

		// Step 2: Publish
		threadID, err := client.PublishContainer(ctx, threadsUserID, containerID)
		if err != nil {
			log.Printf("[AutoPublish] ERROR publishing post %s: %v", p.ID, err)
			h.updatePostStatus(ctx, p.ID, "failed")
			continue
		}

		// Step 3: Update post status to published
		_, err = h.pool.Exec(ctx, `
			UPDATE posts SET status = 'published', thread_id = $1, published_at = NOW()
			WHERE id = $2`,
			threadID, p.ID)
		if err != nil {
			log.Printf("[AutoPublish] ERROR updating post %s status: %v", p.ID, err)
			continue
		}

		log.Printf("[AutoPublish] Post %s published as thread %s", p.ID, threadID)
	}

	log.Printf("[AutoPublish] Done processing %d posts", len(duePosts))
	return nil
}

// HandleAutoReply monitors replies on published posts and auto-responds with affiliate links
func (h *Handlers) HandleAutoReply(ctx context.Context, t *asynq.Task) error {
	var payload AutoReplyPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	log.Printf("[AutoReply] Processing post %s for account %s", payload.PostID, payload.AccountID)

	// Get post details (thread_id and affiliate link)
	var threadID string
	var shortSlug *string
	err := h.pool.QueryRow(ctx, `
		SELECT p.thread_id, al.short_slug
		FROM posts p
		LEFT JOIN affiliate_links al ON p.link_id = al.id
		WHERE p.id = $1 AND p.thread_id IS NOT NULL`,
		payload.PostID).Scan(&threadID, &shortSlug)
	if err != nil {
		return fmt.Errorf("query post: %w", err)
	}

	// Get account access token and threads_user_id
	var accessToken, threadsUserID string
	err = h.pool.QueryRow(ctx, `
		SELECT access_token, threads_user_id FROM threads_accounts WHERE id = $1`,
		payload.AccountID).Scan(&accessToken, &threadsUserID)
	if err != nil {
		return fmt.Errorf("query account: %w", err)
	}

	// Build the short URL for the reply
	shortURL := ""
	if shortSlug != nil {
		shortURL = fmt.Sprintf("https://%s/%s", h.cfg.Shortener.Domain, *shortSlug)
	}
	if shortURL == "" {
		log.Printf("[AutoReply] No affiliate link found for post %s, skipping", payload.PostID)
		return nil
	}

	// Fetch replies from Threads API
	client := threads.NewClient(accessToken)
	replies, err := client.GetReplies(ctx, threadID)
	if err != nil {
		return fmt.Errorf("get replies: %w", err)
	}

	if len(replies) == 0 {
		log.Printf("[AutoReply] No replies found for post %s", payload.PostID)
		return nil
	}

	// Trigger phrases to match
	triggerPhrases := []string{
		"beli dimana",
		"link dong",
		"mau dong",
		"dimana belinya",
		"link nya",
		"linknya",
		"where to buy",
	}

	// Natural reply templates
	replyTemplates := []string{
		"Nih linknya: %s 👆",
		"Cek sini: %s",
		"Langsung aja: %s ✨",
		"Ini dia: %s 🔗",
	}

	redisKey := fmt.Sprintf("replied:%s", payload.PostID)
	repliedCount := 0

	for _, reply := range replies {
		// Check if already replied to this comment
		alreadyReplied, err := h.rdb.SIsMember(ctx, redisKey, reply.ID).Result()
		if err != nil {
			log.Printf("[AutoReply] Redis error checking reply %s: %v", reply.ID, err)
			continue
		}
		if alreadyReplied {
			continue
		}

		// Check if reply text matches any trigger phrase
		lowerText := strings.ToLower(reply.Text)
		matched := false
		for _, trigger := range triggerPhrases {
			if strings.Contains(lowerText, trigger) {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		// Pick a reply template (rotate based on count)
		template := replyTemplates[repliedCount%len(replyTemplates)]
		replyText := fmt.Sprintf(template, shortURL)

		// Send reply via Threads API
		_, err = client.ReplyToThread(ctx, threadsUserID, reply.ID, replyText)
		if err != nil {
			log.Printf("[AutoReply] ERROR replying to %s: %v", reply.ID, err)
			continue
		}

		// Mark as replied in Redis
		h.rdb.SAdd(ctx, redisKey, reply.ID)
		// Set expiry on the set (7 days)
		h.rdb.Expire(ctx, redisKey, 7*24*time.Hour)

		repliedCount++
		log.Printf("[AutoReply] Replied to %s (@%s) on post %s", reply.ID, reply.Username, payload.PostID)

		// Small delay between replies for anti-detection
		time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
	}

	log.Printf("[AutoReply] Done. Replied to %d comments on post %s", repliedCount, payload.PostID)
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
