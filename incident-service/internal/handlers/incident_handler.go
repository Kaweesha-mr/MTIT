package handlers

import (
	"encoding/json"
	"net/http"

	"incident-service/internal/models"
)

type IncidentHandler struct{}

func NewIncidentHandler() *IncidentHandler {
	return &IncidentHandler{}
}

func (h *IncidentHandler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *IncidentHandler) Incidents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode([]models.Incident{})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
