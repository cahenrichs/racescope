package openf1

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBaseURL          = "https://api.openf1.org/v1"
	FirstSupportedYear      = 2023
	DefaultRequestTimeout   = 10 * time.Second
	DefaultRequestInterval  = 2 * time.Second // Enforces the free tier's stricter 30 requests/minute limit.
	DefaultMaxRetries       = 2
	DefaultRetryDelay       = 500 * time.Millisecond
	DefaultMaxRetryDelay    = 5 * time.Second
	DefaultMaxResponseBytes = 2 << 20
	MaxSessionsPerBatch     = 10
	MaxRetriesAllowed       = 5
	liveWindowPadding       = 30 * time.Minute
)

var (
	ErrLiveDataWindow   = errors.New("OpenF1 session detail is unavailable during the live-data window")
	ErrResponseTooLarge = errors.New("OpenF1 response exceeds the configured size limit")
)

// Config controls all provider access bounds. Zero values select production-safe defaults.
type Config struct {
	BaseURL          string
	HTTPClient       *http.Client
	RequestTimeout   time.Duration
	RequestInterval  time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
	MaxRetryDelay    time.Duration
	MaxResponseBytes int64
	Now              func() time.Time
}

// Client is safe for concurrent use. All endpoint requests share one rate limiter.
type Client struct {
	baseURL          *url.URL
	httpClient       *http.Client
	requestTimeout   time.Duration
	maxRetries       int
	retryDelay       time.Duration
	maxRetryDelay    time.Duration
	maxResponseBytes int64
	now              func() time.Time
	limiter          requestLimiter
	recordsMu        sync.Mutex
	requestRecords   []RequestRecord
}

// RequestRecord is sanitized response metadata retained for import auditing, never the response body.
type RequestRecord struct {
	Endpoint       string
	Parameters     map[string][]string
	ResponseStatus int
	FetchedAt      time.Time
	RecordCount    int
	ResponseSHA256 string
}

type requestLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
}

// RequestError includes enough provider context to diagnose a failed import.
type RequestError struct {
	Endpoint   string
	StatusCode int
	Attempt    int
	Body       string
	Err        error
}

func (e *RequestError) Error() string {
	parts := []string{fmt.Sprintf("OpenF1 %s request failed on attempt %d", e.Endpoint, e.Attempt)}
	if e.StatusCode != 0 {
		parts = append(parts, fmt.Sprintf("status %d", e.StatusCode))
	}
	if e.Body != "" {
		parts = append(parts, e.Body)
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, ": ")
}

func (e *RequestError) Unwrap() error { return e.Err }

