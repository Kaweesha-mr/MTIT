package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"resource-service/internal/models"
	"resource-service/internal/services"
)

type ResourceHandler struct {
	service *services.ResourceService
}

type errorResponse struct {
	Message string `json:"message"`
}

func NewResourceHandler(service *services.ResourceService) *ResourceHandler {
	return &ResourceHandler{service: service}
}

func (h *ResourceHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ResourceHandler) Resources(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createResource(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ResourceHandler) ResourceByID(w http.ResponseWriter, r *http.Request) {
	id, isDispatchPath, err := parseIDFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid resource id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		if isDispatchPath {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.getResource(w, r, id)
	case http.MethodPut:
		if !isDispatchPath {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.dispatchResource(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ResourceHandler) createResource(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req models.CreateResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resource, err := h.service.CreateResource(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":        resource.ID,
		"item":      resource.Item,
		"available": resource.Available,
	})
}

func (h *ResourceHandler) getResource(w http.ResponseWriter, r *http.Request, id int) {
	resource, err := h.service.GetResource(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":     resource.ID,
		"item":   resource.Item,
		"weight": resource.Weight,
	})
}

func (h *ResourceHandler) dispatchResource(w http.ResponseWriter, r *http.Request, id int) {
	defer r.Body.Close()

	var req models.DispatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resource, shelter, err := h.service.DispatchResource(r.Context(), id, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"resourceId": resource.ID,
		"status":     resource.Status,
		"message":    fmt.Sprintf("Resources dispatched successfully to shelter %d", shelter.ID),
	})
}

func (h *ResourceHandler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input. check required fields")
	case errors.Is(err, services.ErrInvalidDispatchQty):
		writeError(w, http.StatusBadRequest, "quantity must be greater than zero")
	case errors.Is(err, services.ErrResourceNotFound):
		writeError(w, http.StatusNotFound, "resource not found")
	case errors.Is(err, services.ErrInsufficientStock):
		writeError(w, http.StatusConflict, "insufficient stock")
	case errors.Is(err, services.ErrShelterNotFound):
		writeError(w, http.StatusConflict, "dispatch rejected. shelter not found")
	case errors.Is(err, services.ErrShelterRejected):
		writeError(w, http.StatusConflict, "dispatch rejected. shelter is closed or full")
	case errors.Is(err, services.ErrShelterUnavailable):
		writeError(w, http.StatusBadGateway, "shelter service unavailable")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func parseIDFromPath(path string) (int, bool, error) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 || len(parts) > 3 || parts[0] != "resources" {
		return 0, false, fmt.Errorf("invalid path")
	}

	isDispatchPath := len(parts) == 3 && parts[2] == "dispatch"
	if len(parts) == 3 && !isDispatchPath {
		return 0, false, fmt.Errorf("invalid path")
	}

	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false, err
	}

	return id, isDispatchPath, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Message: message})
}
