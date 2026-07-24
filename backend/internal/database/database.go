package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const URLKey = "DATABASE_URL"

func PoolConfigFromEnv() (*pgxpool.Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv(URLKey))
	if databaseURL == "" {
		return nil, fmt.Errorf("%s is required; set it to a PostgreSQL connection URL", URLKey)
	}

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid PostgreSQL connection URL: %w", URLKey, err)
	}
	if parsedURL.Scheme != "postgres" && parsedURL.Scheme != "postgresql" {
		return nil, fmt.Errorf("%s must use the postgres or postgresql scheme", URLKey)
	}
	if parsedURL.Hostname() == "" {
		return nil, fmt.Errorf("%s must include a database host", URLKey)
	}
	if strings.Trim(parsedURL.Path, "/") == "" {
		return nil, fmt.Errorf("%s must include a database name", URLKey)
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", URLKey, err)
	}
	return config, nil
}

func Open(ctx context.Context) (*pgxpool.Pool, error) {
	config, err := PoolConfigFromEnv()
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL connection pool: %w", err)
	}
	return pool, nil
}
