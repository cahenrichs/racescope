package ingest

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/clint/f1/backend/internal/openf1"
)

func TestTransformRacePreservesClassificationAndValueStates(t *testing.T) {
	t.Parallel()

	drivers := []openf1.Driver{
		raceDriver(16, "Charles", "Leclerc", "Charles LECLERC", "LEC", "Ferrari"),
		raceDriver(81, "Oscar", "Piastri", "Oscar PIASTRI", "PIA", "McLaren"),
		raceDriver(1, "Max", "Verstappen", "Max VERSTAPPEN", "VER", "Red Bull Racing"),
		raceDriver(2, "Logan", "Sargeant", "Logan SARGEANT", "SAR", "Williams"),
		raceDriver(3, "Daniel", "Ricciardo", "Daniel RICCIARDO", "RIC", "RB"),
		raceDriver(4, "Lando", "Norris", "Lando NORRIS", "NOR", "McLaren"),
	}
	zero := 0.0
	one := 1
	results := []openf1.SessionResult{
		{DriverNumber: 16, MeetingKey: 1236, SessionKey: 9500, Position: &one, Duration: openf1.ResultValue{Present: true, Number: &zero}, GapToLeader: openf1.ResultValue{Present: true, Number: &zero}},
		{DriverNumber: 81, MeetingKey: 1236, SessionKey: 9500, DNS: true, Duration: openf1.ResultValue{Present: true}, GapToLeader: openf1.ResultValue{Present: true}},
		{DriverNumber: 1, MeetingKey: 1236, SessionKey: 9500, DNF: true},
		{DriverNumber: 2, MeetingKey: 1236, SessionKey: 9500, DSQ: true},
		{DriverNumber: 3, MeetingKey: 1236, SessionKey: 9500},
	}

	data, err := TransformRace(Target{Season: 2024, MeetingKey: 1236}, testSeason(), testRaceSession(), 9500, drivers, results)
	var quarantined *QuarantineError
	if !errors.As(err, &quarantined) || len(quarantined.Errors) != 1 || quarantined.Errors[0].Code != "missing_result" {
		t.Fatalf("TransformRace() error = %+v, want one missing_result quarantine", err)
	}
	if len(data.Entries) != 6 || len(data.Results) != 6 {
		t.Fatalf("entries/results = %d/%d, want 6/6", len(data.Entries), len(data.Results))
	}

	byNumber := resultsByDriverNumber(data)
	wantStates := map[int]domain.ClassificationState{
		16: domain.ClassificationOrdinary,
		81: domain.ClassificationDNS,
		1:  domain.ClassificationDNF,
		2:  domain.ClassificationDSQ,
		3:  domain.ClassificationUnknown,
		4:  domain.ClassificationMissing,
	}
	for number, want := range wantStates {
		if got := byNumber[number].Classification; got != want {
			t.Errorf("driver %d state = %q, want %q", number, got, want)
		}
	}
	if got := byNumber[16].GapToLeader; got.Kind != domain.ResultValueNumber || got.Number != 0 {
		t.Errorf("zero gap = %+v", got)
	}
	if got := byNumber[81].Duration.Kind; got != domain.ResultValueNull {
		t.Errorf("null duration kind = %q", got)
	}
	if got := byNumber[3].Duration.Kind; got != domain.ResultValueMissing {
		t.Errorf("omitted duration kind = %q", got)
	}
	if got := byNumber[4].Duration.Kind; got != domain.ResultValueMissing {
		t.Errorf("missing classification duration kind = %q", got)
	}
}

func TestTransformRaceQuarantinesAllIdentityMismatches(t *testing.T) {
	t.Parallel()

	drivers := []openf1.Driver{
		raceDriver(16, "Charles", "Leclerc", "Charles LECLERC", "LEC", "Ferrari"),
		raceDriver(99, "Mystery", "Driver", "Mystery DRIVER", "MYS", "Unknown Team"),
		raceDriver(81, "Oscar", "Piastri", "Oscar PIASTRI", "BAD", "Unknown Constructor"),
	}
	one := 1
	data, err := TransformRace(Target{Season: 2024, MeetingKey: 1236}, testSeason(), testRaceSession(), 9500, drivers,
		[]openf1.SessionResult{{DriverNumber: 16, MeetingKey: 1236, SessionKey: 9500, Position: &one}})
	var quarantined *QuarantineError
	if !errors.As(err, &quarantined) {
		t.Fatalf("TransformRace() error = %v, want QuarantineError", err)
	}
	if len(data.Entries) != 1 || data.Entries[0].DriverNumber != 16 {
		t.Fatalf("valid entries = %+v", data.Entries)
	}

	codes := make([]string, 0, len(quarantined.Errors))
	for _, problem := range quarantined.Errors {
		codes = append(codes, problem.Code)
	}
	for _, want := range []string{"unknown_driver", "unknown_constructor", "driver_acronym_mismatch"} {
		if !slices.Contains(codes, want) {
			t.Errorf("quarantine codes = %v, missing %q", codes, want)
		}
	}
	if len(quarantined.Errors) < 4 {
		t.Fatalf("quarantine errors = %v, want all mismatches", quarantined.Errors)
	}
}

