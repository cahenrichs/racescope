package ingest

import (
	"testing"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/clint/f1/backend/internal/openf1"
)

func TestTransformTimingRetainsNullDurationAndSourceContext(t *testing.T) {
	t.Parallel()
	entryID, err := domain.NewPublicID(domain.EntitySessionEntry, "monaco:race:leclerc")
	if err != nil {
		t.Fatal(err)
	}
	target := TimingTarget{MeetingKey: 1236, SessionKey: 9500, EntryIDsByNumber: map[int]domain.PublicID{16: entryID}}
	pitOut := true
	compound := "HARD"
	firstStart, firstEnd, finalStart, finalEnd := 1, 2, 2, 3
	duration := int64(74_123_456)
	laps := []openf1.Lap{
		{MeetingKey: 1236, SessionKey: 9500, DriverNumber: 16, LapNumber: 1, IsPitOutLap: &pitOut},
		{MeetingKey: 1236, SessionKey: 9500, DriverNumber: 16, LapNumber: 2, LapDuration: openf1.MicrosecondDuration{Present: true, Microseconds: &duration}},
		{MeetingKey: 1236, SessionKey: 9500, DriverNumber: 16, LapNumber: 3},
	}
	stints := []openf1.Stint{
		{MeetingKey: 1236, SessionKey: 9500, DriverNumber: 16, StintNumber: 1, Compound: &compound, LapStart: &firstStart, LapEnd: &firstEnd},
		{MeetingKey: 1236, SessionKey: 9500, DriverNumber: 16, StintNumber: 2, Compound: &compound, LapStart: &finalStart, LapEnd: &finalEnd},
	}

	snapshot, err := TransformTiming(target, laps, stints)
	if err != nil {
		t.Fatalf("TransformTiming() error = %v", err)
	}
	if len(snapshot.Laps) != 3 || snapshot.Laps[0].DurationMicroseconds != nil || snapshot.Laps[1].DurationMicroseconds == nil || *snapshot.Laps[1].DurationMicroseconds != duration {
		t.Fatalf("transformed laps = %+v", snapshot.Laps)
	}
	if snapshot.Laps[0].IsPitOutLap == nil || !*snapshot.Laps[0].IsPitOutLap || len(snapshot.Stints) != 2 || snapshot.Stints[0].Compound == nil {
		t.Fatalf("source context = laps %+v stints %+v", snapshot.Laps, snapshot.Stints)
	}
	if !snapshot.Laps[0].IsStintStart || snapshot.Laps[0].IsStintEnd ||
		!snapshot.Laps[1].IsStintStart || !snapshot.Laps[1].IsStintEnd ||
		snapshot.Laps[2].IsStintStart || !snapshot.Laps[2].IsStintEnd {
		t.Fatalf("stint boundary markers = %+v", snapshot.Laps)
	}
	if snapshot.Laps[1].IsPitOutLap != nil || snapshot.Laps[2].IsPitOutLap != nil {
		t.Fatalf("stint boundaries inferred pit-out markers: %+v", snapshot.Laps)
	}
}

func TestTransformTimingQuarantinesDuplicateObservations(t *testing.T) {
	t.Parallel()
	entryID, _ := domain.NewPublicID(domain.EntitySessionEntry, "monaco:race:leclerc")
	target := TimingTarget{MeetingKey: 1236, SessionKey: 9500, EntryIDsByNumber: map[int]domain.PublicID{16: entryID}}
	lap := openf1.Lap{MeetingKey: 1236, SessionKey: 9500, DriverNumber: 16, LapNumber: 1}
	if _, err := TransformTiming(target, []openf1.Lap{lap, lap}, nil); err == nil {
		t.Fatal("TransformTiming() error = nil")
	}
}
