package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a platform user account
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Plan         string    `json:"plan" db:"plan"` // trial, starter, pro, agency
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// ThreadsAccount represents a connected Threads social media account
type ThreadsAccount struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	UserID            uuid.UUID  `json:"user_id" db:"user_id"`
	ThreadsUserID     string     `json:"threads_user_id" db:"threads_user_id"`
	AccessToken       string     `json:"-" db:"access_token"`   // encrypted
	RefreshToken      string     `json:"-" db:"refresh_token"`
	Persona           string     `json:"persona" db:"persona"`
	Niche             string     `json:"niche" db:"niche"`
	Status            string     `json:"status" db:"status"` // active, paused, flagged
	AutoMode          bool       `json:"auto_mode" db:"auto_mode"`
	AutoModeEnabledAt *time.Time `json:"auto_mode_enabled_at,omitempty" db:"auto_mode_enabled_at"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// Product represents an affiliate product
type Product struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	Price     float64   `json:"price" db:"price"`
	Category  string    `json:"category" db:"category"`
	Platform  string    `json:"platform" db:"platform"` // shopee, tiktok
	ImageURL  string    `json:"image_url" db:"image_url"`
	Status    string    `json:"status" db:"status"` // active, needs_info
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// AffiliateLink represents a tracked affiliate link
type AffiliateLink struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ProductID   uuid.UUID `json:"product_id" db:"product_id"`
	OriginalURL string    `json:"original_url" db:"original_url"`
	ShortSlug   string    `json:"short_slug" db:"short_slug"` // unique
	Platform    string    `json:"platform" db:"platform"`
	Status      string    `json:"status" db:"status"` // active, expired
	ClickCount  int       `json:"click_count" db:"click_count"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Post represents a scheduled or published Threads post
type Post struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	AccountID     uuid.UUID  `json:"account_id" db:"account_id"`
	LinkID        *uuid.UUID `json:"link_id,omitempty" db:"link_id"`       // nullable
	Content       string     `json:"content" db:"content"`
	LinkPlacement string     `json:"link_placement" db:"link_placement"`   // direct, reply_drop, bio, question_trigger, thread_end
	Persona       string     `json:"persona" db:"persona"`
	Format        string     `json:"format" db:"format"`                   // single, thread, hot_take, question, story
	ScheduledAt   time.Time  `json:"scheduled_at" db:"scheduled_at"`
	PublishedAt   *time.Time `json:"published_at,omitempty" db:"published_at"` // nullable
	ThreadID      *string    `json:"thread_id,omitempty" db:"thread_id"`       // nullable
	Status        string     `json:"status" db:"status"`                       // draft, pending_review, approved, published, failed
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// PostAnalytics represents engagement metrics for a post
type PostAnalytics struct {
	ID        uuid.UUID `json:"id" db:"id"`
	PostID    uuid.UUID `json:"post_id" db:"post_id"`
	Views     int       `json:"views" db:"views"`
	Likes     int       `json:"likes" db:"likes"`
	Replies   int       `json:"replies" db:"replies"`
	Reposts   int       `json:"reposts" db:"reposts"`
	Clicks    int       `json:"clicks" db:"clicks"`
	FetchedAt time.Time `json:"fetched_at" db:"fetched_at"`
}

// ClickLog represents a single click event on an affiliate link
type ClickLog struct {
	ID        uuid.UUID `json:"id" db:"id"`
	LinkID    uuid.UUID `json:"link_id" db:"link_id"`
	HashedIP  string    `json:"hashed_ip" db:"hashed_ip"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	Referrer  string    `json:"referrer" db:"referrer"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CircuitBreaker represents a safety circuit breaker event for an account
type CircuitBreaker struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	AccountID   uuid.UUID  `json:"account_id" db:"account_id"`
	EventType   string     `json:"event_type" db:"event_type"`
	TriggeredAt time.Time  `json:"triggered_at" db:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty" db:"resolved_at"` // nullable
}
