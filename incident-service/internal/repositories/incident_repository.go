package repositories

import "incident-service/internal/models"

type IncidentRepository interface {
	List() []models.Incident
}
