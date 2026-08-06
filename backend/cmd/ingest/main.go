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
	unit       string
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

type timingImporter interface {
	ImportTiming(context.Context, ingest.Target) (ingest.TimingOutcome, error)
}

type commandRuntime struct {
	importer       weekendImporter
	timingImporter timingImporter
	audit          database.AuditDB
	requestRecords func() []openf1.RequestRecord
	close          func()
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

	target := ingest.Target{Season: options.season, MeetingKey: options.meetingKey}
	if options.unit == "timing" {
		return runTiming(ctx, runtime, target, output, options)
	}
	var runID int64
	if runtime.audit != nil {
		runID, err = database.CreateImportRun(ctx, runtime.audit, target.Season, target.MeetingKey, time.Now().UTC())
		if err != nil {
			return err
		}
	}

	outcome, importErr := runtime.importer.ImportWeekend(ctx, target)
	if runtime.audit != nil {
		auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		auditErr := recordAudit(auditCtx, runtime, runID, outcome, importErr)
		cancel()
		if auditErr != nil {
			importErr = errors.Join(importErr, auditErr)
		}
	}
	err = importErr
	if err != nil {
		var quarantined *ingest.QuarantineError
		if errors.As(err, &quarantined) {
			fmt.Fprintf(errorOutput, "weekend quarantined: meeting_id=%s sessions=%d entries=%d results=%d errors=%d\n",
				outcome.MeetingID, outcome.SessionCount, outcome.EntryCount, outcome.ResultCount, len(quarantined.Errors))
			for _, problem := range quarantined.Errors {
				fmt.Fprintf(errorOutput, "%s: entity=%s source=%q driver_number=%d: %s\n",
					problem.Code, problem.Entity, problem.SourceValue, problem.DriverNumber, problem.Message)
			}
		}
		return fmt.Errorf("import season %d meeting %d: %w", options.season, options.meetingKey, err)
	}
	fmt.Fprintf(output, "weekend published: meeting_id=%s sessions=%d entries=%d results=%d transformed_at=%s published_at=%s\n",
		outcome.MeetingID, outcome.SessionCount, outcome.EntryCount, outcome.ResultCount,
		outcome.TransformedAt.Format(time.RFC3339), outcome.PublishedAt.Format(time.RFC3339))
	return nil
}

func runTiming(ctx context.Context, runtime commandRuntime, target ingest.Target, output io.Writer, options options) error {
	if runtime.timingImporter == nil {
		return errors.New("timing importer is not configured")
	}
	var runID int64
	var err error
	if runtime.audit != nil {
		runID, err = database.CreateTimingImportRun(ctx, runtime.audit, target.Season, target.MeetingKey, time.Now().UTC())
		if err != nil {
			return err
		}
	}
	outcome, importErr := runtime.timingImporter.ImportTiming(ctx, target)
	if runtime.audit != nil {
		auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		auditErr := recordTimingAudit(auditCtx, runtime, runID, outcome, importErr)
		cancel()
		if auditErr != nil {
			importErr = errors.Join(importErr, auditErr)
		}
	}
	if importErr != nil {
		return fmt.Errorf("import timing for season %d meeting %d: %w", options.season, options.meetingKey, importErr)
	}
	fmt.Fprintf(output, "timing published: session_id=%s laps=%d stints=%d published_at=%s\n",
		outcome.SessionID, outcome.LapCount, outcome.StintCount, outcome.PublishedAt.Format(time.RFC3339))
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

	timingStore := database.NewTimingStore(pool)
	return commandRuntime{
		importer:       ingest.NewImporter(client, database.NewWeekendPublisher(pool)),
		timingImporter: ingest.NewTimingImporter(timingStore, client, timingStore), audit: pool,
		requestRecords: client.RequestRecords, close: pool.Close,
	}, nil
}

