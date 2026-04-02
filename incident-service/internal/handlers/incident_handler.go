package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"incident-service/internal/models"
	"incident-service/internal/repositories"
	"incident-service/internal/services"
)

type IncidentHandler struct {
	service      *services.IncidentService
	healthCheck  func(ctx context.Context) error
	healthSource string
}

func NewIncidentHandler(service *services.IncidentService, healthCheck func(ctx context.Context) error, healthSource string) *IncidentHandler {
	return &IncidentHandler{service: service, healthCheck: healthCheck, healthSource: healthSource}
}

func (h *IncidentHandler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	payload := map[string]any{
		"status": "ok",
		"db":     map[string]any{"status": "unknown", "source": h.healthSource},
	}

	if h.healthCheck != nil {
		if err := h.healthCheck(ctx); err != nil {
			payload["status"] = "degraded"
			payload["db"] = map[string]any{"status": "down", "source": h.healthSource, "error": err.Error()}
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(payload)
			return
		}
		payload["db"] = map[string]any{"status": "up", "source": h.healthSource}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *IncidentHandler) Incidents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		incidents := h.service.List()
		response := make([]map[string]any, 0, len(incidents))
		for _, inc := range incidents {
			response = append(response, map[string]any{
				"id":     inc.ID,
				"type":   inc.Type,
				"status": inc.Status,
			})
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	case http.MethodPost:
		h.createIncident(w, r)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *IncidentHandler) IncidentByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	trimmed := strings.TrimPrefix(r.URL.Path, "/incidents/")
	if trimmed == "" {
		h.writeError(w, http.StatusBadRequest, "invalid incident id")
		return
	}

	if strings.HasSuffix(trimmed, "/status") {
		h.updateStatus(w, r, strings.TrimSuffix(trimmed, "/status"))
		return
	}

	id, err := strconv.Atoi(strings.Trim(trimmed, "/"))
	if err != nil || id <= 0 {
		h.writeError(w, http.StatusBadRequest, "invalid incident id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		incident, err := h.service.GetByID(id)
		if err != nil {
			if errors.Is(err, repositories.ErrIncidentNotFound) {
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
			"severity": incident.Severity,
			"status":   incident.Status,
			"type":     incident.Type,
		})
	case http.MethodPut:
		h.updateIncident(w, r, id)
	case http.MethodDelete:
		h.deleteIncident(w, id)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *IncidentHandler) createIncident(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req models.CreateIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	incident, err := h.service.CreateIncident(req)
	if err != nil {
		log.Printf("Error creating incident: %v", err)
		switch {
		case errors.Is(err, services.ErrValidation):
			h.writeError(w, http.StatusBadRequest, "type, location and severity are required")
		case errors.Is(err, services.ErrIntegrationFail):
			h.writeError(w, http.StatusServiceUnavailable, "failed to trigger downstream services")
		default:
			h.writeError(w, http.StatusInternalServerError, "failed to create incident")
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":         incident.ID,
		"status":     incident.Status,
		"message":    "Incident Created",
		"downstream": incident.Downstream,
	})
}

func (h *IncidentHandler) updateStatus(w http.ResponseWriter, r *http.Request, idText string) {
	if r.Method != http.MethodPut {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := strconv.Atoi(strings.Trim(idText, "/"))
	if err != nil || id <= 0 {
		h.writeError(w, http.StatusBadRequest, "invalid incident id")
		return
	}

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	incident, err := h.service.UpdateStatus(id, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrValidation):
			h.writeError(w, http.StatusBadRequest, "status is required")
		case errors.Is(err, repositories.ErrIncidentNotFound):
			h.writeError(w, http.StatusNotFound, "incident not found")
		default:
			h.writeError(w, http.StatusInternalServerError, "failed to update status")
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":     incident.ID,
		"status": incident.Status,
	})
}

func (h *IncidentHandler) updateIncident(w http.ResponseWriter, r *http.Request, id int) {
	defer r.Body.Close()

	var req models.UpdateIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	incident, err := h.service.UpdateIncident(id, req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrValidation):
			h.writeError(w, http.StatusBadRequest, "type, location and severity are required")
		case errors.Is(err, repositories.ErrIncidentNotFound):
			h.writeError(w, http.StatusNotFound, "incident not found")
		default:
			h.writeError(w, http.StatusInternalServerError, "failed to update incident")
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":       incident.ID,
		"location": incident.Location,
		"severity": incident.Severity,
		"status":   incident.Status,
		"type":     incident.Type,
	})
}

func (h *IncidentHandler) deleteIncident(w http.ResponseWriter, id int) {
	err := h.service.DeleteIncident(id)
	if err != nil {
		if errors.Is(err, repositories.ErrIncidentNotFound) {
			h.writeError(w, http.StatusNotFound, "incident not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to delete incident")
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Incident deleted successfully",
	})
}

func (h *IncidentHandler) writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
