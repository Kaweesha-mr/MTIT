package services

import (
	"errors"
	"regexp"
	"strings"

	"volunteer-service/internal/models"
	"volunteer-service/internal/repositories"
)

var (
	ErrValidation            = errors.New("validation failed")
	ErrIncidentUnavailable   = errors.New("incident unavailable")
	ErrIncidentResolved      = errors.New("incident resolved")
	ErrLogisticsUnavailable  = errors.New("logistics information unavailable")
	ErrInvalidVolunteerState = errors.New("volunteer is not available")
	ErrVolunteerBusy         = errors.New("volunteer already has active trip")
)

var phoneRegex = regexp.MustCompile(`^0[0-9]{9}$`)

var allowedRoles = map[string]struct{}{
	"DOCTOR": {},
	"DRIVER": {},
	"RESCUE": {},
	"LEADER": {},
}

type VolunteerService struct {
	repository repositories.VolunteerRepository
	incidents  IncidentVerifier
	logistics  LogisticsChecker
}

func NewVolunteerService(repository repositories.VolunteerRepository, incidents IncidentVerifier, logistics LogisticsChecker) *VolunteerService {
	return &VolunteerService{
		repository: repository,
		incidents:  incidents,
		logistics:  logistics,
	}
}

func (s *VolunteerService) Create(req models.CreateVolunteerRequest) (models.CreateVolunteerResponse, error) {
	name := strings.TrimSpace(req.Name)
	role := strings.ToUpper(strings.TrimSpace(req.Role))
	phone := strings.TrimSpace(req.Phone)

	if name == "" || role == "" || phone == "" {
		return models.CreateVolunteerResponse{}, ErrValidation
	}

	if _, ok := allowedRoles[role]; !ok {
		return models.CreateVolunteerResponse{}, ErrValidation
	}

	if !phoneRegex.MatchString(phone) {
		return models.CreateVolunteerResponse{}, ErrValidation
	}

	licenseValid := true
	if req.LicenseValid != nil {
		licenseValid = *req.LicenseValid
	}

	volunteer, err := s.repository.Create(models.Volunteer{
		Name:         name,
		Role:         role,
		Phone:        phone,
		Status:       "AVAILABLE",
		LicenseValid: licenseValid,
	})
	if err != nil {
		return models.CreateVolunteerResponse{}, err
	}

	return models.CreateVolunteerResponse{
		ID:     volunteer.ID,
		Name:   volunteer.Name,
		Status: volunteer.Status,
	}, nil
}

func (s *VolunteerService) GetByID(id int) (models.Volunteer, error) {
	return s.repository.GetByID(id)
}

func (s *VolunteerService) List() ([]models.Volunteer, error) {
	return s.repository.List()
}

func (s *VolunteerService) UpdateVolunteer(id int, req models.UpdateVolunteerRequest) (models.Volunteer, error) {
	name := strings.TrimSpace(req.Name)
	role := strings.ToUpper(strings.TrimSpace(req.Role))
	phone := strings.TrimSpace(req.Phone)

	if name == "" || role == "" || phone == "" {
		return models.Volunteer{}, ErrValidation
	}

	if _, ok := allowedRoles[role]; !ok {
		return models.Volunteer{}, ErrValidation
	}

	if !phoneRegex.MatchString(phone) {
		return models.Volunteer{}, ErrValidation
	}

	volunteer, err := s.repository.GetByID(id)
	if err != nil {
		return models.Volunteer{}, err
	}

	volunteer.Name = name
	volunteer.Role = role
	volunteer.Phone = phone

	return s.repository.Update(volunteer)
}

func (s *VolunteerService) DeleteVolunteer(id int) error {
	return s.repository.Delete(id)
}

func (s *VolunteerService) Assign(volunteerID int, incidentID int) (models.AssignVolunteerResponse, error) {
	if incidentID <= 0 {
		return models.AssignVolunteerResponse{}, ErrValidation
	}

	volunteer, err := s.repository.GetByID(volunteerID)
	if err != nil {
		return models.AssignVolunteerResponse{}, err
	}

	if volunteer.Status != "AVAILABLE" {
		return models.AssignVolunteerResponse{}, ErrInvalidVolunteerState
	}

	incident, err := s.incidents.GetIncident(incidentID)
	if err != nil {
		if errors.Is(err, ErrIncidentUnavailable) {
			return models.AssignVolunteerResponse{}, ErrIncidentUnavailable
		}
		return models.AssignVolunteerResponse{}, err
	}

	if strings.EqualFold(strings.TrimSpace(incident.Status), "RESOLVED") {
		return models.AssignVolunteerResponse{}, ErrIncidentResolved
	}

	assignment, err := s.logistics.GetAssignment(volunteerID)
	if err != nil {
		if errors.Is(err, ErrLogisticsUnavailable) {
			return models.AssignVolunteerResponse{}, ErrLogisticsUnavailable
		}
		return models.AssignVolunteerResponse{}, err
	}

	if assignment.HasActiveTrip {
		return models.AssignVolunteerResponse{}, ErrVolunteerBusy
	}

	volunteer.Status = "BUSY"
	volunteer.AssignedIncidentID = &incidentID

	if _, err := s.repository.Update(volunteer); err != nil {
		return models.AssignVolunteerResponse{}, err
	}

	return models.AssignVolunteerResponse{
		ID:         volunteer.ID,
		AssignedTo: incidentID,
		Status:     "BUSY",
	}, nil
}
