package statistics

import "fmt"

var lapComparisonStyles = []SeriesStyle{
	{Colour: "#E10600", LineStyle: "solid", Marker: "circle"},
	{Colour: "#3671C6", LineStyle: "dashed", Marker: "square"},
}

func lapComparisonResponse(source LapComparisonSource, canonicalDriverIDs []string) QueryResponse {
	drivers := make(map[string]SourceDriver, len(source.Drivers))
	for _, driver := range source.Drivers {
		drivers[driver.ID] = driver
	}

	series := make([]LapSeries, 0, len(canonicalDriverIDs))
	coverageBySeries := make([]SeriesCoverage, 0, len(canonicalDriverIDs))
	warnings := make([]Warning, 0)
	overallCoverage := CoverageComplete
	usableDurations := 0
	for index, driverID := range canonicalDriverIDs {
		sourceDriver := drivers[driverID]
		observations := make([]LapObservation, 0, len(sourceDriver.Observations))
		for _, lap := range sourceDriver.Observations {
			observation := LapObservation{
				LapNumber: lap.LapNumber, DurationMicroseconds: lap.DurationMicroseconds,
				Compound: lap.Compound, StintNumber: lap.StintNumber, IsPitOutLap: lap.IsPitOutLap,
				IsStintStart: lap.IsStintStart, IsStintEnd: lap.IsStintEnd,
			}
			if lap.DurationMicroseconds == nil {
				reason := MissingSourceDuration
				observation.MissingReason = &reason
			} else {
				usableDurations++
			}
			observations = append(observations, observation)
		}
		seriesCoverage, seriesWarnings := lapSeriesCoverage(sourceDriver)
		if seriesCoverage.Status == CoveragePartial {
			overallCoverage = CoveragePartial
		}
		coverageBySeries = append(coverageBySeries, seriesCoverage)
		warnings = append(warnings, seriesWarnings...)
		series = append(series, LapSeries{
			Driver:       Driver{ID: sourceDriver.ID, Name: sourceDriver.Name, Acronym: sourceDriver.Acronym},
			Style:        lapComparisonStyles[index],
			Coverage:     seriesCoverage,
			Observations: observations,
		})
	}

	if usableDurations == 0 {
		for index := range coverageBySeries {
			coverageBySeries[index].Status = CoveragePartial
			series[index].Coverage.Status = CoveragePartial
		}
		freshness := lapComparisonFreshness(source, &warnings)
		return QueryResponse{
			Kind: ResultNoData, Analysis: AnalysisLapComparison,
			NoData:   &NoData{Code: "no_usable_lap_durations", Message: "Neither selected driver has a usable lap duration."},
			Warnings: warnings, Coverage: Coverage{Status: CoveragePartial, Series: coverageBySeries},
			Freshness: freshness,
		}
	}

	for index, item := range coverageBySeries {
		durationCoverage := item.Fields[0]
		if durationCoverage.Available != 0 {
			continue
		}
		overallCoverage = CoveragePartial
		coverageBySeries[index].Status = CoveragePartial
		series[index].Coverage.Status = CoveragePartial
		warnings = append(warnings, Warning{
			Code: WarningDriverLapsMissing, SeriesID: item.SeriesID,
			Message: "The selected driver has no usable lap durations.",
		})
	}

	freshness := lapComparisonFreshness(source, &warnings)
	return QueryResponse{
		Kind: ResultSuccess, Analysis: AnalysisLapComparison,
		Result: &LapComparisonResult{
			SessionID: source.SessionID, CanonicalDriverIDs: append([]string(nil), canonicalDriverIDs...),
			Title:     fmt.Sprintf("%s and %s lap comparison", series[0].Driver.Name, series[1].Driver.Name),
			Dimension: "lap", Units: "microseconds", PreferredChartType: "line", Series: series,
		},
		Warnings: warnings, Coverage: Coverage{Status: overallCoverage, Series: coverageBySeries},
		Freshness: freshness,
	}
}

func lapComparisonFreshness(source LapComparisonSource, warnings *[]Warning) Freshness {
	status := FreshnessFresh
	if source.ReconciliationFailed {
		status = FreshnessStale
		*warnings = append(*warnings, Warning{
			Code:    WarningReconciliationFailed,
			Message: "A newer source reconciliation failed; the last complete published timing data is being served.",
		})
	}
	return Freshness{Status: status, SourceFetchedAt: source.SourceFetchedAt, PublishedAt: source.PublishedAt}
}

func lapSeriesCoverage(driver SourceDriver) (SeriesCoverage, []Warning) {
	total := len(driver.Observations)
	available := map[string]int{
		"durationMicroseconds": 0,
		"compound":             0,
		"stintNumber":          0,
		"isPitOutLap":          0,
		"isStintStart":         total,
		"isStintEnd":           total,
	}
	for _, lap := range driver.Observations {
		if lap.DurationMicroseconds != nil {
			available["durationMicroseconds"]++
		}
		if lap.Compound != nil {
			available["compound"]++
		}
		if lap.StintNumber != nil {
			available["stintNumber"]++
		}
		if lap.IsPitOutLap != nil {
			available["isPitOutLap"]++
		}
	}

	fieldNames := []string{"durationMicroseconds", "compound", "stintNumber", "isPitOutLap", "isStintStart", "isStintEnd"}
	coverage := SeriesCoverage{SeriesID: driver.ID, Status: CoverageComplete, Fields: make([]FieldCoverage, 0, len(fieldNames))}
	warnings := make([]Warning, 0)
	for _, field := range fieldNames {
		coverage.Fields = append(coverage.Fields, FieldCoverage{Field: field, Available: available[field], Total: total})
		if field != "durationMicroseconds" && available[field] < total {
			coverage.Status = CoveragePartial
			warnings = append(warnings, Warning{
				Code: WarningLapContextMissing, Field: field, SeriesID: driver.ID,
				Message: fmt.Sprintf("%s is missing for %d of %d source laps.", field, total-available[field], total),
			})
		}
	}
	return coverage, warnings
}
