package database

import (
	"context"
	"testing"
	"time"
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
