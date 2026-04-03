package repositories

import (
	"database/sql"
	"errors"

	"alert-service/internal/models"
)

var ErrAlertNotFound = errors.New("alert not found")

type AlertRepository interface {
	Create(alert models.Alert) (models.Alert, error)
	GetAll() ([]models.Alert, error)
	GetByID(id int) (models.Alert, error)
	Update(id int, req models.UpdateAlertRequest) (models.Alert, error)
	Delete(id int) error
}

type PostgresAlertRepository struct {
	db *sql.DB
}

func NewPostgresAlertRepository(db *sql.DB) *PostgresAlertRepository {
	repo := &PostgresAlertRepository{db: db}
	_ = repo.ensureSchema()
	return repo
}

func (r *PostgresAlertRepository) ensureSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS alerts (
		id SERIAL PRIMARY KEY,
		incident_id INT NOT NULL,
		message TEXT NOT NULL,
		severity VARCHAR(20) NOT NULL,
		status VARCHAR(30) NOT NULL,
		timestamp TIMESTAMP NOT NULL
	);`
	_, err := r.db.Exec(query)
	return err
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

func (r *PostgresAlertRepository) GetAll() ([]models.Alert, error) {
	query := `
		SELECT id, incident_id, message, severity, status, timestamp
		FROM alerts
		ORDER BY id ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]models.Alert, 0)
	for rows.Next() {
		var alert models.Alert
		if err := rows.Scan(
			&alert.ID,
			&alert.IncidentID,
			&alert.Message,
			&alert.Severity,
			&alert.Status,
			&alert.Timestamp,
		); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return alerts, nil
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
