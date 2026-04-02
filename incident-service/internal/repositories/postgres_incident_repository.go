package repositories

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"incident-service/internal/models"
)

type PostgresIncidentRepository struct {
	db *sql.DB
}

func (r *PostgresIncidentRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func NewPostgresIncidentRepository(dsn string) (*PostgresIncidentRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	repo := &PostgresIncidentRepository{db: db}
	if err := repo.ensureSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := repo.ensureSequenceStart(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

func BuildPostgresDSN(host, port, user, password, dbName, sslMode string) string {
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

func (r *PostgresIncidentRepository) ensureSchema() error {
	const q = `
	CREATE TABLE IF NOT EXISTS incidents (
		id SERIAL PRIMARY KEY,
		type TEXT NOT NULL,
		location TEXT NOT NULL,
		severity TEXT NOT NULL,
		status TEXT NOT NULL
	);
	`
	_, err := r.db.Exec(q)
	return err
}

func (r *PostgresIncidentRepository) ensureSequenceStart() error {
	// Set sequence to be higher than any existing ID
	const q = `SELECT setval('incidents_id_seq', COALESCE(MAX(id), 100) + 1) FROM incidents;`
	_, err := r.db.Exec(q)
	return err
}

func (r *PostgresIncidentRepository) Create(incident models.Incident) (models.Incident, error) {
	const q = `
	INSERT INTO incidents (type, location, severity, status)
	VALUES ($1, $2, $3, $4)
	RETURNING id;
	`

	err := r.db.QueryRow(q, incident.Type, incident.Location, incident.Severity, incident.Status).Scan(&incident.ID)
	if err != nil {
		return models.Incident{}, err
	}
	return incident, nil
}

func (r *PostgresIncidentRepository) List() []models.Incident {
	const q = `SELECT id, type, location, severity, status FROM incidents ORDER BY id;`
	rows, err := r.db.Query(q)
	if err != nil {
		return []models.Incident{}
	}
	defer rows.Close()

	var incidents []models.Incident
	for rows.Next() {
		var inc models.Incident
		if err := rows.Scan(&inc.ID, &inc.Type, &inc.Location, &inc.Severity, &inc.Status); err == nil {
			incidents = append(incidents, inc)
		}
	}
	return incidents
}

func (r *PostgresIncidentRepository) GetByID(id int) (models.Incident, error) {
	const q = `SELECT id, type, location, severity, status FROM incidents WHERE id = $1;`
	var inc models.Incident
	err := r.db.QueryRow(q, id).Scan(&inc.ID, &inc.Type, &inc.Location, &inc.Severity, &inc.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Incident{}, ErrIncidentNotFound
		}
		return models.Incident{}, err
	}
	return inc, nil
}

func (r *PostgresIncidentRepository) UpdateStatus(id int, status string) (models.Incident, error) {
	const q = `UPDATE incidents SET status = $1 WHERE id = $2;`
	res, err := r.db.Exec(q, status, id)
	if err != nil {
		return models.Incident{}, err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return models.Incident{}, ErrIncidentNotFound
	}
	return r.GetByID(id)
}

func (r *PostgresIncidentRepository) Update(id int, req models.UpdateIncidentRequest) (models.Incident, error) {
	const q = `UPDATE incidents SET type = $1, location = $2, severity = $3 WHERE id = $4;`
	res, err := r.db.Exec(q, req.Type, req.Location, req.Severity, id)
	if err != nil {
		return models.Incident{}, err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return models.Incident{}, ErrIncidentNotFound
	}
	return r.GetByID(id)
}

func (r *PostgresIncidentRepository) Delete(id int) error {
	const q = `DELETE FROM incidents WHERE id = $1;`
	res, err := r.db.Exec(q, id)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrIncidentNotFound
	}
	return nil
}
