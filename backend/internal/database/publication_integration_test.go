package database

import (
	"context"
	"testing"
	"time"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/clint/f1/backend/internal/ingest"
)

func TestReplaceWeekendIsIdempotentAndRemovesDisappearedRows(t *testing.T) {
	pool := newIntegrationPool(t)
	publisher := NewWeekendPublisher(pool)
	ctx := context.Background()
	first := publicationSnapshot(t)
	publisher.now = func() time.Time { return time.Date(2026, time.July, 30, 10, 0, 0, 0, time.UTC) }

	if _, err := publisher.ReplaceWeekend(ctx, first); err != nil {
		t.Fatalf("first ReplaceWeekend() error = %v", err)
	}
	if _, err := publisher.ReplaceWeekend(ctx, first); err != nil {
		t.Fatalf("repeated ReplaceWeekend() error = %v", err)
	}
	assertTableCount(t, pool, "meetings", 1)
	assertTableCount(t, pool, "sessions", 2)
	assertTableCount(t, pool, "session_entries", 1)
	assertTableCount(t, pool, "session_results", 1)
	var meetingFetchedAt, sessionFetchedAt, driverFetchedAt, resultFetchedAt, storedPublishedAt time.Time
	err := pool.QueryRow(ctx, `
		SELECT m.source_fetched_at, s.source_fetched_at, d.source_fetched_at, r.source_fetched_at, m.published_at
		FROM meetings m
		JOIN sessions s ON s.meeting_id = m.id AND s.name = 'Race'
		JOIN session_entries e ON e.session_id = s.id
		JOIN drivers d ON d.id = e.driver_id
		JOIN session_results r ON r.session_entry_id = e.id`).Scan(
		&meetingFetchedAt, &sessionFetchedAt, &driverFetchedAt, &resultFetchedAt, &storedPublishedAt)
	if err != nil {
		t.Fatalf("query source and publication timestamps: %v", err)
	}
	if !meetingFetchedAt.Equal(first.SourceFetchedAt.Meetings) || !sessionFetchedAt.Equal(first.SourceFetchedAt.Sessions) ||
		!driverFetchedAt.Equal(first.SourceFetchedAt.Drivers) || !resultFetchedAt.Equal(first.SourceFetchedAt.Results) ||
		!storedPublishedAt.Equal(publisher.now()) {
		t.Fatalf("stored timestamps = %s/%s/%s/%s published %s", meetingFetchedAt, sessionFetchedAt, driverFetchedAt, resultFetchedAt, storedPublishedAt)
	}

	replacement := first
	replacement.Weekend.Sessions = replacement.Weekend.Sessions[1:]
	delete(replacement.SessionSourceKeys, first.Weekend.Sessions[0].PublicID)
	if _, err := publisher.ReplaceWeekend(ctx, replacement); err != nil {
		t.Fatalf("replacement ReplaceWeekend() error = %v", err)
	}
	assertTableCount(t, pool, "sessions", 1)

	var remainingName, durationKind, gapKind string
	var sourceOrder int
	err = pool.QueryRow(ctx, `
		SELECT s.name, r.duration_kind, r.gap_to_leader_kind, r.source_order
		FROM sessions s
		JOIN session_entries e ON e.session_id = s.id
		JOIN session_results r ON r.session_entry_id = e.id`).Scan(&remainingName, &durationKind, &gapKind, &sourceOrder)
	if err != nil {
		t.Fatalf("query replacement result: %v", err)
	}
	if remainingName != "Race" || durationKind != "null" || gapKind != "number" || sourceOrder != 7 {
		t.Fatalf("replacement values = %q/%q/%q/%d", remainingName, durationKind, gapKind, sourceOrder)
	}
}

