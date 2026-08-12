package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clint/f1/backend/internal/statistics"
)

type statisticsQueryFunc func(context.Context, statistics.QueryRequest) (statistics.QueryResponse, error)

func (fn statisticsQueryFunc) Query(ctx context.Context, request statistics.QueryRequest) (statistics.QueryResponse, error) {
	return fn(ctx, request)
}

func TestStatisticsQuerySuccessAndNoDataContractsUseHTTP200(t *testing.T) {
	t.Parallel()
	duration := int64(74_123_456)
	fetchedAt := time.Date(2024, time.May, 27, 12, 0, 0, 0, time.UTC)
	publishedAt := time.Date(2026, time.August, 6, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		response statistics.QueryResponse
		assert   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success",
			response: statistics.QueryResponse{
				Kind: statistics.ResultSuccess, Analysis: statistics.AnalysisLapComparison,
				Result: &statistics.LapComparisonResult{
					SessionID: "session_monaco_race", CanonicalDriverIDs: []string{"driver_leclerc", "driver_piastri"},
					Title: "Leclerc and Piastri lap comparison", Dimension: "lap", Units: "microseconds", PreferredChartType: "line",
					Series: []statistics.LapSeries{{
						Driver: statistics.Driver{ID: "driver_leclerc", Name: "Charles Leclerc", Acronym: "LEC"},
						Observations: []statistics.LapObservation{
							{LapNumber: 1, DurationMicroseconds: &duration, IsStintStart: true},
							{LapNumber: 2, DurationMicroseconds: nil, MissingReason: statisticsMissingReason(statistics.MissingSourceDuration)},
						},
					}},
				},
				Warnings: []statistics.Warning{}, Coverage: statistics.Coverage{Status: statistics.CoverageComplete, Series: []statistics.SeriesCoverage{}},
				Freshness: statistics.Freshness{Status: statistics.FreshnessFresh, SourceFetchedAt: fetchedAt, PublishedAt: publishedAt},
			},
			assert: func(t *testing.T, response *httptest.ResponseRecorder) {
				var body statistics.QueryResponse
				decodeResponse(t, response, &body)
				if body.Kind != statistics.ResultSuccess || body.Result == nil || body.NoData != nil {
					t.Fatalf("success response = %+v", body)
				}
				observations := body.Result.Series[0].Observations
				if observations[0].DurationMicroseconds == nil || *observations[0].DurationMicroseconds != duration || observations[1].DurationMicroseconds != nil {
					t.Fatalf("duration observations = %+v", observations)
				}
				if strings.Contains(response.Body.String(), "74.123456") || !strings.Contains(response.Body.String(), `"durationMicroseconds":74123456`) {
					t.Fatalf("duration was not encoded as integer microseconds: %s", response.Body.String())
				}
			},
		},
		{
			name: "no data",
			response: statistics.QueryResponse{
				Kind: statistics.ResultNoData, Analysis: statistics.AnalysisLapComparison,
				NoData:   &statistics.NoData{Code: "no_usable_lap_durations", Message: "Neither selected driver has a usable lap duration."},
				Warnings: []statistics.Warning{}, Coverage: statistics.Coverage{Status: statistics.CoveragePartial, Series: []statistics.SeriesCoverage{}},
				Freshness: statistics.Freshness{Status: statistics.FreshnessFresh, SourceFetchedAt: fetchedAt, PublishedAt: publishedAt},
			},
			assert: func(t *testing.T, response *httptest.ResponseRecorder) {
				var body statistics.QueryResponse
				decodeResponse(t, response, &body)
				if body.Kind != statistics.ResultNoData || body.NoData == nil || body.Result != nil || body.NoData.Code != "no_usable_lap_durations" {
					t.Fatalf("no-data response = %+v", body)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service := statisticsQueryFunc(func(_ context.Context, request statistics.QueryRequest) (statistics.QueryResponse, error) {
				if request.Analysis != statistics.AnalysisLapComparison || request.SessionID != "session_monaco_race" || len(request.DriverIDs) != 2 {
					t.Fatalf("request = %+v", request)
				}
				return test.response, nil
			})
			response := serveStatisticsRequest(statisticsQuery(service), `{"analysis":"lap-comparison","sessionId":"session_monaco_race","driverIds":["driver_leclerc","driver_piastri"]}`)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			test.assert(t, response)
		})
	}
}

