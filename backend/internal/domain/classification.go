package domain

// ClassificationState preserves ordinary and nonnumeric result semantics.
type ClassificationState string

const (
	ClassificationOrdinary ClassificationState = "ordinary"
	ClassificationDNS      ClassificationState = "dns"
	ClassificationDNF      ClassificationState = "dnf"
	ClassificationDSQ      ClassificationState = "dsq"
	ClassificationUnknown  ClassificationState = "unknown"
	ClassificationMissing  ClassificationState = "missing"
)
