package database

import (
	"context"
	"fmt"
	"time"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/clint/f1/backend/internal/ingest"
	"github.com/jackc/pgx/v5"
)

type TransactionStarter interface {
	Begin(context.Context) (pgx.Tx, error)
}

// WeekendPublisher replaces all meeting-owned rows in one transaction while upserting shared stable entities.
type WeekendPublisher struct {
	db  TransactionStarter
	now func() time.Time
}

func NewWeekendPublisher(db TransactionStarter) *WeekendPublisher {
	return &WeekendPublisher{db: db, now: time.Now}
}

func (p *WeekendPublisher) ReplaceWeekend(ctx context.Context, snapshot ingest.Snapshot) (time.Time, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("begin weekend replacement: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	publishedAt := p.now().UTC()
	fetchedAt := snapshot.SourceFetchedAt
	if fetchedAt.Meetings.IsZero() || fetchedAt.Sessions.IsZero() || fetchedAt.Drivers.IsZero() || fetchedAt.Results.IsZero() {
		return time.Time{}, fmt.Errorf("publish weekend: source fetch timestamps are incomplete")
	}

	meeting := snapshot.Weekend.Meeting
	var seasonID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO seasons (public_id, year, source_fetched_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (public_id) DO UPDATE SET year = EXCLUDED.year, source_fetched_at = EXCLUDED.source_fetched_at, ingested_at = CURRENT_TIMESTAMP
		RETURNING id`, meeting.Season.PublicID, meeting.Season.Year, fetchedAt.Meetings).Scan(&seasonID)
	if err != nil {
		return time.Time{}, fmt.Errorf("upsert season: %w", err)
	}

	var circuitID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO circuits (public_id, source_key, short_name, country_code, country_name, location, source_fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (public_id) DO UPDATE SET source_key = EXCLUDED.source_key, short_name = EXCLUDED.short_name,
			country_code = EXCLUDED.country_code, country_name = EXCLUDED.country_name, location = EXCLUDED.location,
			source_fetched_at = EXCLUDED.source_fetched_at, ingested_at = CURRENT_TIMESTAMP
		RETURNING id`, meeting.Circuit.PublicID, snapshot.CircuitSourceKey, meeting.Circuit.ShortName,
		meeting.Circuit.CountryCode, meeting.Circuit.CountryName, meeting.Circuit.Location, fetchedAt.Meetings).Scan(&circuitID)
	if err != nil {
		return time.Time{}, fmt.Errorf("upsert circuit: %w", err)
	}

	var meetingID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO meetings (public_id, source_key, season_id, circuit_id, name, official_name, date_start, date_end, is_cancelled, source_fetched_at, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (public_id) DO UPDATE SET source_key = EXCLUDED.source_key, season_id = EXCLUDED.season_id,
			circuit_id = EXCLUDED.circuit_id, name = EXCLUDED.name, official_name = EXCLUDED.official_name,
			date_start = EXCLUDED.date_start, date_end = EXCLUDED.date_end, is_cancelled = EXCLUDED.is_cancelled,
			source_fetched_at = EXCLUDED.source_fetched_at, published_at = EXCLUDED.published_at, ingested_at = CURRENT_TIMESTAMP
		RETURNING id`, meeting.PublicID, snapshot.MeetingSourceKey, seasonID, circuitID, meeting.Name,
		meeting.OfficialName, meeting.DateStart, meeting.DateEnd, meeting.IsCancelled, fetchedAt.Meetings, publishedAt).Scan(&meetingID)
	if err != nil {
		return time.Time{}, fmt.Errorf("upsert meeting: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM session_results WHERE session_entry_id IN (
		SELECT e.id FROM session_entries e JOIN sessions s ON s.id = e.session_id WHERE s.meeting_id = $1)`, meetingID); err != nil {
		return time.Time{}, fmt.Errorf("delete old results: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM session_entries WHERE session_id IN (SELECT id FROM sessions WHERE meeting_id = $1)`, meetingID); err != nil {
		return time.Time{}, fmt.Errorf("delete old entries: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE meeting_id = $1`, meetingID); err != nil {
		return time.Time{}, fmt.Errorf("delete old sessions: %w", err)
	}

	sessionIDs := make(map[domain.PublicID]int64, len(snapshot.Weekend.Sessions))
	for _, session := range snapshot.Weekend.Sessions {
		sourceKey, ok := snapshot.SessionSourceKeys[session.PublicID]
		if !ok {
			return time.Time{}, fmt.Errorf("session %q has no private source key", session.PublicID)
		}
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO sessions (public_id, source_key, meeting_id, name, type, date_start, date_end, is_cancelled, source_fetched_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`, session.PublicID, sourceKey, meetingID,
			session.Name, session.Type, session.DateStart, session.DateEnd, session.IsCancelled, fetchedAt.Sessions).Scan(&id)
		if err != nil {
			return time.Time{}, fmt.Errorf("insert session %q: %w", session.PublicID, err)
		}
		sessionIDs[session.PublicID] = id
	}

	driverIDs := make(map[domain.PublicID]int64, len(snapshot.Drivers))
	for _, driver := range snapshot.Drivers {
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO drivers (public_id, first_name, last_name, full_name, name_acronym, source_fetched_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (public_id) DO UPDATE SET first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name,
				full_name = EXCLUDED.full_name, name_acronym = EXCLUDED.name_acronym,
				source_fetched_at = EXCLUDED.source_fetched_at, ingested_at = CURRENT_TIMESTAMP
			RETURNING id`, driver.PublicID, driver.FirstName, driver.LastName, driver.FullName, driver.NameAcronym, fetchedAt.Drivers).Scan(&id)
		if err != nil {
			return time.Time{}, fmt.Errorf("upsert driver %q: %w", driver.PublicID, err)
		}
		driverIDs[driver.PublicID] = id
	}

	constructorIDs := make(map[domain.PublicID]int64, len(snapshot.Constructors))
	for _, constructor := range snapshot.Constructors {
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO constructor_entrants (public_id, season_id, name, source_fetched_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (public_id) DO UPDATE SET season_id = EXCLUDED.season_id, name = EXCLUDED.name,
				source_fetched_at = EXCLUDED.source_fetched_at, ingested_at = CURRENT_TIMESTAMP
			RETURNING id`, constructor.PublicID, seasonID, constructor.Name, fetchedAt.Drivers).Scan(&id)
		if err != nil {
			return time.Time{}, fmt.Errorf("upsert constructor %q: %w", constructor.PublicID, err)
		}
		constructorIDs[constructor.PublicID] = id
	}

	entryIDs := make(map[domain.PublicID]int64, len(snapshot.Entries))
	for _, entry := range snapshot.Entries {
		sessionID, sessionOK := sessionIDs[entry.SessionID]
		driverID, driverOK := driverIDs[entry.Driver.PublicID]
		constructorID, constructorOK := constructorIDs[entry.Constructor.PublicID]
		if !sessionOK || !driverOK || !constructorOK {
			return time.Time{}, fmt.Errorf("entry %q references unpublished domain records", entry.PublicID)
		}
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO session_entries (public_id, session_id, driver_id, constructor_entrant_id, driver_number, team_colour, source_fetched_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, entry.PublicID, sessionID, driverID,
			constructorID, entry.DriverNumber, entry.TeamColour, fetchedAt.Drivers).Scan(&id)
		if err != nil {
			return time.Time{}, fmt.Errorf("insert entry %q: %w", entry.PublicID, err)
		}
		entryIDs[entry.PublicID] = id
	}

	for _, result := range snapshot.Results {
		entryID, ok := entryIDs[result.SessionEntryID]
		if !ok {
			return time.Time{}, fmt.Errorf("result %q references an unpublished entry", result.PublicID)
		}
		durationNumber, durationText, durationNumbers := resultColumns(result.Duration)
		gapNumber, gapText, gapNumbers := resultColumns(result.GapToLeader)
		_, err := tx.Exec(ctx, `
			INSERT INTO session_results (public_id, session_entry_id, classification_state, position, number_of_laps,
				duration_seconds, duration_text, duration_segments_seconds, duration_kind,
				gap_to_leader_seconds, gap_to_leader_text, gap_to_leader_segments_seconds, gap_to_leader_kind,
				source_order, source_fetched_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
			result.PublicID, entryID, result.Classification, result.Position, result.NumberOfLaps,
			durationNumber, durationText, durationNumbers, result.Duration.Kind,
			gapNumber, gapText, gapNumbers, result.GapToLeader.Kind, result.SourceOrder, fetchedAt.Results)
		if err != nil {
			return time.Time{}, fmt.Errorf("insert result %q: %w", result.PublicID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, fmt.Errorf("commit weekend replacement: %w", err)
	}
	return publishedAt, nil
}

func resultColumns(value domain.ResultValue) (*float64, *string, []*float64) {
	switch value.Kind {
	case domain.ResultValueNumber:
		return &value.Number, nil, nil
	case domain.ResultValueText:
		return nil, &value.Text, nil
	case domain.ResultValueNumbers:
		return nil, nil, value.Numbers
	default:
		return nil, nil, nil
	}
}
