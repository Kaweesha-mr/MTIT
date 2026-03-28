package repositories

import (
	"errors"
	"sync"

	"incident-service/internal/models"
)

var ErrIncidentNotFound = errors.New("incident not found")

type IncidentRepository interface {
	Create(incident models.Incident) models.Incident
	GetByID(id int) (models.Incident, error)
	ListActive() []models.Incident
	UpdateStatus(id int, status string) (models.Incident, error)
}

type InMemoryIncidentRepository struct {
	mu        sync.RWMutex
	nextID    int
	incidents map[int]models.Incident
}

func NewInMemoryIncidentRepository() *InMemoryIncidentRepository {
	return &InMemoryIncidentRepository{
		nextID:    100,
		incidents: make(map[int]models.Incident),
	}
}

func (r *InMemoryIncidentRepository) Create(incident models.Incident) models.Incident {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	incident.ID = r.nextID
	r.incidents[incident.ID] = incident

	return incident
}

func (r *InMemoryIncidentRepository) GetByID(id int) (models.Incident, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	incident, ok := r.incidents[id]
	if !ok {
		return models.Incident{}, ErrIncidentNotFound
	}

	return incident, nil
}

func (r *InMemoryIncidentRepository) ListActive() []models.Incident {
	r.mu.RLock()
	defer r.mu.RUnlock()

	active := make([]models.Incident, 0, len(r.incidents))
	for _, incident := range r.incidents {
		if incident.Status == "ACTIVE" {
			active = append(active, incident)
		}
	}

	return active
}

func (r *InMemoryIncidentRepository) UpdateStatus(id int, status string) (models.Incident, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	incident, ok := r.incidents[id]
	if !ok {
		return models.Incident{}, ErrIncidentNotFound
	}

	incident.Status = status
	r.incidents[id] = incident

	return incident, nil
}
