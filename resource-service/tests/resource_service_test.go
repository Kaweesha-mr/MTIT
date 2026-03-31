package tests

import (
	"context"
	"errors"
	"testing"

	"resource-service/internal/models"
	"resource-service/internal/repositories"
	"resource-service/internal/services"
)

type fakeResourceRepo struct {
	resources map[int]*models.Resource
	nextID    int
}

func (f *fakeResourceRepo) Create(ctx context.Context, r *models.Resource) error {
	r.ID = f.nextID
	f.nextID++
	f.resources[r.ID] = r
	return nil
}

func (f *fakeResourceRepo) GetByID(ctx context.Context, id int) (*models.Resource, error) {
	r, ok := f.resources[id]
	if !ok {
		return nil, repositories.ErrResourceNotFound
	}
	return r, nil
}

func (f *fakeResourceRepo) UpdateDispatch(ctx context.Context, id int, avail int, status string) error {
	r, ok := f.resources[id]
	if !ok {
		return repositories.ErrResourceNotFound
	}
	r.Quantity = avail
	return nil
}

func (f *fakeResourceRepo) Update(ctx context.Context, id int, item string, qty int, unit string) error {
	r, ok := f.resources[id]
	if !ok {
		return repositories.ErrResourceNotFound
	}
	r.Item = item
	r.Quantity = qty
	r.Unit = unit
	return nil
}

func (f *fakeResourceRepo) Delete(ctx context.Context, id int) error {
	delete(f.resources, id)
	return nil
}

func (f *fakeResourceRepo) GetNextID(ctx context.Context) (int, error) {
	return f.nextID, nil
}

func (f *fakeResourceRepo) List(ctx context.Context) ([]models.Resource, error) {
	var list []models.Resource
	for _, r := range f.resources {
		list = append(list, *r)
	}
	return list, nil
}

func TestCreateResource_ValidInput_Success(t *testing.T) {
	repo := &fakeResourceRepo{resources: make(map[int]*models.Resource), nextID: 1}
	svc := services.NewResourceService(repo, nil)

	req := models.CreateResourceRequest{
		Item:     "Water",
		Quantity: 100,
		Unit:     "Liters",
	}

	res, err := svc.CreateResource(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Item != "Water" || res.Quantity != 100 {
		t.Errorf("unexpected resource details: %+v", res)
	}
}

func TestCreateResource_EmptyItem_ReturnsError(t *testing.T) {
	repo := &fakeResourceRepo{resources: make(map[int]*models.Resource), nextID: 1}
	svc := services.NewResourceService(repo, nil)

	req := models.CreateResourceRequest{
		Item:     "",
		Quantity: 100,
		Unit:     "Liters",
	}

	_, err := svc.CreateResource(context.Background(), req)
	if !errors.Is(err, services.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateResource_InvalidQuantity_ReturnsError(t *testing.T) {
	repo := &fakeResourceRepo{resources: make(map[int]*models.Resource), nextID: 1}
	svc := services.NewResourceService(repo, nil)

	req := models.CreateResourceRequest{
		Item:     "Water",
		Quantity: 0,
		Unit:     "Liters",
	}

	_, err := svc.CreateResource(context.Background(), req)
	if !errors.Is(err, services.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetResource_ExistingID_ReturnsResource(t *testing.T) {
	repo := &fakeResourceRepo{resources: make(map[int]*models.Resource), nextID: 1}
	repo.Create(context.Background(), &models.Resource{Item: "Food", Quantity: 50, Unit: "kg"})
	svc := services.NewResourceService(repo, nil)

	res, err := svc.GetResource(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.ID != 1 || res.Item != "Food" {
		t.Errorf("unexpected resource details: %+v", res)
	}
}
