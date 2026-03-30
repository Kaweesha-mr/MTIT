package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"volunteer-service/internal/routes"
)

func TestCreateVolunteer(t *testing.T) {
	h := routes.NewRouter()

	body := map[string]any{
		"name":  "John Doe",
		"role":  "RESCUE",
		"phone": "0771234567",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/volunteers", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if created["status"] != "AVAILABLE" {
		t.Fatalf("expected status AVAILABLE, got %v", created["status"])
	}
}

func TestGetVolunteerByID(t *testing.T) {
	h := routes.NewRouter()

	body := map[string]any{
		"name":  "Jane Doe",
		"role":  "DOCTOR",
		"phone": "0779876543",
	}
	payload, _ := json.Marshal(body)

	createReq := httptest.NewRequest(http.MethodPost, "/volunteers", bytes.NewBuffer(payload))
	createRec := httptest.NewRecorder()
	h.ServeHTTP(createRec, createReq)

	getReq := httptest.NewRequest(http.MethodGet, "/volunteers/501", nil)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRec.Code)
	}

	var volunteer map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &volunteer); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if volunteer["name"] != "Jane Doe" {
		t.Fatalf("expected name Jane Doe, got %v", volunteer["name"])
	}
}
