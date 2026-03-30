package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"volunteer-service/internal/models"
	"volunteer-service/internal/repositories"
	"volunteer-service/internal/services"
)

type VolunteerHandler struct {
	service *services.VolunteerService
}

func NewVolunteerHandler(service *services.VolunteerService) *VolunteerHandler {
	return &VolunteerHandler{service: service}
}

func (h *VolunteerHandler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *VolunteerHandler) VolunteersCollection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

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
}

func (h *VolunteerHandler) VolunteerByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := extractVolunteerID(r.URL.Path)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid volunteer id")
		return
	}

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
		"id":           volunteer.ID,
		"name":         volunteer.Name,
		"role":         volunteer.Role,
		"licenseValid": volunteer.LicenseValid,
	})
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
		case errors.Is(err, services.ErrVehicleNotAssigned):
			h.writeError(w, http.StatusConflict, "volunteer is not assigned to a vehicle")
		case errors.Is(err, services.ErrLogisticsUnavailable):
			h.writeError(w, http.StatusBadGateway, "unable to verify logistics assignment")
		case errors.Is(err, services.ErrInvalidVolunteerState):
			h.writeError(w, http.StatusConflict, "volunteer is not available")
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
