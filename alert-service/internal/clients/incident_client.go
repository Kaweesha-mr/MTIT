package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// IncidentValidator defines the interface for incident service calls
type IncidentValidator interface {
	GetIncident(ctx context.Context, incidentID int) (*IncidentResponse, int, error)
	ValidateIncidentExists(ctx context.Context, incidentID int) error
}

// IncidentResponse represents an incident as returned from Incident Service
type IncidentResponse struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Location  string `json:"location"`
	Severity  string `json:"severity"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// IncidentClient wraps HTTP client for Incident Service calls
type IncidentClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewIncidentClient creates a new incident client with timeout
func NewIncidentClient(baseURL string) *IncidentClient {
	return &IncidentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetIncident retrieves incident details from Incident Service
func (c *IncidentClient) GetIncident(ctx context.Context, incidentID int) (*IncidentResponse, int, error) {
	url := fmt.Sprintf("%s/incidents/%d", c.baseURL, incidentID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to call incident service: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	// If not found or other error status
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("incident service returned status %d", resp.StatusCode)
	}

	// Parse successful response
	var incident IncidentResponse
	if err := json.Unmarshal(body, &incident); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to parse incident response: %w", err)
	}

	return &incident, resp.StatusCode, nil
}

// ValidateIncidentExists checks if incident exists and is ACTIVE
func (c *IncidentClient) ValidateIncidentExists(ctx context.Context, incidentID int) error {
	incident, statusCode, err := c.GetIncident(ctx, incidentID)
	if err != nil {
		if statusCode == http.StatusNotFound {
			return NewValidationError(
				"incident_not_found",
				fmt.Sprintf("Incident with ID %d not found", incidentID),
				statusCode,
			)
		}
		// Service unavailable or other network error
		return NewValidationError(
			"service_unavailable",
			"Incident Service is unavailable",
			http.StatusServiceUnavailable,
		)
	}

	// Check if incident is ACTIVE
	if incident.Status != "ACTIVE" {
		return NewValidationError(
			"incident_not_active",
			fmt.Sprintf("Incident is in %s status, cannot create alert for non-active incident", incident.Status),
			http.StatusConflict,
		)
	}

	return nil
}

// ValidationError is returned when incident validation fails
type ValidationError struct {
	Code       string
	Message    string
	StatusCode int
}

// NewValidationError creates a validation error
func NewValidationError(code, message string, statusCode int) *ValidationError {
	return &ValidationError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Error implements error interface
func (e *ValidationError) Error() string {
	return e.Message
}

// IsValidationError checks if error is a validation error
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// GetValidationError returns validation error if applicable
func GetValidationError(err error) *ValidationError {
	if ve, ok := err.(*ValidationError); ok {
		return ve
	}
	return nil
}
