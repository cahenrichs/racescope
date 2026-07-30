package ingest

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/clint/f1/backend/internal/identity"
	"github.com/clint/f1/backend/internal/openf1"
)

// Target selects one reviewed weekend for import.
type Target struct {
	Season     int
	MeetingKey int
}

// Source is the OpenF1 read boundary needed for weekend metadata import.
type Source interface {
	Meetings(context.Context, int) ([]openf1.Meeting, error)
	Sessions(context.Context, int) ([]openf1.Session, error)
	Drivers(context.Context, []openf1.Session) ([]openf1.Driver, error)
	SessionResults(context.Context, []openf1.Session) ([]openf1.SessionResult, error)
}

// Snapshot keeps private source keys beside provider-independent domain records.
type Snapshot struct {
	Weekend           domain.Weekend
	MeetingSourceKey  int
	SessionSourceKeys map[domain.PublicID]int
	Drivers           []domain.Driver
	Constructors      []domain.ConstructorEntrant
	Entries           []domain.SessionEntry
	Results           []domain.SessionResult
}

// Outcome describes a completed fetch and transformation. Publication is a later import step.
type Outcome struct {
	MeetingID     domain.PublicID
	SessionCount  int
	EntryCount    int
	ResultCount   int
	ErrorCount    int
	TransformedAt time.Time
}

// Importer fetches and transforms one reviewed weekend.
type Importer struct {
	source Source
	now    func() time.Time
}

func NewImporter(source Source) *Importer {
	return &Importer{source: source, now: time.Now}
}

func (i *Importer) ImportWeekend(ctx context.Context, target Target) (Outcome, error) {
	meetings, err := i.source.Meetings(ctx, target.Season)
	if err != nil {
		return Outcome{}, fmt.Errorf("fetch meetings: %w", err)
	}
	sessions, err := i.source.Sessions(ctx, target.MeetingKey)
	if err != nil {
		return Outcome{}, fmt.Errorf("fetch sessions: %w", err)
	}

	snapshot, err := TransformWeekend(target, meetings, sessions)
	if err != nil {
		return Outcome{}, err
	}

	raceSourceKey := snapshot.SessionSourceKeys[snapshot.Weekend.GrandPrixSessionID]
	var raceSource openf1.Session
	for _, session := range sessions {
		if session.SessionKey == raceSourceKey {
			raceSource = session
			break
		}
	}
	drivers, err := i.source.Drivers(ctx, []openf1.Session{raceSource})
	if err != nil {
		return Outcome{}, fmt.Errorf("fetch Grand Prix entries: %w", err)
	}
	results, err := i.source.SessionResults(ctx, []openf1.Session{raceSource})
	if err != nil {
		return Outcome{}, fmt.Errorf("fetch Grand Prix classification: %w", err)
	}

	var raceSession domain.Session
	for _, session := range snapshot.Weekend.Sessions {
		if session.PublicID == snapshot.Weekend.GrandPrixSessionID {
			raceSession = session
			break
		}
	}
	raceData, transformErr := TransformRace(target, snapshot.Weekend.Meeting.Season, raceSession, raceSourceKey, drivers, results)
	snapshot.Drivers = raceData.Drivers
	snapshot.Constructors = raceData.Constructors
	snapshot.Entries = raceData.Entries
	snapshot.Results = raceData.Results
	outcome := Outcome{
		MeetingID:     snapshot.Weekend.Meeting.PublicID,
		SessionCount:  len(snapshot.Weekend.Sessions),
		EntryCount:    len(snapshot.Entries),
		ResultCount:   len(snapshot.Results),
		TransformedAt: i.now().UTC(),
	}
	if transformErr != nil {
		var quarantined *QuarantineError
		if errors.As(transformErr, &quarantined) {
			outcome.ErrorCount = len(quarantined.Errors)
		}
		return outcome, transformErr
	}
	return outcome, nil
}

// TransformWeekend validates exact reviewed identities and creates deterministic domain records.
func TransformWeekend(target Target, meetings []openf1.Meeting, sessions []openf1.Session) (Snapshot, error) {
	mapping, ok := identity.Weekend(target.Season, target.MeetingKey)
	if !ok {
		return Snapshot{}, fmt.Errorf("no reviewed weekend identity for season %d meeting %d", target.Season, target.MeetingKey)
	}

	meeting, err := selectMeeting(meetings, target.MeetingKey)
	if err != nil {
		return Snapshot{}, err
	}
	if err := validateMeeting(meeting, mapping); err != nil {
		return Snapshot{}, err
	}

	seasonID, err := domain.NewPublicID(domain.EntitySeason, fmt.Sprint(mapping.Season))
	if err != nil {
		return Snapshot{}, err
	}
	circuitID, err := domain.NewPublicID(domain.EntityCircuit, mapping.CircuitCanonicalKey)
	if err != nil {
		return Snapshot{}, err
	}
	meetingID, err := domain.NewPublicID(domain.EntityMeeting, mapping.MeetingCanonicalKey)
	if err != nil {
		return Snapshot{}, err
	}

	domainSessions, sourceKeys, grandPrixID, err := transformSessions(sessions, mapping)
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		Weekend: domain.Weekend{
			Meeting: domain.Meeting{
				PublicID:  meetingID,
				StableKey: mapping.MeetingCanonicalKey,
				Season:    domain.Season{PublicID: seasonID, Year: mapping.Season},
				Circuit: domain.Circuit{
					PublicID: circuitID, StableKey: mapping.CircuitCanonicalKey,
					ShortName: meeting.CircuitShortName, CountryCode: meeting.CountryCode,
					CountryName: meeting.CountryName, Location: meeting.Location,
				},
				Name: meeting.MeetingName, OfficialName: meeting.MeetingOfficialName,
				DateStart: meeting.DateStart, DateEnd: meeting.DateEnd, IsCancelled: meeting.IsCancelled,
			},
			Sessions: domainSessions, GrandPrixSessionID: grandPrixID,
		},
		MeetingSourceKey:  meeting.MeetingKey,
		SessionSourceKeys: sourceKeys,
	}, nil
}

