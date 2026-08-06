package ingest

import (
	"context"
	"fmt"
	"time"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/clint/f1/backend/internal/openf1"
)

// DeferredTimingError describes an expected eligibility failure that should be audited, not retried as a failed publication.
type DeferredTimingError struct {
	Reason string
}

func (e *DeferredTimingError) Error() string { return "timing ingestion deferred: " + e.Reason }

// TimingTarget is an already-published, complete Grand Prix resolved from storage.
type TimingTarget struct {
	MeetingKey       int
	SessionID        domain.PublicID
	SessionKey       int
	SessionStart     time.Time
	SessionEnd       time.Time
	EntryIDsByNumber map[int]domain.PublicID
}

type TimingEligibility interface {
	EligibleTimingTarget(context.Context, Target) (TimingTarget, error)
}

type TimingSource interface {
	Laps(context.Context, []openf1.Session) ([]openf1.Lap, error)
	Stints(context.Context, []openf1.Session) ([]openf1.Stint, error)
	RequestRecords() []openf1.RequestRecord
}

type TimingSnapshot struct {
	Target          TimingTarget
	Laps            []domain.Lap
	Stints          []domain.Stint
	LapsFetchedAt   time.Time
	StintsFetchedAt time.Time
}

type TimingPublisher interface {
	ReplaceTiming(context.Context, TimingSnapshot) (time.Time, error)
}

type TimingOutcome struct {
	SessionID   domain.PublicID
	SessionKey  int
	LapCount    int
	StintCount  int
	PublishedAt time.Time
}

type TimingImporter struct {
	eligibility TimingEligibility
	source      TimingSource
	publisher   TimingPublisher
}

func NewTimingImporter(eligibility TimingEligibility, source TimingSource, publisher TimingPublisher) *TimingImporter {
	return &TimingImporter{eligibility: eligibility, source: source, publisher: publisher}
}

func (i *TimingImporter) ImportTiming(ctx context.Context, requested Target) (TimingOutcome, error) {
	target, err := i.eligibility.EligibleTimingTarget(ctx, requested)
	if err != nil {
		return TimingOutcome{}, err
	}
	outcome := TimingOutcome{SessionID: target.SessionID, SessionKey: target.SessionKey}
	session := openf1.Session{
		MeetingKey: target.MeetingKey, SessionKey: target.SessionKey, SessionName: "Race", SessionType: "Race",
		DateStart: target.SessionStart, DateEnd: target.SessionEnd,
	}
	laps, err := i.source.Laps(ctx, []openf1.Session{session})
	if err != nil {
		return outcome, fmt.Errorf("fetch Grand Prix laps: %w", err)
	}
	stints, err := i.source.Stints(ctx, []openf1.Session{session})
	if err != nil {
		return outcome, fmt.Errorf("fetch Grand Prix stints: %w", err)
	}

	snapshot, transformErr := TransformTiming(target, laps, stints)
	outcome.LapCount = len(snapshot.Laps)
	outcome.StintCount = len(snapshot.Stints)
	if transformErr != nil {
		return outcome, transformErr
	}
	for _, record := range i.source.RequestRecords() {
		if record.ResponseStatus < 200 || record.ResponseStatus >= 300 {
			continue
		}
		switch record.Endpoint {
		case "laps":
			snapshot.LapsFetchedAt = record.FetchedAt
		case "stints":
			snapshot.StintsFetchedAt = record.FetchedAt
		}
	}
	publishedAt, err := i.publisher.ReplaceTiming(ctx, snapshot)
	if err != nil {
		return outcome, fmt.Errorf("publish Grand Prix timing: %w", err)
	}
	outcome.PublishedAt = publishedAt
	return outcome, nil
}

