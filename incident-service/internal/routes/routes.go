package routes

import (
	"net/http"
	"os"
	"strings"

	"incident-service/internal/handlers"
	"incident-service/internal/repositories"
	"incident-service/internal/services"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	repo := repositories.NewInMemoryIncidentRepository()
	alertURL := os.Getenv("ALERT_SERVICE_URL")
	if alertURL == "" {
		alertURL = "http://localhost:8085/alerts"
	}

	shelterURL := os.Getenv("SHELTER_SERVICE_URL")
	if shelterURL == "" {
		shelterURL = "http://localhost:8084/shelters/open"
	}

	integrator := services.NewHTTPIntegrator(alertURL, shelterURL)
	service := services.NewIncidentService(repo, integrator)
	h := handlers.NewIncidentHandler(service)

	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/incidents", h.IncidentsCollection)
	mux.HandleFunc("/incidents/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/incidents/" {
			h.IncidentsCollection(w, r)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/status") {
			h.IncidentStatus(w, r)
			return
		}

		h.IncidentByID(w, r)
	})

	return mux
}
