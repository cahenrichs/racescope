package statistics

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

const maximumPublicIDBytes = 128

var (
	ErrSessionNotFound    = errors.New("statistics session not found")
	ErrDriverNotFound     = errors.New("statistics driver not found")
	ErrUnsupportedSession = errors.New("statistics session is unsupported")
	ErrDriverNotInSession = errors.New("statistics driver is not in session")
)

type LapComparisonSource struct {
	SessionID            string
	Drivers              []SourceDriver
	SourceFetchedAt      time.Time
	PublishedAt          time.Time
	ReconciliationFailed bool
}

type SourceDriver struct {
	ID           string
	Name         string
	Acronym      string
	Observations []SourceLapObservation
}

type SourceLapObservation struct {
	LapNumber            int
	DurationMicroseconds *int64
	Compound             *string
	StintNumber          *int
	IsPitOutLap          *bool
	IsStintStart         bool
	IsStintEnd           bool
}

type Repository interface {
	LapComparison(context.Context, string, []string) (LapComparisonSource, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Query(ctx context.Context, request QueryRequest) (QueryResponse, error) {
	if strings.TrimSpace(string(request.Analysis)) == "" {
		return QueryResponse{}, &QueryError{Kind: ErrorInvalidRequest, Message: "The lap comparison request is invalid.", Issues: []ValidationIssue{{Field: "analysis", Code: "required", Message: "Select an analysis."}}}
	}
	if request.Analysis != AnalysisLapComparison {
		return QueryResponse{}, &QueryError{Kind: ErrorUnsupportedCombination, Message: "The requested analysis is not supported."}
	}
	if issues := validateLapComparisonRequest(request); len(issues) > 0 {
		return QueryResponse{}, &QueryError{Kind: ErrorInvalidRequest, Message: "The lap comparison request is invalid.", Issues: issues}
	}

	driverIDs := append([]string(nil), request.DriverIDs...)
	sort.Strings(driverIDs)
	source, err := s.repository.LapComparison(ctx, request.SessionID, driverIDs)
	if err != nil {
		return QueryResponse{}, publicQueryError(err)
	}
	return lapComparisonResponse(source, driverIDs), nil
}

func validateLapComparisonRequest(request QueryRequest) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	if strings.TrimSpace(request.SessionID) == "" || len(request.SessionID) > maximumPublicIDBytes {
		issues = append(issues, ValidationIssue{Field: "sessionId", Code: "invalid_size", Message: "Provide one bounded session public ID."})
	}
	if len(request.DriverIDs) != 2 {
		issues = append(issues, ValidationIssue{Field: "driverIds", Code: "exact_count", Message: "Select exactly two drivers."})
		return issues
	}
	for _, id := range request.DriverIDs {
		if strings.TrimSpace(id) == "" || len(id) > maximumPublicIDBytes {
			issues = append(issues, ValidationIssue{Field: "driverIds", Code: "invalid_size", Message: "Driver public IDs must be non-empty and bounded."})
			break
		}
	}
	if request.DriverIDs[0] == request.DriverIDs[1] {
		issues = append(issues, ValidationIssue{Field: "driverIds", Code: "distinct", Message: "Select two distinct drivers."})
	}
	return issues
}

func publicQueryError(err error) error {
	switch {
	case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrDriverNotFound):
		return &QueryError{Kind: ErrorUnknownPublicID, Message: "A requested public ID was not found."}
	case errors.Is(err, ErrUnsupportedSession), errors.Is(err, ErrDriverNotInSession):
		return &QueryError{Kind: ErrorUnsupportedCombination, Message: "Select two drivers from one Grand Prix session."}
	default:
		return err
	}
}
