package ingest

import (
	"fmt"

	"github.com/clint/f1/backend/internal/domain"
	"github.com/clint/f1/backend/internal/identity"
	"github.com/clint/f1/backend/internal/openf1"
)

// TransformError is one actionable, sanitized source validation failure.
type TransformError struct {
	Code         string
	Entity       string
	SourceValue  string
	DriverNumber int
	Message      string
}

// QuarantineError blocks publication while retaining every error found in the source records.
type QuarantineError struct {
	Errors []TransformError
}

func (e *QuarantineError) Error() string {
	return fmt.Sprintf("weekend quarantined with %d transformation errors", len(e.Errors))
}

type RaceData struct {
	Drivers      []domain.Driver
	Constructors []domain.ConstructorEntrant
	Entries      []domain.SessionEntry
	Results      []domain.SessionResult
}

// TransformRace validates every race entry and result, collecting errors instead of stopping at the first bad identity.
func TransformRace(target Target, season domain.Season, session domain.Session, sessionKey int, sourceDrivers []openf1.Driver, sourceResults []openf1.SessionResult) (RaceData, error) {
	data := RaceData{
		Drivers:      make([]domain.Driver, 0, len(sourceDrivers)),
		Constructors: make([]domain.ConstructorEntrant, 0, 10),
		Entries:      make([]domain.SessionEntry, 0, len(sourceDrivers)),
		Results:      make([]domain.SessionResult, 0, len(sourceDrivers)),
	}
	problems := make([]TransformError, 0)
	entriesByNumber := make(map[int]domain.SessionEntry, len(sourceDrivers))
	sourceDriverNumbers := make(map[int]bool, len(sourceDrivers))
	seenDriverNames := make(map[string]bool, len(sourceDrivers))
	seenDriverNumbers := make(map[int]bool, len(sourceDrivers))
	driversByID := make(map[domain.PublicID]bool, len(sourceDrivers))
	constructorsByID := make(map[domain.PublicID]domain.ConstructorEntrant)
	if len(sourceDrivers) == 0 {
		problems = append(problems, TransformError{Code: "missing_entries", Entity: "session_entry", Message: "Grand Prix has no driver entries"})
	}
	if len(sourceResults) == 0 {
		problems = append(problems, TransformError{Code: "missing_classification", Entity: "session_result", Message: "Grand Prix has no classification rows"})
	}

	for _, source := range sourceDrivers {
		sourceDriverNumbers[source.DriverNumber] = true
		valid := true
		addProblem := func(code, entity, sourceValue, message string) {
			problems = append(problems, TransformError{Code: code, Entity: entity, SourceValue: sourceValue, DriverNumber: source.DriverNumber, Message: message})
			valid = false
		}

		if source.MeetingKey != target.MeetingKey || source.SessionKey != sessionKey {
			addProblem("entry_scope_mismatch", "session_entry", source.FullName, "entry does not belong to the Grand Prix session")
		}
		if source.DriverNumber <= 0 {
			addProblem("invalid_driver_number", "driver", source.FullName, "driver number must be positive")
		}
		if seenDriverNames[source.FullName] || seenDriverNumbers[source.DriverNumber] {
			addProblem("duplicate_entry", "session_entry", source.FullName, "driver name or number is duplicated")
		}
		seenDriverNames[source.FullName] = true
		seenDriverNumbers[source.DriverNumber] = true

		driverMapping, driverKnown := identity.Driver(target.Season, source.FullName)
		if !driverKnown {
			addProblem("unknown_driver", "driver", source.FullName, "full name has no reviewed mapping")
		} else {
			if source.NameAcronym != "" && source.NameAcronym != driverMapping.ExpectedAcronym {
				addProblem("driver_acronym_mismatch", "driver", source.FullName, "acronym does not match the reviewed identity")
			}
			if source.DriverNumber > 0 && source.DriverNumber != driverMapping.ExpectedNumber {
				addProblem("driver_number_mismatch", "driver", source.FullName, "number does not match the reviewed identity")
			}
		}

		constructorMapping, constructorKnown := identity.Constructor(target.Season, source.TeamName)
		if !constructorKnown {
			addProblem("unknown_constructor", "constructor", source.TeamName, "team name has no reviewed mapping")
		}

		var driver domain.Driver
		if driverKnown && (source.NameAcronym == "" || source.NameAcronym == driverMapping.ExpectedAcronym) && source.DriverNumber == driverMapping.ExpectedNumber {
			driverID, err := domain.NewPublicID(domain.EntityDriver, driverMapping.CanonicalKey)
			if err != nil {
				return RaceData{}, err
			}
			driver = domain.Driver{
				PublicID: driverID, StableKey: driverMapping.CanonicalKey,
				FirstName: source.FirstName, LastName: source.LastName,
				FullName: driverMapping.DisplayName, NameAcronym: driverMapping.ExpectedAcronym,
			}
			if !driversByID[driverID] {
				driversByID[driverID] = true
				data.Drivers = append(data.Drivers, driver)
			}
		}

		var constructor domain.ConstructorEntrant
		if constructorKnown {
			constructorID, err := domain.NewPublicID(domain.EntityConstructorEntrant, constructorMapping.CanonicalKey)
			if err != nil {
				return RaceData{}, err
			}
			constructor = domain.ConstructorEntrant{PublicID: constructorID, StableKey: constructorMapping.CanonicalKey, Season: season, Name: constructorMapping.DisplayName}
			if _, exists := constructorsByID[constructorID]; !exists {
				constructorsByID[constructorID] = constructor
				data.Constructors = append(data.Constructors, constructor)
			}
		}

		if !valid || driver.PublicID == "" || constructor.PublicID == "" {
			continue
		}
		stableKey := session.StableKey + ":" + driver.StableKey
		entryID, err := domain.NewPublicID(domain.EntitySessionEntry, stableKey)
		if err != nil {
			return RaceData{}, err
		}
		entry := domain.SessionEntry{
			PublicID: entryID, StableKey: stableKey, SessionID: session.PublicID,
			Driver: driver, Constructor: constructor, DriverNumber: source.DriverNumber, TeamColour: source.TeamColour,
		}
		entriesByNumber[source.DriverNumber] = entry
		data.Entries = append(data.Entries, entry)
	}

	seenResults := make(map[int]bool, len(sourceResults))
	for order, source := range sourceResults {
		entry, entryExists := entriesByNumber[source.DriverNumber]
		if source.MeetingKey != target.MeetingKey || source.SessionKey != sessionKey {
			problems = append(problems, resultProblem("result_scope_mismatch", source.DriverNumber, "result does not belong to the Grand Prix session"))
			continue
		}
		if seenResults[source.DriverNumber] {
			problems = append(problems, resultProblem("duplicate_result", source.DriverNumber, "driver has multiple classification rows"))
			continue
		}
		seenResults[source.DriverNumber] = true
		if !sourceDriverNumbers[source.DriverNumber] {
			problems = append(problems, resultProblem("result_without_entry", source.DriverNumber, "result has no source driver entry"))
			continue
		}
		if !entryExists {
			continue
		}

		state, err := classificationState(source)
		if err != nil {
			problems = append(problems, resultProblem("invalid_classification", source.DriverNumber, err.Error()))
			continue
		}
		data.Results = append(data.Results, makeResult(entry, state, source.Position, source.NumberOfLaps, resultValue(source.Duration), resultValue(source.GapToLeader), order))
	}

	for _, entry := range data.Entries {
		if seenResults[entry.DriverNumber] {
			continue
		}
		data.Results = append(data.Results, makeResult(entry, domain.ClassificationMissing, nil, nil,
			domain.ResultValue{Kind: domain.ResultValueMissing}, domain.ResultValue{Kind: domain.ResultValueMissing}, len(sourceResults)))
		problems = append(problems, resultProblem("missing_result", entry.DriverNumber, "driver has no Grand Prix classification row"))
	}

	if len(problems) != 0 {
		return data, &QuarantineError{Errors: problems}
	}
	return data, nil
}

