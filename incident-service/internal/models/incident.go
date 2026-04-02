package models

type Incident struct {
	ID       int              `json:"id"`
	Type     string           `json:"type"`
	Location string           `json:"location"`
	Severity string           `json:"severity"`
	Status   string           `json:"status"`
	Downstream *DownstreamServices `json:"downstream,omitempty"`
}

type DownstreamServices struct {
	AlertID   int `json:"alertId,omitempty"`
	ShelterID int `json:"shelterId,omitempty"`
}

type CreateIncidentRequest struct {
	Type     string `json:"type"`
	Location string `json:"location"`
	Severity string `json:"severity"`
}

type UpdateIncidentRequest struct {
	Type     string `json:"type"`
	Location string `json:"location"`
	Severity string `json:"severity"`
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}