func TestImporterFetchesDetailsForGrandPrixOnly(t *testing.T) {
	t.Parallel()

	one := 1
	source := &recordingSource{
		meetings: monacoMeetings(), sessions: monacoSessions(),
		drivers: []openf1.Driver{raceDriver(16, "Charles", "Leclerc", "Charles LECLERC", "LEC", "Ferrari")},
		results: []openf1.SessionResult{{DriverNumber: 16, MeetingKey: 1236, SessionKey: 9500, Position: &one}},
	}
	publisher := &recordingPublisher{}
	importer := NewImporter(source, publisher)
	outcome, err := importer.ImportWeekend(context.Background(), Target{Season: 2024, MeetingKey: 1236})
	if err != nil {
		t.Fatalf("ImportWeekend() error = %v", err)
	}
	if outcome.SessionCount != 5 || outcome.EntryCount != 1 || outcome.ResultCount != 1 {
		t.Fatalf("outcome = %+v", outcome)
	}
	if outcome.PublishedAt.IsZero() || !outcome.PublishedAt.Equal(publisher.publishedAt) {
		t.Fatalf("published outcome = %+v", outcome)
	}
	if len(source.driverSessions) != 1 || source.driverSessions[0].SessionName != "Race" || len(source.resultSessions) != 1 || source.resultSessions[0].SessionName != "Race" {
		t.Fatalf("detail requests used drivers=%+v results=%+v", source.driverSessions, source.resultSessions)
	}
	if publisher.calls != 1 || publisher.snapshot.MeetingSourceKey != 1236 || len(publisher.snapshot.Entries) != 1 {
		t.Fatalf("publisher = %+v", publisher)
	}
	if publisher.snapshot.SourceFetchedAt.Meetings.IsZero() || publisher.snapshot.SourceFetchedAt.Results.IsZero() {
		t.Fatalf("source fetch timestamps = %+v", publisher.snapshot.SourceFetchedAt)
	}
}

func TestImporterBlocksQuarantinedWeekend(t *testing.T) {
	t.Parallel()

	source := &recordingSource{
		meetings: monacoMeetings(), sessions: monacoSessions(),
		drivers: []openf1.Driver{
			raceDriver(99, "First", "Unknown", "First UNKNOWN", "ONE", "Unknown One"),
			raceDriver(98, "Second", "Unknown", "Second UNKNOWN", "TWO", "Unknown Two"),
		},
	}
	outcome, err := NewImporter(source).ImportWeekend(context.Background(), Target{Season: 2024, MeetingKey: 1236})
	var quarantined *QuarantineError
	if !errors.As(err, &quarantined) {
		t.Fatalf("ImportWeekend() error = %v, want quarantine", err)
	}
	if outcome.ErrorCount != len(quarantined.Errors) || outcome.ErrorCount < 4 {
		t.Fatalf("outcome/errors = %+v/%+v", outcome, quarantined.Errors)
	}
}

