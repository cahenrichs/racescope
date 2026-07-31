package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var integrationSchemaSequence atomic.Uint64

func TestCompletedWeekendConstraints(t *testing.T) {
	pool := newIntegrationPool(t)
	seed := seedCompletedWeekend(t, pool)
	ctx := context.Background()
	fetchedAt := time.Date(2026, time.July, 28, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		code string
		run  func() error
	}{
		{
			name: "public IDs are unique",
			code: "23505",
			run: func() error {
				_, err := pool.Exec(ctx, `INSERT INTO seasons (public_id, year, source_fetched_at) VALUES ($1, $2, $3)`, seed.seasonID, 2026, fetchedAt)
				return err
			},
		},
		{
			name: "source timestamps are required",
			code: "23502",
			run: func() error {
				_, err := pool.Exec(ctx, `INSERT INTO seasons (public_id, year) VALUES ($1, $2)`, "season_without_source_time", 2026)
				return err
			},
		},
		{
			name: "classification states are bounded",
			code: "23514",
			run: func() error {
				_, err := pool.Exec(ctx, `
					INSERT INTO session_results
						(public_id, session_entry_id, classification_state, source_fetched_at)
					VALUES ($1, $2, 'retired', $3)`, "result_invalid_state", seed.missingEntryID, fetchedAt)
				return err
			},
		},
		{
			name: "nonnumeric states reject positions",
			code: "23514",
			run: func() error {
				_, err := pool.Exec(ctx, `
					INSERT INTO session_results
						(public_id, session_entry_id, classification_state, position, source_fetched_at)
					VALUES ($1, $2, 'dnf', 4, $3)`, "result_dnf_with_position", seed.missingEntryID, fetchedAt)
				return err
			},
		},
		{
			name: "entries have at most one result",
			code: "23505",
			run: func() error {
				_, err := pool.Exec(ctx, `
					INSERT INTO session_results
						(public_id, session_entry_id, classification_state, position, source_fetched_at)
					VALUES ($1, $2, 'ordinary', 2, $3)`, "result_duplicate_entry", seed.ordinaryEntryID, fetchedAt)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPostgresErrorCode(t, tt.run(), tt.code)
		})
	}
}

