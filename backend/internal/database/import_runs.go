package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type AuditDB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type ImportRunRequest struct {
	Endpoint       string
	Parameters     map[string][]string
	ResponseStatus int
	FetchedAt      time.Time
	RecordCount    int
	ResponseSHA256 string
}

type ImportRunError struct {
	Order         int
	Code          string
	Entity        string
	SourceContext map[string]any
	Message       string
}

type ImportRunCompletion struct {
	Status         string
	FinishedAt     time.Time
	SessionCount   int
	EntryCount     int
	ResultCount    int
	ErrorCount     int
	DeferredReason *string
	PublishedAt    *time.Time
}

func CreateImportRun(ctx context.Context, db AuditDB, season, meetingKey int, startedAt time.Time) (int64, error) {
	var id int64
	err := db.QueryRow(ctx, `
		INSERT INTO import_runs (status, season, source_meeting_key, started_at)
		VALUES ('running', $1, $2, $3) RETURNING id`, season, meetingKey, startedAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create import run: %w", err)
	}
	return id, nil
}

func RecordImportRunRequest(ctx context.Context, db AuditDB, runID int64, request ImportRunRequest) error {
	parameters, err := json.Marshal(request.Parameters)
	if err != nil {
		return fmt.Errorf("encode sanitized request parameters: %w", err)
	}
	_, err = db.Exec(ctx, `
		INSERT INTO import_run_requests
			(import_run_id, endpoint, parameters, response_status, fetched_at, record_count, response_sha256)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, runID, request.Endpoint, parameters,
		request.ResponseStatus, request.FetchedAt, request.RecordCount, request.ResponseSHA256)
	if err != nil {
		return fmt.Errorf("record import run request: %w", err)
	}
	return nil
}

func RecordImportRunError(ctx context.Context, db AuditDB, runID int64, runError ImportRunError) error {
	contextJSON, err := json.Marshal(runError.SourceContext)
	if err != nil {
		return fmt.Errorf("encode sanitized error context: %w", err)
	}
	_, err = db.Exec(ctx, `
		INSERT INTO import_run_errors (import_run_id, error_order, code, entity, source_context, message)
		VALUES ($1, $2, $3, $4, $5, $6)`, runID, runError.Order, runError.Code, runError.Entity, contextJSON, runError.Message)
	if err != nil {
		return fmt.Errorf("record import run error: %w", err)
	}
	return nil
}

func FinishImportRun(ctx context.Context, db AuditDB, runID int64, completion ImportRunCompletion) error {
	result, err := db.Exec(ctx, `
		UPDATE import_runs
		SET status = $2, finished_at = $3, session_count = $4, entry_count = $5, result_count = $6,
			error_count = $7, deferred_reason = $8, published_at = $9
		WHERE id = $1 AND status = 'running'`, runID, completion.Status, completion.FinishedAt,
		completion.SessionCount, completion.EntryCount, completion.ResultCount, completion.ErrorCount,
		completion.DeferredReason, completion.PublishedAt)
	if err != nil {
		return fmt.Errorf("finish import run: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("finish import run %d: running row not found", runID)
	}
	return nil
}