func classificationState(result openf1.SessionResult) (domain.ClassificationState, error) {
	flags := 0
	for _, set := range []bool{result.DNS, result.DNF, result.DSQ} {
		if set {
			flags++
		}
	}
	if flags > 1 {
		return "", fmt.Errorf("DNS, DNF, and DSQ flags conflict")
	}
	if result.DNS {
		return domain.ClassificationDNS, nil
	}
	if result.DNF {
		return domain.ClassificationDNF, nil
	}
	if result.DSQ {
		return domain.ClassificationDSQ, nil
	}
	if result.Position == nil {
		return domain.ClassificationUnknown, nil
	}
	if *result.Position <= 0 {
		return "", fmt.Errorf("ordinary position must be positive")
	}
	return domain.ClassificationOrdinary, nil
}

func resultValue(source openf1.ResultValue) domain.ResultValue {
	switch {
	case source.Number != nil:
		return domain.ResultValue{Kind: domain.ResultValueNumber, Number: *source.Number}
	case source.Text != nil:
		return domain.ResultValue{Kind: domain.ResultValueText, Text: *source.Text}
	case source.Numbers != nil:
		return domain.ResultValue{Kind: domain.ResultValueNumbers, Numbers: append([]*float64(nil), source.Numbers...)}
	case source.Present:
		return domain.ResultValue{Kind: domain.ResultValueNull}
	default:
		return domain.ResultValue{Kind: domain.ResultValueMissing}
	}
}

func makeResult(entry domain.SessionEntry, state domain.ClassificationState, position, laps *int, duration, gap domain.ResultValue, order int) domain.SessionResult {
	stableKey := entry.StableKey
	resultID, _ := domain.NewPublicID(domain.EntitySessionResult, stableKey)
	return domain.SessionResult{
		PublicID: resultID, StableKey: stableKey, SessionEntryID: entry.PublicID,
		Classification: state, Position: position, NumberOfLaps: laps,
		Duration: duration, GapToLeader: gap, SourceOrder: order,
	}
}

func resultProblem(code string, number int, message string) TransformError {
	return TransformError{Code: code, Entity: "session_result", SourceValue: fmt.Sprint(number), DriverNumber: number, Message: message}
}
