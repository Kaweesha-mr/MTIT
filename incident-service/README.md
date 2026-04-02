# Incident Service (Go)

Runs on port `8081` and manages incidents, triggering Alert and Shelter services after creation.

## Endpoints
- `POST /incidents`
- `GET /incidents`
- `GET /incidents/{id}`
- `PUT /incidents/{id}`
- `DELETE /incidents/{id}`
- `PUT /incidents/{id}/status`
- `GET /health`

## API Documentation (Swagger)
- Swagger UI: `http://localhost:8081/docs`
- Swagger UI (alias): `http://localhost:8081/swagger`
- OpenAPI YAML: `http://localhost:8081/openapi.yaml`
- OpenAPI alias: `http://localhost:8081/swagger.json`

## Environment
- `PORT` (default `8081`)
- `ALERT_SERVICE_URL` (default `http://localhost:8085/alerts`)
- `SHELTER_SERVICE_URL` (default `http://localhost:8084/shelters`)
- `DB_HOST` (default `localhost`)
- `DB_PORT` (default `5436` to match compose)
- `DB_USER` (default `incident_user`)
- `DB_PASSWORD` (default `incident_pass`)
- `DB_NAME` (default `incidents_db`)
- `DB_SSLMODE` (default `disable`)

## Run with Docker Compose (database)
From `incident-service/`:
```bash
docker compose up -d
```

Then start the service:
```bash
go run ./cmd/server
```

If DB connection fails while `USE_DB` is enabled, the service exits.
