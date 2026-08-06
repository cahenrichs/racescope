package openf1

import (
	"encoding/json"
	"testing"
)

func TestResultValueDistinguishesZeroNullAndMissing(t *testing.T) {
	t.Parallel()

	var results []SessionResult
	if err := json.Unmarshal([]byte(`[
		{"duration":0,"gap_to_leader":null},
		{"duration":null},
		{}
	]`), &results); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !results[0].Duration.Present || results[0].Duration.Number == nil || *results[0].Duration.Number != 0 {
		t.Fatalf("zero duration = %+v", results[0].Duration)
	}
	if !results[0].GapToLeader.Present || results[0].GapToLeader.Number != nil || results[0].GapToLeader.Text != nil || results[0].GapToLeader.Numbers != nil {
		t.Fatalf("null gap = %+v", results[0].GapToLeader)
	}
	if !results[1].Duration.Present {
		t.Fatalf("explicit null duration = %+v", results[1].Duration)
	}
	if results[1].GapToLeader.Present || results[2].Duration.Present {
		t.Fatalf("missing values were marked present: %+v %+v", results[1].GapToLeader, results[2].Duration)
	}
}

func TestLapDurationPreservesExactMicrosecondsAndMissingValues(t *testing.T) {
	t.Parallel()
	var laps []Lap
	if err := json.Unmarshal([]byte(`[
		{"lap_duration":78.123456},
		{"lap_duration":null},
		{}
	]`), &laps); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got := laps[0].LapDuration.Microseconds; got == nil || *got != 78_123_456 {
		t.Fatalf("exact duration = %v, want 78123456", got)
	}
	if !laps[1].LapDuration.Present || laps[1].LapDuration.Microseconds != nil {
		t.Fatalf("null duration = %+v", laps[1].LapDuration)
	}
	if laps[2].LapDuration.Present || laps[2].LapDuration.Microseconds != nil {
		t.Fatalf("missing duration = %+v", laps[2].LapDuration)
	}
}

func TestLapDurationRejectsSubMicrosecondPrecision(t *testing.T) {
	t.Parallel()
	var lap Lap
	if err := json.Unmarshal([]byte(`{"lap_duration":1.0000001}`), &lap); err == nil {
		t.Fatal("json.Unmarshal() error = nil")
	}
}