func TestCompletedWeekendRepresentativeQueries(t *testing.T) {
	pool := newIntegrationPool(t)
	seed := seedCompletedWeekend(t, pool)
	ctx := context.Background()

	weekend, err := WeekendByPublicID(ctx, pool, seed.meetingID)
	if err != nil {
		t.Fatalf("WeekendByPublicID() error = %v", err)
	}
	if weekend.Year != 2025 || weekend.Name != "Monaco Grand Prix" || weekend.CircuitName != "Monaco" {
		t.Fatalf("WeekendByPublicID() = %+v", weekend)
	}
	if _, err := WeekendByPublicID(ctx, pool, domain.PublicID("meeting_missing")); !errors.Is(err, ErrWeekendNotFound) {
		t.Fatalf("WeekendByPublicID() missing error = %v, want ErrWeekendNotFound", err)
	}

	results, err := SessionClassification(ctx, pool, seed.sessionID)
	if err != nil {
		t.Fatalf("SessionClassification() error = %v", err)
	}
	wantStates := []domain.ClassificationState{
		domain.ClassificationOrdinary,
		domain.ClassificationDNS,
		domain.ClassificationDNF,
		domain.ClassificationDSQ,
		domain.ClassificationUnknown,
		domain.ClassificationMissing,
	}
	if len(results) != len(wantStates) {
		t.Fatalf("SessionClassification() returned %d results, want %d", len(results), len(wantStates))
	}
	for i, want := range wantStates {
		if results[i].State != want {
			t.Errorf("result %d state = %q, want %q", i, results[i].State, want)
		}
	}
	if results[0].Position == nil || *results[0].Position != 1 {
		t.Errorf("ordinary position = %v, want 1", results[0].Position)
	}
	if results[0].NumberOfLaps == nil || *results[0].NumberOfLaps != 0 {
		t.Errorf("ordinary laps = %v, want numeric zero", results[0].NumberOfLaps)
	}
	if results[4].NumberOfLaps != nil || results[5].NumberOfLaps != nil {
		t.Errorf("unknown and missing laps = %v, %v; want nil, nil", results[4].NumberOfLaps, results[5].NumberOfLaps)
	}

	emptySessionID := testPublicID(t, domain.EntitySession, "2025-monaco-grand-prix:practice")
	var meetingPK int64
	mustScanID(t, pool.QueryRow(ctx, `SELECT id FROM meetings WHERE public_id = $1`, seed.meetingID), &meetingPK)
	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (public_id, source_key, meeting_id, name, type, date_start, date_end, is_cancelled, source_fetched_at)
		VALUES ($1, 99999, $2, 'Practice', 'Practice', $3, $3, false, $3)`, emptySessionID, meetingPK, time.Now().UTC()); err != nil {
		t.Fatalf("insert session with no entries: %v", err)
	}
	empty, err := SessionClassification(ctx, pool, emptySessionID)
	if err != nil || len(empty) != 0 {
		t.Fatalf("SessionClassification() known empty session = %+v, error %v", empty, err)
	}
	if _, err := SessionClassification(ctx, pool, domain.PublicID("session_missing")); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("SessionClassification() unknown session error = %v, want ErrSessionNotFound", err)
	}
}

func TestPublicIDsReproduceAcrossDatabaseRebuilds(t *testing.T) {
	firstPool := newIntegrationPool(t)
	firstSeed := seedCompletedWeekend(t, firstPool)
	first, err := WeekendByPublicID(context.Background(), firstPool, firstSeed.meetingID)
	if err != nil {
		t.Fatalf("first WeekendByPublicID() error = %v", err)
	}

	secondPool := newIntegrationPool(t)
	secondSeed := seedCompletedWeekend(t, secondPool)
	second, err := WeekendByPublicID(context.Background(), secondPool, secondSeed.meetingID)
	if err != nil {
		t.Fatalf("second WeekendByPublicID() error = %v", err)
	}

	if first.PublicID != second.PublicID || first.CircuitPublicID != second.CircuitPublicID || firstSeed.sessionID != secondSeed.sessionID {
		t.Fatalf("rebuilt public IDs differ: first = %+v/%q, second = %+v/%q", first, firstSeed.sessionID, second, secondSeed.sessionID)
	}
}

type completedWeekendSeed struct {
	seasonID        domain.PublicID
	meetingID       domain.PublicID
	sessionID       domain.PublicID
	ordinaryEntryID int64
	missingEntryID  int64
}

func seedCompletedWeekend(t *testing.T, pool *pgxpool.Pool) completedWeekendSeed {
	t.Helper()
	ctx := context.Background()
	fetchedAt := time.Date(2026, time.July, 28, 10, 0, 0, 0, time.UTC)
	seasonID := testPublicID(t, domain.EntitySeason, "2025")
	circuitID := testPublicID(t, domain.EntityCircuit, "monaco")
	meetingID := testPublicID(t, domain.EntityMeeting, "2025:monaco-grand-prix")
	sessionID := testPublicID(t, domain.EntitySession, "2025:monaco-grand-prix:race")
	constructorID := testPublicID(t, domain.EntityConstructorEntrant, "2025:ferrari")

	var seasonPK, circuitPK, meetingPK, sessionPK, constructorPK int64
	mustScanID(t, pool.QueryRow(ctx, `INSERT INTO seasons (public_id, year, source_fetched_at) VALUES ($1, 2025, $2) RETURNING id`, seasonID, fetchedAt), &seasonPK)
	mustScanID(t, pool.QueryRow(ctx, `
		INSERT INTO circuits (public_id, source_key, short_name, country_code, country_name, location, source_fetched_at)
		VALUES ($1, 22, 'Monaco', 'MON', 'Monaco', 'Monte Carlo', $2) RETURNING id`, circuitID, fetchedAt), &circuitPK)
	mustScanID(t, pool.QueryRow(ctx, `
		INSERT INTO meetings
			(public_id, source_key, season_id, circuit_id, name, official_name, date_start, date_end, is_cancelled, source_fetched_at)
		VALUES ($1, 1250, $2, $3, 'Monaco Grand Prix', 'Formula 1 Grand Prix de Monaco 2025',
		        '2025-05-23T11:00:00Z', '2025-05-25T16:00:00Z', false, $4) RETURNING id`, meetingID, seasonPK, circuitPK, fetchedAt), &meetingPK)
	mustScanID(t, pool.QueryRow(ctx, `
		INSERT INTO sessions
			(public_id, source_key, meeting_id, name, type, date_start, date_end, is_cancelled, source_fetched_at)
		VALUES ($1, 9800, $2, 'Race', 'Race', '2025-05-25T13:00:00Z', '2025-05-25T15:00:00Z', false, $3) RETURNING id`, sessionID, meetingPK, fetchedAt), &sessionPK)
	mustScanID(t, pool.QueryRow(ctx, `
		INSERT INTO constructor_entrants (public_id, season_id, name, source_fetched_at)
		VALUES ($1, $2, 'Ferrari', $3) RETURNING id`, constructorID, seasonPK, fetchedAt), &constructorPK)

	participants := []struct {
		key    string
		name   string
		number int
		state  domain.ClassificationState
	}{
		{key: "ordinary", name: "Ordinary Driver", number: 1, state: domain.ClassificationOrdinary},
		{key: "dns", name: "DNS Driver", number: 2, state: domain.ClassificationDNS},
		{key: "dnf", name: "DNF Driver", number: 3, state: domain.ClassificationDNF},
		{key: "dsq", name: "DSQ Driver", number: 4, state: domain.ClassificationDSQ},
		{key: "unknown", name: "Unknown Driver", number: 5, state: domain.ClassificationUnknown},
		{key: "missing", name: "Missing Driver", number: 6, state: domain.ClassificationMissing},
	}

	seed := completedWeekendSeed{seasonID: seasonID, meetingID: meetingID, sessionID: sessionID}
	for _, participant := range participants {
		driverID := testPublicID(t, domain.EntityDriver, participant.key)
		entryID := testPublicID(t, domain.EntitySessionEntry, sessionID.String()+":"+participant.key)
		var driverPK, entryPK int64
		mustScanID(t, pool.QueryRow(ctx, `
			INSERT INTO drivers (public_id, first_name, last_name, full_name, name_acronym, source_fetched_at)
			VALUES ($1, $2, 'Driver', $2, $3, $4) RETURNING id`, driverID, participant.name, strings.ToUpper(participant.key[:3]), fetchedAt), &driverPK)
		mustScanID(t, pool.QueryRow(ctx, `
			INSERT INTO session_entries
				(public_id, session_id, driver_id, constructor_entrant_id, driver_number, team_colour, source_fetched_at)
			VALUES ($1, $2, $3, $4, $5, 'E80020', $6) RETURNING id`, entryID, sessionPK, driverPK, constructorPK, participant.number, fetchedAt), &entryPK)

		switch participant.state {
		case domain.ClassificationOrdinary:
			seed.ordinaryEntryID = entryPK
			resultID := testPublicID(t, domain.EntitySessionResult, entryID.String())
			if _, err := pool.Exec(ctx, `
				INSERT INTO session_results
					(public_id, session_entry_id, classification_state, position, number_of_laps, source_fetched_at)
				VALUES ($1, $2, $3, 1, 0, $4)`, resultID, entryPK, participant.state, fetchedAt); err != nil {
				t.Fatalf("insert ordinary result: %v", err)
			}
		case domain.ClassificationMissing:
			seed.missingEntryID = entryPK
		default:
			resultID := testPublicID(t, domain.EntitySessionResult, entryID.String())
			if _, err := pool.Exec(ctx, `
				INSERT INTO session_results (public_id, session_entry_id, classification_state, source_fetched_at)
				VALUES ($1, $2, $3, $4)`, resultID, entryPK, participant.state, fetchedAt); err != nil {
				t.Fatalf("insert %s result: %v", participant.state, err)
			}
		}
	}
	return seed
}

func newIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv(URLKey))
	if databaseURL == "" {
		t.Skipf("set %s to run PostgreSQL integration tests", URLKey)
	}

	ctx := context.Background()
	baseConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	basePool, err := pgxpool.NewWithConfig(ctx, baseConfig)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	if err := basePool.Ping(ctx); err != nil {
		basePool.Close()
		t.Fatalf("ping integration database: %v", err)
	}

	schema := fmt.Sprintf("task7_%d_%d", os.Getpid(), integrationSchemaSequence.Add(1))
	if _, err := basePool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		basePool.Close()
		t.Fatalf("create integration schema: %v", err)
	}

	var pool *pgxpool.Pool
	t.Cleanup(func() {
		if pool != nil {
			pool.Close()
		}
		if _, err := basePool.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Errorf("drop integration schema: %v", err)
		}
		basePool.Close()
	})

	testConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test pool config: %v", err)
	}
	testConfig.ConnConfig.RuntimeParams["search_path"] = schema
	migrationDB := sql.OpenDB(stdlib.GetConnector(*testConfig.ConnConfig))
	if err := goose.SetDialect("postgres"); err != nil {
		migrationDB.Close()
		t.Fatalf("configure Goose: %v", err)
	}
	migrationDirectory := filepath.Join("..", "..", "migrations")
	if err := goose.UpContext(ctx, migrationDB, migrationDirectory, goose.WithNoColor(true)); err != nil {
		migrationDB.Close()
		t.Fatalf("migrate integration schema: %v", err)
	}
	if err := migrationDB.Close(); err != nil {
		t.Fatalf("close migration database: %v", err)
	}

	pool, err = pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		t.Fatalf("open migrated integration schema: %v", err)
	}
	return pool
}

func testPublicID(t *testing.T, kind domain.EntityKind, key string) domain.PublicID {
	t.Helper()
	id, err := domain.NewPublicID(kind, key)
	if err != nil {
		t.Fatalf("NewPublicID(%q, %q): %v", kind, key, err)
	}
	return id
}

func mustScanID(t *testing.T, row pgx.Row, destination *int64) {
	t.Helper()
	if err := row.Scan(destination); err != nil {
		t.Fatalf("scan inserted ID: %v", err)
	}
}

func assertPostgresErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != code {
		t.Fatalf("error = %v, want PostgreSQL code %s", err, code)
	}
}
