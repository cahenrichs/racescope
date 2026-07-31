package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/clint/f1/backend/internal/domain"
)

var (
	ErrRaceNotFound   = errors.New("race not found")
	ErrRaceIncomplete = errors.New("race incomplete")
)

type Coverage struct {
	SourceFetchedAt time.Time
	PublishedAt     time.Time
}

type Circuit struct {
	PublicID    domain.PublicID
	Name        string
	CountryCode string
	CountryName string
	Location    string
}

type RaceSummary struct {
	PublicID     domain.PublicID
	Season       int
	Name         string
	OfficialName string
	Circuit      Circuit
	StartAt      time.Time
	EndAt        time.Time
	Coverage     Coverage
}

type SessionSummary struct {
	PublicID    domain.PublicID
	Name        string
	Type        string
	StartAt     time.Time
	EndAt       time.Time
	IsCancelled bool
}

type Race struct {
	RaceSummary
	Sessions []SessionSummary
}

type ResultValue struct {
	Kind    domain.ResultValueKind
	Number  *float64
	Text    *string
	Numbers []*float64
}

type ClassificationRow struct {
	DriverPublicID      domain.PublicID
	DriverName          string
	DriverAcronym       string
	ConstructorPublicID domain.PublicID
	ConstructorName     string
	DriverNumber        int
	Position            *int
	State               domain.ClassificationState
	Laps                *int
	Duration            ResultValue
	GapToLeader         ResultValue
}

type RaceResults struct {
	RacePublicID    domain.PublicID
	SessionPublicID domain.PublicID
	Classification  []ClassificationRow
	Coverage        Coverage
}

const completeRaceCTEs = `
	WITH grand_prix AS (
		SELECT meeting_id, MIN(id) AS session_id, COUNT(*) AS session_count
		FROM sessions
		WHERE type = 'Race' AND name = 'Race' AND NOT is_cancelled
		GROUP BY meeting_id
	), race_completeness AS (
		SELECT gp.meeting_id, gp.session_id, gp.session_count,
		       COUNT(e.id) AS entry_count, COUNT(r.id) AS result_count
		FROM grand_prix gp
		LEFT JOIN session_entries e ON e.session_id = gp.session_id
		LEFT JOIN session_results r ON r.session_entry_id = e.id
		GROUP BY gp.meeting_id, gp.session_id, gp.session_count
	)`

const sourceFetchedAtSQL = `GREATEST(
	m.source_fetched_at,
	season.source_fetched_at,
	c.source_fetched_at,
	COALESCE((SELECT MAX(s2.source_fetched_at) FROM sessions s2 WHERE s2.meeting_id = m.id), m.source_fetched_at),
	COALESCE((SELECT MAX(e2.source_fetched_at) FROM sessions s2 JOIN session_entries e2 ON e2.session_id = s2.id WHERE s2.meeting_id = m.id), m.source_fetched_at),
	COALESCE((SELECT MAX(d2.source_fetched_at) FROM sessions s2 JOIN session_entries e2 ON e2.session_id = s2.id JOIN drivers d2 ON d2.id = e2.driver_id WHERE s2.meeting_id = m.id), m.source_fetched_at),
	COALESCE((SELECT MAX(ce2.source_fetched_at) FROM sessions s2 JOIN session_entries e2 ON e2.session_id = s2.id JOIN constructor_entrants ce2 ON ce2.id = e2.constructor_entrant_id WHERE s2.meeting_id = m.id), m.source_fetched_at),
	COALESCE((SELECT MAX(r2.source_fetched_at) FROM sessions s2 JOIN session_entries e2 ON e2.session_id = s2.id JOIN session_results r2 ON r2.session_entry_id = e2.id WHERE s2.meeting_id = m.id), m.source_fetched_at)
)`

