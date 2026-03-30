package models

type Alert struct {
	ID         int    `json:"id"`
	IncidentID int    `json:"incidentId"`
	Message    string `json:"message"`
	Severity   string `json:"severity"`
	Status     string `json:"status"`
	Timestamp  string `json:"timestamp"`
}

type CreateAlertRequest struct {
	IncidentID int    `json:"incidentId"`
	Message    string `json:"message"`
	Severity   string `json:"severity"`
}

type CreateAlertResponse struct {
	ID        int    `json:"id"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type UpdateAlertRequest struct {
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type AlertDetailResponse struct {
	ID      int    `json:"id"`
	Message string `json:"message"`
	Status  string `json:"status"`
}
