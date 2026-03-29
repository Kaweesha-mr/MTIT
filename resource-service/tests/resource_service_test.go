package tests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"resource-service/internal/clients"
	"resource-service/internal/models"
	"resource-service/internal/repositories"
	"resource-service/internal/services"
)

type fakeResourceRepo struct {
	resource *models.Resource
	nextID   int
}

func (f *fakeResourceRepo) Create(_ context.Context, resource *models.Resource) error {
	f.resource = resource
	return nil
}

func (f *fakeResourceRepo) GetByID(_ context.Context, id int) (*models.Resource, error) {
	if f.resource == nil || f.resource.ID != id {
		return nil, repositories.ErrResourceNotFound
	}
	return f.resource, nil
}

func (f *fakeResourceRepo) UpdateDispatch(_ context.Context, id int, available int, status string) error {
	if f.resource == nil || f.resource.ID != id {
		return repositories.ErrResourceNotFound
	}
	f.resource.Available = available
	f.resource.Status = status
	return nil
}

func (f *fakeResourceRepo) GetNextID(_ context.Context) (int, error) {
	if f.nextID == 0 {
		return 701, nil
	}
	return f.nextID, nil
}

func TestCreateResource_DefaultsAreSet(t *testing.T) {
	repo := &fakeResourceRepo{nextID: 701}
	svc := services.NewResourceService(repo, nil)

	resource, err := svc.CreateResource(context.Background(), models.CreateResourceRequest{
		Item:     "WATER_BOTTLES",
		Quantity: 500,
		Unit:     "PACKS",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resource.ID != 701 {
		t.Fatalf("expected id 701, got %d", resource.ID)
	}
	if resource.Available != 500 {
		t.Fatalf("expected available 500, got %d", resource.Available)
	}
	if resource.Status != models.ResourceStatusAvailable {
		t.Fatalf("expected status AVAILABLE, got %s", resource.Status)
	}
	if resource.Weight != "500kg" {
		t.Fatalf("expected weight 500kg, got %s", resource.Weight)
	}
}

func TestDispatchResource_RejectsInsufficientStock(t *testing.T) {
	repo := &fakeResourceRepo{resource: &models.Resource{ID: 701, Available: 10, Status: models.ResourceStatusAvailable}}
	svc := services.NewResourceService(repo, nil)

	_, _, err := svc.DispatchResource(context.Background(), 701, models.DispatchRequest{ShelterID: 301, Quantity: 100})
	if !errors.Is(err, services.ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestDispatchResource_RejectsClosedShelter(t *testing.T) {
	shelterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":301,"name":"Central School","currentOccupancy":10,"maxCapacity":500,"status":"CLOSED"}`))
	}))
	defer shelterServer.Close()

	repo := &fakeResourceRepo{resource: &models.Resource{ID: 701, Available: 500, Status: models.ResourceStatusAvailable}}
	client := clients.NewShelterClient(shelterServer.URL, 2)
	svc := services.NewResourceService(repo, client)

	_, _, err := svc.DispatchResource(context.Background(), 701, models.DispatchRequest{ShelterID: 301, Quantity: 100})
	if !errors.Is(err, services.ErrShelterRejected) {
		t.Fatalf("expected ErrShelterRejected for closed shelter, got %v", err)
	}
}

func TestDispatchResource_RejectsFullShelter(t *testing.T) {
	shelterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":301,"name":"Central School","currentOccupancy":500,"maxCapacity":500,"status":"OPEN"}`))
	}))
	defer shelterServer.Close()

	repo := &fakeResourceRepo{resource: &models.Resource{ID: 701, Available: 500, Status: models.ResourceStatusAvailable}}
	client := clients.NewShelterClient(shelterServer.URL, 2)
	svc := services.NewResourceService(repo, client)

	_, _, err := svc.DispatchResource(context.Background(), 701, models.DispatchRequest{ShelterID: 301, Quantity: 100})
	if !errors.Is(err, services.ErrShelterRejected) {
		t.Fatalf("expected ErrShelterRejected for full shelter, got %v", err)
	}
}
