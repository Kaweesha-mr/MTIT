# Shelter Service (Node.js)

Runs on port `8084` and manages shelters for incidents.

## Endpoints
- `POST /shelters` — create shelter after verifying incident (Incident Service `GET /incidents/{id}` must return ACTIVE)
- `GET /shelters` — list shelters
- `GET /shelters/{id}` — shelter details
- `PUT /shelters/{id}/capacity` — update `currentOccupancy`
- `GET /health`

## Environment
- `PORT` (default `8084`)
- `INCIDENT_SERVICE_URL` (default `http://localhost:8081`)
- `DB_HOST` (default `localhost`)
- `DB_PORT` (default `5435` to match compose)
- `DB_USER` (default `shelter_user`)
- `DB_PASSWORD` (default `shelter_pass`)
- `DB_NAME` (default `shelters_db`)
- `DB_SSLMODE` (default `disable`)

## Run with Docker Compose (database)
From `shelter-service/`:
```bash
docker compose up -d
```

Then start the service:
```bash
npm install
DB_HOST=localhost DB_PORT=5435 npm start
```

If DB connection fails, the service exits (no in-memory fallback).
