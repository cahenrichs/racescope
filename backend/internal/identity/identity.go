package identity

import "time"

// DriverMapping identifies a driver by the exact source identity for a season.
type DriverMapping struct {
	CanonicalKey    string
	DisplayName     string
	ExpectedAcronym string
	ExpectedNumber  int
}

// ConstructorMapping identifies a season-specific constructor entrant.
type ConstructorMapping struct {
	CanonicalKey string
	DisplayName  string
}

// SessionMapping defines the exact source name and type used in one weekend.
type SessionMapping struct {
	Name            string
	Type            string
	CanonicalSuffix string
}

// WeekendMapping contains reviewed source fields and stable identity inputs.
type WeekendMapping struct {
	Season              int
	SourceMeetingKey    int
	MeetingCanonicalKey string
	CircuitCanonicalKey string
	CircuitKey          int
	CircuitShortName    string
	CountryCode         string
	CountryName         string
	Location            string
	MeetingName         string
	MeetingOfficialName string
	DateStart           time.Time
	DateEnd             time.Time
	IsCancelled         bool
	Sessions            map[string]SessionMapping
}

// Weekend returns the reviewed mapping for an exact season and source meeting key.
func Weekend(season, meetingKey int) (WeekendMapping, bool) {
	mapping, ok := weekendMappings[weekendLookup{season: season, meetingKey: meetingKey}]
	return mapping, ok
}

// Driver returns a reviewed driver mapping using an exact, case-sensitive source full name.
func Driver(season int, fullName string) (DriverMapping, bool) {
	mapping, ok := driverMappings[driverLookup{season: season, fullName: fullName}]
	return mapping, ok
}

// Constructor returns a reviewed constructor mapping using an exact, case-sensitive source team name.
func Constructor(season int, teamName string) (ConstructorMapping, bool) {
	mapping, ok := constructorMappings[constructorLookup{season: season, teamName: teamName}]
	return mapping, ok
}

type weekendLookup struct {
	season     int
	meetingKey int
}

type driverLookup struct {
	season   int
	fullName string
}

type constructorLookup struct {
	season   int
	teamName string
}
