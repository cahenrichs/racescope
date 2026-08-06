package openf1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"time"
)

// Meeting is the subset of an OpenF1 meeting used to identify and display a weekend.
type Meeting struct {
	CircuitKey          int       `json:"circuit_key"`
	CircuitShortName    string    `json:"circuit_short_name"`
	CountryCode         string    `json:"country_code"`
	CountryName         string    `json:"country_name"`
	DateEnd             time.Time `json:"date_end"`
	DateStart           time.Time `json:"date_start"`
	Location            string    `json:"location"`
	MeetingKey          int       `json:"meeting_key"`
	MeetingName         string    `json:"meeting_name"`
	MeetingOfficialName string    `json:"meeting_official_name"`
	Year                int       `json:"year"`
	IsCancelled         bool      `json:"is_cancelled"`
}

// Session is the subset of an OpenF1 session needed to import a weekend.
type Session struct {
	CircuitKey  int       `json:"circuit_key"`
	DateEnd     time.Time `json:"date_end"`
	DateStart   time.Time `json:"date_start"`
	MeetingKey  int       `json:"meeting_key"`
	SessionKey  int       `json:"session_key"`
	SessionName string    `json:"session_name"`
	SessionType string    `json:"session_type"`
	Year        int       `json:"year"`
	IsCancelled bool      `json:"is_cancelled"`
}

// Driver describes a driver's session-specific identity and constructor entry.
type Driver struct {
	DriverNumber int    `json:"driver_number"`
	FirstName    string `json:"first_name"`
	FullName     string `json:"full_name"`
	LastName     string `json:"last_name"`
	MeetingKey   int    `json:"meeting_key"`
	NameAcronym  string `json:"name_acronym"`
	SessionKey   int    `json:"session_key"`
	TeamColour   string `json:"team_colour"`
	TeamName     string `json:"team_name"`
}

// ResultValue preserves whether an OpenF1 field was missing, null, numeric, text, or an array.
// Exactly one of Number, Text, or Numbers is populated. Present distinguishes JSON null from omission.
type ResultValue struct {
	Present bool
	Number  *float64
	Text    *string
	Numbers []*float64
}

func (v *ResultValue) UnmarshalJSON(data []byte) error {
	*v = ResultValue{Present: true}
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return nil
	}

	var number float64
	if err := json.Unmarshal(data, &number); err == nil {
		v.Number = &number
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		v.Text = &text
		return nil
	}

	var numbers []*float64
	if err := json.Unmarshal(data, &numbers); err == nil {
		v.Numbers = numbers
		return nil
	}

	return fmt.Errorf("expected number, string, null, or array of nullable numbers")
}

// SessionResult preserves provider classification flags and nullable/non-numeric values.
type SessionResult struct {
	DNF          bool        `json:"dnf"`
	DNS          bool        `json:"dns"`
	DSQ          bool        `json:"dsq"`
	DriverNumber int         `json:"driver_number"`
	Duration     ResultValue `json:"duration"`
	GapToLeader  ResultValue `json:"gap_to_leader"`
	MeetingKey   int         `json:"meeting_key"`
	NumberOfLaps *int        `json:"number_of_laps"`
	Position     *int        `json:"position"`
	SessionKey   int         `json:"session_key"`
}

// MicrosecondDuration decodes source seconds exactly, without a float64 round trip.
type MicrosecondDuration struct {
	Present      bool
	Microseconds *int64
}

func (d *MicrosecondDuration) UnmarshalJSON(data []byte) error {
	*d = MicrosecondDuration{Present: true}
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	seconds, ok := new(big.Rat).SetString(string(data))
	if !ok {
		return fmt.Errorf("expected duration seconds as a number or null")
	}
	microseconds := new(big.Rat).Mul(seconds, big.NewRat(1_000_000, 1))
	if !microseconds.IsInt() || !microseconds.Num().IsInt64() || microseconds.Sign() < 0 {
		return fmt.Errorf("duration must be a non-negative whole number of microseconds")
	}
	value := microseconds.Num().Int64()
	d.Microseconds = &value
	return nil
}

// Lap is the source timing observation retained for one driver and lap number.
type Lap struct {
	MeetingKey   int                 `json:"meeting_key"`
	SessionKey   int                 `json:"session_key"`
	DriverNumber int                 `json:"driver_number"`
	LapNumber    int                 `json:"lap_number"`
	LapDuration  MicrosecondDuration `json:"lap_duration"`
	IsPitOutLap  *bool               `json:"is_pit_out_lap"`
}

// Stint is the source-reported tire context and endpoints for one driver stint.
type Stint struct {
	MeetingKey   int     `json:"meeting_key"`
	SessionKey   int     `json:"session_key"`
	DriverNumber int     `json:"driver_number"`
	StintNumber  int     `json:"stint_number"`
	Compound     *string `json:"compound"`
	LapStart     *int    `json:"lap_start"`
	LapEnd       *int    `json:"lap_end"`
}
