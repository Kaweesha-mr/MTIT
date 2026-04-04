# MTIT - Multi-Threat Incident Tracking

[![CI - Tests & Linting](https://github.com/kaweeshamarasinghe/MTIT/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/kaweeshamarasinghe/MTIT/actions/workflows/ci.yml)
[![Docker Build & Push](https://github.com/kaweeshamarasinghe/MTIT/actions/workflows/docker-build.yml/badge.svg?branch=main)](https://github.com/kaweeshamarasinghe/MTIT/actions/workflows/docker-build.yml)
[![Deploy API Docs](https://github.com/kaweeshamarasinghe/MTIT/actions/workflows/docs.yml/badge.svg?branch=main)](https://github.com/kaweeshamarasinghe/MTIT/actions/workflows/docs.yml)

[📚 View Complete API Documentation](https://kaweeshamarasinghe.github.io/MTIT)

## Overview

MTIT is a microservices-based platform for managing multi-threat incidents with comprehensive resource allocation, volunteer management, and real-time alerting capabilities.

## Architecture

The application consists of 8 independent microservices:

### Core Services

- **Alert Service** (Go) - Manages alerts and notifications
- **Incident Service** (Go) - Incident tracking and management
- **Resource Service** (Go) - Resource management and allocation
- **Volunteer Service** (Go) - Volunteer coordination and scheduling

### Support Services

- **Auth Service** (Go) - Authentication and authorization
- **Gateway** (Go) - API Gateway and request routing
- **Fleet Service** (Python) - Fleet and vehicle management
- **Shelter Service** (Node.js) - Shelter and accommodation services

## Getting Started

### Prerequisites

- Docker & Docker Compose
- Go 1.21+
- Node.js 18+
- Python 3.9+

### Running the Application

```bash
# Start all services using Docker Compose
docker compose up

# Services will be available at:
# - Gateway: http://localhost:8000
# - Alert Service: http://localhost:8001
# - Incident Service: http://localhost:8002
# - Resource Service: http://localhost:8003
# - Volunteer Service: http://localhost:8004
# - Auth Service: http://localhost:8005
# - Fleet Service: http://localhost:8006
# - Shelter Service: http://localhost:8007
```

## API Documentation

Complete API documentation is automatically generated and published at:
**[https://kaweeshamarasinghe.github.io/MTIT](https://kaweeshamarasinghe.github.io/MTIT)**

Each service has its OpenAPI specification documented at `<service>/api/openapi.yaml`

## Development

### Running Tests

Each service can be tested independently:

```bash
# Go services
cd alert-service && go test ./...

# Python services
cd fleet-service && pytest tests/

# Node.js services
cd shelter-service && npm test
```

### Code Quality

The project uses:
- **golangci-lint** for Go services
- **flake8** for Python services
- **eslint** for Node.js services

Run linters through CI/CD or locally using the respective service's development setup.

## CI/CD Pipeline

This repository uses GitHub Actions for:

1. **Continuous Integration** - Runs tests and linting on every push/PR
2. **Docker Build & Push** - Builds and pushes Docker images to GitHub Container Registry
3. **Documentation** - Auto-generates and publishes API docs to GitHub Pages

See `.github/workflows/` for workflow definitions.

## Project Structure

```
.
├── .github/workflows/          # GitHub Actions workflows
├── alert-service/              # Go service
├── auth-service/               # Go service
├── gateway/                    # Go service
├── fleet-service/              # Python service
├── incident-service/           # Go service
├── resource-service/           # Go service
├── shelter-service/            # Node.js service
├── volunteer-service/          # Go service
├── docker-compose.yml          # Local development setup
└── .golangci.yml              # Go linting configuration
```

## Contributing

1. Create a feature branch
2. Make your changes
3. Ensure tests pass locally
4. Push to GitHub - CI/CD pipeline will verify
5. Create a pull request

## License

[Add your license information here]

## Contact

For questions or support, please contact the development team.
