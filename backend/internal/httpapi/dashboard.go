package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/clint/f1/backend/internal/database"
)

func dashboard(db database.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		races, err := database.PublishedRaces(r.Context(), db)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "The dashboard could not be loaded.")
			return
		}

		response := dashboardResponse{Races: make([]raceSummaryResponse, 0, len(races))}
		for _, race := range races {
			response.Races = append(response.Races, raceSummaryContract(race))
			if response.Coverage == nil {
				coverage := coverageContract(race.Coverage)
				response.Coverage = &coverage
				continue
			}
			if race.Coverage.SourceFetchedAt.After(response.Coverage.SourceFetchedAt) {
				response.Coverage.SourceFetchedAt = race.Coverage.SourceFetchedAt
			}
			if race.Coverage.PublishedAt.After(response.Coverage.PublishedAt) {
				response.Coverage.PublishedAt = race.Coverage.PublishedAt
			}
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func raceSummaryContract(race database.RaceSummary) raceSummaryResponse {
	return raceSummaryResponse{
		ID: race.PublicID.String(), Season: race.Season, Name: race.Name,
		OfficialName: race.OfficialName, Circuit: circuitContract(race.Circuit),
		StartAt: race.StartAt, EndAt: race.EndAt, Coverage: coverageContract(race.Coverage),
	}
}

func circuitContract(circuit database.Circuit) circuitResponse {
	return circuitResponse{
		ID: circuit.PublicID.String(), Name: circuit.Name, CountryCode: circuit.CountryCode,
		CountryName: circuit.CountryName, Location: circuit.Location,
	}
}

func coverageContract(coverage database.Coverage) coverageResponse {
	return coverageResponse{
		Status: "complete", SourceFetchedAt: coverage.SourceFetchedAt, PublishedAt: coverage.PublishedAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message}})
}
