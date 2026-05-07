package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// Task types
const (
	TaskGenerateContent     = "content:generate"
	TaskPublishPost         = "post:publish"
	TaskReplyDrop           = "post:reply_drop"
	TaskCheckReplies        = "engagement:check_replies"
	TaskCollectAnalytics    = "analytics:collect"
	TaskHealthCheckLinks    = "links:health_check"
	TaskLinkHealthCheck     = "link:health_check"
	TaskWeeklyLearning      = "ai:weekly_learning"
	TaskAutoPublish         = "auto:publish"
	TaskAutoReply           = "auto:reply"
	TaskCircuitBreakerCheck = "circuit:check"
)

// GenerateContentPayload is the payload for content generation tasks
type GenerateContentPayload struct {
	LinkID      uuid.UUID `json:"link_id"`
	ProductName string    `json:"product_name"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	Platform    string    `json:"platform"`
	ShortURL    string    `json:"short_url"`
	UserID      uuid.UUID `json:"user_id"`
	AccountID   uuid.UUID `json:"account_id"`
}

// PublishPostPayload is the payload for publishing tasks
type PublishPostPayload struct {
	PostID    uuid.UUID `json:"post_id"`
	AccountID uuid.UUID `json:"account_id"`
	Content   string    `json:"content"`
	ThreadsUserID string `json:"threads_user_id"`
	AccessToken   string `json:"access_token"`
}

// ReplyDropPayload is the payload for reply drop tasks
type ReplyDropPayload struct {
	PostID        uuid.UUID `json:"post_id"`
	ThreadID      string    `json:"thread_id"`
	ReplyContent  string    `json:"reply_content"`
	ThreadsUserID string    `json:"threads_user_id"`
	AccessToken   string    `json:"access_token"`
}

// NewGenerateContentTask creates a new content generation task
func NewGenerateContentTask(payload *GenerateContentPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return asynq.NewTask(TaskGenerateContent, data, asynq.MaxRetry(3), asynq.Timeout(60*time.Second)), nil
}

// NewPublishPostTask creates a new post publishing task scheduled at a specific time
func NewPublishPostTask(payload *PublishPostPayload, publishAt time.Time) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return asynq.NewTask(TaskPublishPost, data, asynq.MaxRetry(3), asynq.Timeout(60*time.Second), asynq.ProcessAt(publishAt)), nil
}

// NewReplyDropTask creates a delayed reply task (5-15 min after publish)
func NewReplyDropTask(payload *ReplyDropPayload, delay time.Duration) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return asynq.NewTask(TaskReplyDrop, data, asynq.MaxRetry(2), asynq.Timeout(30*time.Second), asynq.ProcessIn(delay)), nil
}

// AutoReplyPayload is the payload for auto-reply tasks
type AutoReplyPayload struct {
	AccountID uuid.UUID `json:"account_id"`
	PostID    uuid.UUID `json:"post_id"`
}

// NewAutoReplyTask creates a new auto-reply task
func NewAutoReplyTask(payload *AutoReplyPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return asynq.NewTask(TaskAutoReply, data, asynq.MaxRetry(2), asynq.Timeout(60*time.Second)), nil
}

// AutoPublishPayload is the payload for auto-publish tasks (can be empty for periodic)
type AutoPublishPayload struct{}

// LinkHealthCheckPayload is the payload for link health check tasks (empty, runs for all links)
type LinkHealthCheckPayload struct{}

// NewLinkHealthCheckTask creates a periodic task to check affiliate link health
func NewLinkHealthCheckTask() *asynq.Task {
	return asynq.NewTask(TaskLinkHealthCheck, nil, asynq.MaxRetry(1), asynq.Timeout(120*time.Second))
}

// NewAutoPublishTask creates a periodic task to auto-publish approved posts on schedule
func NewAutoPublishTask() *asynq.Task {
	return asynq.NewTask(TaskAutoPublish, nil, asynq.MaxRetry(2), asynq.Timeout(120*time.Second))
}

// NewCheckRepliesTask creates a periodic task to check for new replies
func NewCheckRepliesTask() *asynq.Task {
	return asynq.NewTask(TaskCheckReplies, nil, asynq.MaxRetry(1), asynq.Timeout(30*time.Second))
}

// NewCircuitBreakerCheckTask creates a periodic task to check circuit breaker conditions
func NewCircuitBreakerCheckTask() *asynq.Task {
	return asynq.NewTask(TaskCircuitBreakerCheck, nil, asynq.MaxRetry(1), asynq.Timeout(60*time.Second))
}

// NewCollectAnalyticsTask creates a periodic analytics collection task
func NewCollectAnalyticsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectAnalytics, nil, asynq.MaxRetry(2), asynq.Timeout(60*time.Second))
}

// Scheduler sets up periodic tasks
type Scheduler struct {
	scheduler *asynq.Scheduler
}

// NewScheduler creates a new task scheduler
func NewScheduler(redisAddr string) (*Scheduler, error) {
	scheduler := asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: redisAddr},
		&asynq.SchedulerOpts{
			Location: time.FixedZone("WIB", 7*60*60), // UTC+7
		},
	)
	return &Scheduler{scheduler: scheduler}, nil
}

// RegisterPeriodicTasks registers all cron-based periodic tasks
func (s *Scheduler) RegisterPeriodicTasks() error {
	// Check replies every 10 minutes
	_, err := s.scheduler.Register("*/10 * * * *", NewCheckRepliesTask())
	if err != nil {
		return fmt.Errorf("register check_replies: %w", err)
	}

	// Auto-publish approved posts every 5 minutes
	_, err = s.scheduler.Register("*/5 * * * *", NewAutoPublishTask())
	if err != nil {
		return fmt.Errorf("register auto_publish: %w", err)
	}

	// Collect analytics every 4 hours
	_, err = s.scheduler.Register("0 */4 * * *", NewCollectAnalyticsTask())
	if err != nil {
		return fmt.Errorf("register collect_analytics: %w", err)
	}

	// Weekly learning - Monday 02:00 WIB
	_, err = s.scheduler.Register("0 2 * * 1", asynq.NewTask(TaskWeeklyLearning, nil, asynq.MaxRetry(3), asynq.Timeout(300*time.Second)))
	if err != nil {
		return fmt.Errorf("register weekly_learning: %w", err)
	}

	// Link health check every hour
	_, err = s.scheduler.Register("0 * * * *", asynq.NewTask(TaskHealthCheckLinks, nil, asynq.MaxRetry(1), asynq.Timeout(30*time.Second)))
	if err != nil {
		return fmt.Errorf("register health_check_links: %w", err)
	}

	// Circuit breaker check every 15 minutes
	_, err = s.scheduler.Register("*/15 * * * *", NewCircuitBreakerCheckTask())
	if err != nil {
		return fmt.Errorf("register circuit_breaker_check: %w", err)
	}

	return nil
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	return s.scheduler.Start()
}

// Shutdown stops the scheduler
func (s *Scheduler) Shutdown() {
	s.scheduler.Shutdown()
}

// Client wraps asynq.Client for enqueuing tasks
type Client struct {
	client *asynq.Client
}

// NewClient creates a new queue client
func NewClient(redisAddr string) *Client {
	return &Client{
		client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr}),
	}
}

// Enqueue enqueues a task
func (c *Client) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.client.Enqueue(task, opts...)
}

// EnqueueContext enqueues a task with context
func (c *Client) EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.client.EnqueueContext(ctx, task, opts...)
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.client.Close()
}
