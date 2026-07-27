package openf1

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// ResultValue preserves OpenF1's number, text, null, or qualifying-array result values.
// Exactly one of Number, Text, or Numbers is populated; all nil represents JSON null.
type ResultValue struct {
	Number  *float64
	Text    *string
	Numbers []*float64
}

func (v *ResultValue) UnmarshalJSON(data []byte) error {
	*v = ResultValue{}
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
