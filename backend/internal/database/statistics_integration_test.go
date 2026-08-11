package database

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/clint/f1/backend/internal/ingest"
)

func TestStatisticsLapComparisonReturnsAllSourceLapsIncludingNullDurations(t *testing.T) {
	pool := newIntegrationPool(t)
	ctx := context.Background()
	snapshot := publicationSnapshot(t)
	race := snapshot.Weekend.Sessions[1]
	piastri := domain.Driver{
		PublicID: testPublicID(t, domain.EntityDriver, "oscar-piastri"), StableKey: "oscar-piastri",
		FirstName: "Oscar", LastName: "Piastri", FullName: "Oscar Piastri", NameAcronym: "PIA",
	}
	piastriEntry := domain.SessionEntry{
		PublicID: testPublicID(t, domain.EntitySessionEntry, race.StableKey+":"+piastri.StableKey), StableKey: race.StableKey + ":" + piastri.StableKey,
		SessionID: race.PublicID, Driver: piastri, Constructor: snapshot.Constructors[0], DriverNumber: 81, TeamColour: "FF8700",
	}
	position, laps := 2, 78
	piastriResult := domain.SessionResult{
		PublicID: testPublicID(t, domain.EntitySessionResult, piastriEntry.StableKey), StableKey: piastriEntry.StableKey,
		SessionEntryID: piastriEntry.PublicID, Classification: domain.ClassificationOrdinary, Position: &position, NumberOfLaps: &laps,
		Duration: domain.ResultValue{Kind: domain.ResultValueNull}, GapToLeader: domain.ResultValue{Kind: domain.ResultValueNull}, SourceOrder: 8,
	}
	snapshot.Drivers = append(snapshot.Drivers, piastri)
	snapshot.Entries = append(snapshot.Entries, piastriEntry)
	snapshot.Results = append(snapshot.Results, piastriResult)
	if _, err := NewWeekendPublisher(pool).ReplaceWeekend(ctx, snapshot); err != nil {
		t.Fatalf("seed weekend: %v", err)
	}

	timing := NewTimingStore(pool)
	timing.now = func() time.Time { return time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC) }
	target, err := timing.EligibleTimingTarget(ctx, ingest.Target{Season: 2024, MeetingKey: 1236})
	if err != nil {
		t.Fatalf("EligibleTimingTarget() error = %v", err)
	}
	duration := int64(74_123_456)
	pitOut, notPitOut := true, false
	hard, soft := "HARD", "SOFT"
	start, end := 1, 2
	one := 1
	fetchedAt := time.Date(2026, time.August, 5, 11, 0, 0, 0, time.UTC)
	_, err = timing.ReplaceTiming(ctx, ingest.TimingSnapshot{
		Target: target, LapsFetchedAt: fetchedAt, StintsFetchedAt: fetchedAt,
		Laps: []domain.Lap{
			{SessionEntryID: target.EntryIDsByNumber[16], SourceDriverNumber: 16, LapNumber: 1, IsPitOutLap: &pitOut, IsStintStart: true},
			{SessionEntryID: target.EntryIDsByNumber[16], SourceDriverNumber: 16, LapNumber: 2, DurationMicroseconds: &duration, IsPitOutLap: &notPitOut, IsStintEnd: true},
			{SessionEntryID: target.EntryIDsByNumber[81], SourceDriverNumber: 81, LapNumber: 1, DurationMicroseconds: &duration, IsPitOutLap: &pitOut, IsStintStart: true, IsStintEnd: true},
		},
		Stints: []domain.Stint{
			{SessionEntryID: target.EntryIDsByNumber[16], SourceDriverNumber: 16, StintNumber: 1, Compound: &hard, LapStart: &start, LapEnd: &end},
			{SessionEntryID: target.EntryIDsByNumber[81], SourceDriverNumber: 81, StintNumber: 1, Compound: &soft, LapStart: &one, LapEnd: &one},
		},
	})
	if err != nil {
		t.Fatalf("ReplaceTiming() error = %v", err)
	}

	driverIDs := []string{snapshot.Drivers[0].PublicID.String(), piastri.PublicID.String()}
	sort.Strings(driverIDs)
	comparison, err := NewStatisticsStore(pool).LapComparison(ctx, race.PublicID.String(), driverIDs)
	if err != nil {
		t.Fatalf("LapComparison() error = %v", err)
	}
	if len(comparison.Drivers) != 2 {
		t.Fatalf("drivers = %+v", comparison.Drivers)
	}
	observations := make(map[string][]domain.Lap, len(comparison.Drivers))
	for _, driver := range comparison.Drivers {
		for _, lap := range driver.Observations {
			observations[driver.ID] = append(observations[driver.ID], domain.Lap{
				LapNumber: lap.LapNumber, DurationMicroseconds: lap.DurationMicroseconds, IsPitOutLap: lap.IsPitOutLap,
				IsStintStart: lap.IsStintStart, IsStintEnd: lap.IsStintEnd,
			})
		}
	}
	leclercLaps := observations[snapshot.Drivers[0].PublicID.String()]
	if len(leclercLaps) != 2 || leclercLaps[0].DurationMicroseconds != nil || leclercLaps[1].DurationMicroseconds == nil || *leclercLaps[1].DurationMicroseconds != duration {
		t.Fatalf("Leclerc lap durations = %v", leclercLaps)
	}
	if !leclercLaps[0].IsStintStart || leclercLaps[0].IsStintEnd || leclercLaps[0].IsPitOutLap == nil || !*leclercLaps[0].IsPitOutLap {
		t.Fatalf("Leclerc lap 1 context = %+v", leclercLaps[0])
	}
	for _, driver := range comparison.Drivers {
		if driver.ID != snapshot.Drivers[0].PublicID.String() {
			continue
		}
		if driver.Observations[0].Compound == nil || *driver.Observations[0].Compound != hard || driver.Observations[0].StintNumber == nil || *driver.Observations[0].StintNumber != 1 {
			t.Fatalf("Leclerc stint context = %+v", driver.Observations[0])
		}
	}
}
