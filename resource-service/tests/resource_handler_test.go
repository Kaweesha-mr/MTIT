package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"resource-service/internal/handlers"
)

func TestHealth(t *testing.T) {
	h := handlers.NewResourceHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestResources_MethodNotAllowed_ReturnsJSON(t *testing.T) {
	h := handlers.NewResourceHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/resources", nil)
	w := httptest.NewRecorder()

	h.Resources(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}

	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content type, got %s", got)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response must be valid json: %v", err)
	}

	if body["message"] == "" {
		t.Fatalf("expected non-empty message in json error body")
	}
}

func TestResourceByID_InvalidID_ReturnsJSON(t *testing.T) {
	h := handlers.NewResourceHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/resources/abc", nil)
	w := httptest.NewRecorder()

	h.ResourceByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content type, got %s", got)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response must be valid json: %v", err)
	}

	if body["message"] == "" {
		t.Fatalf("expected non-empty message in json error body")
	}
}
