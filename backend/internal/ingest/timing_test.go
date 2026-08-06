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
	start, end := 1, 77
	duration := int64(74_123_456)
	laps := []openf1.Lap{
		{MeetingKey: 1236, SessionKey: 9500, DriverNumber: 16, LapNumber: 1, IsPitOutLap: &pitOut},
		{MeetingKey: 1236, SessionKey: 9500, DriverNumber: 16, LapNumber: 2, LapDuration: openf1.MicrosecondDuration{Present: true, Microseconds: &duration}},
	}
	stints := []openf1.Stint{{MeetingKey: 1236, SessionKey: 9500, DriverNumber: 16, StintNumber: 1, Compound: &compound, LapStart: &start, LapEnd: &end}}

	snapshot, err := TransformTiming(target, laps, stints)
	if err != nil {
		t.Fatalf("TransformTiming() error = %v", err)
	}
	if len(snapshot.Laps) != 2 || snapshot.Laps[0].DurationMicroseconds != nil || snapshot.Laps[1].DurationMicroseconds == nil || *snapshot.Laps[1].DurationMicroseconds != duration {
		t.Fatalf("transformed laps = %+v", snapshot.Laps)
	}
	if snapshot.Laps[0].IsPitOutLap == nil || !*snapshot.Laps[0].IsPitOutLap || len(snapshot.Stints) != 1 || snapshot.Stints[0].Compound == nil {
		t.Fatalf("source context = laps %+v stints %+v", snapshot.Laps, snapshot.Stints)
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
