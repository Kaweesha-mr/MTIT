package repositories

import (
	"errors"
	"sync"

	"volunteer-service/internal/models"
)

var ErrVolunteerNotFound = errors.New("volunteer not found")

type VolunteerRepository interface {
	Create(volunteer models.Volunteer) (models.Volunteer, error)
	GetByID(id int) (models.Volunteer, error)
	Update(volunteer models.Volunteer) (models.Volunteer, error)
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