func recordTimingAudit(ctx context.Context, runtime commandRuntime, runID int64, outcome ingest.TimingOutcome, importErr error) error {
	var auditErrors []error
	if runtime.requestRecords != nil {
		for _, request := range runtime.requestRecords() {
			if err := database.RecordImportRunRequest(ctx, runtime.audit, runID, database.ImportRunRequest{
				Endpoint: request.Endpoint, Parameters: request.Parameters, ResponseStatus: request.ResponseStatus,
				FetchedAt: request.FetchedAt, RecordCount: request.RecordCount, ResponseSHA256: request.ResponseSHA256,
			}); err != nil {
				auditErrors = append(auditErrors, err)
			}
		}
	}
	status := "succeeded"
	var publishedAt *time.Time
	var deferredReason *string
	if importErr == nil && !outcome.PublishedAt.IsZero() {
		published := outcome.PublishedAt
		publishedAt = &published
	} else if importErr == nil {
		status = "failed"
		auditErrors = append(auditErrors, errors.New("successful timing import did not report publication time"))
	} else {
		status = "failed"
		var deferred *ingest.DeferredTimingError
		if errors.As(importErr, &deferred) || errors.Is(importErr, openf1.ErrLiveDataWindow) {
			status = "deferred"
			reason := importErr.Error()
			deferredReason = &reason
		}
		var quarantined *ingest.QuarantineError
		if errors.As(importErr, &quarantined) {
			status = "quarantined"
			for order, problem := range quarantined.Errors {
				if err := database.RecordImportRunError(ctx, runtime.audit, runID, database.ImportRunError{
					Order: order, Code: problem.Code, Entity: problem.Entity,
					SourceContext: map[string]any{"source_value": problem.SourceValue, "driver_number": problem.DriverNumber}, Message: problem.Message,
				}); err != nil {
					auditErrors = append(auditErrors, err)
				}
			}
		}
	}
	if err := database.FinishImportRun(ctx, runtime.audit, runID, database.ImportRunCompletion{
		Status: status, FinishedAt: time.Now().UTC(), SessionCount: boolCount(outcome.SessionKey > 0), LapCount: outcome.LapCount,
		StintCount: outcome.StintCount, SourceSessionKey: optionalPositiveInt(outcome.SessionKey), DeferredReason: deferredReason, PublishedAt: publishedAt,
	}); err != nil {
		auditErrors = append(auditErrors, err)
	}
	return errors.Join(auditErrors...)
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func optionalPositiveInt(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func recordAudit(ctx context.Context, runtime commandRuntime, runID int64, outcome ingest.Outcome, importErr error) error {
	var auditErrors []error
	if runtime.requestRecords != nil {
		for _, request := range runtime.requestRecords() {
			err := database.RecordImportRunRequest(ctx, runtime.audit, runID, database.ImportRunRequest{
				Endpoint: request.Endpoint, Parameters: request.Parameters, ResponseStatus: request.ResponseStatus,
				FetchedAt: request.FetchedAt, RecordCount: request.RecordCount, ResponseSHA256: request.ResponseSHA256,
			})
			if err != nil {
				auditErrors = append(auditErrors, err)
			}
		}
	}

	status := "succeeded"
	var publishedAt *time.Time
	var deferredReason *string
	if importErr == nil {
		if outcome.PublishedAt.IsZero() {
			status = "failed"
			auditErrors = append(auditErrors, errors.New("successful import did not report publication time"))
		} else {
			published := outcome.PublishedAt
			publishedAt = &published
		}
	} else {
		status = "failed"
		if errors.Is(importErr, openf1.ErrLiveDataWindow) {
			status = "deferred"
			reason := "OpenF1 session detail is not yet available"
			deferredReason = &reason
		}
		var quarantined *ingest.QuarantineError
		if errors.As(importErr, &quarantined) {
			status = "quarantined"
			for order, problem := range quarantined.Errors {
				err := database.RecordImportRunError(ctx, runtime.audit, runID, database.ImportRunError{
					Order: order, Code: problem.Code, Entity: problem.Entity,
					SourceContext: map[string]any{"source_value": problem.SourceValue, "driver_number": problem.DriverNumber},
					Message:       problem.Message,
				})
				if err != nil {
					auditErrors = append(auditErrors, err)
				}
			}
		}
	}
	if err := database.FinishImportRun(ctx, runtime.audit, runID, database.ImportRunCompletion{
		Status: status, FinishedAt: time.Now().UTC(), SessionCount: outcome.SessionCount,
		EntryCount: outcome.EntryCount, ResultCount: outcome.ResultCount, ErrorCount: outcome.ErrorCount,
		DeferredReason: deferredReason, PublishedAt: publishedAt,
	}); err != nil {
		auditErrors = append(auditErrors, err)
	}
	return errors.Join(auditErrors...)
}

func parseOptions(args []string, output io.Writer, currentYear int) (options, error) {
	flags := flag.NewFlagSet("ingest", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.Usage = func() {
		fmt.Fprintln(output, "Usage: ingest --season YEAR --meeting OPENF1_MEETING_KEY [--unit weekend|timing]")
		flags.PrintDefaults()
	}

	var parsed options
	flags.IntVar(&parsed.season, "season", 0, "season year to import")
	flags.IntVar(&parsed.meetingKey, "meeting", 0, "OpenF1 meeting key to import")
	flags.StringVar(&parsed.unit, "unit", "weekend", "publication unit to import: weekend or timing")
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
	if parsed.unit != "weekend" && parsed.unit != "timing" {
		return options{}, errors.New("unit must be weekend or timing")
	}

	return parsed, nil
}
