package httpapi

import (
	"fmt"
	"net/http"
)

func NewRouter(db databasePinger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintln(w, "RaceScope API")
	})
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /ready", ready(db))
	return mux
}
