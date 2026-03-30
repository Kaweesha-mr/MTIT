package models

type Volunteer struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Role               string `json:"role"`
	Phone              string `json:"phone"`
	Status             string `json:"status"`
	LicenseValid       bool   `json:"licenseValid"`
	AssignedIncidentID *int   `json:"assignedIncidentId,omitempty"`
}

type CreateVolunteerRequest struct {
	Name         string `json:"name"`
	Role         string `json:"role"`
	Phone        string `json:"phone"`
	LicenseValid *bool  `json:"licenseValid,omitempty"`
}

type CreateVolunteerResponse struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type AssignVolunteerRequest struct {
	IncidentID int `json:"incidentId"`
}

type AssignVolunteerResponse struct {
	VolunteerID int    `json:"volunteerId"`
	IncidentID  int    `json:"incidentId"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

type IncidentSummary struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

type LogisticsAssignment struct {
	VolunteerID       int  `json:"volunteerId"`
	AssignedToVehicle bool `json:"assignedToVehicle"`
}