func NewClient(config Config) (*Client, error) {
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Host == "" {
		return nil, fmt.Errorf("invalid OpenF1 base URL %q", config.BaseURL)
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/")

	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = DefaultRequestTimeout
	}
	if config.RequestInterval == 0 {
		config.RequestInterval = DefaultRequestInterval
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = DefaultRetryDelay
	}
	if config.MaxRetryDelay == 0 {
		config.MaxRetryDelay = DefaultMaxRetryDelay
	}
	if config.MaxResponseBytes == 0 {
		config.MaxResponseBytes = DefaultMaxResponseBytes
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.MaxRetries < 0 || config.MaxRetries > MaxRetriesAllowed {
		return nil, fmt.Errorf("OpenF1 max retries must be between 0 and %d", MaxRetriesAllowed)
	}
	if config.RequestTimeout < 0 || config.RequestInterval < 0 || config.RetryDelay < 0 || config.MaxRetryDelay < 0 || config.MaxResponseBytes < 1 {
		return nil, errors.New("OpenF1 client bounds must not be negative and max response bytes must be positive")
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = DefaultMaxRetries
	}

	return &Client{
		baseURL:          baseURL,
		httpClient:       config.HTTPClient,
		requestTimeout:   config.RequestTimeout,
		maxRetries:       config.MaxRetries,
		retryDelay:       config.RetryDelay,
		maxRetryDelay:    config.MaxRetryDelay,
		maxResponseBytes: config.MaxResponseBytes,
		now:              config.Now,
		limiter:          requestLimiter{interval: config.RequestInterval},
	}, nil
}

// Meetings fetches the bounded set of meetings for one supported season.
func (c *Client) Meetings(ctx context.Context, year int) ([]Meeting, error) {
	if year < FirstSupportedYear || year > c.now().UTC().Year() {
		return nil, fmt.Errorf("OpenF1 meeting year must be between %d and %d", FirstSupportedYear, c.now().UTC().Year())
	}
	var meetings []Meeting
	err := c.get(ctx, "meetings", url.Values{"year": {strconv.Itoa(year)}}, &meetings)
	return meetings, err
}

// Sessions fetches all sessions belonging to one meeting.
func (c *Client) Sessions(ctx context.Context, meetingKey int) ([]Session, error) {
	if meetingKey <= 0 {
		return nil, errors.New("OpenF1 meeting key must be positive")
	}
	var sessions []Session
	err := c.get(ctx, "sessions", url.Values{"meeting_key": {strconv.Itoa(meetingKey)}}, &sessions)
	return sessions, err
}

// Drivers fetches session-specific driver entries in a bounded batch.
func (c *Client) Drivers(ctx context.Context, sessions []Session) ([]Driver, error) {
	if err := c.validateDetailBatch(sessions); err != nil {
		return nil, err
	}
	drivers := make([]Driver, 0, len(sessions)*20)
	for _, session := range sessions {
		var batch []Driver
		if err := c.get(ctx, "drivers", url.Values{"session_key": {strconv.Itoa(session.SessionKey)}}, &batch); err != nil {
			return nil, fmt.Errorf("fetch drivers for session %d: %w", session.SessionKey, err)
		}
		drivers = append(drivers, batch...)
	}
	return drivers, nil
}

// SessionResults fetches classifications in a bounded, session-at-a-time batch.
func (c *Client) SessionResults(ctx context.Context, sessions []Session) ([]SessionResult, error) {
	if err := c.validateDetailBatch(sessions); err != nil {
		return nil, err
	}
	results := make([]SessionResult, 0, len(sessions)*20)
	for _, session := range sessions {
		var batch []SessionResult
		if err := c.get(ctx, "session_result", url.Values{"session_key": {strconv.Itoa(session.SessionKey)}}, &batch); err != nil {
			return nil, fmt.Errorf("fetch results for session %d: %w", session.SessionKey, err)
		}
		results = append(results, batch...)
	}
	return results, nil
}

func (c *Client) RequestRecords() []RequestRecord {
	c.recordsMu.Lock()
	defer c.recordsMu.Unlock()
	records := make([]RequestRecord, len(c.requestRecords))
	copy(records, c.requestRecords)
	return records
}

func (c *Client) validateDetailBatch(sessions []Session) error {
	if len(sessions) == 0 || len(sessions) > MaxSessionsPerBatch {
		return fmt.Errorf("OpenF1 session batch size must be between 1 and %d", MaxSessionsPerBatch)
	}
	now := c.now()
	for _, session := range sessions {
		if session.SessionKey <= 0 {
			return errors.New("OpenF1 session key must be positive")
		}
		if session.DateStart.IsZero() || session.DateEnd.IsZero() || session.DateEnd.Before(session.DateStart) {
			return fmt.Errorf("OpenF1 session %d has invalid start or end times", session.SessionKey)
		}
		availableAt := session.DateEnd.Add(liveWindowPadding)
		if now.Before(availableAt) {
			return fmt.Errorf("%w: session %d is historical after %s", ErrLiveDataWindow, session.SessionKey, availableAt.UTC().Format(time.RFC3339))
		}
	}
	return nil
}

func (c *Client) get(ctx context.Context, endpoint string, query url.Values, destination any) error {
	requestURL := *c.baseURL
	requestURL.Path += "/" + endpoint
	requestURL.RawQuery = query.Encode()

	for attempt := 1; attempt <= c.maxRetries+1; attempt++ {
		if err := c.limiter.wait(ctx, c.now); err != nil {
			return &RequestError{Endpoint: endpoint, Attempt: attempt, Err: err}
		}

		requestCtx, cancel := context.WithTimeout(ctx, c.requestTimeout)
		req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, requestURL.String(), nil)
		if err != nil {
			cancel()
			return &RequestError{Endpoint: endpoint, Attempt: attempt, Err: err}
		}
		req.Header.Set("Accept", "application/json")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			cancel()
			if attempt <= c.maxRetries && isRetryableTransportError(err, ctx) {
				if err := c.waitForRetry(ctx, attempt, 0); err != nil {
					return &RequestError{Endpoint: endpoint, Attempt: attempt, Err: err}
				}
				continue
			}
			return &RequestError{Endpoint: endpoint, Attempt: attempt, Err: err}
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBytes+1))
		resp.Body.Close()
		cancel()
		if readErr != nil {
			return &RequestError{Endpoint: endpoint, StatusCode: resp.StatusCode, Attempt: attempt, Err: readErr}
		}
		if int64(len(body)) > c.maxResponseBytes {
			c.recordRequest(endpoint, query, resp.StatusCode, body, 0)
			return &RequestError{Endpoint: endpoint, StatusCode: resp.StatusCode, Attempt: attempt, Err: ErrResponseTooLarge}
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			c.recordRequest(endpoint, query, resp.StatusCode, body, 0)
			requestErr := &RequestError{Endpoint: endpoint, StatusCode: resp.StatusCode, Attempt: attempt, Body: responseMessage(body)}
			if attempt <= c.maxRetries && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError) {
				if err := c.waitForRetry(ctx, attempt, retryAfter(resp.Header.Get("Retry-After"), c.now())); err != nil {
					requestErr.Err = err
					return requestErr
				}
				continue
			}
			return requestErr
		}

		decoder := json.NewDecoder(bytes.NewReader(body))
		if err := decoder.Decode(destination); err != nil {
			c.recordRequest(endpoint, query, resp.StatusCode, body, 0)
			return &RequestError{Endpoint: endpoint, StatusCode: resp.StatusCode, Attempt: attempt, Err: fmt.Errorf("decode response: %w", err)}
		}
		if err := ensureJSONEnd(decoder); err != nil {
			c.recordRequest(endpoint, query, resp.StatusCode, body, 0)
			return &RequestError{Endpoint: endpoint, StatusCode: resp.StatusCode, Attempt: attempt, Err: err}
		}
		c.recordRequest(endpoint, query, resp.StatusCode, body, decodedRecordCount(destination))
		return nil
	}
	panic("unreachable")
}

