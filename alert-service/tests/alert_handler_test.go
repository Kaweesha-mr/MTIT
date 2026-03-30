package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"alert-service/internal/handlers"
	"alert-service/internal/models"
	"alert-service/internal/repositories"
	"alert-service/internal/services"
)

type mockAlertRepository struct {
	store  map[int]models.Alert
	nextID int
}

func newMockAlertRepository() *mockAlertRepository {
	return &mockAlertRepository{
		store:  make(map[int]models.Alert),
		nextID: 1,
	}
}

func (m *mockAlertRepository) Create(alert models.Alert) (models.Alert, error) {
	alert.ID = m.nextID
	m.nextID++
	m.store[alert.ID] = alert
	return alert, nil
}

func (m *mockAlertRepository) GetAll() ([]models.Alert, error) {
	alerts := make([]models.Alert, 0, len(m.store))
	for i := 1; i < m.nextID; i++ {
		if alert, ok := m.store[i]; ok {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (m *mockAlertRepository) GetByID(id int) (models.Alert, error) {
	alert, ok := m.store[id]
	if !ok {
		return models.Alert{}, repositories.ErrAlertNotFound
	}
	return alert, nil
}

func (m *mockAlertRepository) Update(id int, req models.UpdateAlertRequest) (models.Alert, error) {
	alert, ok := m.store[id]
	if !ok {
		return models.Alert{}, repositories.ErrAlertNotFound
	}
	alert.Message = req.Message
	alert.Severity = req.Severity
	m.store[id] = alert
	return alert, nil
}

func (m *mockAlertRepository) Delete(id int) error {
	if _, ok := m.store[id]; !ok {
		return repositories.ErrAlertNotFound
	}
	delete(m.store, id)
	return nil
}

func setupHandler() *handlers.AlertHandler {
	repo := newMockAlertRepository()
	svc := services.NewAlertService(repo)
	return handlers.NewAlertHandler(svc)
}

func TestHealth(t *testing.T) {
	h := setupHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, w.Code)
	}
}

func TestCreateAlert(t *testing.T) {
	h := setupHandler()
	body := models.CreateAlertRequest{
		IncidentID: 101,
		Message:    "Evacuate Immediately",
		Severity:   "high",
	}

	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/alerts", bytes.NewReader(raw))
	w := httptest.NewRecorder()

	h.AlertsCollection(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestGetAlertInvalidID(t *testing.T) {
	h := setupHandler()
	req := httptest.NewRequest(http.MethodGet, "/alerts/abc", nil)
	w := httptest.NewRecorder()

	h.AlertByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetAllAlerts(t *testing.T) {
	h := setupHandler()

	body := `{"incidentId":101,"message":"Evacuate Immediately","severity":"HIGH"}`
	createReq := httptest.NewRequest(http.MethodPost, "/alerts", strings.NewReader(body))
	createW := httptest.NewRecorder()
	h.AlertsCollection(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, createW.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	getW := httptest.NewRecorder()
	h.AlertsCollection(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, getW.Code)
	}

	var alerts []models.Alert
	if err := json.Unmarshal(getW.Body.Bytes(), &alerts); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(alerts) == 0 {
		t.Fatal("expected at least one alert")
	}
}
