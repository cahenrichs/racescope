package database

import (
	"context"
	"testing"
	"time"

	"github.com/clint/f1/backend/internal/domain"
)

func TestPublishedRaceQueries(t *testing.T) {
	pool := newIntegrationPool(t)
	ctx := context.Background()
	snapshot := publicationSnapshot(t)
	publishedAt := time.Date(2026, time.July, 30, 15, 0, 0, 0, time.UTC)
	publisher := NewWeekendPublisher(pool)
	publisher.now = func() time.Time { return publishedAt }
	if _, err := publisher.ReplaceWeekend(ctx, snapshot); err != nil {
		t.Fatalf("ReplaceWeekend() error = %v", err)
	}

	races, err := PublishedRaces(ctx, pool)
	if err != nil {
		t.Fatalf("PublishedRaces() error = %v", err)
	}
	if len(races) != 1 || races[0].PublicID != snapshot.Weekend.Meeting.PublicID {
		t.Fatalf("PublishedRaces() = %+v", races)
	}
	if !races[0].Coverage.SourceFetchedAt.Equal(snapshot.SourceFetchedAt.Results) ||
		!races[0].Coverage.PublishedAt.Equal(publishedAt) {
		t.Fatalf("dashboard coverage = %+v", races[0].Coverage)
	}

	race, err := RaceByPublicID(ctx, pool, snapshot.Weekend.Meeting.PublicID)
	if err != nil {
		t.Fatalf("RaceByPublicID() error = %v", err)
	}
	if len(race.Sessions) != 2 || race.Sessions[0].Name != "Practice 1" || race.Sessions[1].Name != "Race" {
		t.Fatalf("RaceByPublicID() sessions = %+v", race.Sessions)
	}

	results, err := ResultsByRacePublicID(ctx, pool, snapshot.Weekend.Meeting.PublicID)
	if err != nil {
		t.Fatalf("ResultsByRacePublicID() error = %v", err)
	}
	if results.SessionPublicID != snapshot.Weekend.GrandPrixSessionID || len(results.Classification) != 1 {
		t.Fatalf("ResultsByRacePublicID() = %+v", results)
	}
	row := results.Classification[0]
	if row.DriverName != "Charles Leclerc" || row.ConstructorName != "Ferrari" ||
		row.Duration.Kind != "null" || row.GapToLeader.Number == nil || *row.GapToLeader.Number != 0 {
		t.Fatalf("classification row = %+v", row)
	}
}

func TestRaceResultsOrdersNumericPositionsBeforeNonnumericProviderOrder(t *testing.T) {
	pool := newIntegrationPool(t)
	ctx := context.Background()
	snapshot := publicationSnapshot(t)
	if _, err := NewWeekendPublisher(pool).ReplaceWeekend(ctx, snapshot); err != nil {
		t.Fatalf("ReplaceWeekend() error = %v", err)
	}

	var sessionPK, constructorPK int64
	if err := pool.QueryRow(ctx, `SELECT id FROM sessions WHERE public_id = $1`, snapshot.Weekend.GrandPrixSessionID).Scan(&sessionPK); err != nil {
		t.Fatalf("query race session: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM constructor_entrants WHERE public_id = $1`, snapshot.Constructors[0].PublicID).Scan(&constructorPK); err != nil {
		t.Fatalf("query constructor: %v", err)
	}

	rows := []struct {
		key         string
		name        string
		number      int
		state       string
		position    *int
		sourceOrder int
	}{
		{key: "second-driver", name: "Second Driver", number: 2, state: "ordinary", position: intPointer(2), sourceOrder: 0},
		{key: "first-nonnumeric", name: "First Nonnumeric", number: 3, state: "dnf", sourceOrder: 3},
		{key: "second-nonnumeric", name: "Second Nonnumeric", number: 4, state: "dns", sourceOrder: 9},
	}
	for _, seed := range rows {
		driverID := testPublicID(t, domain.EntityDriver, seed.key)
		entryID := testPublicID(t, domain.EntitySessionEntry, "race:"+seed.key)
		resultID := testPublicID(t, domain.EntitySessionResult, "race:"+seed.key)
		var driverPK, entryPK int64
		if err := pool.QueryRow(ctx, `
			INSERT INTO drivers (public_id, first_name, last_name, full_name, name_acronym, source_fetched_at)
			VALUES ($1, $2, 'Driver', $2, 'TST', $3) RETURNING id`, driverID, seed.name, snapshot.SourceFetchedAt.Drivers).Scan(&driverPK); err != nil {
			t.Fatalf("insert %s: %v", seed.name, err)
		}
		if err := pool.QueryRow(ctx, `
			INSERT INTO session_entries (public_id, session_id, driver_id, constructor_entrant_id, driver_number, team_colour, source_fetched_at)
			VALUES ($1, $2, $3, $4, $5, '000000', $6) RETURNING id`, entryID, sessionPK, driverPK, constructorPK, seed.number, snapshot.SourceFetchedAt.Drivers).Scan(&entryPK); err != nil {
			t.Fatalf("insert entry for %s: %v", seed.name, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_results (public_id, session_entry_id, classification_state, position, duration_kind,
				gap_to_leader_kind, source_order, source_fetched_at)
			VALUES ($1, $2, $3, $4, 'missing', 'missing', $5, $6)`, resultID, entryPK, seed.state, seed.position, seed.sourceOrder, snapshot.SourceFetchedAt.Results); err != nil {
			t.Fatalf("insert result for %s: %v", seed.name, err)
		}
	}

	results, err := ResultsByRacePublicID(ctx, pool, snapshot.Weekend.Meeting.PublicID)
	if err != nil {
		t.Fatalf("ResultsByRacePublicID() error = %v", err)
	}
	want := []string{"Charles Leclerc", "Second Driver", "First Nonnumeric", "Second Nonnumeric"}
	if len(results.Classification) != len(want) {
		t.Fatalf("classification length = %d, want %d", len(results.Classification), len(want))
	}
	for i, name := range want {
		if results.Classification[i].DriverName != name {
			t.Errorf("classification[%d] = %q, want %q", i, results.Classification[i].DriverName, name)
		}
	}
}

func intPointer(value int) *int {
	return &value
}
