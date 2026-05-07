package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CircuitBreakerEvent represents a circuit breaker event record
type CircuitBreakerEvent struct {
	ID           uuid.UUID
	AccountID    uuid.UUID
	EventType    string
	TriggeredAt  time.Time
	ResolvedAt   *time.Time
	Severity     string
	CooldownUntil *time.Time
	AutoResolved bool
	Notes        string
}

// FlaggedAccountInfo holds info about a flagged account for circuit breaker checks
type FlaggedAccountInfo struct {
	AccountID uuid.UUID
	UserID    uuid.UUID
	Status    string
}

// UnresolvedCBEvent holds info about an unresolved circuit breaker event
type UnresolvedCBEvent struct {
	ID            uuid.UUID
	AccountID     uuid.UUID
	UserID        uuid.UUID
	CooldownUntil *time.Time
}

// GetFlaggedAccounts returns all accounts with status = 'flagged'.
func GetFlaggedAccounts(ctx context.Context, pool *pgxpool.Pool) ([]FlaggedAccountInfo, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, user_id, status FROM threads_accounts WHERE status = 'flagged'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []FlaggedAccountInfo
	for rows.Next() {
		var a FlaggedAccountInfo
		if err := rows.Scan(&a.AccountID, &a.UserID, &a.Status); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// HasRecentCBEvent checks if a circuit breaker event already exists for an account in the last 24h.
func HasRecentCBEvent(ctx context.Context, pool *pgxpool.Pool, accountID uuid.UUID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM circuit_breaker
			WHERE account_id = $1
			  AND triggered_at > NOW() - INTERVAL '24 hours'
			  AND resolved_at IS NULL
		)`, accountID).Scan(&exists)
	return exists, err
}

// InsertCircuitBreakerEvent inserts a new circuit breaker event.
func InsertCircuitBreakerEvent(ctx context.Context, pool *pgxpool.Pool, accountID uuid.UUID, eventType, severity, notes string, cooldownUntil time.Time) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO circuit_breaker (id, account_id, event_type, triggered_at, severity, cooldown_until, auto_resolved, notes)
		VALUES (gen_random_uuid(), $1, $2, NOW(), $3, $4, false, $5)`,
		accountID, eventType, severity, cooldownUntil, notes)
	return err
}

// PauseAccount sets an account's status to 'paused'.
func PauseAccount(ctx context.Context, pool *pgxpool.Pool, accountID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE threads_accounts SET status = 'paused' WHERE id = $1`, accountID)
	return err
}

// PauseAllUserAccounts pauses all accounts for a given user.
func PauseAllUserAccounts(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE threads_accounts SET status = 'paused' WHERE user_id = $1`, userID)
	return err
}

// CountFlaggedAccountsForUserLast24h counts how many accounts for a user were flagged in the last 24h.
func CountFlaggedAccountsForUserLast24h(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT account_id) FROM circuit_breaker cb
		JOIN threads_accounts ta ON cb.account_id = ta.id
		WHERE ta.user_id = $1
		  AND cb.triggered_at > NOW() - INTERVAL '24 hours'
		  AND cb.resolved_at IS NULL`, userID).Scan(&count)
	return count, err
}

// GetUnresolvedExpiredCBEvents returns circuit breaker events where cooldown has expired and not yet resolved.
func GetUnresolvedExpiredCBEvents(ctx context.Context, pool *pgxpool.Pool) ([]UnresolvedCBEvent, error) {
	rows, err := pool.Query(ctx, `
		SELECT cb.id, cb.account_id, ta.user_id, cb.cooldown_until
		FROM circuit_breaker cb
		JOIN threads_accounts ta ON cb.account_id = ta.id
		WHERE cb.resolved_at IS NULL
		  AND cb.cooldown_until IS NOT NULL
		  AND cb.cooldown_until < NOW()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []UnresolvedCBEvent
	for rows.Next() {
		var e UnresolvedCBEvent
		if err := rows.Scan(&e.ID, &e.AccountID, &e.UserID, &e.CooldownUntil); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// HasNewFlagsSinceCooldown checks if the account has been flagged again since the cooldown started.
func HasNewFlagsSinceCooldown(ctx context.Context, pool *pgxpool.Pool, accountID uuid.UUID, cbEventID uuid.UUID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM circuit_breaker
			WHERE account_id = $1
			  AND id != $2
			  AND triggered_at > (SELECT triggered_at FROM circuit_breaker WHERE id = $2)
			  AND resolved_at IS NULL
		)`, accountID, cbEventID).Scan(&exists)
	return exists, err
}

// ResolveCBEvent marks a circuit breaker event as resolved.
func ResolveCBEvent(ctx context.Context, pool *pgxpool.Pool, cbEventID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE circuit_breaker SET resolved_at = NOW(), auto_resolved = true WHERE id = $1`, cbEventID)
	return err
}

// ReactivateAccountWithReducedLimit sets account status to 'active' and reduces max_daily_posts by 50% (min 5).
func ReactivateAccountWithReducedLimit(ctx context.Context, pool *pgxpool.Pool, accountID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE threads_accounts
		SET status = 'active',
		    max_daily_posts = GREATEST(5, max_daily_posts / 2)
		WHERE id = $1`, accountID)
	return err
}

// ResetDailyPostCountsIfNeeded resets daily_post_count for accounts where last_post_reset_at is not today.
func ResetDailyPostCountsIfNeeded(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		UPDATE threads_accounts
		SET daily_post_count = 0, last_post_reset_at = CURRENT_DATE
		WHERE last_post_reset_at IS NULL OR last_post_reset_at < CURRENT_DATE`)
	return err
}
