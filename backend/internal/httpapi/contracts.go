package httpapi

import "time"

type coverageResponse struct {
	Status          string    `json:"status"`
	SourceFetchedAt time.Time `json:"sourceFetchedAt"`
	PublishedAt     time.Time `json:"publishedAt"`
}

type circuitResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CountryCode string `json:"countryCode"`
	CountryName string `json:"countryName"`
	Location    string `json:"location"`
}

type raceSummaryResponse struct {
	ID           string           `json:"id"`
	Season       int              `json:"season"`
	Name         string           `json:"name"`
	OfficialName string           `json:"officialName"`
	Circuit      circuitResponse  `json:"circuit"`
	StartAt      time.Time        `json:"startAt"`
	EndAt        time.Time        `json:"endAt"`
	Coverage     coverageResponse `json:"coverage"`
}

type dashboardResponse struct {
	Races    []raceSummaryResponse `json:"races"`
	Coverage *coverageResponse     `json:"coverage"`
}

type sessionSummaryResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	StartAt     time.Time `json:"startAt"`
	EndAt       time.Time `json:"endAt"`
	IsCancelled bool      `json:"isCancelled"`
}

type raceResponse struct {
	ID           string                   `json:"id"`
	Season       int                      `json:"season"`
	Name         string                   `json:"name"`
	OfficialName string                   `json:"officialName"`
	Circuit      circuitResponse          `json:"circuit"`
	StartAt      time.Time                `json:"startAt"`
	EndAt        time.Time                `json:"endAt"`
	Sessions     []sessionSummaryResponse `json:"sessions"`
	Coverage     coverageResponse         `json:"coverage"`
}

type resultValueResponse struct {
	Kind            string     `json:"kind"`
	Seconds         *float64   `json:"seconds,omitempty"`
	Text            *string    `json:"text,omitempty"`
	SegmentsSeconds []*float64 `json:"segmentsSeconds,omitempty"`
}

type driverResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Acronym string `json:"acronym"`
}

type constructorResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type classificationRowResponse struct {
	Driver       driverResponse      `json:"driver"`
	Constructor  constructorResponse `json:"constructor"`
	DriverNumber int                 `json:"driverNumber"`
	Position     *int                `json:"position"`
	State        string              `json:"state"`
	Laps         *int                `json:"laps"`
	Duration     resultValueResponse `json:"duration"`
	Gap          resultValueResponse `json:"gap"`
}

type raceResultsResponse struct {
	RaceID         string                      `json:"raceId"`
	SessionID      string                      `json:"sessionId"`
	Classification []classificationRowResponse `json:"classification"`
	Coverage       coverageResponse            `json:"coverage"`
}

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
