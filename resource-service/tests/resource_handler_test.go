package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"resource-service/internal/clients"
	"resource-service/internal/handlers"
	"resource-service/internal/models"
	"resource-service/internal/repositories"
	"resource-service/internal/services"
)

// ── In-memory resource repository for testing ────────────────────────────────

type inMemResourceRepo struct {
	mu       sync.RWMutex
	nextID   int
	store    map[int]*models.Resource
}

func newInMemResourceRepo() *inMemResourceRepo {
	return &inMemResourceRepo{nextID: 1, store: make(map[int]*models.Resource)}
}

func (r *inMemResourceRepo) Create(ctx context.Context, resource *models.Resource) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	resource.ID = r.nextID
	r.nextID++
	r.store[resource.ID] = resource
	return nil
}

func (r *inMemResourceRepo) GetByID(ctx context.Context, id int) (*models.Resource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.store[id]
	if !ok {
		return nil, repositories.ErrResourceNotFound
	}
	return res, nil
}

func (r *inMemResourceRepo) Update(ctx context.Context, id int, item string, quantity int, unit string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.store[id]
	if !ok {
		return repositories.ErrResourceNotFound
	}
	res.Item = item
	res.Quantity = quantity
	res.Unit = unit
	res.UpdatedAt = time.Now()
	return nil
}

func (r *inMemResourceRepo) Delete(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[id]; !ok {
		return repositories.ErrResourceNotFound
	}
	delete(r.store, id)
	return nil
}

func (r *inMemResourceRepo) List(ctx context.Context) ([]models.Resource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]models.Resource, 0, len(r.store))
	for _, res := range r.store {
		out = append(out, *res)
	}
	return out, nil
}

func (r *inMemResourceRepo) GetNextID(ctx context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.nextID, nil
}

func (r *inMemResourceRepo) UpdateDispatch(ctx context.Context, id int, available int, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.store[id]
	if !ok {
		return repositories.ErrResourceNotFound
	}
	res.Quantity = available
	res.UpdatedAt = time.Now()
	return nil
}

// ── Helper ────────────────────────────────────────────────────────────────────

func newTestResourceHandler() *handlers.ResourceHandler {
	repo := newInMemResourceRepo()
	shelterClient := clients.NewShelterClient("http://fake-shelter", 1)
	svc := services.NewResourceService(repo, shelterClient)
	return handlers.NewResourceHandler(svc)
}

func itoa(n int) string {
	return fmt.Sprint(n)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestResourceHealth(t *testing.T) {
	h := newTestResourceHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.Health(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestCreateResource(t *testing.T) {
	h := newTestResourceHandler()
	body := bytes.NewBufferString(`{"item":"WATER","quantity":100,"unit":"PACKS"}`)
	req := httptest.NewRequest(http.MethodPost, "/resources", body)
	rr := httptest.NewRecorder()
	h.Resources(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}
}

func TestListResources(t *testing.T) {
	h := newTestResourceHandler()
	req := httptest.NewRequest(http.MethodGet, "/resources", nil)
	rr := httptest.NewRecorder()
	h.Resources(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestGetResourceByIDNotFound(t *testing.T) {
	h := newTestResourceHandler()
	req := httptest.NewRequest(http.MethodGet, "/resources/9999", nil)
	rr := httptest.NewRecorder()
	h.ResourceByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateResourceNotFound(t *testing.T) {
	h := newTestResourceHandler()
	body := bytes.NewBufferString(`{"item":"WATER","quantity":200,"unit":"PACKS"}`)
	req := httptest.NewRequest(http.MethodPut, "/resources/9999", body)
	rr := httptest.NewRecorder()
	h.ResourceByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteResourceNotFound(t *testing.T) {
	h := newTestResourceHandler()
	req := httptest.NewRequest(http.MethodDelete, "/resources/9999", nil)
	rr := httptest.NewRecorder()
	h.ResourceByID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestResourceFullCRUD(t *testing.T) {
	h := newTestResourceHandler()
	
	// Create
	body := bytes.NewBufferString(`{"item":"WATER","quantity":100,"unit":"PACKS"}`)
	req := httptest.NewRequest(http.MethodPost, "/resources", body)
	rr := httptest.NewRecorder()
	h.Resources(rr, req)
	
	var res map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &res)
	idFloat, ok := res["id"].(float64)
	if !ok {
		t.Fatalf("id not found in response: %v", res)
	}
	id := int(idFloat)

	// Get by ID
	req = httptest.NewRequest(http.MethodGet, "/resources/"+itoa(id), nil)
	rr = httptest.NewRecorder()
	h.ResourceByID(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("get by id failed: %d", rr.Code)
	}

	// Update
	upBody := bytes.NewBufferString(`{"item":"FOOD","quantity":50,"unit":"KG"}`)
	req = httptest.NewRequest(http.MethodPut, "/resources/" + itoa(id), upBody)
	rr = httptest.NewRecorder()
	h.ResourceByID(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("update failed: %d", rr.Code)
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/resources/" + itoa(id), nil)
	rr = httptest.NewRecorder()
	h.ResourceByID(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("delete failed: %d", rr.Code)
	}
}
