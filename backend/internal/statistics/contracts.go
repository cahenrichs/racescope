package statistics

import "time"

type Analysis string

const AnalysisLapComparison Analysis = "lap-comparison"

// QueryRequest is discriminated by Analysis. Each registered analysis owns the
// remaining fields it accepts rather than exposing a general query language.
type QueryRequest struct {
	Analysis Analysis `json:"analysis"`
	LapComparisonRequest
}

type LapComparisonRequest struct {
	SessionID string   `json:"sessionId"`
	DriverIDs []string `json:"driverIds"`
}

type ResultKind string

const (
	ResultSuccess ResultKind = "success"
	ResultNoData  ResultKind = "no-data"
)

type CoverageStatus string

const (
	CoverageComplete CoverageStatus = "complete"
	CoveragePartial  CoverageStatus = "partial"
)

type FreshnessStatus string

const (
	FreshnessFresh FreshnessStatus = "fresh"
	FreshnessStale FreshnessStatus = "stale"
)

// QueryResponse has the same envelope for complete, partial, stale, and no-data results.
// Coverage and freshness are intentionally independent dimensions.
type QueryResponse struct {
	Kind      ResultKind           `json:"kind"`
	Analysis  Analysis             `json:"analysis"`
	Result    *LapComparisonResult `json:"result,omitempty"`
	NoData    *NoData              `json:"noData,omitempty"`
	Warnings  []Warning            `json:"warnings"`
	Coverage  Coverage             `json:"coverage"`
	Freshness Freshness            `json:"freshness"`
}

type LapComparisonResult struct {
	SessionID          string      `json:"sessionId"`
	CanonicalDriverIDs []string    `json:"driverIds"`
	Title              string      `json:"title"`
	Dimension          string      `json:"dimension"`
	Units              string      `json:"units"`
	PreferredChartType string      `json:"preferredChartType"`
	Series             []LapSeries `json:"series"`
}

type LapSeries struct {
	Driver       Driver           `json:"driver"`
	Style        SeriesStyle      `json:"style"`
	Coverage     SeriesCoverage   `json:"coverage"`
	Observations []LapObservation `json:"observations"`
}

type Driver struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Acronym string `json:"acronym"`
}

type SeriesStyle struct {
	Colour    string `json:"colour"`
	LineStyle string `json:"lineStyle"`
	Marker    string `json:"marker"`
}

type LapObservation struct {
	LapNumber            int            `json:"lapNumber"`
	DurationMicroseconds *int64         `json:"durationMicroseconds"`
	MissingReason        *MissingReason `json:"missingReason"`
	Compound             *string        `json:"compound"`
	StintNumber          *int           `json:"stintNumber"`
	IsPitOutLap          *bool          `json:"isPitOutLap"`
	IsStintStart         bool           `json:"isStintStart"`
	IsStintEnd           bool           `json:"isStintEnd"`
}

type MissingReason string

const MissingSourceDuration MissingReason = "source-duration-missing"

type Warning struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Field    string `json:"field,omitempty"`
	SeriesID string `json:"seriesId,omitempty"`
}

type FieldCoverage struct {
	Field     string `json:"field"`
	Available int    `json:"available"`
	Total     int    `json:"total"`
}

type SeriesCoverage struct {
	Status CoverageStatus  `json:"status"`
	Fields []FieldCoverage `json:"fields"`
}

type Coverage struct {
	Status CoverageStatus   `json:"status"`
	Series []SeriesCoverage `json:"series"`
}

type Freshness struct {
	Status          FreshnessStatus `json:"status"`
	SourceFetchedAt time.Time       `json:"sourceFetchedAt"`
	PublishedAt     time.Time       `json:"publishedAt"`
}

type NoData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationIssue struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
