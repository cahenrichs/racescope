package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type pingFunc func(context.Context) error

func (fn pingFunc) Ping(ctx context.Context) error {
	return fn(ctx)
}

func TestHealthDoesNotPingDatabase(t *testing.T) {
	router := NewRouter(pingFunc(func(context.Context) error {
		t.Fatal("database pinged by liveness check")
		return nil
	}))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	assertStatusResponse(t, response, http.StatusOK, "{\"status\":\"ok\"}\n")
}

func TestReady(t *testing.T) {
	tests := map[string]struct {
		ping     pingFunc
		wantCode int
		wantBody string
	}{
		"available": {
			ping:     func(context.Context) error { return nil },
			wantCode: http.StatusOK,
			wantBody: "{\"status\":\"ok\"}\n",
		},
		"unavailable": {
			ping:     func(context.Context) error { return errors.New("database unavailable") },
			wantCode: http.StatusServiceUnavailable,
			wantBody: "{\"status\":\"unavailable\"}\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			NewRouter(test.ping).ServeHTTP(
				response,
				httptest.NewRequest(http.MethodGet, "/ready", nil),
			)

			assertStatusResponse(t, response, test.wantCode, test.wantBody)
		})
	}
}

func TestReadyBoundsDatabasePing(t *testing.T) {
	response := httptest.NewRecorder()
	NewRouter(pingFunc(func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("database ping context has no deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > readinessTimeout {
			t.Fatalf("database ping deadline remaining = %v, want at most %v", remaining, readinessTimeout)
		}
		return nil
	})).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))

	assertStatusResponse(t, response, http.StatusOK, "{\"status\":\"ok\"}\n")
}

func TestHealthRoutesRejectUnsupportedMethods(t *testing.T) {
	for _, path := range []string{"/health", "/ready"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			NewRouter(pingFunc(func(context.Context) error {
				t.Fatal("database pinged for unsupported method")
				return nil
			})).ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))

			if response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func assertStatusResponse(t *testing.T, response *httptest.ResponseRecorder, wantCode int, wantBody string) {
	t.Helper()
	if response.Code != wantCode {
		t.Fatalf("status = %d, want %d", response.Code, wantCode)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
	if body := response.Body.String(); body != wantBody {
		t.Fatalf("body = %q, want %q", body, wantBody)
	}
}
