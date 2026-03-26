package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"volunteer-service/internal/handlers"
	"volunteer-service/internal/models"
	"volunteer-service/internal/repositories"
	"volunteer-service/internal/services"
)

// ── Mocks ─────────────────────────────────────────────────────────────────────

type mockIncidentVerifier struct{}

func (m *mockIncidentVerifier) GetIncident(id int) (models.IncidentSummary, error) {
	return models.IncidentSummary{ID: id, Status: "ACTIVE"}, nil
}

type mockLogisticsChecker struct{}

func (m *mockLogisticsChecker) GetAssignment(id int) (models.LogisticsAssignment, error) {
	return models.LogisticsAssignment{VolunteerID: id, HasActiveTrip: false}, nil
}

// ── Helper ────────────────────────────────────────────────────────────────────

func newTestVolunteerHandler() *handlers.VolunteerHandler {
	repo := repositories.NewInMemoryVolunteerRepository()
	// Use the pointer to the mock struct which implements the interface
	svc := services.NewVolunteerService(repo, &mockIncidentVerifier{}, &mockLogisticsChecker{})
	return handlers.NewVolunteerHandler(svc, nil, "in-memory")
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestVolunteerHealth(t *testing.T) {
	h := newTestVolunteerHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.Health(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestCreateVolunteer(t *testing.T) {
	h := newTestVolunteerHandler()
	body := bytes.NewBufferString(`{"name":"Jane Doe","role":"DOCTOR","phone":"0771234567"}`)
	req := httptest.NewRequest(http.MethodPost, "/volunteers", body)
	rr := httptest.NewRecorder()
	h.VolunteersCollection(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("response not valid JSON: %v", err)
	}
	if resp["id"] == nil {
		t.Error("expected id in response")
	}
}

func TestCreateVolunteerValidation(t *testing.T) {
	h := newTestVolunteerHandler()
	body := bytes.NewBufferString(`{"name":"","role":"","phone":""}`)
	req := httptest.NewRequest(http.MethodPost, "/volunteers", body)
	rr := httptest.NewRecorder()
	h.VolunteersCollection(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", rr.Code)
	}
}

func TestListVolunteers(t *testing.T) {
	h := newTestVolunteerHandler()
	req := httptest.NewRequest(http.MethodGet, "/volunteers", nil)
	rr := httptest.NewRecorder()
	h.VolunteersCollection(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var body []interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("response not JSON array: %v", err)
	}
}

func TestGetVolunteerByIDNotFound(t *testing.T) {
	h := newTestVolunteerHandler()
	req := httptest.NewRequest(http.MethodGet, "/volunteers/9999", nil)
	rr := httptest.NewRecorder()
	h.VolunteerByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateVolunteerNotFound(t *testing.T) {
	h := newTestVolunteerHandler()
	body := bytes.NewBufferString(`{"name":"Jane Smith","role":"DOCTOR","phone":"0779999999"}`)
	req := httptest.NewRequest(http.MethodPut, "/volunteers/9999", body)
	rr := httptest.NewRecorder()
	h.VolunteerByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteVolunteerNotFound(t *testing.T) {
	h := newTestVolunteerHandler()
	req := httptest.NewRequest(http.MethodDelete, "/volunteers/9999", nil)
	rr := httptest.NewRecorder()
	h.VolunteerByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestVolunteerMethodNotAllowed(t *testing.T) {
	h := newTestVolunteerHandler()
	req := httptest.NewRequest(http.MethodDelete, "/volunteers", nil)
	rr := httptest.NewRecorder()
	h.VolunteersCollection(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}
