package services

import "incident-service/internal/models"

type IncidentService struct{}

func NewIncidentService() *IncidentService {
	return &IncidentService{}
}

func (s *IncidentService) List() []models.Incident {
	return []models.Incident{}
}
