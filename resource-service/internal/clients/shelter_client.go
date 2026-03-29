package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"resource-service/internal/models"
)

type ShelterClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewShelterClient(baseURL string, timeoutSeconds int) *ShelterClient {
	return &ShelterClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

func (c *ShelterClient) GetShelter(ctx context.Context, shelterID int) (*models.Shelter, int, error) {
	url := fmt.Sprintf("%s/shelters/%d", c.baseURL, shelterID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create shelter request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("call shelter service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("shelter service returned status %d", resp.StatusCode)
	}

	var shelter models.Shelter
	if err := json.NewDecoder(resp.Body).Decode(&shelter); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode shelter response: %w", err)
	}

	return &shelter, resp.StatusCode, nil
}
