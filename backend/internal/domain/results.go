package domain

// ResultValueKind preserves missing, null, zero-valued, text, and array source values.
type ResultValueKind string

const (
	ResultValueMissing ResultValueKind = "missing"
	ResultValueNull    ResultValueKind = "null"
	ResultValueNumber  ResultValueKind = "number"
	ResultValueText    ResultValueKind = "text"
	ResultValueNumbers ResultValueKind = "numbers"
)

type ResultValue struct {
	Kind    ResultValueKind
	Number  float64
	Text    string
	Numbers []*float64
}

type Driver struct {
	PublicID    PublicID
	StableKey   string
	FirstName   string
	LastName    string
	FullName    string
	NameAcronym string
}

type ConstructorEntrant struct {
	PublicID  PublicID
	StableKey string
	Season    Season
	Name      string
}

type SessionEntry struct {
	PublicID     PublicID
	StableKey    string
	SessionID    PublicID
	Driver       Driver
	Constructor  ConstructorEntrant
	DriverNumber int
	TeamColour   string
}

type SessionResult struct {
	PublicID       PublicID
	StableKey      string
	SessionEntryID PublicID
	Classification ClassificationState
	Position       *int
	NumberOfLaps   *int
	Duration       ResultValue
	GapToLeader    ResultValue
	SourceOrder    int
}
