package database

import (
	"context"
	"fmt"
	"time"

	"github.com/clint/f1/backend/internal/statistics"
	"github.com/jackc/pgx/v5"
)

type StatisticsStore struct {
	db Querier
}

func NewStatisticsStore(db Querier) *StatisticsStore {
	return &StatisticsStore{db: db}
}

func (s *StatisticsStore) LapComparison(ctx context.Context, sessionPublicID string, driverPublicIDs []string) (statistics.LapComparisonSource, error) {
	var sessionID int64
	var name, sessionType string
	var cancelled bool
	var sourceFetchedAt, publishedAt *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT s.id, s.name, s.type, s.is_cancelled,
		       GREATEST(p.laps_source_fetched_at, p.stints_source_fetched_at), p.published_at
		FROM sessions s
		LEFT JOIN session_timing_publications p ON p.session_id = s.id
		WHERE s.public_id = $1`, sessionPublicID).Scan(&sessionID, &name, &sessionType, &cancelled, &sourceFetchedAt, &publishedAt)
	if err == pgx.ErrNoRows {
		return statistics.LapComparisonSource{}, statistics.ErrSessionNotFound
	}
	if err != nil {
		return statistics.LapComparisonSource{}, fmt.Errorf("query statistics session %q: %w", sessionPublicID, err)
	}
	if name != "Race" || sessionType != "Race" || cancelled || sourceFetchedAt == nil || publishedAt == nil {
		return statistics.LapComparisonSource{}, statistics.ErrUnsupportedSession
	}

	rows, err := s.db.Query(ctx, `
		SELECT d.public_id, d.full_name, d.name_acronym, l.lap_number, l.duration_microseconds,
		       stint.compound, stint.stint_number, l.is_pit_out_lap, l.is_stint_start, l.is_stint_end
		FROM session_entries e
		JOIN drivers d ON d.id = e.driver_id
		LEFT JOIN grand_prix_laps l ON l.session_entry_id = e.id
		LEFT JOIN LATERAL (
			SELECT st.compound, st.stint_number
			FROM grand_prix_stints st
			WHERE st.session_entry_id = e.id
			  AND l.lap_number IS NOT NULL
			  AND (st.lap_start IS NOT NULL OR st.lap_end IS NOT NULL)
			  AND (st.lap_start IS NULL OR st.lap_start <= l.lap_number)
			  AND (st.lap_end IS NULL OR st.lap_end >= l.lap_number)
			ORDER BY
			  (st.lap_start = l.lap_number OR st.lap_end = l.lap_number) DESC,
			  st.stint_number
			LIMIT 1
		) stint ON true
		WHERE e.session_id = $1 AND d.public_id = ANY($2::text[])
		ORDER BY d.public_id, l.lap_number`, sessionID, driverPublicIDs)
	if err != nil {
		return statistics.LapComparisonSource{}, fmt.Errorf("query lap comparison: %w", err)
	}
	defer rows.Close()

	drivers := make([]statistics.SourceDriver, 0, len(driverPublicIDs))
	byID := make(map[string]int, len(driverPublicIDs))
	for rows.Next() {
		var driverID, driverName, acronym string
		var lapNumber *int
		var duration *int64
		var compound *string
		var stintNumber *int
		var isPitOutLap *bool
		var isStintStart, isStintEnd *bool
		if err := rows.Scan(
			&driverID, &driverName, &acronym, &lapNumber, &duration,
			&compound, &stintNumber, &isPitOutLap, &isStintStart, &isStintEnd,
		); err != nil {
			return statistics.LapComparisonSource{}, fmt.Errorf("scan lap comparison: %w", err)
		}
		index, exists := byID[driverID]
		if !exists {
			index = len(drivers)
			byID[driverID] = index
			drivers = append(drivers, statistics.SourceDriver{ID: driverID, Name: driverName, Acronym: acronym, Observations: []statistics.SourceLapObservation{}})
		}
		if lapNumber != nil {
			drivers[index].Observations = append(drivers[index].Observations, statistics.SourceLapObservation{
				LapNumber: *lapNumber, DurationMicroseconds: duration, Compound: compound, StintNumber: stintNumber,
				IsPitOutLap: isPitOutLap, IsStintStart: *isStintStart, IsStintEnd: *isStintEnd,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return statistics.LapComparisonSource{}, fmt.Errorf("read lap comparison: %w", err)
	}
	if len(drivers) != len(driverPublicIDs) {
		return statistics.LapComparisonSource{}, s.missingDriverError(ctx, driverPublicIDs, byID)
	}
	return statistics.LapComparisonSource{
		SessionID: sessionPublicID, Drivers: drivers, SourceFetchedAt: *sourceFetchedAt, PublishedAt: *publishedAt,
	}, nil
}

func (s *StatisticsStore) missingDriverError(ctx context.Context, requested []string, inSession map[string]int) error {
	for _, id := range requested {
		if _, ok := inSession[id]; ok {
			continue
		}
		var exists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM drivers WHERE public_id = $1)`, id).Scan(&exists); err != nil {
			return fmt.Errorf("query statistics driver %q: %w", id, err)
		}
		if !exists {
			return statistics.ErrDriverNotFound
		}
	}
	return statistics.ErrDriverNotInSession
}