func (c *Client) recordRequest(endpoint string, query url.Values, status int, body []byte, recordCount int) {
	hash := sha256.Sum256(body)
	parameters := make(map[string][]string, len(query))
	for key, values := range query {
		parameters[key] = append([]string(nil), values...)
	}
	record := RequestRecord{
		Endpoint: endpoint, Parameters: parameters, ResponseStatus: status, FetchedAt: c.now().UTC(),
		RecordCount: recordCount, ResponseSHA256: fmt.Sprintf("%x", hash),
	}
	c.recordsMu.Lock()
	c.requestRecords = append(c.requestRecords, record)
	c.recordsMu.Unlock()
}

func decodedRecordCount(destination any) int {
	value := reflect.ValueOf(destination)
	if value.Kind() == reflect.Pointer && !value.IsNil() {
		value = value.Elem()
	}
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		return value.Len()
	}
	return 0
}

func (l *requestLimiter) wait(ctx context.Context, now func() time.Time) error {
	l.mu.Lock()
	current := now()
	slot := current
	if l.next.After(slot) {
		slot = l.next
	}
	l.next = slot.Add(l.interval)
	l.mu.Unlock()
	return wait(ctx, slot.Sub(current))
}

func (c *Client) waitForRetry(ctx context.Context, attempt int, requested time.Duration) error {
	delay := c.retryDelay << (attempt - 1)
	if requested > delay {
		delay = requested
	}
	if delay > c.maxRetryDelay {
		delay = c.maxRetryDelay
	}
	return wait(ctx, delay)
}

func wait(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func retryAfter(value string, now time.Time) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil && date.After(now) {
		return date.Sub(now)
	}
	return 0
}

func isRetryableTransportError(err error, parent context.Context) bool {
	return parent.Err() == nil && !errors.Is(err, context.Canceled)
}

func responseMessage(body []byte) string {
	const maxMessageBytes = 512
	body = []byte(strings.TrimSpace(string(body)))
	if len(body) > maxMessageBytes {
		body = body[:maxMessageBytes]
	}
	return string(body)
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode response: multiple JSON values")
		}
		return fmt.Errorf("decode response trailer: %w", err)
	}
	return nil
}
