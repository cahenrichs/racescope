package httpapi

import (
	"fmt"
	"net/http"

	"github.com/clint/f1/backend/internal/database"
)

type routerDatabase interface {
	databasePinger
	database.Querier
}

func NewRouter(db routerDatabase) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintln(w, "RaceScope API")
	})
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /ready", ready(db))
	mux.HandleFunc("GET /api/dashboard", dashboard(db))
	mux.HandleFunc("GET /api/races/{meetingID}", raceDetail(db))
	mux.HandleFunc("GET /api/races/{meetingID}/results", raceResults(db))
	return mux
}