func selectMeeting(meetings []openf1.Meeting, meetingKey int) (openf1.Meeting, error) {
	var selected openf1.Meeting
	count := 0
	for _, meeting := range meetings {
		if meeting.MeetingKey == meetingKey {
			selected = meeting
			count++
		}
	}
	if count != 1 {
		return openf1.Meeting{}, fmt.Errorf("expected exactly one source meeting %d, got %d", meetingKey, count)
	}
	return selected, nil
}

func validateMeeting(got openf1.Meeting, want identity.WeekendMapping) error {
	if got.Year != want.Season || got.CircuitKey != want.CircuitKey ||
		got.CircuitShortName != want.CircuitShortName || got.CountryCode != want.CountryCode ||
		got.CountryName != want.CountryName || got.Location != want.Location ||
		got.MeetingName != want.MeetingName || got.MeetingOfficialName != want.MeetingOfficialName ||
		!got.DateStart.Equal(want.DateStart) || !got.DateEnd.Equal(want.DateEnd) ||
		got.IsCancelled != want.IsCancelled {
		return fmt.Errorf("source meeting %d does not match its reviewed identity", got.MeetingKey)
	}
	if got.DateStart.IsZero() || got.DateEnd.Before(got.DateStart) {
		return fmt.Errorf("source meeting %d has invalid start or end times", got.MeetingKey)
	}
	return nil
}

func transformSessions(sessions []openf1.Session, mapping identity.WeekendMapping) ([]domain.Session, map[domain.PublicID]int, domain.PublicID, error) {
	if len(sessions) != len(mapping.Sessions) {
		return nil, nil, "", fmt.Errorf("expected %d reviewed sessions, got %d", len(mapping.Sessions), len(sessions))
	}

	seenNames := make(map[string]bool, len(sessions))
	seenKeys := make(map[int]bool, len(sessions))
	domainSessions := make([]domain.Session, 0, len(sessions))
	sourceKeys := make(map[domain.PublicID]int, len(sessions))
	var grandPrixID domain.PublicID
	grandPrixCount := 0

	for _, session := range sessions {
		if session.MeetingKey != mapping.SourceMeetingKey || session.Year != mapping.Season || session.CircuitKey != mapping.CircuitKey {
			return nil, nil, "", fmt.Errorf("session %d does not belong to the reviewed meeting", session.SessionKey)
		}
		if session.SessionKey <= 0 || seenKeys[session.SessionKey] {
			return nil, nil, "", fmt.Errorf("session source key %d is invalid or duplicated", session.SessionKey)
		}
		seenKeys[session.SessionKey] = true
		if seenNames[session.SessionName] {
			return nil, nil, "", fmt.Errorf("duplicate session name %q", session.SessionName)
		}
		seenNames[session.SessionName] = true

		reviewed, ok := mapping.Sessions[session.SessionName]
		if !ok {
			return nil, nil, "", fmt.Errorf("unexpected session name %q", session.SessionName)
		}
		if session.SessionType != reviewed.Type {
			return nil, nil, "", fmt.Errorf("session %q type %q does not match reviewed type %q", session.SessionName, session.SessionType, reviewed.Type)
		}
		if session.DateStart.IsZero() || session.DateEnd.Before(session.DateStart) {
			return nil, nil, "", fmt.Errorf("session %q has invalid start or end times", session.SessionName)
		}

		stableKey := mapping.MeetingCanonicalKey + ":" + reviewed.CanonicalSuffix
		publicID, err := domain.NewPublicID(domain.EntitySession, stableKey)
		if err != nil {
			return nil, nil, "", err
		}
		domainSessions = append(domainSessions, domain.Session{
			PublicID: publicID, StableKey: stableKey, Name: session.SessionName, Type: session.SessionType,
			DateStart: session.DateStart, DateEnd: session.DateEnd, IsCancelled: session.IsCancelled,
		})
		sourceKeys[publicID] = session.SessionKey
		if !session.IsCancelled && session.SessionName == "Race" && session.SessionType == "Race" {
			grandPrixID = publicID
			grandPrixCount++
		}
	}

	for name := range mapping.Sessions {
		if !seenNames[name] {
			return nil, nil, "", fmt.Errorf("missing reviewed session %q", name)
		}
	}
	if grandPrixCount != 1 {
		return nil, nil, "", fmt.Errorf("expected exactly one non-cancelled Grand Prix race session, got %d", grandPrixCount)
	}

	sort.Slice(domainSessions, func(i, j int) bool {
		if domainSessions[i].DateStart.Equal(domainSessions[j].DateStart) {
			return domainSessions[i].Name < domainSessions[j].Name
		}
		return domainSessions[i].DateStart.Before(domainSessions[j].DateStart)
	})
	return domainSessions, sourceKeys, grandPrixID, nil
}
