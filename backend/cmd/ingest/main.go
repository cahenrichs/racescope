package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/clint/f1/backend/internal/database"
	"github.com/clint/f1/backend/internal/ingest"
	"github.com/clint/f1/backend/internal/openf1"
)

const openF1BaseURLKey = "OPENF1_BASE_URL"

type options struct {
	season     int
	meetingKey int
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr, time.Now().UTC().Year(), newRuntime); err != nil {
		logger.Error("weekend import failed", "error", err)
		os.Exit(1)
	}
}

type weekendImporter interface {
	ImportWeekend(context.Context, ingest.Target) (ingest.Outcome, error)
}

type commandRuntime struct {
	importer weekendImporter
	close    func()
}

type runtimeFactory func(context.Context) (commandRuntime, error)

func run(ctx context.Context, args []string, output, errorOutput io.Writer, currentYear int, build runtimeFactory) error {
	options, err := parseOptions(args, errorOutput, currentYear)
	if err != nil {
		return err
	}

	runtime, err := build(ctx)
	if err != nil {
		return err
	}
	defer runtime.close()

	outcome, err := runtime.importer.ImportWeekend(ctx, ingest.Target{Season: options.season, MeetingKey: options.meetingKey})
	if err != nil {
		return fmt.Errorf("import season %d meeting %d: %w", options.season, options.meetingKey, err)
	}
	fmt.Fprintf(output, "weekend transformed: meeting_id=%s sessions=%d transformed_at=%s\n",
		outcome.MeetingID, outcome.SessionCount, outcome.TransformedAt.Format(time.RFC3339))
	return nil
}

func newRuntime(ctx context.Context) (commandRuntime, error) {
	client, err := openf1.NewClient(openf1.Config{BaseURL: strings.TrimSpace(os.Getenv(openF1BaseURLKey))})
	if err != nil {
		return commandRuntime{}, fmt.Errorf("configure OpenF1: %w", err)
	}

	pool, err := database.Open(ctx)
	if err != nil {
		return commandRuntime{}, fmt.Errorf("open database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return commandRuntime{}, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	return commandRuntime{importer: ingest.NewImporter(client), close: pool.Close}, nil
}

func parseOptions(args []string, output io.Writer, currentYear int) (options, error) {
	flags := flag.NewFlagSet("ingest", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.Usage = func() {
		fmt.Fprintln(output, "Usage: ingest --season YEAR --meeting OPENF1_MEETING_KEY")
		flags.PrintDefaults()
	}

	var parsed options
	flags.IntVar(&parsed.season, "season", 0, "season year to import")
	flags.IntVar(&parsed.meetingKey, "meeting", 0, "OpenF1 meeting key to import")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if parsed.season < openf1.FirstSupportedYear || parsed.season > currentYear {
		return options{}, fmt.Errorf("season must be between %d and %d", openf1.FirstSupportedYear, currentYear)
	}
	if parsed.meetingKey <= 0 {
		return options{}, errors.New("meeting must be a positive OpenF1 meeting key")
	}

	return parsed, nil
}
