package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"alert-service/internal/models"
	"alert-service/internal/repositories"
	"alert-service/internal/services"
)

type AlertHandler struct {
	service *services.AlertService
}

func NewAlertHandler(service *services.AlertService) *AlertHandler {
	return &AlertHandler{service: service}
}

func (h *AlertHandler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *AlertHandler) AlertsCollection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		alerts, err := h.service.GetAll()
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to fetch alerts")
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(alerts)

	case http.MethodPost:
		var req models.CreateAlertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		res, err := h.service.Create(req)
		if err != nil {
			if err == services.ErrValidation {
				h.writeError(w, http.StatusBadRequest, "incidentId, message and severity are required")
				return
			}
			h.writeError(w, http.StatusInternalServerError, "failed to create alert")
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res)

	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *AlertHandler) AlertByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, err := extractAlertID(r.URL.Path, "/alerts/")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid alert id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		alert, err := h.service.GetByID(id)
		if err != nil {
			if err == repositories.ErrAlertNotFound {
				h.writeError(w, http.StatusNotFound, "alert not found")
				return
			}
			h.writeError(w, http.StatusInternalServerError, "failed to fetch alert")
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(alert)

	case http.MethodPut:
		var req models.UpdateAlertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		updated, err := h.service.Update(id, req)
		if err != nil {
			if err == services.ErrValidation {
				h.writeError(w, http.StatusBadRequest, "message and severity are required")
				return
			}
			if err == repositories.ErrAlertNotFound {
				h.writeError(w, http.StatusNotFound, "alert not found")
				return
			}
			h.writeError(w, http.StatusInternalServerError, "failed to update alert")
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(updated)

	case http.MethodDelete:
		err := h.service.Delete(id)
		if err != nil {
			if err == repositories.ErrAlertNotFound {
				h.writeError(w, http.StatusNotFound, "alert not found")
				return
			}
			h.writeError(w, http.StatusInternalServerError, "failed to delete alert")
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Alert deleted successfully",
		})

	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func extractAlertID(path string, prefix string) (int, error) {
	idText := strings.TrimPrefix(path, prefix)
	idText = strings.Trim(idText, "/")
	return strconv.Atoi(idText)
}

func (h *AlertHandler) writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
