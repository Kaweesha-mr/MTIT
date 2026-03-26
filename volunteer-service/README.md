# Volunteer Service (Go)

Volunteer Service runs on port `8082` and manages helpers (doctors, drivers, rescue workers).

## Run

```bash
go run ./cmd/server
```

Default port is `8082` and can be overridden with `PORT`.

## Environment Variables

- `PORT` (default: `8082`)
- `INCIDENT_SERVICE_URL` (default: `http://localhost:8081`)
- `LOGISTICS_SERVICE_URL` (default: `http://localhost:8086`)
- `DB_HOST` (default: `localhost`)
- `DB_PORT` (default: `5434` for the provided Docker Compose database)
- `DB_USER` (default: `volunteer_user`)
- `DB_PASSWORD` (default: `volunteer_pass`)
- `DB_NAME` (default: `volunteers_db`)
- `DB_SSLMODE` (default: `disable`)

## Docker Compose (Database)

Start the Volunteer database:

```bash
docker compose up -d
```

Run the service (PostgreSQL is required; it will fail fast if the DB is unreachable):

```bash
DB_HOST=localhost DB_PORT=5434 go run ./cmd/server
```

## Endpoints

### 1) Register Volunteer

- `POST /volunteers`

Request:

```json
{
	"name": "John Doe",
	"role": "RESCUE",
	"phone": "0771234567"
}
```

Response:

```json
{
	"id": 501,
	"name": "John Doe",
	"status": "AVAILABLE"
}
```

### 2) Assign Volunteer to Incident

- `PUT /volunteers/{id}/assign`

Request:

```json
{
	"incidentId": 101
}
```

Response:

```json
{
	"volunteerId": 501,
	"incidentId": 101,
	"status": "ASSIGNED",
	"message": "Volunteer assigned successfully"
}
```

Assignment rules:

- Calls Incident Service: `GET http://localhost:8081/incidents/{incidentId}`.
- Rejects assignment if incident is not found (`404`) or incident status is `RESOLVED`.
- Calls Logistics Service: `GET <LOGISTICS_SERVICE_URL>/volunteers/{id}`.
- Rejects assignment if volunteer is not assigned to a vehicle in Logistics.

### 3) Get Volunteer Details (Inbound for other services)

- `GET /volunteers/{id}`

Response:

```json
{
	"id": 501,
	"name": "John Doe",
	"role": "RESCUE",
	"licenseValid": true
}
```

## Test

```bash
go test ./...
```