func PublishedRaces(ctx context.Context, db Querier) ([]RaceSummary, error) {
	query := completeRaceCTEs + `
		SELECT m.public_id, season.year, m.name, m.official_name,
		       c.public_id, c.short_name, c.country_code, c.country_name, c.location,
		       m.date_start, m.date_end, ` + sourceFetchedAtSQL + `, m.published_at
		FROM meetings m
		JOIN seasons season ON season.id = m.season_id
		JOIN circuits c ON c.id = m.circuit_id
		JOIN race_completeness complete ON complete.meeting_id = m.id
		WHERE m.published_at IS NOT NULL
		  AND complete.session_count = 1
		  AND complete.entry_count > 0
		  AND complete.result_count = complete.entry_count
		ORDER BY m.date_start DESC, m.public_id`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query published races: %w", err)
	}
	defer rows.Close()

	races := make([]RaceSummary, 0)
	for rows.Next() {
		var race RaceSummary
		if err := rows.Scan(
			&race.PublicID, &race.Season, &race.Name, &race.OfficialName,
			&race.Circuit.PublicID, &race.Circuit.Name, &race.Circuit.CountryCode,
			&race.Circuit.CountryName, &race.Circuit.Location,
			&race.StartAt, &race.EndAt, &race.Coverage.SourceFetchedAt, &race.Coverage.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan published race: %w", err)
		}
		races = append(races, race)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read published races: %w", err)
	}
	return races, nil
}