func TransformTiming(target TimingTarget, sourceLaps []openf1.Lap, sourceStints []openf1.Stint) (TimingSnapshot, error) {
	snapshot := TimingSnapshot{Target: target, Laps: make([]domain.Lap, 0, len(sourceLaps)), Stints: make([]domain.Stint, 0, len(sourceStints))}
	problems := make([]TransformError, 0)
	seenLaps := make(map[[2]int]bool, len(sourceLaps))
	lapIndexes := make(map[[2]int]int, len(sourceLaps))
	for _, source := range sourceLaps {
		entryID, known := target.EntryIDsByNumber[source.DriverNumber]
		key := [2]int{source.DriverNumber, source.LapNumber}
		switch {
		case source.MeetingKey != target.MeetingKey || source.SessionKey != target.SessionKey:
			problems = append(problems, timingProblem("lap_scope_mismatch", "lap", source.DriverNumber, "lap does not belong to the Grand Prix session"))
		case !known:
			problems = append(problems, timingProblem("lap_without_entry", "lap", source.DriverNumber, "lap has no published session entry"))
		case source.LapNumber <= 0:
			problems = append(problems, timingProblem("invalid_lap_number", "lap", source.DriverNumber, "lap number must be positive"))
		case seenLaps[key]:
			problems = append(problems, timingProblem("duplicate_lap", "lap", source.DriverNumber, "driver lap number is duplicated"))
		default:
			seenLaps[key] = true
			snapshot.Laps = append(snapshot.Laps, domain.Lap{SessionEntryID: entryID, SourceDriverNumber: source.DriverNumber, LapNumber: source.LapNumber, DurationMicroseconds: source.LapDuration.Microseconds, IsPitOutLap: source.IsPitOutLap})
			lapIndexes[key] = len(snapshot.Laps) - 1
		}
	}

	seenStints := make(map[[2]int]bool, len(sourceStints))
	for _, source := range sourceStints {
		entryID, known := target.EntryIDsByNumber[source.DriverNumber]
		key := [2]int{source.DriverNumber, source.StintNumber}
		switch {
		case source.MeetingKey != target.MeetingKey || source.SessionKey != target.SessionKey:
			problems = append(problems, timingProblem("stint_scope_mismatch", "stint", source.DriverNumber, "stint does not belong to the Grand Prix session"))
		case !known:
			problems = append(problems, timingProblem("stint_without_entry", "stint", source.DriverNumber, "stint has no published session entry"))
		case source.StintNumber <= 0:
			problems = append(problems, timingProblem("invalid_stint_number", "stint", source.DriverNumber, "stint number must be positive"))
		case seenStints[key]:
			problems = append(problems, timingProblem("duplicate_stint", "stint", source.DriverNumber, "driver stint number is duplicated"))
		case source.LapStart != nil && *source.LapStart <= 0, source.LapEnd != nil && *source.LapEnd <= 0,
			source.LapStart != nil && source.LapEnd != nil && *source.LapEnd < *source.LapStart:
			problems = append(problems, timingProblem("invalid_stint_endpoints", "stint", source.DriverNumber, "stint endpoints are invalid"))
		default:
			seenStints[key] = true
			snapshot.Stints = append(snapshot.Stints, domain.Stint{SessionEntryID: entryID, SourceDriverNumber: source.DriverNumber, StintNumber: source.StintNumber, Compound: source.Compound, LapStart: source.LapStart, LapEnd: source.LapEnd})
			if source.LapStart != nil {
				if index, ok := lapIndexes[[2]int{source.DriverNumber, *source.LapStart}]; ok {
					snapshot.Laps[index].IsStintStart = true
				}
			}
			if source.LapEnd != nil {
				if index, ok := lapIndexes[[2]int{source.DriverNumber, *source.LapEnd}]; ok {
					snapshot.Laps[index].IsStintEnd = true
				}
			}
		}
	}
	if len(problems) != 0 {
		return snapshot, &QuarantineError{Errors: problems}
	}
	return snapshot, nil
}

func timingProblem(code, entity string, driverNumber int, message string) TransformError {
	return TransformError{Code: code, Entity: entity, DriverNumber: driverNumber, Message: message}
}
