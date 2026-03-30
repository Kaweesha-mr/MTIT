# Fleet (Logistics) Service — Python/Flask

Runs on port `8086` and manages vehicles and trips.

## Endpoints
- `POST /vehicles` — body `{ "type", "plate", "capacity" }` → `{ "id", "type", "status": "AVAILABLE" }`
- `POST /trips` — body `{ "vehicleId", "driverId", "cargoId", "destinationId" }` → `{ "tripId", "status": "SCHEDULED" }` (verifies Volunteer, Resource, Shelter services)
- `GET /trips` — list trips
- `GET /trips/volunteer/{id}` — `{ "volunteerId", "hasActiveTrip" }`
- `GET /health`

## API Documentation (Swagger)
- Swagger UI: `http://localhost:8086/docs`
- Swagger UI (alias): `http://localhost:8086/swagger`
- OpenAPI YAML: `http://localhost:8086/openapi.yaml`
- OpenAPI alias: `http://localhost:8086/swagger.json`

## Environment
- `PORT` (default `8086`)
- `VOLUNTEER_SERVICE_URL` (default `http://localhost:8082`)
- `RESOURCE_SERVICE_URL` (default `http://localhost:8083`)
- `SHELTER_SERVICE_URL` (default `http://localhost:8084`)
- `DB_HOST` (default `localhost`)
- `DB_PORT` (default `3307` to match compose)
- `DB_USER` (default `fleet_user`)
- `DB_PASSWORD` (default `fleet_pass`)
- `DB_NAME` (default `fleet_db`)

## Run with Docker Compose (MySQL)
From `fleet-service/`:
```bash
docker compose up -d
```

Install deps and start:
```bash
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
DB_HOST=localhost DB_PORT=3307 python app.py
```

If DB connection fails, the service exits (no in-memory fallback).
