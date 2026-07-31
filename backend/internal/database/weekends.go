package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

var (
	ErrWeekendNotFound = errors.New("weekend not found")
	ErrSessionNotFound = errors.New("session not found")
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Weekend struct {
	PublicID        domain.PublicID
	Year            int
	Name            string
	OfficialName    string
	DateStart       time.Time
	DateEnd         time.Time
	IsCancelled     bool
	CircuitPublicID domain.PublicID
	CircuitName     string
	CountryCode     string
	CountryName     string
	Location        string
	IngestedAt      time.Time
}

type ClassificationResult struct {
	EntryPublicID  domain.PublicID
	DriverPublicID domain.PublicID
	DriverName     string
	DriverNumber   int
	State          domain.ClassificationState
	Position       *int
	NumberOfLaps   *int
}

func WeekendByPublicID(ctx context.Context, db Querier, publicID domain.PublicID) (Weekend, error) {
	const query = `
		SELECT m.public_id, s.year, m.name, m.official_name, m.date_start, m.date_end,
		       m.is_cancelled, c.public_id, c.short_name, c.country_code, c.country_name,
		       c.location, m.ingested_at
		FROM meetings m
		JOIN seasons s ON s.id = m.season_id
		JOIN circuits c ON c.id = m.circuit_id
		WHERE m.public_id = $1`

	var weekend Weekend
	err := db.QueryRow(ctx, query, publicID.String()).Scan(
		&weekend.PublicID,
		&weekend.Year,
		&weekend.Name,
		&weekend.OfficialName,
		&weekend.DateStart,
		&weekend.DateEnd,
		&weekend.IsCancelled,
		&weekend.CircuitPublicID,
		&weekend.CircuitName,
		&weekend.CountryCode,
		&weekend.CountryName,
		&weekend.Location,
		&weekend.IngestedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Weekend{}, ErrWeekendNotFound
	}
	if err != nil {
		return Weekend{}, fmt.Errorf("query weekend %q: %w", publicID, err)
	}
	return weekend, nil
}

func SessionClassification(ctx context.Context, db Querier, sessionPublicID domain.PublicID) ([]ClassificationResult, error) {
	const query = `
		SELECT e.public_id, d.public_id, d.full_name, e.driver_number,
		       COALESCE(r.classification_state, 'missing'), r.position, r.number_of_laps
		FROM sessions s
		JOIN session_entries e ON e.session_id = s.id
		JOIN drivers d ON d.id = e.driver_id
		LEFT JOIN session_results r ON r.session_entry_id = e.id
		WHERE s.public_id = $1
		ORDER BY r.position NULLS LAST, e.driver_number`

	rows, err := db.Query(ctx, query, sessionPublicID.String())
	if err != nil {
		return nil, fmt.Errorf("query session classification %q: %w", sessionPublicID, err)
	}
	defer rows.Close()

	results := make([]ClassificationResult, 0)
	for rows.Next() {
		var result ClassificationResult
		if err := rows.Scan(
			&result.EntryPublicID,
			&result.DriverPublicID,
			&result.DriverName,
			&result.DriverNumber,
			&result.State,
			&result.Position,
			&result.NumberOfLaps,
		); err != nil {
			return nil, fmt.Errorf("scan session classification %q: %w", sessionPublicID, err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read session classification %q: %w", sessionPublicID, err)
	}
	if len(results) == 0 {
		var exists bool
		if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sessions WHERE public_id = $1)`, sessionPublicID.String()).Scan(&exists); err != nil {
			return nil, fmt.Errorf("check session %q: %w", sessionPublicID, err)
		}
		if !exists {
			return nil, ErrSessionNotFound
		}
	}
	return results, nil
}
