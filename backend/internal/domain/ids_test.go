package domain

import (
	"strings"
	"testing"
)

func TestNewPublicIDIsDeterministicAndScoped(t *testing.T) {
	t.Parallel()

	first, err := NewPublicID(EntityMeeting, "2025:belgian-grand-prix")
	if err != nil {
		t.Fatalf("NewPublicID() error = %v", err)
	}
	second, err := NewPublicID(EntityMeeting, "2025:belgian-grand-prix")
	if err != nil {
		t.Fatalf("NewPublicID() second error = %v", err)
	}
	seasonID, err := NewPublicID(EntitySeason, "2025:belgian-grand-prix")
	if err != nil {
		t.Fatalf("NewPublicID() scoped error = %v", err)
	}

	if first != second {
		t.Fatalf("NewPublicID() = %q, second call = %q", first, second)
	}
	if first == seasonID {
		t.Fatalf("meeting ID %q equals season ID %q", first, seasonID)
	}
	if !strings.HasPrefix(first.String(), "meeting_") {
		t.Fatalf("NewPublicID() = %q, want meeting_ prefix", first)
	}
}

func TestNewPublicIDRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		kind      EntityKind
		stableKey string
	}{
		{name: "unsupported kind", kind: EntityKind("unknown"), stableKey: "key"},
		{name: "empty key", kind: EntityDriver, stableKey: ""},
		{name: "blank key", kind: EntityDriver, stableKey: "  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewPublicID(tt.kind, tt.stableKey); err == nil {
				t.Fatal("NewPublicID() error = nil, want error")
			}
		})
	}
}
