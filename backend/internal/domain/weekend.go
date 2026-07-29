package domain

import "time"

// Season is the stable domain identity for a championship year.
type Season struct {
	PublicID PublicID
	Year     int
}

// Circuit is a provider-independent circuit record.
type Circuit struct {
	PublicID    PublicID
	StableKey   string
	ShortName   string
	CountryCode string
	CountryName string
	Location    string
}

// Meeting is a completed race weekend and its circuit.
type Meeting struct {
	PublicID     PublicID
	StableKey    string
	Season       Season
	Circuit      Circuit
	Name         string
	OfficialName string
	DateStart    time.Time
	DateEnd      time.Time
	IsCancelled  bool
}

// Session is one scheduled session within a meeting.
type Session struct {
	PublicID    PublicID
	StableKey   string
	Name        string
	Type        string
	DateStart   time.Time
	DateEnd     time.Time
	IsCancelled bool
}

// Weekend contains complete meeting and session metadata.
type Weekend struct {
	Meeting            Meeting
	Sessions           []Session
	GrandPrixSessionID PublicID
}
