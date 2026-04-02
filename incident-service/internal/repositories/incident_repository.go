package repositories

import (
	"context"
	"errors"
	"sync"

	"incident-service/internal/models"
)

var ErrIncidentNotFound = errors.New("incident not found")

type Pinger interface {
	Ping(ctx context.Context) error
}

type IncidentRepository interface {
	Create(models.Incident) (models.Incident, error)
	List() []models.Incident
	GetByID(id int) (models.Incident, error)
	Update(id int, req models.UpdateIncidentRequest) (models.Incident, error)
	UpdateStatus(id int, status string) (models.Incident, error)
	Delete(id int) error
}

type InMemoryIncidentRepository struct {
	mu     sync.RWMutex
	nextID int
	byID   map[int]models.Incident
}

func NewInMemoryIncidentRepository() *InMemoryIncidentRepository {
	return &InMemoryIncidentRepository{
		nextID: 101,
		byID:   make(map[int]models.Incident),
	}
}

func (r *InMemoryIncidentRepository) Create(incident models.Incident) (models.Incident, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	incident.ID = r.nextID
	r.nextID++
	r.byID[incident.ID] = incident
	return incident, nil
}

func (r *InMemoryIncidentRepository) List() []models.Incident {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.Incident, 0, len(r.byID))
	for _, incident := range r.byID {
		result = append(result, incident)
	}
	return result
}

func (r *InMemoryIncidentRepository) GetByID(id int) (models.Incident, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	incident, ok := r.byID[id]
	if !ok {
		return models.Incident{}, ErrIncidentNotFound
	}
	return incident, nil
}

func (r *InMemoryIncidentRepository) UpdateStatus(id int, status string) (models.Incident, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	incident, ok := r.byID[id]
	if !ok {
		return models.Incident{}, ErrIncidentNotFound
	}

	incident.Status = status
	r.byID[id] = incident
	return incident, nil
}

func (r *InMemoryIncidentRepository) Update(id int, req models.UpdateIncidentRequest) (models.Incident, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	incident, ok := r.byID[id]
	if !ok {
		return models.Incident{}, ErrIncidentNotFound
	}

	incident.Type = req.Type
	incident.Location = req.Location
	incident.Severity = req.Severity
	r.byID[id] = incident
	return incident, nil
}

func (r *InMemoryIncidentRepository) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byID[id]; !ok {
		return ErrIncidentNotFound
	}

	delete(r.byID, id)
	return nil
}

func (r *InMemoryIncidentRepository) Ping(_ context.Context) error {
	return nil
}
