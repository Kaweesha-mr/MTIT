package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"volunteer-service/internal/models"
	"volunteer-service/internal/repositories"
	"volunteer-service/internal/services"
)

type VolunteerHandler struct {
	service      *services.VolunteerService
	healthCheck  func(ctx context.Context) error
	healthSource string
}

func NewVolunteerHandler(service *services.VolunteerService, healthCheck func(ctx context.Context) error, healthSource string) *VolunteerHandler {
	return &VolunteerHandler{service: service, healthCheck: healthCheck, healthSource: healthSource}
}

func (h *VolunteerHandler) Health(w http.ResponseWriter, _ *http.Request) {
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

func (h *VolunteerHandler) VolunteersCollection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		var req models.CreateVolunteerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		res, err := h.service.Create(req)
		if err != nil {
			if errors.Is(err, services.ErrValidation) {
				h.writeError(w, http.StatusBadRequest, "name, role and valid phone are required")
				return
			}
			h.writeError(w, http.StatusInternalServerError, "failed to create volunteer")
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res)
	case http.MethodGet:
		volunteers, err := h.service.List()
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to list volunteers")
			return
		}

		list := make([]map[string]any, 0, len(volunteers))
		for _, v := range volunteers {
			list = append(list, map[string]any{
				"id":     v.ID,
				"name":   v.Name,
				"role":   v.Role,
				"status": v.Status,
			})
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(list)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *VolunteerHandler) VolunteerByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, err := extractVolunteerID(r.URL.Path)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid volunteer id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		volunteer, err := h.service.GetByID(id)
		if err != nil {
			if errors.Is(err, repositories.ErrVolunteerNotFound) {
				h.writeError(w, http.StatusNotFound, "volunteer not found")
				return
			}
			h.writeError(w, http.StatusInternalServerError, "failed to fetch volunteer")
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     volunteer.ID,
			"name":   volunteer.Name,
			"role":   volunteer.Role,
			"status": volunteer.Status,
		})

	case http.MethodPut:
		var req models.UpdateVolunteerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		volunteer, err := h.service.UpdateVolunteer(id, req)
		if err != nil {
			if errors.Is(err, repositories.ErrVolunteerNotFound) {
				h.writeError(w, http.StatusNotFound, "volunteer not found")
				return
			}
			if errors.Is(err, services.ErrValidation) {
				h.writeError(w, http.StatusBadRequest, "name, role and valid phone are required")
				return
			}
			h.writeError(w, http.StatusInternalServerError, "failed to update volunteer")
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    volunteer.ID,
			"name":  volunteer.Name,
			"role":  volunteer.Role,
			"phone": volunteer.Phone,
			"status": volunteer.Status,
		})

	case http.MethodDelete:
		err := h.service.DeleteVolunteer(id)
		if err != nil {
			if errors.Is(err, repositories.ErrVolunteerNotFound) {
				h.writeError(w, http.StatusNotFound, "volunteer not found")
				return
			}
			h.writeError(w, http.StatusInternalServerError, "failed to delete volunteer")
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *VolunteerHandler) AssignVolunteer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := extractVolunteerID(strings.TrimSuffix(r.URL.Path, "/assign"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid volunteer id")
		return
	}

	var req models.AssignVolunteerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	res, err := h.service.Assign(id, req.IncidentID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrVolunteerNotFound):
			h.writeError(w, http.StatusNotFound, "volunteer not found")
		case errors.Is(err, services.ErrValidation):
			h.writeError(w, http.StatusBadRequest, "incidentId must be greater than 0")
		case errors.Is(err, services.ErrIncidentUnavailable):
			h.writeError(w, http.StatusBadRequest, "incident not found")
		case errors.Is(err, services.ErrIncidentResolved):
			h.writeError(w, http.StatusBadRequest, "incident is resolved")
		case errors.Is(err, services.ErrLogisticsUnavailable):
			h.writeError(w, http.StatusBadGateway, "unable to verify logistics assignment")
		case errors.Is(err, services.ErrInvalidVolunteerState):
			h.writeError(w, http.StatusConflict, "volunteer is not available")
		case errors.Is(err, services.ErrVolunteerBusy):
			h.writeError(w, http.StatusConflict, "volunteer already has an active trip")
		default:
			h.writeError(w, http.StatusInternalServerError, "failed to assign volunteer")
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func extractVolunteerID(path string) (int, error) {
	idText := strings.TrimPrefix(path, "/volunteers/")
	idText = strings.Trim(idText, "/")
	return strconv.Atoi(idText)
}

func (h *VolunteerHandler) writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
