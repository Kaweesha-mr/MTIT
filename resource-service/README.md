# Resource Service (Go)

Resource Service manages disaster relief supplies and dispatches resources to shelters.

This service:
- Runs independently on port `8083` by default
- Uses MongoDB Atlas (`resource_db`, `resources` collection)
- Exposes stable JSON contracts with camelCase fields
- Validates shelter state via Shelter Service before dispatching resources
- Is compatible for direct calls now and API Gateway routing later

## Tech Stack

- Go 1.22
- `net/http` (ServeMux)
- MongoDB official Go driver
- `godotenv` for local `.env` loading

## Default Port

- `8083`

## Environment Variables

Copy `.env.example` to `.env` and set values.

```env
PORT=8083
MONGODB_URI=mongodb+srv://<username>:<password>@<cluster-url>/?retryWrites=true&w=majority
DB_NAME=resource_db
COLLECTION_NAME=resources
SHELTER_SERVICE_URL=http://localhost:8084
MONGO_TIMEOUT_SECONDS=10
SHELTER_TIMEOUT_SECONDS=5
```

## Project Structure

```text
resource-service/
	cmd/server/main.go
	internal/
		clients/
		config/
		handlers/
		models/
		repositories/
		routes/
		services/
	api/openapi.yaml
	configs/config.yaml
	tests/resource_handler_test.go
	.env.example
	.gitignore
	go.mod
```

## Run the Service

```bash
cd resource-service
go mod tidy
go run ./cmd/server
```

## API Endpoints

### 1) Health

- `GET /health`

Response:

```json
{
	"status": "ok"
}
```

### 2) Create Resource

- `POST /resources`

Request:

```json
{
	"item": "WATER_BOTTLES",
	"quantity": 500,
	"unit": "PACKS"
}
```

Response (`201`):

```json
{
	"id": 701,
	"item": "WATER_BOTTLES",
	"available": 500
}
```

### 3) Get Resource by ID

- `GET /resources/{id}`

Response (`200`):

```json
{
	"id": 701,
	"item": "WATER_BOTTLES",
	"weight": "500kg"
}
```

### 4) Dispatch Resource

- `PUT /resources/{id}/dispatch`

Request:

```json
{
	"shelterId": 301,
	"quantity": 100
}
```

Success response (`200`):

```json
{
	"resourceId": 701,
	"status": "DISPATCHED",
	"message": "Resources dispatched successfully to shelter 301"
}
```

Conflict response (`409`) example:

```json
{
	"message": "dispatch rejected. shelter is closed or full"
}
```

## Dispatch Business Rules

When dispatching, the service validates:
- Resource exists
- Requested quantity is positive
- Available stock is sufficient
- Shelter Service returns `200`
- Shelter status is not `CLOSED`
- Shelter is not full (`currentOccupancy < maxCapacity`)

If shelter service is unavailable, the API returns JSON error with `502`.

## Sample cURL Commands

```bash
curl -X GET http://localhost:8083/health
```

```bash
curl -X POST http://localhost:8083/resources \
	-H "Content-Type: application/json" \
	-d '{"item":"WATER_BOTTLES","quantity":500,"unit":"PACKS"}'
```

```bash
curl -X GET http://localhost:8083/resources/701
```

```bash
curl -X PUT http://localhost:8083/resources/701/dispatch \
	-H "Content-Type: application/json" \
	-d '{"shelterId":301,"quantity":100}'
```

## API Gateway Note

These endpoints are designed to work directly on this service now.
Later, the same endpoints can be exposed through API Gateway routing without changing the service contracts.
