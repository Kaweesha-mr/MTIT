# Alert Service

Alert service implemented in Go with PostgreSQL.

## Endpoints

- `POST /alerts`
- `GET /alerts/{id}`
- `PUT /alerts/{id}`
- `DELETE /alerts/{id}`
- `GET /health`

## Run locally

1. Copy `.env.example` to `.env` and adjust values if needed.
2. Start database stack:

```bash
docker compose up -d
```

3. Run the API:

```bash
go run ./cmd/server
```

Default port is `8085`.
