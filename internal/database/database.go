package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates a PostgreSQL connection pool using the provided configuration.
func Connect(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// RunMigrations reads .up.sql files from the specified directory and executes them in order.
// Only runs files ending in .up.sql to avoid executing down migrations.
func RunMigrations(pool *pgxpool.Pool, dir string) error {
	sqlFiles, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	sort.Strings(sqlFiles)

	ctx := context.Background()
	for _, file := range sqlFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		sql := strings.TrimSpace(string(content))
		if sql == "" {
			continue
		}

		if _, err := pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}

	return nil
}
