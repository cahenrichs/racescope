package ingest

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/clint/f1/backend/internal/openf1"
)

func TestTransformWeekend(t *testing.T) {
	t.Parallel()

	snapshot, err := TransformWeekend(Target{Season: 2024, MeetingKey: 1235}, monacoMeetings(), monacoSessions())
	if err != nil {
		t.Fatalf("TransformWeekend() error = %v", err)
	}

	meeting := snapshot.Weekend.Meeting
	if meeting.StableKey != "2024-monaco-grand-prix" || meeting.Circuit.StableKey != "circuit-de-monaco" {
		t.Fatalf("meeting = %+v", meeting)
	}
	if got := len(snapshot.Weekend.Sessions); got != 5 {
		t.Fatalf("sessions = %d, want 5", got)
	}
	wantNames := []string{"Practice 1", "Practice 2", "Practice 3", "Qualifying", "Race"}
	gotNames := make([]string, 0, 5)
	for _, session := range snapshot.Weekend.Sessions {
		gotNames = append(gotNames, session.Name)
		if session.PublicID == "" || session.StableKey == "" {
			t.Fatalf("session lacks stable identity: %+v", session)
		}
	}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("session order = %v, want %v", gotNames, wantNames)
	}
	if snapshot.Weekend.GrandPrixSessionID != snapshot.Weekend.Sessions[4].PublicID {
		t.Fatalf("Grand Prix session ID = %q, want %q", snapshot.Weekend.GrandPrixSessionID, snapshot.Weekend.Sessions[4].PublicID)
	}

	wantMeetingID, _ := domain.NewPublicID(domain.EntityMeeting, "2024-monaco-grand-prix")
	if meeting.PublicID != wantMeetingID {
		t.Fatalf("meeting ID = %q, want %q", meeting.PublicID, wantMeetingID)
	}
}

func TestTransformWeekendIDsIgnoreSourceKeysAndOrder(t *testing.T) {
	t.Parallel()

	first, err := TransformWeekend(Target{Season: 2024, MeetingKey: 1235}, monacoMeetings(), monacoSessions())
	if err != nil {
		t.Fatal(err)
	}
	sessions := monacoSessions()
	for left, right := 0, len(sessions)-1; left < right; left, right = left+1, right-1 {
		sessions[left], sessions[right] = sessions[right], sessions[left]
	}
	for index := range sessions {
		sessions[index].SessionKey += 10000
	}
	second, err := TransformWeekend(Target{Season: 2024, MeetingKey: 1235}, monacoMeetings(), sessions)
	if err != nil {
		t.Fatal(err)
	}

	if first.Weekend.Meeting.PublicID != second.Weekend.Meeting.PublicID {
		t.Fatal("meeting public ID changed with source record order")
	}
	for index := range first.Weekend.Sessions {
		if first.Weekend.Sessions[index].PublicID != second.Weekend.Sessions[index].PublicID {
			t.Fatalf("session %d public ID changed with private source keys", index)
		}
	}
}

