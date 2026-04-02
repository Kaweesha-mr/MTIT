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

var (
	ErrValidation      = errors.New("validation failed")
	ErrIntegrationFail = errors.New("integration failed")
)

type IncidentService struct {
	repo       repositories.IncidentRepository
	alertURL   string
	shelterURL string
	httpClient *http.Client
}

func NewIncidentService(repo repositories.IncidentRepository, alertURL, shelterURL string) *IncidentService {
	client := &http.Client{Timeout: 5 * time.Second}
	return &IncidentService{repo: repo, alertURL: strings.TrimRight(alertURL, "/"), shelterURL: strings.TrimRight(shelterURL, "/"), httpClient: client}
}

func (s *IncidentService) CreateIncident(req models.CreateIncidentRequest) (models.Incident, error) {
	if strings.TrimSpace(req.Type) == "" || strings.TrimSpace(req.Location) == "" || strings.TrimSpace(req.Severity) == "" {
		return models.Incident{}, ErrValidation
	}

	incident := models.Incident{
		Type:     strings.ToUpper(strings.TrimSpace(req.Type)),
		Location: strings.TrimSpace(req.Location),
		Severity: strings.ToUpper(strings.TrimSpace(req.Severity)),
		Status:   "ACTIVE",
	}

	created, err := s.repo.Create(incident)
	if err != nil {
		return models.Incident{}, err
	}

	downstream := &models.DownstreamServices{}

	if alertID, err := s.triggerAlert(created); err == nil {
		downstream.AlertID = alertID
	} else {
		return models.Incident{}, fmt.Errorf("%w: alert", ErrIntegrationFail)
	}

	if shelterID, err := s.triggerShelter(created); err == nil {
		downstream.ShelterID = shelterID
	} else {
		return models.Incident{}, fmt.Errorf("%w: shelter", ErrIntegrationFail)
	}

	created.Downstream = downstream
	return created, nil
}

func (s *IncidentService) List() []models.Incident {
	return s.repo.List()
}

func (s *IncidentService) GetByID(id int) (models.Incident, error) {
	return s.repo.GetByID(id)
}

func (s *IncidentService) UpdateIncident(id int, req models.UpdateIncidentRequest) (models.Incident, error) {
	if strings.TrimSpace(req.Type) == "" || strings.TrimSpace(req.Location) == "" || strings.TrimSpace(req.Severity) == "" {
		return models.Incident{}, ErrValidation
	}

	update := models.UpdateIncidentRequest{
		Type:     strings.ToUpper(strings.TrimSpace(req.Type)),
		Location: strings.TrimSpace(req.Location),
		Severity: strings.ToUpper(strings.TrimSpace(req.Severity)),
	}

	return s.repo.Update(id, update)
}

func (s *IncidentService) UpdateStatus(id int, status string) (models.Incident, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		return models.Incident{}, ErrValidation
	}

	switch status {
	case "ACTIVE", "RESOLVED", "IN_PROGRESS":
	default:
		return models.Incident{}, ErrValidation
	}

	return s.repo.UpdateStatus(id, status)
}

func (s *IncidentService) DeleteIncident(id int) error {
	return s.repo.Delete(id)
}

func (s *IncidentService) triggerAlert(incident models.Incident) (int, error) {
	body := map[string]any{
		"incidentId": incident.ID,
		"message":    fmt.Sprintf("%s in %s", incident.Type, incident.Location),
		"severity":   incident.Severity,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}

	resp, err := s.httpClient.Post(s.alertURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("received status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if id, ok := result["id"].(float64); ok {
		return int(id), nil
	}

	return 0, nil
}

func (s *IncidentService) triggerShelter(incident models.Incident) (int, error) {
	body := map[string]any{
		"incidentId": incident.ID,
		"name":       "Emergency Shelter",
		"capacity":   500,
		"status":     "REQUEST",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}

	resp, err := s.httpClient.Post(s.shelterURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("received status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if id, ok := result["id"].(float64); ok {
		return int(id), nil
	}

	return 0, nil
}
