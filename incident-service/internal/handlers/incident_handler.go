package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"incident-service/internal/models"
	"incident-service/internal/repositories"
	"incident-service/internal/services"
)

type IncidentHandler struct {
	service *services.IncidentService
}

func NewIncidentHandler(service *services.IncidentService) *IncidentHandler {
	return &IncidentHandler{service: service}
}

func (h *IncidentHandler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *IncidentHandler) IncidentsCollection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		var req models.CreateIncidentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		res, err := h.service.Create(req)
		if err != nil {
			if err == services.ErrValidation {
				h.writeError(w, http.StatusBadRequest, "type, location and severity are required")
				return
			}
			h.writeError(w, http.StatusInternalServerError, "failed to create incident")
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res)
	case http.MethodGet:
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(h.service.ListActive())
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *IncidentHandler) IncidentByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := extractIncidentID(r.URL.Path, "/incidents/")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid incident id")
		return
	}

	incident, err := h.service.GetByID(id)
	if err != nil {
		if err == repositories.ErrIncidentNotFound {
			h.writeError(w, http.StatusNotFound, "incident not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to fetch incident")
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":       incident.ID,
		"location": incident.Location,
		"status":   incident.Status,
	})
}

func (h *IncidentHandler) IncidentStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := extractIncidentID(strings.TrimSuffix(r.URL.Path, "/status"), "/incidents/")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid incident id")
		return
	}

	status := "RESOLVED"
	if r.ContentLength > 0 {
		var req models.UpdateIncidentStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}
		if strings.TrimSpace(req.Status) != "" {
			status = req.Status
		}
	}

	updated, err := h.service.UpdateStatus(id, status)
	if err != nil {
		if err == repositories.ErrIncidentNotFound {
			h.writeError(w, http.StatusNotFound, "incident not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to update incident")
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":      updated.ID,
		"status":  updated.Status,
		"message": "Incident status updated",
	})
}

func extractIncidentID(path string, prefix string) (int, error) {
	idText := strings.TrimPrefix(path, prefix)
	idText = strings.Trim(idText, "/")

	return strconv.Atoi(idText)
}

func (h *IncidentHandler) writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
