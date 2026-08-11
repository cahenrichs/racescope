package statistics

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type repositoryFunc func(context.Context, string, []string) (LapComparisonSource, error)

func (fn repositoryFunc) LapComparison(ctx context.Context, sessionID string, driverIDs []string) (LapComparisonSource, error) {
	return fn(ctx, sessionID, driverIDs)
}

func TestServiceCanonicalizesDriversAndReturnsEveryLap(t *testing.T) {
	t.Parallel()
	duration := int64(74_123_456)
	compound := "HARD"
	stint := 1
	pitOut := true
	repository := repositoryFunc(func(_ context.Context, sessionID string, driverIDs []string) (LapComparisonSource, error) {
		if sessionID != "session_monaco" || !reflect.DeepEqual(driverIDs, []string{"driver_leclerc", "driver_piastri"}) {
			t.Fatalf("repository request = %q, %v", sessionID, driverIDs)
		}
		return LapComparisonSource{
			SessionID: sessionID,
			Drivers: []SourceDriver{
				{ID: "driver_piastri", Name: "Oscar Piastri", Acronym: "PIA", Observations: []SourceLapObservation{{LapNumber: 1, DurationMicroseconds: &duration, Compound: &compound, StintNumber: &stint, IsPitOutLap: &pitOut, IsStintStart: true, IsStintEnd: true}}},
				{ID: "driver_leclerc", Name: "Charles Leclerc", Acronym: "LEC", Observations: []SourceLapObservation{
					{LapNumber: 1, Compound: &compound, StintNumber: &stint, IsPitOutLap: &pitOut, IsStintStart: true},
					{LapNumber: 2, DurationMicroseconds: &duration, StintNumber: &stint, IsPitOutLap: &pitOut, IsStintEnd: true},
				}},
			},
		}, nil
	})

	response, err := NewService(repository).Query(context.Background(), QueryRequest{
		Analysis:             AnalysisLapComparison,
		LapComparisonRequest: LapComparisonRequest{SessionID: "session_monaco", DriverIDs: []string{"driver_piastri", "driver_leclerc"}},
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if response.Result == nil || !reflect.DeepEqual(response.Result.CanonicalDriverIDs, []string{"driver_leclerc", "driver_piastri"}) {
		t.Fatalf("canonical driver IDs = %+v", response.Result)
	}
	if response.Result.Series[0].Driver.ID != "driver_leclerc" || len(response.Result.Series[0].Observations) != 2 {
		t.Fatalf("first series = %+v", response.Result.Series[0])
	}
	missing := response.Result.Series[0].Observations[0]
	if missing.DurationMicroseconds != nil || missing.MissingReason == nil || *missing.MissingReason != MissingSourceDuration {
		t.Fatalf("missing duration observation = %+v", missing)
	}
	if response.Result.Series[0].Observations[1].MissingReason != nil {
		t.Fatalf("available duration has missing reason = %+v", response.Result.Series[0].Observations[1])
	}
	if response.Result.Title != "Charles Leclerc and Oscar Piastri lap comparison" || response.Result.Dimension != "lap" || response.Result.Units != "microseconds" || response.Result.PreferredChartType != "line" {
		t.Fatalf("chart metadata = %+v", response.Result)
	}
	if response.Result.Series[0].Style != lapComparisonStyles[0] || response.Result.Series[1].Style != lapComparisonStyles[1] {
		t.Fatalf("series styles = %+v", response.Result.Series)
	}
	if response.Coverage.Status != CoveragePartial || len(response.Coverage.Series) != 2 || response.Coverage.Series[0].SeriesID != "driver_leclerc" {
		t.Fatalf("coverage = %+v", response.Coverage)
	}
	if len(response.Warnings) != 1 || response.Warnings[0].Code != WarningLapContextMissing || response.Warnings[0].Field != "compound" || response.Warnings[0].SeriesID != "driver_leclerc" {
		t.Fatalf("warnings = %+v", response.Warnings)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), "pitIn") {
		t.Fatalf("response inferred pit-in context: %s", encoded)
	}
}

func TestServiceRejectsInvalidLapComparisonBeforeRepositoryAccess(t *testing.T) {
	t.Parallel()
	repository := repositoryFunc(func(context.Context, string, []string) (LapComparisonSource, error) {
		t.Fatal("repository called for invalid request")
		return LapComparisonSource{}, nil
	})
	tests := []struct {
		name    string
		request QueryRequest
		code    string
	}{
		{name: "missing session", request: validRequest(), code: "invalid_size"},
		{name: "one driver", request: requestWithDrivers("driver_leclerc"), code: "exact_count"},
		{name: "duplicate driver", request: requestWithDrivers("driver_leclerc", "driver_leclerc"), code: "distinct"},
		{name: "oversized driver ID", request: requestWithDrivers("driver_leclerc", strings.Repeat("x", maximumPublicIDBytes+1)), code: "invalid_size"},
		{name: "missing analysis", request: QueryRequest{}, code: "required"},
	}
	tests[0].request.SessionID = ""

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewService(repository).Query(context.Background(), test.request)
			var queryError *QueryError
			if !errors.As(err, &queryError) || queryError.Kind != ErrorInvalidRequest {
				t.Fatalf("Query() error = %v", err)
			}
			found := false
			for _, issue := range queryError.Issues {
				found = found || issue.Code == test.code
			}
			if !found {
				t.Fatalf("issues = %+v, want code %q", queryError.Issues, test.code)
			}
		})
	}
}

func TestServiceMapsRepositoryIdentityAndCombinationErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		source error
		kind   ErrorKind
	}{
		{source: ErrSessionNotFound, kind: ErrorUnknownPublicID},
		{source: ErrDriverNotFound, kind: ErrorUnknownPublicID},
		{source: ErrUnsupportedSession, kind: ErrorUnsupportedCombination},
		{source: ErrDriverNotInSession, kind: ErrorUnsupportedCombination},
	}
	for _, test := range tests {
		repository := repositoryFunc(func(context.Context, string, []string) (LapComparisonSource, error) {
			return LapComparisonSource{}, test.source
		})
		_, err := NewService(repository).Query(context.Background(), validRequest())
		var queryError *QueryError
		if !errors.As(err, &queryError) || queryError.Kind != test.kind {
			t.Fatalf("source %v: Query() error = %v", test.source, err)
		}
	}
}

func validRequest() QueryRequest {
	return requestWithDrivers("driver_leclerc", "driver_piastri")
}

func requestWithDrivers(driverIDs ...string) QueryRequest {
	return QueryRequest{Analysis: AnalysisLapComparison, LapComparisonRequest: LapComparisonRequest{SessionID: "session_monaco", DriverIDs: driverIDs}}
}
