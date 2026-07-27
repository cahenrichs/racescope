package openf1

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var historicalNow = time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

func TestClientFetchesCompletedWeekendResources(t *testing.T) {
	fixtures := map[string]string{
		"/v1/meetings":       "meetings.json",
		"/v1/sessions":       "sessions.json",
		"/v1/drivers":        "drivers.json",
		"/v1/session_result": "session_results.json",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixture, ok := fixtures[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept header = %q, want application/json", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(readFixture(t, fixture))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+"/v1", nil)
	ctx := context.Background()
	meetings, err := client.Meetings(ctx, 2024)
	if err != nil {
		t.Fatalf("Meetings() error = %v", err)
	}
	if len(meetings) != 1 || meetings[0].MeetingKey != 1235 || meetings[0].MeetingName != "Monaco Grand Prix" {
		t.Fatalf("Meetings() = %#v", meetings)
	}

	sessions, err := client.Sessions(ctx, meetings[0].MeetingKey)
	if err != nil {
		t.Fatalf("Sessions() error = %v", err)
	}
	if len(sessions) != 1 || sessions[0].SessionKey != 9500 {
		t.Fatalf("Sessions() = %#v", sessions)
	}

	drivers, err := client.Drivers(ctx, sessions)
	if err != nil {
		t.Fatalf("Drivers() error = %v", err)
	}
	if len(drivers) != 2 || drivers[0].TeamName != "Ferrari" {
		t.Fatalf("Drivers() = %#v", drivers)
	}

	results, err := client.SessionResults(ctx, sessions)
	if err != nil {
		t.Fatalf("SessionResults() error = %v", err)
	}
	if len(results) != 2 || results[0].Duration.Number == nil || *results[0].Duration.Number != 8345.411 {
		t.Fatalf("SessionResults() = %#v", results)
	}
	if results[1].GapToLeader.Text == nil || *results[1].GapToLeader.Text != "+1 LAP" || results[1].Duration.Number != nil {
		t.Fatalf("SessionResults() did not preserve nonnumeric/null values: %#v", results[1])
	}
}

func TestClientRejectsMalformedResponse(t *testing.T) {
	server := fixtureServer(t, http.StatusOK, "malformed.json")
	defer server.Close()
	client := newTestClient(t, server.URL, nil)

	_, err := client.Meetings(context.Background(), 2024)
	if err == nil || !strings.Contains(err.Error(), "decode response") || !strings.Contains(err.Error(), "meetings") {
		t.Fatalf("Meetings() error = %v, want actionable decode error", err)
	}
}

func TestClientRetriesRateLimitedResponse(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write(readFixture(t, "rate_limited.json"))
			return
		}
		_, _ = w.Write(readFixture(t, "meetings.json"))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, nil)

	meetings, err := client.Meetings(context.Background(), 2024)
	if err != nil {
		t.Fatalf("Meetings() error = %v", err)
	}
	if len(meetings) != 1 || requests.Load() != 2 {
		t.Fatalf("Meetings() count = %d, requests = %d", len(meetings), requests.Load())
	}
}

func TestClientReturnsFailedResponseAfterBoundedRetries(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(readFixture(t, "failed.json"))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, nil)

	_, err := client.Meetings(context.Background(), 2024)
	var requestErr *RequestError
	if !errors.As(err, &requestErr) {
		t.Fatalf("Meetings() error = %v, want RequestError", err)
	}
	if requestErr.StatusCode != http.StatusServiceUnavailable || requestErr.Attempt != 3 || requests.Load() != 3 {
		t.Fatalf("request error = %#v, requests = %d", requestErr, requests.Load())
	}
	if !strings.Contains(err.Error(), "provider unavailable") {
		t.Fatalf("error = %v, want provider response context", err)
	}
}

func TestClientRejectsSessionDetailsUntilLiveWindowEnds(t *testing.T) {
	now := time.Date(2026, time.July, 27, 12, 0, 0, 0, time.UTC)
	session := Session{
		SessionKey: 1,
		DateStart:  now.Add(-time.Hour),
		DateEnd:    now.Add(-15 * time.Minute),
	}
	client := newTestClient(t, "http://unused.example", func() time.Time { return now })

	_, err := client.Drivers(context.Background(), []Session{session})
	if !errors.Is(err, ErrLiveDataWindow) || !strings.Contains(err.Error(), "historical after") {
		t.Fatalf("Drivers() error = %v, want live-window error", err)
	}
}

func TestClientPropagatesRequestTimeout(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	client := newTestClient(t, "http://openf1.example/v1", nil)
	client.httpClient = &http.Client{Transport: transport}
	client.requestTimeout = time.Millisecond
	client.maxRetries = 0

	_, err := client.Meetings(context.Background(), 2024)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Meetings() error = %v, want deadline exceeded", err)
	}
}

func TestClientRateLimitsAllEndpointsCentrally(t *testing.T) {
	var mu sync.Mutex
	var requestedAt []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		requestedAt = append(requestedAt, time.Now())
		mu.Unlock()
		_, _ = io.WriteString(w, "[]")
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, nil)
	client.limiter.interval = 15 * time.Millisecond

	if _, err := client.Meetings(context.Background(), 2024); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Sessions(context.Background(), 1235); err != nil {
		t.Fatal(err)
	}
	if elapsed := requestedAt[1].Sub(requestedAt[0]); elapsed < 12*time.Millisecond {
		t.Fatalf("requests were %s apart, want shared rate limit", elapsed)
	}
}

func TestClientValidatesInputsAndBatchBounds(t *testing.T) {
	client := newTestClient(t, "http://unused.example", nil)
	if _, err := client.Meetings(context.Background(), 2022); err == nil {
		t.Fatal("Meetings() error = nil for unsupported year")
	}
	if _, err := client.Sessions(context.Background(), 0); err == nil {
		t.Fatal("Sessions() error = nil for invalid meeting key")
	}
	if _, err := client.SessionResults(context.Background(), nil); err == nil {
		t.Fatal("SessionResults() error = nil for empty batch")
	}
	tooMany := make([]Session, MaxSessionsPerBatch+1)
	if _, err := client.Drivers(context.Background(), tooMany); err == nil {
		t.Fatal("Drivers() error = nil for oversized batch")
	}
}

func newTestClient(t *testing.T, baseURL string, now func() time.Time) *Client {
	t.Helper()
	if now == nil {
		now = func() time.Time { return historicalNow }
	}
	client, err := NewClient(Config{
		BaseURL:          baseURL,
		RequestInterval:  time.Nanosecond,
		RetryDelay:       time.Nanosecond,
		MaxRetryDelay:    time.Nanosecond,
		MaxResponseBytes: 64 << 10,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func fixtureServer(t *testing.T, status int, fixture string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write(readFixture(t, fixture))
	}))
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
