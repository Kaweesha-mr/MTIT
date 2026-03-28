package models

type Incident struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Location string `json:"location"`
	Severity string `json:"severity"`
	Status   string `json:"status"`
}

type CreateIncidentRequest struct {
	Type     string `json:"type"`
	Location string `json:"location"`
	Severity string `json:"severity"`
}

type CreateIncidentResponse struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type UpdateIncidentStatusRequest struct {
	Status string `json:"status"`
}
