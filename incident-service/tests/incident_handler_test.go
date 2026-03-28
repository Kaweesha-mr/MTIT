package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"incident-service/internal/handlers"
	"incident-service/internal/models"
	"incident-service/internal/repositories"
	"incident-service/internal/services"
)

type mockIntegrator struct {
	alertCalls   int
	shelterCalls int
}

func (m *mockIntegrator) NotifyAlert(_ models.Incident) error {
	m.alertCalls++
	return nil
}

func (m *mockIntegrator) OpenShelter(_ models.Incident) error {
	m.shelterCalls++
	return nil
}

func newTestHandler() (*handlers.IncidentHandler, *mockIntegrator) {
	repo := repositories.NewInMemoryIncidentRepository()
	integration := &mockIntegrator{}
	service := services.NewIncidentService(repo, integration)

	return handlers.NewIncidentHandler(service), integration
}

func TestCreateIncident(t *testing.T) {
	h, integration := newTestHandler()

	body := []byte(`{"type":"FLOOD","location":"Kelaniya","severity":"HIGH"}`)
	req := httptest.NewRequest(http.MethodPost, "/incidents", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.IncidentsCollection(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var res models.CreateIncidentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.ID != 101 {
		t.Fatalf("expected id 101, got %d", res.ID)
	}
	if res.Status != "ACTIVE" {
		t.Fatalf("expected ACTIVE status, got %s", res.Status)
	}
	if integration.alertCalls != 1 || integration.shelterCalls != 1 {
		t.Fatalf("expected integrations to be called once each, got alert=%d shelter=%d", integration.alertCalls, integration.shelterCalls)
	}
}

func TestGetIncidentByID(t *testing.T) {
	h, _ := newTestHandler()

	createReq := httptest.NewRequest(http.MethodPost, "/incidents", bytes.NewBufferString(`{"type":"FIRE","location":"Colombo","severity":"MEDIUM"}`))
	createRec := httptest.NewRecorder()
	h.IncidentsCollection(createRec, createReq)

	req := httptest.NewRequest(http.MethodGet, "/incidents/101", nil)
	rec := httptest.NewRecorder()
	h.IncidentByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if int(res["id"].(float64)) != 101 {
		t.Fatalf("expected id 101, got %v", res["id"])
	}
	if res["status"] != "ACTIVE" {
		t.Fatalf("expected ACTIVE status, got %v", res["status"])
	}
}

func TestListActiveIncidentsAndResolve(t *testing.T) {
	h, _ := newTestHandler()

	h.IncidentsCollection(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/incidents", bytes.NewBufferString(`{"type":"EARTHQUAKE","location":"Galle","severity":"HIGH"}`)))
	h.IncidentsCollection(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/incidents", bytes.NewBufferString(`{"type":"FLOOD","location":"Kelaniya","severity":"HIGH"}`)))

	statusBody := []byte(`{"status":"RESOLVED"}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/incidents/101/status", bytes.NewReader(statusBody))
	updateRec := httptest.NewRecorder()
	h.IncidentStatus(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, updateRec.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/incidents", nil)
	listRec := httptest.NewRecorder()
	h.IncidentsCollection(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listRec.Code)
	}

	var incidents []models.Incident
	if err := json.Unmarshal(listRec.Body.Bytes(), &incidents); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(incidents) != 1 {
		t.Fatalf("expected 1 active incident, got %d", len(incidents))
	}
	if incidents[0].ID != 102 {
		t.Fatalf("expected active incident id 102, got %d", incidents[0].ID)
	}
}
