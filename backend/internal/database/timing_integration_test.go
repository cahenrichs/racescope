package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/clint/f1/backend/internal/ingest"
)

func TestTimingPublicationIsExactIdempotentAndSurvivesWeekendRepublication(t *testing.T) {
	pool := newIntegrationPool(t)
	ctx := context.Background()
	weekendPublisher := NewWeekendPublisher(pool)
	weekend := publicationSnapshot(t)
	if _, err := weekendPublisher.ReplaceWeekend(ctx, weekend); err != nil {
		t.Fatalf("seed ReplaceWeekend() error = %v", err)
	}

	store := NewTimingStore(pool)
	store.now = func() time.Time { return time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC) }
	target, err := store.EligibleTimingTarget(ctx, ingest.Target{Season: 2024, MeetingKey: 1236})
	if err != nil {
		t.Fatalf("EligibleTimingTarget() error = %v", err)
	}
	entryID := target.EntryIDsByNumber[16]
	var originalSessionID, originalEntryID int64
	if err := pool.QueryRow(ctx, `SELECT s.id, e.id FROM sessions s JOIN session_entries e ON e.session_id = s.id WHERE s.public_id = $1 AND e.public_id = $2`, target.SessionID, entryID).Scan(&originalSessionID, &originalEntryID); err != nil {
		t.Fatalf("query original timing identities: %v", err)
	}
	duration := int64(74_123_456)
	pitOut := true
	compound := "HARD"
	start, end := 1, 78
	fetchedAt := time.Date(2026, time.August, 5, 11, 0, 0, 0, time.UTC)
	snapshot := ingest.TimingSnapshot{
		Target: target, LapsFetchedAt: fetchedAt, StintsFetchedAt: fetchedAt.Add(time.Minute),
		Laps: []domain.Lap{
			{SessionEntryID: entryID, SourceDriverNumber: 16, LapNumber: 1, DurationMicroseconds: nil, IsPitOutLap: &pitOut, IsStintStart: true},
			{SessionEntryID: entryID, SourceDriverNumber: 16, LapNumber: 2, DurationMicroseconds: &duration, IsStintEnd: true},
		},
		Stints: []domain.Stint{{SessionEntryID: entryID, SourceDriverNumber: 16, StintNumber: 1, Compound: &compound, LapStart: &start, LapEnd: &end}},
	}
	if _, err := store.ReplaceTiming(ctx, snapshot); err != nil {
		t.Fatalf("ReplaceTiming() error = %v", err)
	}
	if _, err := store.ReplaceTiming(ctx, snapshot); err != nil {
		t.Fatalf("repeated ReplaceTiming() error = %v", err)
	}
	assertTableCount(t, pool, "grand_prix_laps", 2)
	assertTableCount(t, pool, "grand_prix_stints", 1)
	assertTableCount(t, pool, "session_timing_publications", 1)
	var nullDuration, exactDuration *int64
	if err := pool.QueryRow(ctx, `SELECT
		MAX(duration_microseconds) FILTER (WHERE lap_number = 1),
		MAX(duration_microseconds) FILTER (WHERE lap_number = 2)
		FROM grand_prix_laps`).Scan(&nullDuration, &exactDuration); err != nil {
		t.Fatalf("query stored durations: %v", err)
	}
	if nullDuration != nil || exactDuration == nil || *exactDuration != duration {
		t.Fatalf("stored durations = %v/%v", nullDuration, exactDuration)
	}
	var storedPitOut *bool
	var storedStart, storedEnd bool
	if err := pool.QueryRow(ctx, `SELECT is_pit_out_lap, is_stint_start, is_stint_end FROM grand_prix_laps WHERE lap_number = 1`).Scan(&storedPitOut, &storedStart, &storedEnd); err != nil {
		t.Fatalf("query stored markers: %v", err)
	}
	if storedPitOut == nil || !*storedPitOut || !storedStart || storedEnd {
		t.Fatalf("stored lap 1 markers = pit-out %v, start %t, end %t", storedPitOut, storedStart, storedEnd)
	}

	broken := snapshot
	broken.Laps = append(append([]domain.Lap(nil), snapshot.Laps...), snapshot.Laps[1])
	if _, err := store.ReplaceTiming(ctx, broken); err == nil {
		t.Fatal("broken ReplaceTiming() error = nil")
	}
	assertTableCount(t, pool, "grand_prix_laps", 2)

	if _, err := weekendPublisher.ReplaceWeekend(ctx, weekend); err != nil {
		t.Fatalf("metadata republication error = %v", err)
	}
	assertTableCount(t, pool, "grand_prix_laps", 2)
	assertTableCount(t, pool, "session_timing_publications", 1)
	var republishedSessionID, republishedEntryID int64
	if err := pool.QueryRow(ctx, `SELECT s.id, e.id FROM sessions s JOIN session_entries e ON e.session_id = s.id WHERE s.public_id = $1 AND e.public_id = $2`, target.SessionID, entryID).Scan(&republishedSessionID, &republishedEntryID); err != nil {
		t.Fatalf("query republished timing identities: %v", err)
	}
	if republishedSessionID != originalSessionID || republishedEntryID != originalEntryID {
		t.Fatalf("republication changed stable identities from %d/%d to %d/%d", originalSessionID, originalEntryID, republishedSessionID, republishedEntryID)
	}

	changedIdentity := weekend
	changedIdentity.SessionSourceKeys = cloneSourceKeys(weekend.SessionSourceKeys)
	changedIdentity.SessionSourceKeys[target.SessionID] = target.SessionKey + 1
	if _, err := weekendPublisher.ReplaceWeekend(ctx, changedIdentity); err != nil {
		t.Fatalf("changed-identity republication error = %v", err)
	}
	assertTableCount(t, pool, "grand_prix_laps", 0)
	assertTableCount(t, pool, "grand_prix_stints", 0)
	assertTableCount(t, pool, "session_timing_publications", 0)
}

func TestTimingEligibilityDefersUnpublishedAndIncompleteWeekends(t *testing.T) {
	pool := newIntegrationPool(t)
	ctx := context.Background()
	weekend := publicationSnapshot(t)
	if _, err := NewWeekendPublisher(pool).ReplaceWeekend(ctx, weekend); err != nil {
		t.Fatalf("seed ReplaceWeekend() error = %v", err)
	}
	store := NewTimingStore(pool)
	store.now = func() time.Time { return time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC) }

	if _, err := pool.Exec(ctx, `UPDATE meetings SET published_at = NULL`); err != nil {
		t.Fatal(err)
	}
	_, err := store.EligibleTimingTarget(ctx, ingest.Target{Season: 2024, MeetingKey: 1236})
	var deferred *ingest.DeferredTimingError
	if !errors.As(err, &deferred) {
		t.Fatalf("unpublished eligibility error = %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE meetings SET published_at = CURRENT_TIMESTAMP; DELETE FROM session_results`); err != nil {
		t.Fatal(err)
	}
	_, err = store.EligibleTimingTarget(ctx, ingest.Target{Season: 2024, MeetingKey: 1236})
	if !errors.As(err, &deferred) {
		t.Fatalf("incomplete eligibility error = %v", err)
	}
}
