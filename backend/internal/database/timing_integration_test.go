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
	duration := int64(74_123_456)
	pitOut := true
	compound := "HARD"
	start, end := 1, 78
	fetchedAt := time.Date(2026, time.August, 5, 11, 0, 0, 0, time.UTC)
	snapshot := ingest.TimingSnapshot{
		Target: target, LapsFetchedAt: fetchedAt, StintsFetchedAt: fetchedAt.Add(time.Minute),
		Laps: []domain.Lap{
			{SessionEntryID: entryID, SourceDriverNumber: 16, LapNumber: 1, DurationMicroseconds: nil, IsPitOutLap: &pitOut},
			{SessionEntryID: entryID, SourceDriverNumber: 16, LapNumber: 2, DurationMicroseconds: &duration},
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
