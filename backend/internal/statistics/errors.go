package statistics

import "fmt"

type ErrorKind string

const (
	ErrorInvalidRequest         ErrorKind = "invalid_request"
	ErrorUnsupportedCombination ErrorKind = "unsupported_combination"
	ErrorUnknownPublicID        ErrorKind = "unknown_public_id"
	ErrorRateLimited            ErrorKind = "rate_limited"
)

// QueryError carries a stable public failure category without coupling the query layer to HTTP.
type QueryError struct {
	Kind              ErrorKind
	Message           string
	Issues            []ValidationIssue
	RetryAfterSeconds int
}

func (e *QueryError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("statistics query failed: %s", e.Kind)
}
