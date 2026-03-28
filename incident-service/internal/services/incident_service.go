package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"incident-service/internal/models"
	"incident-service/internal/repositories"
)

var ErrValidation = errors.New("validation failed")

type Integrator interface {
	NotifyAlert(incident models.Incident) error
	OpenShelter(incident models.Incident) error
}

type HTTPIntegrator struct {
	client          *http.Client
	alertEndpoint   string
	shelterEndpoint string
}

func NewHTTPIntegrator(alertEndpoint, shelterEndpoint string) *HTTPIntegrator {
	return &HTTPIntegrator{
		client:          &http.Client{Timeout: 5 * time.Second},
		alertEndpoint:   alertEndpoint,
		shelterEndpoint: shelterEndpoint,
	}
}

func (i *HTTPIntegrator) NotifyAlert(incident models.Incident) error {
	payload := map[string]any{
		"incidentId": incident.ID,
		"message":    fmt.Sprintf("New %s Reported in %s", strings.Title(strings.ToLower(incident.Type)), incident.Location),
		"severity":   incident.Severity,
	}

	return i.postJSON(i.alertEndpoint, payload)
}

func (i *HTTPIntegrator) OpenShelter(incident models.Incident) error {
	payload := map[string]any{
		"incidentId": incident.ID,
		"type":       incident.Type,
		"location":   incident.Location,
		"severity":   incident.Severity,
		"status":     incident.Status,
	}

	return i.postJSON(i.shelterEndpoint, payload)
}

func (i *HTTPIntegrator) postJSON(url string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := i.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("downstream request failed with status %d", resp.StatusCode)
	}

	return nil
}

type IncidentService struct {
	repository repositories.IncidentRepository
	integrator Integrator
}

func NewIncidentService(repository repositories.IncidentRepository, integrator Integrator) *IncidentService {
	return &IncidentService{repository: repository, integrator: integrator}
}

func (s *IncidentService) Create(req models.CreateIncidentRequest) (models.CreateIncidentResponse, error) {
	if strings.TrimSpace(req.Type) == "" || strings.TrimSpace(req.Location) == "" || strings.TrimSpace(req.Severity) == "" {
		return models.CreateIncidentResponse{}, ErrValidation
	}

	incident := s.repository.Create(models.Incident{
		Type:     strings.ToUpper(strings.TrimSpace(req.Type)),
		Location: strings.TrimSpace(req.Location),
		Severity: strings.ToUpper(strings.TrimSpace(req.Severity)),
		Status:   "ACTIVE",
	})

	if s.integrator != nil {
		_ = s.integrator.NotifyAlert(incident)
		_ = s.integrator.OpenShelter(incident)
	}

	return models.CreateIncidentResponse{
		ID:      incident.ID,
		Type:    incident.Type,
		Status:  incident.Status,
		Message: "Incident Reported Successfully",
	}, nil
}

func (s *IncidentService) GetByID(id int) (models.Incident, error) {
	return s.repository.GetByID(id)
}

func (s *IncidentService) ListActive() []models.Incident {
	return s.repository.ListActive()
}

func (s *IncidentService) UpdateStatus(id int, status string) (models.Incident, error) {
	normalizedStatus := strings.ToUpper(strings.TrimSpace(status))
	if normalizedStatus == "" {
		normalizedStatus = "RESOLVED"
	}

	return s.repository.UpdateStatus(id, normalizedStatus)
}