func TestTransformWeekendRejectsUnreviewedRecords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change func([]openf1.Meeting, []openf1.Session) ([]openf1.Meeting, []openf1.Session)
		want   string
	}{
		{name: "unknown mapping", change: func(m []openf1.Meeting, s []openf1.Session) ([]openf1.Meeting, []openf1.Session) { return m, s }, want: "no reviewed weekend identity"},
		{name: "meeting mismatch", change: func(m []openf1.Meeting, s []openf1.Session) ([]openf1.Meeting, []openf1.Session) {
			m[0].Location = "Monaco"
			return m, s
		}, want: "does not match"},
		{name: "missing session", change: func(m []openf1.Meeting, s []openf1.Session) ([]openf1.Meeting, []openf1.Session) { return m, s[:4] }, want: "expected 5 reviewed sessions"},
		{name: "duplicate session", change: func(m []openf1.Meeting, s []openf1.Session) ([]openf1.Meeting, []openf1.Session) {
			s[4] = s[3]
			s[4].SessionKey = 9999
			return m, s
		}, want: "duplicate session name"},
		{name: "unexpected session", change: func(m []openf1.Meeting, s []openf1.Session) ([]openf1.Meeting, []openf1.Session) {
			s[2].SessionName = "Sprint"
			return m, s
		}, want: "unexpected session name"},
		{name: "wrong type", change: func(m []openf1.Meeting, s []openf1.Session) ([]openf1.Meeting, []openf1.Session) {
			s[3].SessionType = "Race"
			return m, s
		}, want: "does not match reviewed type"},
		{name: "cancelled race", change: func(m []openf1.Meeting, s []openf1.Session) ([]openf1.Meeting, []openf1.Session) {
			s[4].IsCancelled = true
			return m, s
		}, want: "got 0"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			meetings, sessions := test.change(monacoMeetings(), monacoSessions())
			target := Target{Season: 2024, MeetingKey: 1235}
			if test.name == "unknown mapping" {
				target.MeetingKey = 9999
			}
			_, err := TransformWeekend(target, meetings, sessions)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("TransformWeekend() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestImporterReportsFetchFailure(t *testing.T) {
	t.Parallel()

	importer := NewImporter(fakeSource{err: errors.New("source unavailable")})
	_, err := importer.ImportWeekend(context.Background(), Target{Season: 2024, MeetingKey: 1235})
	if err == nil || !strings.Contains(err.Error(), "fetch meetings") {
		t.Fatalf("ImportWeekend() error = %v", err)
	}
}

type fakeSource struct {
	meetings []openf1.Meeting
	sessions []openf1.Session
	err      error
}

func (source fakeSource) Meetings(context.Context, int) ([]openf1.Meeting, error) {
	return source.meetings, source.err
}

func (source fakeSource) Sessions(context.Context, int) ([]openf1.Session, error) {
	return source.sessions, source.err
}

func monacoMeetings() []openf1.Meeting {
	return []openf1.Meeting{{
		CircuitKey: 22, CircuitShortName: "Monte Carlo", CountryCode: "MON", CountryName: "Monaco",
		DateStart: time.Date(2024, time.May, 24, 11, 30, 0, 0, time.UTC),
		DateEnd:   time.Date(2024, time.May, 26, 15, 0, 0, 0, time.UTC),
		Location:  "Monte Carlo", MeetingKey: 1235, MeetingName: "Monaco Grand Prix",
		MeetingOfficialName: "FORMULA 1 GRAND PRIX DE MONACO 2024", Year: 2024,
	}}
}

func monacoSessions() []openf1.Session {
	return []openf1.Session{
		monacoSession(9496, "Practice 1", "Practice", time.Date(2024, time.May, 24, 11, 30, 0, 0, time.UTC), time.Date(2024, time.May, 24, 12, 30, 0, 0, time.UTC)),
		monacoSession(9497, "Practice 2", "Practice", time.Date(2024, time.May, 24, 15, 0, 0, 0, time.UTC), time.Date(2024, time.May, 24, 16, 0, 0, 0, time.UTC)),
		monacoSession(9498, "Practice 3", "Practice", time.Date(2024, time.May, 25, 10, 30, 0, 0, time.UTC), time.Date(2024, time.May, 25, 11, 30, 0, 0, time.UTC)),
		monacoSession(9499, "Qualifying", "Qualifying", time.Date(2024, time.May, 25, 14, 0, 0, 0, time.UTC), time.Date(2024, time.May, 25, 15, 0, 0, 0, time.UTC)),
		monacoSession(9500, "Race", "Race", time.Date(2024, time.May, 26, 13, 0, 0, 0, time.UTC), time.Date(2024, time.May, 26, 15, 0, 0, 0, time.UTC)),
	}
}

func monacoSession(key int, name, sessionType string, start, end time.Time) openf1.Session {
	return openf1.Session{CircuitKey: 22, DateStart: start, DateEnd: end, MeetingKey: 1235, SessionKey: key, SessionName: name, SessionType: sessionType, Year: 2024}
}