func TestStatisticsQueryStatusContracts(t *testing.T) {
	t.Parallel()
	issue := statistics.ValidationIssue{Field: "driverIds", Code: "distinct", Message: "Select two distinct drivers."}
	tests := []struct {
		name       string
		body       string
		err        error
		status     int
		code       string
		retryAfter string
	}{
		{name: "malformed", body: `{"analysis":`, status: http.StatusBadRequest, code: "malformed_request"},
		{name: "invalid", body: `{}`, err: &statistics.QueryError{Kind: statistics.ErrorInvalidRequest, Message: "The request is invalid.", Issues: []statistics.ValidationIssue{issue}}, status: http.StatusUnprocessableEntity, code: "invalid_request"},
		{name: "unsupported", body: `{}`, err: &statistics.QueryError{Kind: statistics.ErrorUnsupportedCombination, Message: "The analysis is unsupported."}, status: http.StatusUnprocessableEntity, code: "unsupported_combination"},
		{name: "unknown ID", body: `{}`, err: &statistics.QueryError{Kind: statistics.ErrorUnknownPublicID, Message: "A public ID was not found."}, status: http.StatusNotFound, code: "unknown_public_id"},
		{name: "rate limited", body: `{}`, err: &statistics.QueryError{Kind: statistics.ErrorRateLimited, Message: "Too many statistics queries.", RetryAfterSeconds: 30}, status: http.StatusTooManyRequests, code: "rate_limited", retryAfter: "30"},
		{name: "internal", body: `{}`, err: errors.New("database unavailable"), status: http.StatusInternalServerError, code: "internal_error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			called := false
			service := statisticsQueryFunc(func(context.Context, statistics.QueryRequest) (statistics.QueryResponse, error) {
				called = true
				return statistics.QueryResponse{}, test.err
			})
			response := serveStatisticsRequest(statisticsQuery(service), test.body)
			assertErrorContract(t, response, test.status, test.code)
			if test.name == "malformed" && called {
				t.Fatal("service was called for malformed JSON")
			}
			if got := response.Header().Get("Retry-After"); got != test.retryAfter {
				t.Fatalf("Retry-After = %q, want %q", got, test.retryAfter)
			}
			if test.name == "invalid" {
				var body errorResponse
				decodeResponse(t, response, &body)
				if len(body.Error.Issues) != 1 || body.Error.Issues[0] != issue {
					t.Fatalf("validation issues = %+v", body.Error.Issues)
				}
			}
		})
	}
}

func TestStatisticsQueryPartialAndStaleUsesHTTP200(t *testing.T) {
	t.Parallel()
	service := statisticsQueryFunc(func(context.Context, statistics.QueryRequest) (statistics.QueryResponse, error) {
		return statistics.QueryResponse{
			Kind: statistics.ResultSuccess, Analysis: statistics.AnalysisLapComparison,
			Result: &statistics.LapComparisonResult{}, Warnings: []statistics.Warning{
				{Code: statistics.WarningDriverLapsMissing, Message: "One driver has no usable laps."},
				{Code: statistics.WarningReconciliationFailed, Message: "A newer source reconciliation failed."},
			},
			Coverage:  statistics.Coverage{Status: statistics.CoveragePartial, Series: []statistics.SeriesCoverage{}},
			Freshness: statistics.Freshness{Status: statistics.FreshnessStale},
		}, nil
	})
	response := serveStatisticsRequest(statisticsQuery(service), `{}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body statistics.QueryResponse
	decodeResponse(t, response, &body)
	if body.Kind != statistics.ResultSuccess || body.Coverage.Status != statistics.CoveragePartial || body.Freshness.Status != statistics.FreshnessStale {
		t.Fatalf("partial response = %+v", body)
	}
}

func TestStatisticsQueryRejectsUnknownFieldsAndMultipleValues(t *testing.T) {
	t.Parallel()
	service := statisticsQueryFunc(func(context.Context, statistics.QueryRequest) (statistics.QueryResponse, error) {
		t.Fatal("service called for malformed request")
		return statistics.QueryResponse{}, nil
	})
	for _, body := range []string{`{"analysis":"lap-comparison","unknown":true}`, `{} {}`} {
		response := serveStatisticsRequest(statisticsQuery(service), body)
		assertErrorContract(t, response, http.StatusBadRequest, "malformed_request")
	}
}

func serveStatisticsRequest(handler http.Handler, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/statistics/query", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(response, request)
	return response
}

func statisticsMissingReason(value statistics.MissingReason) *statistics.MissingReason { return &value }
