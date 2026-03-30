package repositories

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"volunteer-service/internal/models"
)

type PostgresVolunteerRepository struct {
	db *sql.DB
}

func NewPostgresVolunteerRepository(dsn string) (*PostgresVolunteerRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	repo := &PostgresVolunteerRepository{db: db}
	if err := repo.ensureSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

func BuildPostgresDSN(host, port, user, password, dbName string, sslMode string) string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host,
		port,
		user,
		password,
		dbName,
		sslMode,
	)
}

func (r *PostgresVolunteerRepository) ensureSchema() error {
	const q = `
	CREATE TABLE IF NOT EXISTS volunteers (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		role TEXT NOT NULL,
		phone TEXT NOT NULL,
		status TEXT NOT NULL,
		license_valid BOOLEAN NOT NULL DEFAULT TRUE,
		assigned_incident_id INTEGER NULL
	);
	`

	_, err := r.db.Exec(q)
	return err
}

func (r *PostgresVolunteerRepository) Create(volunteer models.Volunteer) (models.Volunteer, error) {
	const q = `
	INSERT INTO volunteers (name, role, phone, status, license_valid, assigned_incident_id)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id;
	`

	var assignedID any
	if volunteer.AssignedIncidentID != nil {
		assignedID = *volunteer.AssignedIncidentID
	}

	err := r.db.QueryRow(
		q,
		volunteer.Name,
		volunteer.Role,
		volunteer.Phone,
		volunteer.Status,
		volunteer.LicenseValid,
		assignedID,
	).Scan(&volunteer.ID)
	if err != nil {
		return models.Volunteer{}, err
	}

	return volunteer, nil
}

func (r *PostgresVolunteerRepository) GetByID(id int) (models.Volunteer, error) {
	const q = `
	SELECT id, name, role, phone, status, license_valid, assigned_incident_id
	FROM volunteers
	WHERE id = $1;
	`

	var volunteer models.Volunteer
	var assignedIncidentID sql.NullInt64

	err := r.db.QueryRow(q, id).Scan(
		&volunteer.ID,
		&volunteer.Name,
		&volunteer.Role,
		&volunteer.Phone,
		&volunteer.Status,
		&volunteer.LicenseValid,
		&assignedIncidentID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Volunteer{}, ErrVolunteerNotFound
		}
		return models.Volunteer{}, err
	}

	if assignedIncidentID.Valid {
		v := int(assignedIncidentID.Int64)
		volunteer.AssignedIncidentID = &v
	}

	return volunteer, nil
}

func (r *PostgresVolunteerRepository) Update(volunteer models.Volunteer) (models.Volunteer, error) {
	const q = `
	UPDATE volunteers
	SET name = $1, role = $2, phone = $3, status = $4, license_valid = $5, assigned_incident_id = $6
	WHERE id = $7;
	`

	var assignedID any
	if volunteer.AssignedIncidentID != nil {
		assignedID = *volunteer.AssignedIncidentID
	}

	res, err := r.db.Exec(
		q,
		volunteer.Name,
		volunteer.Role,
		volunteer.Phone,
		volunteer.Status,
		volunteer.LicenseValid,
		assignedID,
		volunteer.ID,
	)
	if err != nil {
		return models.Volunteer{}, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return models.Volunteer{}, err
	}
	if affected == 0 {
		return models.Volunteer{}, ErrVolunteerNotFound
	}

	return volunteer, nil
}