func TestInterruptedReplacementLeavesPublishedWeekendReadable(t *testing.T) {
	pool := newIntegrationPool(t)
	publisher := NewWeekendPublisher(pool)
	ctx := context.Background()
	snapshot := publicationSnapshot(t)
	if _, err := publisher.ReplaceWeekend(ctx, snapshot); err != nil {
		t.Fatalf("initial ReplaceWeekend() error = %v", err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := publisher.ReplaceWeekend(cancelled, snapshot); err == nil {
		t.Fatal("interrupted ReplaceWeekend() error = nil")
	}
	weekend, err := WeekendByPublicID(ctx, pool, snapshot.Weekend.Meeting.PublicID)
	if err != nil || weekend.Name != snapshot.Weekend.Meeting.Name {
		t.Fatalf("published weekend after interruption = %+v, error %v", weekend, err)
	}
}

func TestFailedReplacementRollsBackAndPreservesAuditRows(t *testing.T) {
	pool := newIntegrationPool(t)
	publisher := NewWeekendPublisher(pool)
	ctx := context.Background()
	snapshot := publicationSnapshot(t)
	publishedAt := time.Date(2026, time.July, 30, 10, 0, 0, 0, time.UTC)
	publisher.now = func() time.Time { return publishedAt }
	if _, err := publisher.ReplaceWeekend(ctx, snapshot); err != nil {
		t.Fatalf("initial ReplaceWeekend() error = %v", err)
	}

	runID, err := CreateImportRun(ctx, pool, 2024, 1235, publishedAt.Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateImportRun() error = %v", err)
	}
	if err := RecordImportRunRequest(ctx, pool, runID, ImportRunRequest{
		Endpoint: "sessions", Parameters: map[string][]string{"meeting_key": {"1235"}}, ResponseStatus: 200,
		FetchedAt: publishedAt.Add(time.Hour), RecordCount: 2,
		ResponseSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}); err != nil {
		t.Fatalf("RecordImportRunRequest() error = %v", err)
	}
	if err := RecordImportRunError(ctx, pool, runID, ImportRunError{
		Order: 0, Code: "unknown_driver", Entity: "driver",
		SourceContext: map[string]any{"full_name": "Unknown DRIVER", "driver_number": 99},
		Message:       "full name has no reviewed mapping",
	}); err != nil {
		t.Fatalf("RecordImportRunError() error = %v", err)
	}
	if err := FinishImportRun(ctx, pool, runID, ImportRunCompletion{
		Status: "quarantined", FinishedAt: publishedAt.Add(90 * time.Minute), SessionCount: 2, ErrorCount: 1,
	}); err != nil {
		t.Fatalf("FinishImportRun() error = %v", err)
	}

	broken := snapshot
	broken.Weekend.Meeting.Name = "Uncommitted replacement"
	broken.Weekend.Sessions = append([]domain.Session(nil), broken.Weekend.Sessions...)
	duplicate := broken.Weekend.Sessions[0]
	duplicate.PublicID = testPublicID(t, domain.EntitySession, "2024-monaco-grand-prix:duplicate")
	broken.Weekend.Sessions = append(broken.Weekend.Sessions, duplicate)
	broken.SessionSourceKeys = cloneSourceKeys(broken.SessionSourceKeys)
	broken.SessionSourceKeys[duplicate.PublicID] = broken.SessionSourceKeys[broken.Weekend.Sessions[0].PublicID]
	if _, err := publisher.ReplaceWeekend(ctx, broken); err == nil {
		t.Fatal("broken ReplaceWeekend() error = nil")
	}

	weekend, err := WeekendByPublicID(ctx, pool, snapshot.Weekend.Meeting.PublicID)
	if err != nil {
		t.Fatalf("WeekendByPublicID() after rollback error = %v", err)
	}
	if weekend.Name != snapshot.Weekend.Meeting.Name {
		t.Fatalf("weekend name after rollback = %q", weekend.Name)
	}
	assertTableCount(t, pool, "sessions", 2)
	assertTableCount(t, pool, "import_runs", 1)
	assertTableCount(t, pool, "import_run_requests", 1)
	assertTableCount(t, pool, "import_run_errors", 1)
}

func publicationSnapshot(t *testing.T) ingest.Snapshot {
	t.Helper()
	season := domain.Season{PublicID: testPublicID(t, domain.EntitySeason, "2024"), Year: 2024}
	circuit := domain.Circuit{
		PublicID: testPublicID(t, domain.EntityCircuit, "circuit-de-monaco"), StableKey: "circuit-de-monaco",
		ShortName: "Monte Carlo", CountryCode: "MON", CountryName: "Monaco", Location: "Monte Carlo",
	}
	meeting := domain.Meeting{
		PublicID: testPublicID(t, domain.EntityMeeting, "2024-monaco-grand-prix"), StableKey: "2024-monaco-grand-prix",
		Season: season, Circuit: circuit, Name: "Monaco Grand Prix", OfficialName: "FORMULA 1 GRAND PRIX DE MONACO 2024",
		DateStart: time.Date(2024, time.May, 24, 11, 30, 0, 0, time.UTC), DateEnd: time.Date(2024, time.May, 26, 15, 0, 0, 0, time.UTC),
	}
	practice := domain.Session{
		PublicID: testPublicID(t, domain.EntitySession, "2024-monaco-grand-prix:practice-1"), StableKey: "2024-monaco-grand-prix:practice-1",
		Name: "Practice 1", Type: "Practice", DateStart: meeting.DateStart, DateEnd: meeting.DateStart.Add(time.Hour),
	}
	race := domain.Session{
		PublicID: testPublicID(t, domain.EntitySession, "2024-monaco-grand-prix:race"), StableKey: "2024-monaco-grand-prix:race",
		Name: "Race", Type: "Race", DateStart: time.Date(2024, time.May, 26, 13, 0, 0, 0, time.UTC), DateEnd: meeting.DateEnd,
	}
	driver := domain.Driver{
		PublicID: testPublicID(t, domain.EntityDriver, "charles-leclerc"), StableKey: "charles-leclerc",
		FirstName: "Charles", LastName: "Leclerc", FullName: "Charles Leclerc", NameAcronym: "LEC",
	}
	constructor := domain.ConstructorEntrant{
		PublicID: testPublicID(t, domain.EntityConstructorEntrant, "2024-ferrari"), StableKey: "2024-ferrari", Season: season, Name: "Ferrari",
	}
	entry := domain.SessionEntry{
		PublicID:  testPublicID(t, domain.EntitySessionEntry, race.StableKey+":"+driver.StableKey),
		StableKey: race.StableKey + ":" + driver.StableKey, SessionID: race.PublicID,
		Driver: driver, Constructor: constructor, DriverNumber: 16, TeamColour: "E80020",
	}
	position, laps, zero := 1, 78, 0.0
	result := domain.SessionResult{
		PublicID: testPublicID(t, domain.EntitySessionResult, entry.StableKey), StableKey: entry.StableKey,
		SessionEntryID: entry.PublicID, Classification: domain.ClassificationOrdinary, Position: &position, NumberOfLaps: &laps,
		Duration:    domain.ResultValue{Kind: domain.ResultValueNull},
		GapToLeader: domain.ResultValue{Kind: domain.ResultValueNumber, Number: zero}, SourceOrder: 7,
	}
	return ingest.Snapshot{
		Weekend:          domain.Weekend{Meeting: meeting, Sessions: []domain.Session{practice, race}, GrandPrixSessionID: race.PublicID},
		MeetingSourceKey: 1235, CircuitSourceKey: 22,
		SessionSourceKeys: map[domain.PublicID]int{practice.PublicID: 9496, race.PublicID: 9500},
		Drivers:           []domain.Driver{driver}, Constructors: []domain.ConstructorEntrant{constructor},
		Entries: []domain.SessionEntry{entry}, Results: []domain.SessionResult{result},
		SourceFetchedAt: ingest.SourceFetchTimes{
			Meetings: meeting.DateEnd.Add(time.Hour), Sessions: meeting.DateEnd.Add(2 * time.Hour),
			Drivers: meeting.DateEnd.Add(3 * time.Hour), Results: meeting.DateEnd.Add(4 * time.Hour),
		},
	}
}

func cloneSourceKeys(source map[domain.PublicID]int) map[domain.PublicID]int {
	cloned := make(map[domain.PublicID]int, len(source)+1)
	for id, key := range source {
		cloned[id] = key
	}
	return cloned
}

func assertTableCount(t *testing.T, pool Querier, table string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}
