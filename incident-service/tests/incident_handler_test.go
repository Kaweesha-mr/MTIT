package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"incident-service/internal/handlers"
	"incident-service/internal/repositories"
	"incident-service/internal/services"
)

func newTestIncidentHandler() *handlers.IncidentHandler {
	repo := repositories.NewInMemoryIncidentRepository()
	svc := services.NewIncidentService(repo, "http://fake-alert", "http://fake-shelter")
	return handlers.NewIncidentHandler(svc, nil, "in-memory")
}

func TestIncidentHealth(t *testing.T) {
	h := newTestIncidentHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.Health(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestCreateIncidentValidation(t *testing.T) {
	h := newTestIncidentHandler()
	body := bytes.NewBufferString(`{"type":"","location":"","severity":""}`)
	req := httptest.NewRequest(http.MethodPost, "/incidents", body)
	rr := httptest.NewRecorder()
	h.Incidents(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", rr.Code)
	}
}

func TestListIncidents(t *testing.T) {
	h := newTestIncidentHandler()
	req := httptest.NewRequest(http.MethodGet, "/incidents", nil)
	rr := httptest.NewRecorder()
	h.Incidents(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var body []interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("response not JSON array: %v", err)
	}
}

func TestGetIncidentByIDNotFound(t *testing.T) {
	h := newTestIncidentHandler()
	req := httptest.NewRequest(http.MethodGet, "/incidents/9999", nil)
	rr := httptest.NewRecorder()
	h.IncidentByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateIncidentNotFound(t *testing.T) {
	h := newTestIncidentHandler()
	body := bytes.NewBufferString(`{"type":"FLOOD","location":"Lakeville","severity":"HIGH"}`)
	req := httptest.NewRequest(http.MethodPut, "/incidents/9999", body)
	rr := httptest.NewRecorder()
	h.IncidentByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteIncidentNotFound(t *testing.T) {
	h := newTestIncidentHandler()
	req := httptest.NewRequest(http.MethodDelete, "/incidents/9999", nil)
	rr := httptest.NewRecorder()
	h.IncidentByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateStatusValidation(t *testing.T) {
	h := newTestIncidentHandler()
	body := bytes.NewBufferString(`{"status":""}`)
	req := httptest.NewRequest(http.MethodPut, "/incidents/101/status", body)
	rr := httptest.NewRecorder()
	h.IncidentByID(rr, req)
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected 400 or 404 for empty status, got %d", rr.Code)
	}
}

func TestMethodNotAllowedOnIncidents(t *testing.T) {
	h := newTestIncidentHandler()
	req := httptest.NewRequest(http.MethodDelete, "/incidents", nil)
	rr := httptest.NewRecorder()
	h.Incidents(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}