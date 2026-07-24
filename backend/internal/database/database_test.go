package database

import (
	"strings"
	"testing"
)

func TestPoolConfigFromEnv(t *testing.T) {
	t.Setenv(URLKey, "postgres://user:password@localhost:5432/racescope?sslmode=disable")

	config, err := PoolConfigFromEnv()
	if err != nil {
		t.Fatalf("PoolConfigFromEnv() error = %v", err)
	}
	if config.ConnConfig.Host != "localhost" {
		t.Fatalf("host = %q, want localhost", config.ConnConfig.Host)
	}
	if config.ConnConfig.Database != "racescope" {
		t.Fatalf("database = %q, want racescope", config.ConnConfig.Database)
	}
}

func TestPoolConfigFromEnvRejectsMissingURL(t *testing.T) {
	t.Setenv(URLKey, "")

	_, err := PoolConfigFromEnv()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("error = %v, want required DATABASE_URL error", err)
	}
}

func TestPoolConfigFromEnvRejectsInvalidURL(t *testing.T) {
	tests := map[string]string{
		"wrong scheme": "mysql://user:password@localhost:3306/racescope",
		"missing host": "postgres:///racescope",
		"missing name": "postgres://user:password@localhost:5432",
	}

	for name, databaseURL := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(URLKey, databaseURL)
			if _, err := PoolConfigFromEnv(); err == nil {
				t.Fatalf("PoolConfigFromEnv() error = nil, want validation error")
			}
		})
	}
}