func TestImporterRetriesAfterQuarantine(t *testing.T) {
	t.Parallel()

	one := 1
	source := &recordingSource{
		meetings: monacoMeetings(), sessions: monacoSessions(),
		drivers: []openf1.Driver{raceDriver(16, "Charles", "Leclerc", "Charles LECLERC", "LEC", "Ferrari")},
		results: []openf1.SessionResult{{DriverNumber: 16, MeetingKey: 1236, SessionKey: 9500, Position: &one}},
	}
	publisher := &recordingPublisher{}
	importer := NewImporter(source, publisher)
	if _, err := importer.ImportWeekend(context.Background(), Target{Season: 2024, MeetingKey: 1236}); err != nil {
		t.Fatalf("initial ImportWeekend() error = %v", err)
	}
	publishedSnapshot := publisher.snapshot

	source.drivers = []openf1.Driver{raceDriver(99, "Mystery", "Driver", "Mystery DRIVER", "MYS", "Unknown Team")}
	source.results = nil
	if _, err := importer.ImportWeekend(context.Background(), Target{Season: 2024, MeetingKey: 1236}); err == nil {
		t.Fatal("quarantined ImportWeekend() error = nil")
	}
	if publisher.calls != 1 || publisher.snapshot.Entries[0].Driver.PublicID != publishedSnapshot.Entries[0].Driver.PublicID {
		t.Fatalf("quarantine replaced the published snapshot: %+v", publisher)
	}

	source.drivers = []openf1.Driver{raceDriver(16, "Charles", "Leclerc", "Charles LECLERC", "LEC", "Ferrari")}
	source.results = []openf1.SessionResult{{DriverNumber: 16, MeetingKey: 1236, SessionKey: 9500, Position: &one}}
	outcome, err := importer.ImportWeekend(context.Background(), Target{Season: 2024, MeetingKey: 1236})
	if err != nil {
		t.Fatalf("retry ImportWeekend() error = %v", err)
	}
	if publisher.calls != 2 || outcome.PublishedAt.IsZero() {
		t.Fatalf("retry outcome/publisher = %+v/%+v", outcome, publisher)
	}
}

type recordingSource struct {
	meetings       []openf1.Meeting
	sessions       []openf1.Session
	drivers        []openf1.Driver
	results        []openf1.SessionResult
	driverSessions []openf1.Session
	resultSessions []openf1.Session
}

type recordingPublisher struct {
	calls       int
	snapshot    Snapshot
	publishedAt time.Time
	err         error
}

func (publisher *recordingPublisher) ReplaceWeekend(_ context.Context, snapshot Snapshot) (time.Time, error) {
	publisher.calls++
	publisher.snapshot = snapshot
	if publisher.publishedAt.IsZero() {
		publisher.publishedAt = time.Date(2026, time.July, 30, 12, 1, 0, 0, time.UTC)
	}
	return publisher.publishedAt, publisher.err
}

func (source *recordingSource) Meetings(context.Context, int) ([]openf1.Meeting, error) {
	return source.meetings, nil
}

func (source *recordingSource) Sessions(context.Context, int) ([]openf1.Session, error) {
	return source.sessions, nil
}

func (source *recordingSource) Drivers(_ context.Context, sessions []openf1.Session) ([]openf1.Driver, error) {
	source.driverSessions = append([]openf1.Session(nil), sessions...)
	return source.drivers, nil
}

func (source *recordingSource) SessionResults(_ context.Context, sessions []openf1.Session) ([]openf1.SessionResult, error) {
	source.resultSessions = append([]openf1.Session(nil), sessions...)
	return source.results, nil
}

func (source *recordingSource) RequestRecords() []openf1.RequestRecord {
	base := time.Date(2026, time.July, 30, 11, 0, 0, 0, time.UTC)
	return []openf1.RequestRecord{
		{Endpoint: "meetings", ResponseStatus: 200, FetchedAt: base},
		{Endpoint: "sessions", ResponseStatus: 200, FetchedAt: base.Add(time.Minute)},
		{Endpoint: "drivers", ResponseStatus: 200, FetchedAt: base.Add(2 * time.Minute)},
		{Endpoint: "session_result", ResponseStatus: 200, FetchedAt: base.Add(3 * time.Minute)},
	}
}

func raceDriver(number int, firstName, lastName, fullName, acronym, team string) openf1.Driver {
	return openf1.Driver{
		DriverNumber: number, FirstName: firstName, LastName: lastName, FullName: fullName,
		NameAcronym: acronym, TeamName: team, TeamColour: "ABCDEF", MeetingKey: 1236, SessionKey: 9500,
	}
}

func testSeason() domain.Season {
	id, _ := domain.NewPublicID(domain.EntitySeason, "2024")
	return domain.Season{PublicID: id, Year: 2024}
}

func testRaceSession() domain.Session {
	id, _ := domain.NewPublicID(domain.EntitySession, "2024-monaco-grand-prix:race")
	return domain.Session{PublicID: id, StableKey: "2024-monaco-grand-prix:race", Name: "Race", Type: "Race"}
}

func resultsByDriverNumber(data RaceData) map[int]domain.SessionResult {
	entries := make(map[domain.PublicID]int, len(data.Entries))
	for _, entry := range data.Entries {
		entries[entry.PublicID] = entry.DriverNumber
	}
	results := make(map[int]domain.SessionResult, len(data.Results))
	for _, result := range data.Results {
		results[entries[result.SessionEntryID]] = result
	}
	return results
}
