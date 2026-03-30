package repositories

import (
	"database/sql"
	"errors"

	"alert-service/internal/models"
)

var ErrAlertNotFound = errors.New("alert not found")

type AlertRepository interface {
	Create(alert models.Alert) (models.Alert, error)
	GetByID(id int) (models.Alert, error)
	Update(id int, req models.UpdateAlertRequest) (models.Alert, error)
	Delete(id int) error
}

type PostgresAlertRepository struct {
	db *sql.DB
}

func NewPostgresAlertRepository(db *sql.DB) *PostgresAlertRepository {
	return &PostgresAlertRepository{db: db}
}

func (r *PostgresAlertRepository) Create(alert models.Alert) (models.Alert, error) {
	query := `
		INSERT INTO alerts (incident_id, message, severity, status, timestamp)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := r.db.QueryRow(
		query,
		alert.IncidentID,
		alert.Message,
		alert.Severity,
		alert.Status,
		alert.Timestamp,
	).Scan(&alert.ID)

	if err != nil {
		return models.Alert{}, err
	}

	return alert, nil
}

func (r *PostgresAlertRepository) GetByID(id int) (models.Alert, error) {
	query := `
		SELECT id, incident_id, message, severity, status, timestamp
		FROM alerts
		WHERE id = $1
	`

	var alert models.Alert
	err := r.db.QueryRow(query, id).Scan(
		&alert.ID,
		&alert.IncidentID,
		&alert.Message,
		&alert.Severity,
		&alert.Status,
		&alert.Timestamp,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.Alert{}, ErrAlertNotFound
		}
		return models.Alert{}, err
	}

	return alert, nil
}

func (r *PostgresAlertRepository) Update(id int, req models.UpdateAlertRequest) (models.Alert, error) {
	query := `
		UPDATE alerts
		SET message = $1, severity = $2
		WHERE id = $3
		RETURNING id, incident_id, message, severity, status, timestamp
	`

	var alert models.Alert
	err := r.db.QueryRow(query, req.Message, req.Severity, id).Scan(
		&alert.ID,
		&alert.IncidentID,
		&alert.Message,
		&alert.Severity,
		&alert.Status,
		&alert.Timestamp,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.Alert{}, ErrAlertNotFound
		}
		return models.Alert{}, err
	}

	return alert, nil
}

func (r *PostgresAlertRepository) Delete(id int) error {
	result, err := r.db.Exec(`DELETE FROM alerts WHERE id = $1`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrAlertNotFound
	}

	return nil
}
