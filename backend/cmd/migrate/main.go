package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/clint/f1/backend/internal/database"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const migrationDirectory = "migrations"

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: migrate <up|down|status>")
	}

	config, err := database.PoolConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	db := sql.OpenDB(stdlib.GetConnector(*config.ConnConfig))
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect to PostgreSQL: %v", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("configure Goose: %v", err)
	}

	switch os.Args[1] {
	case "up":
		err = goose.UpContext(ctx, db, migrationDirectory)
	case "down":
		err = goose.DownContext(ctx, db, migrationDirectory)
	case "status":
		err = goose.StatusContext(ctx, db, migrationDirectory)
	default:
		err = fmt.Errorf("unknown migration command %q; use up, down, or status", os.Args[1])
	}
	if err != nil {
		log.Fatal(err)
	}
}