func RaceByPublicID(ctx context.Context, db Querier, publicID domain.PublicID) (Race, error) {
	query := completeRaceCTEs + `
		SELECT m.public_id, season.year, m.name, m.official_name,
		       c.public_id, c.short_name, c.country_code, c.country_name, c.location,
		       m.date_start, m.date_end, ` + sourceFetchedAtSQL + `, m.published_at,
		       m.published_at IS NOT NULL
		         AND COALESCE(complete.session_count = 1 AND complete.entry_count > 0
		         AND complete.result_count = complete.entry_count, false) AS is_complete,
		       s.public_id, s.name, s.type, s.date_start, s.date_end, s.is_cancelled
		FROM meetings m
		JOIN seasons season ON season.id = m.season_id
		JOIN circuits c ON c.id = m.circuit_id
		LEFT JOIN race_completeness complete ON complete.meeting_id = m.id
		LEFT JOIN sessions s ON s.meeting_id = m.id
		WHERE m.public_id = $1
		ORDER BY s.date_start, s.name, s.public_id`

	rows, err := db.Query(ctx, query, publicID.String())
	if err != nil {
		return Race{}, fmt.Errorf("query race %q: %w", publicID, err)
	}
	defer rows.Close()

	var race Race
	racesFound := false
	isComplete := false
	for rows.Next() {
		var publishedAt *time.Time
		var sessionID *domain.PublicID
		var sessionName, sessionType *string
		var sessionStart, sessionEnd *time.Time
		var sessionCancelled *bool
		if err := rows.Scan(
			&race.PublicID, &race.Season, &race.Name, &race.OfficialName,
			&race.Circuit.PublicID, &race.Circuit.Name, &race.Circuit.CountryCode,
			&race.Circuit.CountryName, &race.Circuit.Location,
			&race.StartAt, &race.EndAt, &race.Coverage.SourceFetchedAt, &publishedAt,
			&isComplete, &sessionID, &sessionName, &sessionType, &sessionStart, &sessionEnd, &sessionCancelled,
		); err != nil {
			return Race{}, fmt.Errorf("scan race %q: %w", publicID, err)
		}
		racesFound = true
		if publishedAt != nil {
			race.Coverage.PublishedAt = *publishedAt
		}
		if sessionID != nil {
			race.Sessions = append(race.Sessions, SessionSummary{
				PublicID: *sessionID, Name: *sessionName, Type: *sessionType,
				StartAt: *sessionStart, EndAt: *sessionEnd, IsCancelled: *sessionCancelled,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return Race{}, fmt.Errorf("read race %q: %w", publicID, err)
	}
	if !racesFound {
		return Race{}, ErrRaceNotFound
	}
	if !isComplete {
		return Race{}, ErrRaceIncomplete
	}
	if race.Sessions == nil {
		race.Sessions = make([]SessionSummary, 0)
	}
	return race, nil
}

func ResultsByRacePublicID(ctx context.Context, db Querier, publicID domain.PublicID) (RaceResults, error) {
	query := completeRaceCTEs + `
		SELECT m.public_id, gp.public_id, ` + sourceFetchedAtSQL + `, m.published_at,
		       m.published_at IS NOT NULL
		         AND COALESCE(complete.session_count = 1 AND complete.entry_count > 0
		         AND complete.result_count = complete.entry_count, false) AS is_complete,
		       d.public_id, d.full_name, d.name_acronym,
		       ce.public_id, ce.name, e.driver_number,
		       r.position, r.classification_state, r.number_of_laps,
		       r.duration_kind, r.duration_seconds, r.duration_text, r.duration_segments_seconds,
		       r.gap_to_leader_kind, r.gap_to_leader_seconds, r.gap_to_leader_text, r.gap_to_leader_segments_seconds
		FROM meetings m
		JOIN seasons season ON season.id = m.season_id
		JOIN circuits c ON c.id = m.circuit_id
		LEFT JOIN race_completeness complete ON complete.meeting_id = m.id
		LEFT JOIN sessions gp ON gp.id = complete.session_id AND complete.session_count = 1
		LEFT JOIN session_entries e ON e.session_id = gp.id
		LEFT JOIN drivers d ON d.id = e.driver_id
		LEFT JOIN constructor_entrants ce ON ce.id = e.constructor_entrant_id
		LEFT JOIN session_results r ON r.session_entry_id = e.id
		WHERE m.public_id = $1
		ORDER BY r.position NULLS LAST, r.source_order, e.driver_number`

	rows, err := db.Query(ctx, query, publicID.String())
	if err != nil {
		return RaceResults{}, fmt.Errorf("query race results %q: %w", publicID, err)
	}
	defer rows.Close()

	var results RaceResults
	racesFound := false
	isComplete := false
	for rows.Next() {
		var sessionID *domain.PublicID
		var publishedAt *time.Time
		var driverID, constructorID *domain.PublicID
		var driverName, acronym, constructorName *string
		var driverNumber *int
		var state *domain.ClassificationState
		var durationKind, gapKind *domain.ResultValueKind
		var row ClassificationRow
		if err := rows.Scan(
			&results.RacePublicID, &sessionID,
			&results.Coverage.SourceFetchedAt, &publishedAt, &isComplete,
			&driverID, &driverName, &acronym, &constructorID, &constructorName, &driverNumber,
			&row.Position, &state, &row.Laps,
			&durationKind, &row.Duration.Number, &row.Duration.Text, &row.Duration.Numbers,
			&gapKind, &row.GapToLeader.Number, &row.GapToLeader.Text, &row.GapToLeader.Numbers,
		); err != nil {
			return RaceResults{}, fmt.Errorf("scan race results %q: %w", publicID, err)
		}
		racesFound = true
		if sessionID != nil {
			results.SessionPublicID = *sessionID
		}
		if publishedAt != nil {
			results.Coverage.PublishedAt = *publishedAt
		}
		if driverID != nil {
			row.DriverPublicID = *driverID
			row.DriverName = *driverName
			row.DriverAcronym = *acronym
			row.ConstructorPublicID = *constructorID
			row.ConstructorName = *constructorName
			row.DriverNumber = *driverNumber
			row.State = *state
			row.Duration.Kind = *durationKind
			row.GapToLeader.Kind = *gapKind
			results.Classification = append(results.Classification, row)
		}
	}
	if err := rows.Err(); err != nil {
		return RaceResults{}, fmt.Errorf("read race results %q: %w", publicID, err)
	}
	if !racesFound {
		return RaceResults{}, ErrRaceNotFound
	}
	if !isComplete {
		return RaceResults{}, ErrRaceIncomplete
	}
	if results.Classification == nil {
		results.Classification = make([]ClassificationRow, 0)
	}
	return results, nil
}
