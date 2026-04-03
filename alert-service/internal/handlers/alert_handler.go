package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"alert-service/internal/clients"
	"alert-service/internal/models"
	"alert-service/internal/repositories"
	"alert-service/internal/services"
)

type AlertHandler struct {
	service      *services.AlertService
	healthCheck  func(ctx context.Context) error
	healthSource string
}

func NewAlertHandler(service *services.AlertService, healthCheck func(ctx context.Context) error, healthSource string) *AlertHandler {
	return &AlertHandler{service: service, healthCheck: healthCheck, healthSource: healthSource}
}

func (h *AlertHandler) Health(w http.ResponseWriter, _ *http.Request) {
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

		// Create context with timeout for the service call
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := h.service.Create(ctx, req)
		if err != nil {
			if err == services.ErrValidation {
				h.writeError(w, http.StatusBadRequest, "incidentId, message and severity are required")
				return
			}

			// Handle ValidationError from incident service call
			if validationErr, ok := err.(*clients.ValidationError); ok {
				h.writeError(w, validationErr.StatusCode, validationErr.Message)
				return
			}

			h.writeError(w, http.StatusInternalServerError, "failed to create alert: "+err.Error())
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
