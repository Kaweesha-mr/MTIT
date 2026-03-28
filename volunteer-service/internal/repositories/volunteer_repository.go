package repositories

import (
	"context"
	"errors"
	"sync"

	"volunteer-service/internal/models"
)

var ErrVolunteerNotFound = errors.New("volunteer not found")

type Pinger interface {
	Ping(ctx context.Context) error
}

type VolunteerRepository interface {
	Create(volunteer models.Volunteer) (models.Volunteer, error)
	GetByID(id int) (models.Volunteer, error)
	Update(volunteer models.Volunteer) (models.Volunteer, error)
	Delete(id int) error
	List() ([]models.Volunteer, error)
}

type InMemoryVolunteerRepository struct {
	mu         sync.RWMutex
	nextID     int
	volunteers map[int]models.Volunteer
}

func NewInMemoryVolunteerRepository() *InMemoryVolunteerRepository {
	return &InMemoryVolunteerRepository{
		nextID:     500,
		volunteers: make(map[int]models.Volunteer),
	}
}

func (r *InMemoryVolunteerRepository) Create(volunteer models.Volunteer) (models.Volunteer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	volunteer.ID = r.nextID
	r.volunteers[volunteer.ID] = volunteer

	return volunteer, nil
}

func (r *InMemoryVolunteerRepository) GetByID(id int) (models.Volunteer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	volunteer, ok := r.volunteers[id]
	if !ok {
		return models.Volunteer{}, ErrVolunteerNotFound
	}

	return volunteer, nil
}

func (r *InMemoryVolunteerRepository) Update(volunteer models.Volunteer) (models.Volunteer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.volunteers[volunteer.ID]; !ok {
		return models.Volunteer{}, ErrVolunteerNotFound
	}

	r.volunteers[volunteer.ID] = volunteer
	return volunteer, nil
}

func (r *InMemoryVolunteerRepository) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.volunteers[id]; !ok {
		return ErrVolunteerNotFound
	}

	delete(r.volunteers, id)
	return nil
}

func (r *InMemoryVolunteerRepository) List() ([]models.Volunteer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.Volunteer, 0, len(r.volunteers))
	for _, v := range r.volunteers {
		result = append(result, v)
	}
	return result, nil
}

func (r *InMemoryVolunteerRepository) Ping(_ context.Context) error {
	return nil
}
