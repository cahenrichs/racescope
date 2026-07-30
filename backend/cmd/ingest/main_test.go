package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/clint/f1/backend/internal/database"
	"github.com/clint/f1/backend/internal/ingest"
	"github.com/clint/f1/backend/internal/openf1"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestParseOptions(t *testing.T) {
	t.Parallel()

	got, err := parseOptions([]string{"--season", "2024", "--meeting", "1242"}, &bytes.Buffer{}, 2026)
	if err != nil {
		t.Fatalf("parseOptions() error = %v", err)
	}
	if got.season != 2024 || got.meetingKey != 1242 {
		t.Fatalf("parseOptions() = %+v, want season 2024 and meeting 1242", got)
	}
}

func TestRunReportsSuccessfulOutcome(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	closed := false
	build := func(context.Context) (commandRuntime, error) {
		return commandRuntime{
			importer: stubImporter{outcome: ingest.Outcome{
				MeetingID: "meeting_test", SessionCount: 5, EntryCount: 20, ResultCount: 20,
				TransformedAt: time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC),
				PublishedAt:   time.Date(2026, time.July, 29, 12, 0, 1, 0, time.UTC),
			}},
			close: func() { closed = true },
		}, nil
	}

	err := run(context.Background(), []string{"--season", "2024", "--meeting", "1235"}, &output, &bytes.Buffer{}, 2026, build)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !closed {
		t.Fatal("run() did not close its runtime")
	}
	if got := output.String(); !strings.Contains(got, "meeting_id=meeting_test sessions=5 entries=20 results=20") || !strings.Contains(got, "2026-07-29T12:00:00Z") {
		t.Fatalf("run() output = %q", got)
	}
}

func TestRunReturnsSetupAndImportFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		build runtimeFactory
		want  string
	}{
		{
			name: "setup",
			build: func(context.Context) (commandRuntime, error) {
				return commandRuntime{}, errors.New("database unavailable")
			},
			want: "database unavailable",
		},
		{
			name: "import",
			build: func(context.Context) (commandRuntime, error) {
				return commandRuntime{importer: stubImporter{err: errors.New("identity mismatch")}, close: func() {}}, nil
			},
			want: "identity mismatch",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := run(context.Background(), []string{"--season", "2024", "--meeting", "1235"}, &bytes.Buffer{}, &bytes.Buffer{}, 2026, test.build)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("run() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestRunReportsQuarantineErrors(t *testing.T) {
	t.Parallel()

	var errorOutput bytes.Buffer
	quarantine := &ingest.QuarantineError{Errors: []ingest.TransformError{{
		Code: "unknown_driver", Entity: "driver", SourceValue: "Mystery DRIVER", DriverNumber: 99,
		Message: "full name has no reviewed mapping",
	}}}
	build := func(context.Context) (commandRuntime, error) {
		return commandRuntime{
			importer: stubImporter{outcome: ingest.Outcome{MeetingID: "meeting_test", SessionCount: 5, ErrorCount: 1}, err: quarantine},
			close:    func() {},
		}, nil
	}

	err := run(context.Background(), []string{"--season", "2024", "--meeting", "1235"}, &bytes.Buffer{}, &errorOutput, 2026, build)
	if !errors.Is(err, quarantine) {
		t.Fatalf("run() error = %v, want quarantine", err)
	}
	if got := errorOutput.String(); !strings.Contains(got, "weekend quarantined") || !strings.Contains(got, `unknown_driver: entity=driver source="Mystery DRIVER" driver_number=99`) {
		t.Fatalf("run() error output = %q", got)
	}
}

func TestRunAuditsDeferredImport(t *testing.T) {
	t.Parallel()

	audit := &recordingAudit{}
	fetchedAt := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	build := func(context.Context) (commandRuntime, error) {
		return commandRuntime{
			importer: stubImporter{
				outcome: ingest.Outcome{SessionCount: 5},
				err:     fmt.Errorf("fetch Grand Prix entries: %w", openf1.ErrLiveDataWindow),
			},
			audit: audit,
			requestRecords: func() []openf1.RequestRecord {
				return []openf1.RequestRecord{{
					Endpoint: "sessions", Parameters: map[string][]string{"meeting_key": {"1235"}},
					ResponseStatus: 200, FetchedAt: fetchedAt, RecordCount: 5,
					ResponseSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				}}
			},
			close: func() {},
		}, nil
	}

	err := run(context.Background(), []string{"--season", "2024", "--meeting", "1235"}, &bytes.Buffer{}, &bytes.Buffer{}, 2026, build)
	if !errors.Is(err, openf1.ErrLiveDataWindow) {
		t.Fatalf("run() error = %v, want live-window error", err)
	}
	completion := audit.completion(t)
	if completion.status != "deferred" || completion.deferredReason != "OpenF1 session detail is not yet available" || completion.sessionCount != 5 {
		t.Fatalf("completion = %+v", completion)
	}
	if audit.requestCount != 1 {
		t.Fatalf("request audit count = %d, want 1", audit.requestCount)
	}
}

func TestRunFinalizesInterruptedImport(t *testing.T) {
	t.Parallel()

	audit := &recordingAudit{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	build := func(context.Context) (commandRuntime, error) {
		return commandRuntime{importer: stubImporter{err: context.Canceled}, audit: audit, close: func() {}}, nil
	}

	err := run(ctx, []string{"--season", "2024", "--meeting", "1235"}, &bytes.Buffer{}, &bytes.Buffer{}, 2026, build)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("run() error = %v, want context cancellation", err)
	}
	if completion := audit.completion(t); completion.status != "failed" {
		t.Fatalf("completion = %+v, want failed", completion)
	}
}

func TestRunAuditsSuccessfulPublicationTime(t *testing.T) {
	t.Parallel()

	audit := &recordingAudit{}
	publishedAt := time.Date(2026, time.July, 30, 12, 5, 0, 0, time.UTC)
	build := func(context.Context) (commandRuntime, error) {
		return commandRuntime{
			importer: stubImporter{outcome: ingest.Outcome{
				MeetingID: "meeting_test", SessionCount: 5, EntryCount: 20, ResultCount: 20,
				TransformedAt: publishedAt.Add(-time.Minute), PublishedAt: publishedAt,
			}},
			audit: audit, close: func() {},
		}, nil
	}

	if err := run(context.Background(), []string{"--season", "2024", "--meeting", "1235"}, &bytes.Buffer{}, &bytes.Buffer{}, 2026, build); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	completion := audit.completion(t)
	if completion.status != "succeeded" || completion.publishedAt == nil || !completion.publishedAt.Equal(publishedAt) {
		t.Fatalf("completion = %+v", completion)
	}
}

type stubImporter struct {
	outcome ingest.Outcome
	err     error
}

type recordingAudit struct {
	requestCount int
	updates      [][]any
}

type recordedCompletion struct {
	status         string
	sessionCount   int
	deferredReason string
	publishedAt    *time.Time
}

func (audit *recordingAudit) QueryRow(context.Context, string, ...any) pgx.Row {
	return scanRow(func(destinations ...any) error {
		*destinations[0].(*int64) = 42
		return nil
	})
}

func (audit *recordingAudit) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(query, "INSERT INTO import_run_requests") {
		audit.requestCount++
	}
	if strings.Contains(query, "UPDATE import_runs") {
		audit.updates = append(audit.updates, append([]any(nil), args...))
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (audit *recordingAudit) completion(t *testing.T) recordedCompletion {
	t.Helper()
	if len(audit.updates) != 1 {
		t.Fatalf("completion updates = %d, want 1", len(audit.updates))
	}
	args := audit.updates[0]
	completion := recordedCompletion{status: args[1].(string), sessionCount: args[3].(int)}
	if reason := args[7].(*string); reason != nil {
		completion.deferredReason = *reason
	}
	completion.publishedAt = args[8].(*time.Time)
	return completion
}

type scanRow func(...any) error

func (row scanRow) Scan(destinations ...any) error { return row(destinations...) }

var _ database.AuditDB = (*recordingAudit)(nil)

func (importer stubImporter) ImportWeekend(context.Context, ingest.Target) (ingest.Outcome, error) {
	return importer.outcome, importer.err
}

func TestParseOptionsRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing season", args: []string{"--meeting", "1242"}, want: "season must be between 2023 and 2026"},
		{name: "unsupported season", args: []string{"--season", "2022", "--meeting", "1242"}, want: "season must be between 2023 and 2026"},
		{name: "future season", args: []string{"--season", "2027", "--meeting", "1242"}, want: "season must be between 2023 and 2026"},
		{name: "missing meeting", args: []string{"--season", "2024"}, want: "meeting must be a positive OpenF1 meeting key"},
		{name: "invalid meeting", args: []string{"--season", "2024", "--meeting", "-1"}, want: "meeting must be a positive OpenF1 meeting key"},
		{name: "positional argument", args: []string{"--season", "2024", "--meeting", "1242", "extra"}, want: "unexpected positional arguments"},
		{name: "unknown option", args: []string{"--season", "2024", "--meeting", "1242", "--all"}, want: "flag provided but not defined"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseOptions(test.args, &bytes.Buffer{}, 2026)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("parseOptions() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}
