package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/clint/f1/backend/internal/ingest"
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

type stubImporter struct {
	outcome ingest.Outcome
	err     error
}

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
