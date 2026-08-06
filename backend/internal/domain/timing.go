package domain

// Lap is one source observation. A nil duration remains source-missing rather than becoming zero.
type Lap struct {
	SessionEntryID       PublicID
	SourceDriverNumber   int
	LapNumber            int
	DurationMicroseconds *int64
	IsPitOutLap          *bool
	IsStintStart         bool
	IsStintEnd           bool
}

// Stint preserves the source-reported context for a numbered driver stint.
type Stint struct {
	SessionEntryID     PublicID
	SourceDriverNumber int
	StintNumber        int
	Compound           *string
	LapStart           *int
	LapEnd             *int
}
