package database

import (
	"context"
	"fmt"
	"time"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/clint/f1/backend/internal/ingest"
	"github.com/jackc/pgx/v5"
)

type timingDB interface {
	Querier
	TransactionStarter
}

// TimingStore resolves timing eligibility and atomically replaces one session timing unit.
type TimingStore struct {
	db  timingDB
	now func() time.Time
}

func NewTimingStore(db timingDB) *TimingStore {
	return &TimingStore{db: db, now: time.Now}
}

func (s *TimingStore) EligibleTimingTarget(ctx context.Context, requested ingest.Target) (ingest.TimingTarget, error) {
	var meetingID int64
	var meetingKey int
	var publishedAt *time.Time
	var meetingCancelled bool
	err := s.db.QueryRow(ctx, `
		SELECT m.id, m.source_key, m.published_at, m.is_cancelled
		FROM meetings m JOIN seasons season ON season.id = m.season_id
		WHERE season.year = $1 AND m.source_key = $2`, requested.Season, requested.MeetingKey).Scan(
		&meetingID, &meetingKey, &publishedAt, &meetingCancelled)
	if err == pgx.ErrNoRows {
		return ingest.TimingTarget{}, &ingest.DeferredTimingError{Reason: "weekend is not published"}
	}
	if err != nil {
		return ingest.TimingTarget{}, fmt.Errorf("query timing weekend: %w", err)
	}
	if publishedAt == nil {
		return ingest.TimingTarget{}, &ingest.DeferredTimingError{Reason: "weekend is not published"}
	}
	if meetingCancelled {
		return ingest.TimingTarget{}, &ingest.DeferredTimingError{Reason: "Grand Prix meeting is cancelled"}
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, public_id, source_key, date_start, date_end, is_cancelled
		FROM sessions WHERE meeting_id = $1 AND name = 'Race' AND type = 'Race'
		ORDER BY id`, meetingID)
	if err != nil {
		return ingest.TimingTarget{}, fmt.Errorf("query Grand Prix session: %w", err)
	}
	defer rows.Close()
	target := ingest.TimingTarget{MeetingKey: meetingKey}
	var sessionID int64
	count := 0
	var cancelled bool
	for rows.Next() {
		count++
		if err := rows.Scan(&sessionID, &target.SessionID, &target.SessionKey, &target.SessionStart, &target.SessionEnd, &cancelled); err != nil {
			return ingest.TimingTarget{}, fmt.Errorf("scan Grand Prix session: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return ingest.TimingTarget{}, fmt.Errorf("read Grand Prix session: %w", err)
	}
	if count != 1 {
		return ingest.TimingTarget{}, &ingest.DeferredTimingError{Reason: "weekend does not have exactly one Race/Race Grand Prix"}
	}
	if cancelled {
		return ingest.TimingTarget{}, &ingest.DeferredTimingError{Reason: "Grand Prix session is cancelled"}
	}
	if s.now().Before(target.SessionEnd.Add(30 * time.Minute)) {
		return ingest.TimingTarget{}, &ingest.DeferredTimingError{Reason: "Grand Prix is inside the provider live-data window"}
	}

	entryRows, err := s.db.Query(ctx, `
		SELECT e.public_id, e.driver_number, r.id IS NOT NULL
		FROM session_entries e LEFT JOIN session_results r ON r.session_entry_id = e.id
		WHERE e.session_id = $1 ORDER BY e.driver_number`, sessionID)
	if err != nil {
		return ingest.TimingTarget{}, fmt.Errorf("query Grand Prix classification completeness: %w", err)
	}
	defer entryRows.Close()
	target.EntryIDsByNumber = make(map[int]domain.PublicID)
	complete := true
	for entryRows.Next() {
		var publicID domain.PublicID
		var number int
		var hasResult bool
		if err := entryRows.Scan(&publicID, &number, &hasResult); err != nil {
			return ingest.TimingTarget{}, fmt.Errorf("scan Grand Prix entry: %w", err)
		}
		target.EntryIDsByNumber[number] = publicID
		complete = complete && hasResult
	}
	if err := entryRows.Err(); err != nil {
		return ingest.TimingTarget{}, fmt.Errorf("read Grand Prix entries: %w", err)
	}
	if len(target.EntryIDsByNumber) == 0 || !complete {
		return ingest.TimingTarget{}, &ingest.DeferredTimingError{Reason: "Grand Prix classification is incomplete"}
	}
	return target, nil
}

func (s *TimingStore) ReplaceTiming(ctx context.Context, snapshot ingest.TimingSnapshot) (time.Time, error) {
	if snapshot.LapsFetchedAt.IsZero() || snapshot.StintsFetchedAt.IsZero() {
		return time.Time{}, fmt.Errorf("publish timing: source fetch timestamps are incomplete")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("begin timing replacement: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var sessionID int64
	err = tx.QueryRow(ctx, `SELECT id FROM sessions WHERE public_id = $1 AND source_key = $2 FOR UPDATE`, snapshot.Target.SessionID, snapshot.Target.SessionKey).Scan(&sessionID)
	if err != nil {
		return time.Time{}, fmt.Errorf("lock timing session identity: %w", err)
	}
	entryIDs := make(map[domain.PublicID]int64, len(snapshot.Target.EntryIDsByNumber))
	rows, err := tx.Query(ctx, `SELECT id, public_id, driver_number FROM session_entries WHERE session_id = $1`, sessionID)
	if err != nil {
		return time.Time{}, fmt.Errorf("query timing entries: %w", err)
	}
	for rows.Next() {
		var id int64
		var publicID domain.PublicID
		var number int
		if err := rows.Scan(&id, &publicID, &number); err != nil {
			rows.Close()
			return time.Time{}, fmt.Errorf("scan timing entry: %w", err)
		}
		if expected, ok := snapshot.Target.EntryIDsByNumber[number]; ok && expected == publicID {
			entryIDs[publicID] = id
		}
	}
	rows.Close()
	if len(entryIDs) != len(snapshot.Target.EntryIDsByNumber) {
		return time.Time{}, fmt.Errorf("publish timing: session entry identities changed before publication")
	}

	if _, err := tx.Exec(ctx, `DELETE FROM grand_prix_laps WHERE session_entry_id IN (SELECT id FROM session_entries WHERE session_id = $1)`, sessionID); err != nil {
		return time.Time{}, fmt.Errorf("delete previous laps: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM grand_prix_stints WHERE session_entry_id IN (SELECT id FROM session_entries WHERE session_id = $1)`, sessionID); err != nil {
		return time.Time{}, fmt.Errorf("delete previous stints: %w", err)
	}
	for _, lap := range snapshot.Laps {
		entryID, ok := entryIDs[lap.SessionEntryID]
		if !ok {
			return time.Time{}, fmt.Errorf("lap references changed session entry %q", lap.SessionEntryID)
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO grand_prix_laps (session_entry_id, source_session_key, source_driver_number, lap_number,
				duration_microseconds, is_pit_out_lap, is_stint_start, is_stint_end, source_fetched_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, entryID, snapshot.Target.SessionKey, lap.SourceDriverNumber,
			lap.LapNumber, lap.DurationMicroseconds, lap.IsPitOutLap, lap.IsStintStart, lap.IsStintEnd, snapshot.LapsFetchedAt)
		if err != nil {
			return time.Time{}, fmt.Errorf("insert driver %d lap %d: %w", lap.SourceDriverNumber, lap.LapNumber, err)
		}
	}
	for _, stint := range snapshot.Stints {
		entryID, ok := entryIDs[stint.SessionEntryID]
		if !ok {
			return time.Time{}, fmt.Errorf("stint references changed session entry %q", stint.SessionEntryID)
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO grand_prix_stints (session_entry_id, source_session_key, source_driver_number, stint_number,
				compound, lap_start, lap_end, source_fetched_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, entryID, snapshot.Target.SessionKey, stint.SourceDriverNumber,
			stint.StintNumber, stint.Compound, stint.LapStart, stint.LapEnd, snapshot.StintsFetchedAt)
		if err != nil {
			return time.Time{}, fmt.Errorf("insert driver %d stint %d: %w", stint.SourceDriverNumber, stint.StintNumber, err)
		}
	}
	publishedAt := s.now().UTC()
	_, err = tx.Exec(ctx, `
		INSERT INTO session_timing_publications (session_id, source_session_key, laps_source_fetched_at, stints_source_fetched_at, published_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (session_id) DO UPDATE SET source_session_key = EXCLUDED.source_session_key,
			laps_source_fetched_at = EXCLUDED.laps_source_fetched_at, stints_source_fetched_at = EXCLUDED.stints_source_fetched_at,
			published_at = EXCLUDED.published_at, ingested_at = CURRENT_TIMESTAMP`, sessionID, snapshot.Target.SessionKey,
		snapshot.LapsFetchedAt, snapshot.StintsFetchedAt, publishedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("publish timing unit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, fmt.Errorf("commit timing replacement: %w", err)
	}
	return publishedAt, nil
}
