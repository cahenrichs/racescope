package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clint/f1/backend/internal/database"
	"github.com/clint/f1/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var apiSchemaSequence atomic.Uint64

type apiSeed struct {
	raceID       domain.PublicID
	incompleteID domain.PublicID
	driverIDs    []domain.PublicID
}

func TestRaceAPIContracts(t *testing.T) {
	pool := newAPIIntegrationPool(t)
	seed := seedRaceAPI(t, pool)
	router := NewRouter(pool)

	t.Run("dashboard", func(t *testing.T) {
		response := serveRequest(router, "/api/dashboard")
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
		var body dashboardResponse
		decodeResponse(t, response, &body)
		if len(body.Races) != 1 || body.Races[0].ID != seed.raceID.String() {
			t.Fatalf("races = %+v", body.Races)
		}
		if body.Coverage == nil || body.Coverage.Status != "complete" || body.Races[0].Coverage.Status != "complete" {
			t.Fatalf("coverage = %+v, race coverage = %+v", body.Coverage, body.Races[0].Coverage)
		}
	})

	t.Run("race detail", func(t *testing.T) {
		response := serveRequest(router, "/api/races/"+seed.raceID.String())
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
		assertNoPrivateKeys(t, response.Body.String())
		var body raceResponse
		decodeResponse(t, response, &body)
		if body.ID != seed.raceID.String() || body.Circuit.ID == "" || len(body.Sessions) != 2 {
			t.Fatalf("race = %+v", body)
		}
		if body.Sessions[0].Name != "Practice 1" || body.Sessions[1].Name != "Race" {
			t.Fatalf("sessions = %+v", body.Sessions)
		}
	})

	t.Run("race results", func(t *testing.T) {
		response := serveRequest(router, "/api/races/"+seed.raceID.String()+"/results")
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
		assertNoPrivateKeys(t, response.Body.String())
		var body raceResultsResponse
		decodeResponse(t, response, &body)
		if body.RaceID != seed.raceID.String() || len(body.Classification) != 3 {
			t.Fatalf("results = %+v", body)
		}
		for i, driverID := range seed.driverIDs {
			if body.Classification[i].Driver.ID != driverID.String() {
				t.Errorf("classification[%d] driver = %q, want %q", i, body.Classification[i].Driver.ID, driverID)
			}
		}
		if body.Classification[0].Position == nil || *body.Classification[0].Position != 1 ||
			body.Classification[1].Position == nil || *body.Classification[1].Position != 2 ||
			body.Classification[2].Position != nil || body.Classification[2].State != "dnf" {
			t.Fatalf("classification order = %+v", body.Classification)
		}
	})

	for _, path := range []string{"/api/races/meeting_missing", "/api/races/meeting_missing/results"} {
		t.Run("not found "+path, func(t *testing.T) {
			response := serveRequest(router, path)
			assertErrorContract(t, response, http.StatusNotFound, "race_not_found")
		})
	}

	for _, suffix := range []string{"", "/results"} {
		t.Run("incomplete "+suffix, func(t *testing.T) {
			response := serveRequest(router, "/api/races/"+seed.incompleteID.String()+suffix)
			assertErrorContract(t, response, http.StatusConflict, "race_incomplete")
		})
	}
}

func TestDashboardIsEmptyWithoutPublishedRaces(t *testing.T) {
	response := serveRequest(NewRouter(newAPIIntegrationPool(t)), "/api/dashboard")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body dashboardResponse
	decodeResponse(t, response, &body)
	if len(body.Races) != 0 || body.Coverage != nil {
		t.Fatalf("empty dashboard = %+v", body)
	}
}

func serveRequest(router http.Handler, path string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	return response
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q", contentType)
	}
	if err := json.Unmarshal(response.Body.Bytes(), destination); err != nil {
		t.Fatalf("decode response %q: %v", response.Body.String(), err)
	}
}

func assertErrorContract(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body.String())
	}
	var body errorResponse
	decodeResponse(t, response, &body)
	if body.Error.Code != code || body.Error.Message == "" {
		t.Fatalf("error = %+v", body.Error)
	}
}

