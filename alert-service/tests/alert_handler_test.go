package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"alert-service/internal/clients"
	"alert-service/internal/handlers"
	"alert-service/internal/models"
	"alert-service/internal/repositories"
	"alert-service/internal/services"
)

// ── In-memory alert repository for testing ────────────────────────────────────

type inMemAlertRepo struct {
	mu     sync.RWMutex
	nextID int
	store  map[int]models.Alert
}

func newInMemAlertRepo() *inMemAlertRepo {
	return &inMemAlertRepo{nextID: 1, store: make(map[int]models.Alert)}
}

func (r *inMemAlertRepo) Create(alert models.Alert) (models.Alert, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	alert.ID = r.nextID
	r.nextID++
	r.store[alert.ID] = alert
	return alert, nil
}

func (r *inMemAlertRepo) GetAll() ([]models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]models.Alert, 0, len(r.store))
	for _, a := range r.store {
		out = append(out, a)
	}
	return out, nil
}

func (r *inMemAlertRepo) GetByID(id int) (models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.store[id]
	if !ok {
		return models.Alert{}, repositories.ErrAlertNotFound
	}
	return a, nil
}

func (r *inMemAlertRepo) Update(id int, req models.UpdateAlertRequest) (models.Alert, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.store[id]
	if !ok {
		return models.Alert{}, repositories.ErrAlertNotFound
	}
	a.Message = req.Message
	a.Severity = req.Severity
	r.store[id] = a
	return a, nil
}

func (r *inMemAlertRepo) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[id]; !ok {
		return repositories.ErrAlertNotFound
	}
	delete(r.store, id)
	return nil
}

// ── Mock incident validator for testing ───────────────────────────────────────

type alwaysValidIncidentClient struct{}

func (c *alwaysValidIncidentClient) GetIncident(_ context.Context, id int) (*clients.IncidentResponse, int, error) {
	return &clients.IncidentResponse{ID: id, Status: "ACTIVE"}, 200, nil
}

func (c *alwaysValidIncidentClient) ValidateIncidentExists(_ context.Context, _ int) error {
	return nil
}

// ── Helper ────────────────────────────────────────────────────────────────────

func newTestAlertHandler() *handlers.AlertHandler {
	repo := newInMemAlertRepo()
	svc := services.NewAlertService(repo, &alwaysValidIncidentClient{})
	return handlers.NewAlertHandler(svc, nil, "in-memory")
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestAlertHealth(t *testing.T) {
	h := newTestAlertHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.Health(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestCreateAlert(t *testing.T) {
	h := newTestAlertHandler()
	body := bytes.NewBufferString(`{"incidentId":1,"message":"Evacuate now","severity":"CRITICAL"}`)
	req := httptest.NewRequest(http.MethodPost, "/alerts", body)
	rr := httptest.NewRecorder()
	h.AlertsCollection(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("response not JSON: %v", err)
	}
	if resp["id"] == nil {
		t.Error("expected id in response")
	}
}

func TestCreateAlertValidation(t *testing.T) {
	h := newTestAlertHandler()
	body := bytes.NewBufferString(`{"incidentId":0,"message":"","severity":""}`)
	req := httptest.NewRequest(http.MethodPost, "/alerts", body)
	rr := httptest.NewRecorder()
	h.AlertsCollection(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid payload, got %d", rr.Code)
	}
}

func TestListAlerts(t *testing.T) {
	h := newTestAlertHandler()
	req := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	rr := httptest.NewRecorder()
	h.AlertsCollection(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var body []interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("response not JSON array: %v", err)
	}
}

func TestGetAlertByIDNotFound(t *testing.T) {
	h := newTestAlertHandler()
	req := httptest.NewRequest(http.MethodGet, "/alerts/9999", nil)
	rr := httptest.NewRecorder()
	h.AlertByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateAlertNotFound(t *testing.T) {
	h := newTestAlertHandler()
	body := bytes.NewBufferString(`{"message":"Updated msg","severity":"HIGH"}`)
	req := httptest.NewRequest(http.MethodPut, "/alerts/9999", body)
	rr := httptest.NewRecorder()
	h.AlertByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteAlertNotFound(t *testing.T) {
	h := newTestAlertHandler()
	req := httptest.NewRequest(http.MethodDelete, "/alerts/9999", nil)
	rr := httptest.NewRecorder()
	h.AlertByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestAlertFullCRUD(t *testing.T) {
	h := newTestAlertHandler()

	// Create
	body := bytes.NewBufferString(`{"incidentId":1,"message":"Initial alert","severity":"HIGH"}`)
	req := httptest.NewRequest(http.MethodPost, "/alerts", body)
	rr := httptest.NewRecorder()
	h.AlertsCollection(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create failed: %d — %s", rr.Code, rr.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rr.Body).Decode(&created)
	idFloat, ok := created["id"].(float64)
	if !ok {
		t.Fatalf("id not returned from create: %v", created)
	}
	id := int(idFloat)

	// Read by ID
	req = httptest.NewRequest(http.MethodGet, "/alerts/"+itoa(id), nil)
	rr = httptest.NewRecorder()
	h.AlertByID(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("get by id failed: %d", rr.Code)
	}

	// Update
	upBody := bytes.NewBufferString(`{"message":"Updated alert","severity":"CRITICAL"}`)
	req = httptest.NewRequest(http.MethodPut, "/alerts/"+itoa(id), upBody)
	rr = httptest.NewRecorder()
	h.AlertByID(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("update failed: %d — %s", rr.Code, rr.Body.String())
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/alerts/"+itoa(id), nil)
	rr = httptest.NewRecorder()
	h.AlertByID(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("delete failed: %d", rr.Code)
	}

	// Confirm gone
	req = httptest.NewRequest(http.MethodGet, "/alerts/"+itoa(id), nil)
	rr = httptest.NewRecorder()
	h.AlertByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rr.Code)
	}

	_ = errors.New("") // suppress unused import
}

func itoa(n int) string {
	return fmt.Sprint(n)
}
