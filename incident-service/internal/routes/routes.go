package routes

import (
	"net/http"

	"incident-service/internal/handlers"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	h := handlers.NewIncidentHandler()

	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/incidents", h.Incidents)

	return mux
}
