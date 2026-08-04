package identity

import "testing"

func TestMonaco2024IdentityMappings(t *testing.T) {
	t.Parallel()

	weekend, ok := Weekend(2024, 1236)
	if !ok {
		t.Fatal("Weekend(2024, 1236) was not found")
	}
	if weekend.MeetingCanonicalKey != "2024-monaco-grand-prix" || weekend.CircuitCanonicalKey != "circuit-de-monaco" {
		t.Fatalf("Weekend(2024, 1236) = %+v", weekend)
	}
	if len(weekend.Sessions) != 5 {
		t.Fatalf("session mappings = %d, want 5", len(weekend.Sessions))
	}

	driver, ok := Driver(2024, "Charles LECLERC")
	if !ok || driver.CanonicalKey != "charles-leclerc" || driver.ExpectedAcronym != "LEC" || driver.ExpectedNumber != 16 {
		t.Fatalf("Driver(Charles LECLERC) = %+v, %v", driver, ok)
	}
	constructor, ok := Constructor(2024, "Ferrari")
	if !ok || constructor.CanonicalKey != "2024-ferrari" {
		t.Fatalf("Constructor(Ferrari) = %+v, %v", constructor, ok)
	}

	if len(driverMappings) != 20 {
		t.Fatalf("driver mappings = %d, want 20", len(driverMappings))
	}
	if len(constructorMappings) != 10 {
		t.Fatalf("constructor mappings = %d, want 10", len(constructorMappings))
	}
}

func TestIdentityLookupsAreExact(t *testing.T) {
	t.Parallel()

	if _, ok := Driver(2024, "Charles Leclerc"); ok {
		t.Fatal("Driver accepted a normalized source name")
	}
	if _, ok := Constructor(2024, "ferrari"); ok {
		t.Fatal("Constructor accepted a case-insensitive source name")
	}
	if _, ok := Weekend(2024, 9999); ok {
		t.Fatal("Weekend accepted an unreviewed source meeting key")
	}
}
