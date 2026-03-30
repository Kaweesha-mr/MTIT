package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"resource-service/internal/clients"
	"resource-service/internal/models"
	"resource-service/internal/repositories"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidDispatchQty = errors.New("dispatch quantity must be positive")
	ErrInsufficientStock  = errors.New("insufficient stock")
	ErrShelterUnavailable = errors.New("shelter service unavailable")
	ErrShelterRejected    = errors.New("dispatch rejected. shelter is closed or full")
	ErrShelterNotFound    = errors.New("shelter not found")
	ErrResourceNotFound   = repositories.ErrResourceNotFound
)

type ResourceService struct {
	repo          repositories.ResourceRepository
	shelterClient *clients.ShelterClient
}

func NewResourceService(repo repositories.ResourceRepository, shelterClient *clients.ShelterClient) *ResourceService {
	return &ResourceService{repo: repo, shelterClient: shelterClient}
}

func (s *ResourceService) CreateResource(ctx context.Context, req models.CreateResourceRequest) (*models.Resource, error) {
	if strings.TrimSpace(req.Item) == "" || strings.TrimSpace(req.Unit) == "" || req.Quantity <= 0 {
		return nil, ErrInvalidInput
	}

	nextID, err := s.repo.GetNextID(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}

	resource := &models.Resource{
		ID:        nextID,
		Item:      strings.TrimSpace(req.Item),
		Quantity:  req.Quantity,
		Unit:      strings.TrimSpace(req.Unit),
		Available: req.Quantity,
		Weight:    fmt.Sprintf("%dkg", req.Quantity),
		Status:    models.ResourceStatusAvailable,
	}

	if err := s.repo.Create(ctx, resource); err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	return resource, nil
}

func (s *ResourceService) GetResource(ctx context.Context, id int) (*models.Resource, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}

	resource, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (s *ResourceService) DispatchResource(ctx context.Context, resourceID int, req models.DispatchRequest) (*models.Resource, *models.Shelter, error) {
	if resourceID <= 0 || req.ShelterID <= 0 {
		return nil, nil, ErrInvalidInput
	}
	if req.Quantity <= 0 {
		return nil, nil, ErrInvalidDispatchQty
	}

	resource, err := s.repo.GetByID(ctx, resourceID)
	if err != nil {
		return nil, nil, err
	}

	if resource.Available < req.Quantity {
		return nil, nil, ErrInsufficientStock
	}

	callCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	shelter, statusCode, err := s.shelterClient.GetShelter(callCtx, req.ShelterID)
	if err != nil {
		if statusCode == 404 {
			return nil, nil, ErrShelterNotFound
		}
		if statusCode >= 400 && statusCode < 500 {
			return nil, nil, fmt.Errorf("%w: invalid shelter", ErrShelterRejected)
		}
		return nil, nil, fmt.Errorf("%w: %v", ErrShelterUnavailable, err)
	}

	if strings.EqualFold(strings.TrimSpace(shelter.Status), "CLOSED") {
		return nil, nil, ErrShelterRejected
	}
	if shelter.CurrentOccupancy >= shelter.MaxCapacity {
		return nil, nil, ErrShelterRejected
	}

	resource.Available -= req.Quantity
	resource.Status = models.ResourceStatusDispatched
	if resource.Available == 0 {
		resource.Status = models.ResourceStatusDispatched
	}

	if err := s.repo.UpdateDispatch(ctx, resource.ID, resource.Available, resource.Status); err != nil {
		return nil, nil, err
	}

	return resource, shelter, nil
}
