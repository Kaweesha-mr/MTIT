package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"volunteer-service/internal/models"
)

type IncidentVerifier interface {
	GetIncident(incidentID int) (models.IncidentSummary, error)
}

type LogisticsChecker interface {
	GetAssignment(volunteerID int) (models.LogisticsAssignment, error)
}

type HTTPIncidentVerifier struct {
	client  *http.Client
	baseURL string
}

type HTTPLogisticsChecker struct {
	client  *http.Client
	baseURL string
}

func NewHTTPIncidentVerifier(baseURL string) *HTTPIncidentVerifier {
	return &HTTPIncidentVerifier{
		client:  &http.Client{Timeout: 5 * time.Second},
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

func NewHTTPLogisticsChecker(baseURL string) *HTTPLogisticsChecker {
	return &HTTPLogisticsChecker{
		client:  &http.Client{Timeout: 5 * time.Second},
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

func (v *HTTPIncidentVerifier) GetIncident(incidentID int) (models.IncidentSummary, error) {
	url := fmt.Sprintf("%s/incidents/%d", v.baseURL, incidentID)

	resp, err := v.client.Get(url)
	if err != nil {
		return models.IncidentSummary{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return models.IncidentSummary{}, ErrIncidentUnavailable
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return models.IncidentSummary{}, fmt.Errorf("incident service returned status %d", resp.StatusCode)
	}

	var incident models.IncidentSummary
	if err := json.NewDecoder(resp.Body).Decode(&incident); err != nil {
		return models.IncidentSummary{}, err
	}

	return incident, nil
}

func (c *HTTPLogisticsChecker) GetAssignment(volunteerID int) (models.LogisticsAssignment, error) {
	url := fmt.Sprintf("%s/volunteers/%d", c.baseURL, volunteerID)

	resp, err := c.client.Get(url)
	if err != nil {
		return models.LogisticsAssignment{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return models.LogisticsAssignment{}, ErrLogisticsUnavailable
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return models.LogisticsAssignment{}, fmt.Errorf("logistics service returned status %d", resp.StatusCode)
	}

	var payload struct {
		VolunteerID       int   `json:"volunteerId"`
		AssignedToVehicle *bool `json:"assignedToVehicle"`
		Assigned          *bool `json:"assigned"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return models.LogisticsAssignment{}, err
	}

	assigned := false
	if payload.AssignedToVehicle != nil {
		assigned = *payload.AssignedToVehicle
	} else if payload.Assigned != nil {
		assigned = *payload.Assigned
	}

	if payload.VolunteerID == 0 {
		payload.VolunteerID = volunteerID
	}

	return models.LogisticsAssignment{
		VolunteerID:       payload.VolunteerID,
		AssignedToVehicle: assigned,
	}, nil
}