func assertNoPrivateKeys(t *testing.T, body string) {
	t.Helper()
	for _, privateValue := range []string{`"sourceKey"`, `"meetingKey"`, `"databaseId"`, "1235", "9496", "9500"} {
		if strings.Contains(body, privateValue) {
			t.Errorf("response leaked private value %q: %s", privateValue, body)
		}
	}
}

func seedRaceAPI(t *testing.T, pool *pgxpool.Pool) apiSeed {
	t.Helper()
	ctx := context.Background()
	fetchedAt := time.Date(2024, time.May, 27, 12, 0, 0, 0, time.UTC)
	publishedAt := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	id := func(kind domain.EntityKind, key string) domain.PublicID {
		value, err := domain.NewPublicID(kind, key)
		if err != nil {
			t.Fatalf("NewPublicID(%q): %v", key, err)
		}
		return value
	}
	seasonID := id(domain.EntitySeason, "2024")
	circuitID := id(domain.EntityCircuit, "circuit-de-monaco")
	raceID := id(domain.EntityMeeting, "2024-monaco-grand-prix")
	incompleteID := id(domain.EntityMeeting, "2024-incomplete-grand-prix")
	practiceID := id(domain.EntitySession, "2024-monaco-grand-prix:practice-1")
	raceSessionID := id(domain.EntitySession, "2024-monaco-grand-prix:race")
	incompleteSessionID := id(domain.EntitySession, "2024-incomplete-grand-prix:race")
	constructorID := id(domain.EntityConstructorEntrant, "2024-ferrari")

	var seasonPK, circuitPK, racePK, incompletePK, raceSessionPK, constructorPK int64
	mustScanAPIID(t, pool.QueryRow(ctx, `INSERT INTO seasons (public_id, year, source_fetched_at) VALUES ($1, 2024, $2) RETURNING id`, seasonID, fetchedAt), &seasonPK)
	mustScanAPIID(t, pool.QueryRow(ctx, `
		INSERT INTO circuits (public_id, source_key, short_name, country_code, country_name, location, source_fetched_at)
		VALUES ($1, 22, 'Monte Carlo', 'MON', 'Monaco', 'Monte Carlo', $2) RETURNING id`, circuitID, fetchedAt), &circuitPK)
	mustScanAPIID(t, pool.QueryRow(ctx, `
		INSERT INTO meetings (public_id, source_key, season_id, circuit_id, name, official_name, date_start, date_end,
			is_cancelled, source_fetched_at, published_at)
		VALUES ($1, 1235, $2, $3, 'Monaco Grand Prix', 'FORMULA 1 GRAND PRIX DE MONACO 2024', $4, $5, false, $6, $7)
		RETURNING id`, raceID, seasonPK, circuitPK, time.Date(2024, 5, 24, 11, 30, 0, 0, time.UTC), time.Date(2024, 5, 26, 15, 0, 0, 0, time.UTC), fetchedAt, publishedAt), &racePK)
	mustScanAPIID(t, pool.QueryRow(ctx, `
		INSERT INTO meetings (public_id, source_key, season_id, circuit_id, name, official_name, date_start, date_end,
			is_cancelled, source_fetched_at, published_at)
		VALUES ($1, 1236, $2, $3, 'Incomplete Grand Prix', 'INCOMPLETE GRAND PRIX', $4, $5, false, $6, $7)
		RETURNING id`, incompleteID, seasonPK, circuitPK, time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC), time.Date(2024, 5, 2, 10, 0, 0, 0, time.UTC), fetchedAt, publishedAt), &incompletePK)
	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (public_id, source_key, meeting_id, name, type, date_start, date_end, is_cancelled, source_fetched_at)
		VALUES ($1, 9496, $2, 'Practice 1', 'Practice', $3, $4, false, $5)`, practiceID, racePK, time.Date(2024, 5, 24, 11, 30, 0, 0, time.UTC), time.Date(2024, 5, 24, 12, 30, 0, 0, time.UTC), fetchedAt); err != nil {
		t.Fatalf("insert practice session: %v", err)
	}
	mustScanAPIID(t, pool.QueryRow(ctx, `
		INSERT INTO sessions (public_id, source_key, meeting_id, name, type, date_start, date_end, is_cancelled, source_fetched_at)
		VALUES ($1, 9500, $2, 'Race', 'Race', $3, $4, false, $5) RETURNING id`, raceSessionID, racePK, time.Date(2024, 5, 26, 13, 0, 0, 0, time.UTC), time.Date(2024, 5, 26, 15, 0, 0, 0, time.UTC), fetchedAt), &raceSessionPK)
	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (public_id, source_key, meeting_id, name, type, date_start, date_end, is_cancelled, source_fetched_at)
		VALUES ($1, 9501, $2, 'Race', 'Race', $3, $4, false, $5)`, incompleteSessionID, incompletePK, time.Date(2024, 5, 2, 8, 0, 0, 0, time.UTC), time.Date(2024, 5, 2, 10, 0, 0, 0, time.UTC), fetchedAt); err != nil {
		t.Fatalf("insert incomplete race session: %v", err)
	}
	mustScanAPIID(t, pool.QueryRow(ctx, `
		INSERT INTO constructor_entrants (public_id, season_id, name, source_fetched_at)
		VALUES ($1, $2, 'Ferrari', $3) RETURNING id`, constructorID, seasonPK, fetchedAt), &constructorPK)

	participants := []struct {
		key, name, acronym, state string
		number, sourceOrder       int
		position                  *int
	}{
		{key: "charles-leclerc", name: "Charles Leclerc", acronym: "LEC", state: "ordinary", number: 16, position: apiIntPointer(1), sourceOrder: 7},
		{key: "oscar-piastri", name: "Oscar Piastri", acronym: "PIA", state: "ordinary", number: 81, position: apiIntPointer(2), sourceOrder: 2},
		{key: "first-retirement", name: "First Retirement", acronym: "RET", state: "dnf", number: 99, sourceOrder: 3},
	}
	seed := apiSeed{raceID: raceID, incompleteID: incompleteID, driverIDs: make([]domain.PublicID, 0, len(participants))}
	for _, participant := range participants {
		driverID := id(domain.EntityDriver, participant.key)
		entryID := id(domain.EntitySessionEntry, "2024-monaco-grand-prix:race:"+participant.key)
		resultID := id(domain.EntitySessionResult, "2024-monaco-grand-prix:race:"+participant.key)
		seed.driverIDs = append(seed.driverIDs, driverID)
		var driverPK, entryPK int64
		mustScanAPIID(t, pool.QueryRow(ctx, `
			INSERT INTO drivers (public_id, first_name, last_name, full_name, name_acronym, source_fetched_at)
			VALUES ($1, $2, $2, $2, $3, $4) RETURNING id`, driverID, participant.name, participant.acronym, fetchedAt), &driverPK)
		mustScanAPIID(t, pool.QueryRow(ctx, `
			INSERT INTO session_entries (public_id, session_id, driver_id, constructor_entrant_id, driver_number, team_colour, source_fetched_at)
			VALUES ($1, $2, $3, $4, $5, 'E80020', $6) RETURNING id`, entryID, raceSessionPK, driverPK, constructorPK, participant.number, fetchedAt), &entryPK)
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_results (public_id, session_entry_id, classification_state, position, number_of_laps,
				duration_kind, gap_to_leader_kind, source_order, source_fetched_at)
			VALUES ($1, $2, $3, $4, 78, 'missing', 'missing', $5, $6)`, resultID, entryPK, participant.state, participant.position, participant.sourceOrder, fetchedAt); err != nil {
			t.Fatalf("insert result for %s: %v", participant.name, err)
		}
	}
	return seed
}

func newAPIIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv(database.URLKey))
	if databaseURL == "" {
		t.Skipf("set %s to run PostgreSQL API contract tests", database.URLKey)
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
	schema := fmt.Sprintf("task9_%d_%d", os.Getpid(), apiSchemaSequence.Add(1))
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
	if err := goose.UpContext(ctx, migrationDB, filepath.Join("..", "..", "migrations"), goose.WithNoColor(true)); err != nil {
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

func mustScanAPIID(t *testing.T, row pgx.Row, destination *int64) {
	t.Helper()
	if err := row.Scan(destination); err != nil {
		t.Fatalf("scan inserted ID: %v", err)
	}
}

func apiIntPointer(value int) *int {
	return &value
}
