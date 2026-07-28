package domain

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
)

const publicIDNamespace = "racescope:public-id:v1"

// EntityKind separates otherwise identical stable keys across domain entities.
type EntityKind string

const (
	EntitySeason             EntityKind = "season"
	EntityCircuit            EntityKind = "circuit"
	EntityMeeting            EntityKind = "meeting"
	EntitySession            EntityKind = "session"
	EntityDriver             EntityKind = "driver"
	EntityConstructorEntrant EntityKind = "constructor_entrant"
	EntitySessionEntry       EntityKind = "session_entry"
	EntitySessionResult      EntityKind = "session_result"
)

// PublicID is an opaque identifier safe to expose outside the application.
type PublicID string

func (id PublicID) String() string {
	return string(id)
}

// NewPublicID derives a reproducible public ID from a reviewed domain key.
// stableKey must describe domain identity rather than a database or source key.
func NewPublicID(kind EntityKind, stableKey string) (PublicID, error) {
	if !kind.valid() {
		return "", fmt.Errorf("unsupported public ID entity kind %q", kind)
	}
	if strings.TrimSpace(stableKey) == "" {
		return "", errors.New("public ID stable key must not be empty")
	}

	sum := sha256.Sum256([]byte(publicIDNamespace + "\x00" + string(kind) + "\x00" + stableKey))
	return PublicID(fmt.Sprintf("%s_%x", kind, sum[:16])), nil
}

func (kind EntityKind) valid() bool {
	switch kind {
	case EntitySeason,
		EntityCircuit,
		EntityMeeting,
		EntitySession,
		EntityDriver,
		EntityConstructorEntrant,
		EntitySessionEntry,
		EntitySessionResult:
		return true
	default:
		return false
	}
}
