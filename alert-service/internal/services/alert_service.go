package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"alert-service/internal/clients"
	"alert-service/internal/models"
	"alert-service/internal/repositories"
)

var ErrValidation = errors.New("validation failed")

type AlertService struct {
	repository      repositories.AlertRepository
	incidentClient  clients.IncidentValidator
}

func NewAlertService(repository repositories.AlertRepository, incidentClient clients.IncidentValidator) *AlertService {
	return &AlertService{
		repository:     repository,
		incidentClient: incidentClient,
	}
}

func (s *AlertService) Create(ctx context.Context, req models.CreateAlertRequest) (models.CreateAlertResponse, error) {
	if req.IncidentID <= 0 || strings.TrimSpace(req.Message) == "" || strings.TrimSpace(req.Severity) == "" {
		return models.CreateAlertResponse{}, ErrValidation
	}

	// Validate incident exists and is ACTIVE
	if err := s.incidentClient.ValidateIncidentExists(ctx, req.IncidentID); err != nil {
		return models.CreateAlertResponse{}, err
	}

	alert := models.Alert{
		IncidentID: req.IncidentID,
		Message:    strings.TrimSpace(req.Message),
		Severity:   strings.ToUpper(strings.TrimSpace(req.Severity)),
		Status:     "BROADCASTED",
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	saved, err := s.repository.Create(alert)
	if err != nil {
		return models.CreateAlertResponse{}, err
	}

	return models.CreateAlertResponse{
		ID:        saved.ID,
		Status:    saved.Status,
		Timestamp: saved.Timestamp,
	}, nil
}

func (s *AlertService) GetByID(id int) (models.AlertDetailResponse, error) {
	alert, err := s.repository.GetByID(id)
	if err != nil {
		return models.AlertDetailResponse{}, err
	}

	return models.AlertDetailResponse{
		ID:      alert.ID,
		Message: alert.Message,
		Status:  alert.Status,
	}, nil
}

func (s *AlertService) GetAll() ([]models.Alert, error) {
	return s.repository.GetAll()
}

func (s *AlertService) Update(id int, req models.UpdateAlertRequest) (models.Alert, error) {
	if strings.TrimSpace(req.Message) == "" || strings.TrimSpace(req.Severity) == "" {
		return models.Alert{}, ErrValidation
	}

	req.Message = strings.TrimSpace(req.Message)
	req.Severity = strings.ToUpper(strings.TrimSpace(req.Severity))

	return s.repository.Update(id, req)
}

func (s *AlertService) Delete(id int) error {
	return s.repository.Delete(id)
}
