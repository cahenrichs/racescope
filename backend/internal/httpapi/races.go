package httpapi

import (
	"errors"
	"net/http"

	"github.com/clint/f1/backend/internal/database"
	"github.com/clint/f1/backend/internal/domain"
)

func raceDetail(db database.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		race, err := database.RaceByPublicID(r.Context(), db, domain.PublicID(r.PathValue("meetingID")))
		if err != nil {
			writeRaceError(w, err)
			return
		}

		sessions := make([]sessionSummaryResponse, 0, len(race.Sessions))
		for _, session := range race.Sessions {
			sessions = append(sessions, sessionSummaryResponse{
				ID: session.PublicID.String(), Name: session.Name, Type: session.Type,
				StartAt: session.StartAt, EndAt: session.EndAt, IsCancelled: session.IsCancelled,
			})
		}
		writeJSON(w, http.StatusOK, raceResponse{
			ID: race.PublicID.String(), Season: race.Season, Name: race.Name,
			OfficialName: race.OfficialName, Circuit: circuitContract(race.Circuit),
			StartAt: race.StartAt, EndAt: race.EndAt, Sessions: sessions,
			Coverage: coverageContract(race.Coverage),
		})
	}
}

func raceResults(db database.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, err := database.ResultsByRacePublicID(r.Context(), db, domain.PublicID(r.PathValue("meetingID")))
		if err != nil {
			writeRaceError(w, err)
			return
		}

		classification := make([]classificationRowResponse, 0, len(results.Classification))
		for _, row := range results.Classification {
			classification = append(classification, classificationRowResponse{
				Driver: driverResponse{
					ID: row.DriverPublicID.String(), Name: row.DriverName, Acronym: row.DriverAcronym,
				},
				Constructor: constructorResponse{
					ID: row.ConstructorPublicID.String(), Name: row.ConstructorName,
				},
				DriverNumber: row.DriverNumber, Position: row.Position, State: string(row.State), Laps: row.Laps,
				Duration: resultValueContract(row.Duration), Gap: resultValueContract(row.GapToLeader),
			})
		}
		writeJSON(w, http.StatusOK, raceResultsResponse{
			RaceID: results.RacePublicID.String(), SessionID: results.SessionPublicID.String(),
			Classification: classification, Coverage: coverageContract(results.Coverage),
		})
	}
}

func resultValueContract(value database.ResultValue) resultValueResponse {
	return resultValueResponse{
		Kind: string(value.Kind), Seconds: value.Number, Text: value.Text, SegmentsSeconds: value.Numbers,
	}
}

func writeRaceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, database.ErrRaceNotFound):
		writeAPIError(w, http.StatusNotFound, "race_not_found", "The requested race was not found.")
	case errors.Is(err, database.ErrRaceIncomplete):
		writeAPIError(w, http.StatusConflict, "race_incomplete", "The requested race does not have a complete published classification.")
	default:
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "The race could not be loaded.")
	}
}
