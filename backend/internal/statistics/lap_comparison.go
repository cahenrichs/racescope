package statistics

func lapComparisonResponse(source LapComparisonSource, canonicalDriverIDs []string) QueryResponse {
	drivers := make(map[string]SourceDriver, len(source.Drivers))
	for _, driver := range source.Drivers {
		drivers[driver.ID] = driver
	}

	series := make([]LapSeries, 0, len(canonicalDriverIDs))
	for _, driverID := range canonicalDriverIDs {
		sourceDriver := drivers[driverID]
		observations := make([]LapObservation, 0, len(sourceDriver.Observations))
		for _, lap := range sourceDriver.Observations {
			observation := LapObservation{LapNumber: lap.LapNumber, DurationMicroseconds: lap.DurationMicroseconds}
			if lap.DurationMicroseconds == nil {
				reason := MissingSourceDuration
				observation.MissingReason = &reason
			}
			observations = append(observations, observation)
		}
		series = append(series, LapSeries{
			Driver:       Driver{ID: sourceDriver.ID, Name: sourceDriver.Name, Acronym: sourceDriver.Acronym},
			Observations: observations,
		})
	}

	return QueryResponse{
		Kind: ResultSuccess, Analysis: AnalysisLapComparison,
		Result:   &LapComparisonResult{SessionID: source.SessionID, CanonicalDriverIDs: append([]string(nil), canonicalDriverIDs...), Series: series},
		Warnings: []Warning{}, Coverage: Coverage{Status: CoverageComplete, Series: []SeriesCoverage{}},
		Freshness: Freshness{Status: FreshnessFresh, SourceFetchedAt: source.SourceFetchedAt, PublishedAt: source.PublishedAt},
	}
}
