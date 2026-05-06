package shortener

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const slugAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// ClickData contains click tracking details stored for a shortened link.
type ClickData struct {
	LinkID    string    `json:"link_id"`
	HashedIP  string    `json:"hashed_ip"`
	UserAgent string    `json:"user_agent"`
	Referrer  string    `json:"referrer"`
	Timestamp time.Time `json:"timestamp"`
}

// GenerateSlug returns a random lowercase alphanumeric slug of the requested length.
func GenerateSlug(length int) string {
	if length <= 0 {
		return ""
	}

	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(fmt.Errorf("failed to generate slug: %w", err))
	}

	slug := make([]byte, length)
	for i, b := range randomBytes {
		slug[i] = slugAlphabet[int(b)%len(slugAlphabet)]
	}

	return string(slug)
}

// RegisterLink stores the original URL and link ID for a slug in Redis.
func RegisterLink(ctx context.Context, rdb *redis.Client, slug string, originalURL string, linkID string) error {
	return rdb.HSet(ctx, linkKey(slug), map[string]string{
		"url":     originalURL,
		"link_id": linkID,
	}).Err()
}

// Resolve returns the original URL and link ID registered for a slug.
func Resolve(ctx context.Context, rdb *redis.Client, slug string) (originalURL string, linkID string, err error) {
	values, err := rdb.HMGet(ctx, linkKey(slug), "url", "link_id").Result()
	if err != nil {
		return "", "", err
	}

	if values[0] != nil {
		originalURL, _ = values[0].(string)
	}
	if values[1] != nil {
		linkID, _ = values[1].(string)
	}

	return originalURL, linkID, nil
}

// TrackClick appends click tracking data for a link ID as JSON in Redis and persists it to PostgreSQL.
func TrackClick(ctx context.Context, rdb *redis.Client, pool *pgxpool.Pool, linkID string, hashedIP string, userAgent string, referrer string) error {
	data := ClickData{
		LinkID:    linkID,
		HashedIP:  hashedIP,
		UserAgent: userAgent,
		Referrer:  referrer,
		Timestamp: time.Now().UTC(),
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	redisErr := rdb.LPush(ctx, clicksKey(linkID), payload).Err()

	// Persist to PostgreSQL if pool is available
	if pool == nil {
		return redisErr
	}

	parsedLinkID, err := uuid.Parse(linkID)
	if err != nil {
		return fmt.Errorf("invalid link id: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO click_logs (id, link_id, hashed_ip, user_agent, referrer, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), parsedLinkID, hashedIP, userAgent, referrer, data.Timestamp)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE affiliate_links
		SET click_count = click_count + 1
		WHERE id = $1
	`, parsedLinkID)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return redisErr
}

func linkKey(slug string) string {
	return "link:" + slug
}

func clicksKey(linkID string) string {
	return "clicks:" + linkID
}
