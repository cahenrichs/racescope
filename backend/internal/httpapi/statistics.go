package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/clint/f1/backend/internal/statistics"
)

const maximumStatisticsRequestBytes = 64 * 1024

type statisticsQueryService interface {
	Query(context.Context, statistics.QueryRequest) (statistics.QueryResponse, error)
}

func statisticsQuery(service statisticsQueryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, err := decodeStatisticsRequest(w, r)
		if err != nil {
			writeStatisticsError(w, http.StatusBadRequest, "malformed_request", "The statistics request is not valid JSON.", nil)
			return
		}

		response, err := service.Query(r.Context(), request)
		if err != nil {
			writeStatisticsQueryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func decodeStatisticsRequest(w http.ResponseWriter, r *http.Request) (statistics.QueryRequest, error) {
	deferredBody := http.MaxBytesReader(w, r.Body, maximumStatisticsRequestBytes)
	decoder := json.NewDecoder(deferredBody)
	decoder.DisallowUnknownFields()
	var request statistics.QueryRequest
	if err := decoder.Decode(&request); err != nil {
		return statistics.QueryRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return statistics.QueryRequest{}, err
	}
	return request, nil
}

func writeStatisticsQueryError(w http.ResponseWriter, err error) {
	var queryError *statistics.QueryError
	if !errors.As(err, &queryError) {
		writeStatisticsError(w, http.StatusInternalServerError, "internal_error", "The statistics query could not be completed.", nil)
		return
	}

	switch queryError.Kind {
	case statistics.ErrorInvalidRequest:
		writeStatisticsError(w, http.StatusUnprocessableEntity, string(queryError.Kind), queryError.Message, queryError.Issues)
	case statistics.ErrorUnsupportedCombination:
		writeStatisticsError(w, http.StatusUnprocessableEntity, string(queryError.Kind), queryError.Message, queryError.Issues)
	case statistics.ErrorUnknownPublicID:
		writeStatisticsError(w, http.StatusNotFound, string(queryError.Kind), queryError.Message, queryError.Issues)
	case statistics.ErrorRateLimited:
		if queryError.RetryAfterSeconds > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(queryError.RetryAfterSeconds))
		}
		writeStatisticsError(w, http.StatusTooManyRequests, string(queryError.Kind), queryError.Message, queryError.Issues)
	default:
		writeStatisticsError(w, http.StatusInternalServerError, "internal_error", "The statistics query could not be completed.", nil)
	}
}

func writeStatisticsError(w http.ResponseWriter, status int, code, message string, issues []statistics.ValidationIssue) {
	if message == "" {
		message = fmt.Sprintf("The statistics query failed with code %s.", code)
	}
	writeJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message, Issues: issues}})
}
