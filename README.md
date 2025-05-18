# Wallet Service

A microservice for managing platform tokens (wallet system) built with Go and Fiber.

## Features

- Get wallet information
- Exchange game tokens for platform tokens
- Spend platform tokens
- Track wallet transaction history
- Authentication and authorization
- Swagger API documentation
- Structured logging and observability

## Tech Stack

- **Go**: Programming language
- **Fiber**: Web framework
- **PostgreSQL**: Database
- **Uber FX**: Dependency injection
- **go-validator**: Request validation
- **Swagger**: API documentation
- **golang-migrate**: Database migrations
- **zap**: Structured logging
- **Prometheus**: Metrics collection
- **OpenTelemetry**: Distributed tracing

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Docker (optional)
- Prometheus (optional, for metrics)
- Jaeger/OpenTelemetry Collector (optional, for tracing)

### Installation

1. Clone the repository:
   ```bash
   git clone github.com/playconomy/wallet-service
   cd wallet-service
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Create database:
   ```bash
   make db-create
   ```

4. Run migrations:
   ```bash
   make migrate-up
   ```

5. Generate Swagger documentation:
   ```bash
   make swagger-init
   ```

### Running

```bash
make run
```

Or with Docker:

```bash
make docker-build
make docker-run
```

## API Documentation

Swagger UI is available at: [http://localhost:3000/swagger/](http://localhost:3000/swagger/)

### Authentication

All API endpoints (except health check) require the following headers:

- `X-User-Id`: User ID (numeric)
- `X-User-Email`: User email
- `X-User-Role`: User role (user/admin)

### Available Endpoints

- `GET /:user_id` - Get wallet information
- `GET /:user_id/logs` - Get wallet transaction history
- `POST /exchange` - Exchange game tokens for platform tokens
- `POST /spend` - Spend tokens from wallet
- `GET /health` - Health check (unprotected)

## Development

### Testing

The service includes unit and integration tests:

- **Unit Tests**: Test individual components in isolation using mocks.
- **Integration Tests**: Test end-to-end functionality with a real database.

#### Running Tests

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report
make test-coverage

# Skip integration tests for faster testing
go test -short ./internal/...
```

#### Test Structure

- `*_test.go` - Unit tests alongside the code they test
- `/internal/test/integration/` - Integration tests with Docker-based PostgreSQL instance

### Makefile Commands

- `make build` - Build application
- `make run` - Run application
- `make clean` - Clean build files
- `make test` - Run all tests
- `make test-verbose` - Run tests with verbose output
- `make test-coverage` - Run tests with coverage report
- `make swagger-init` - Generate Swagger documentation
- `make migrate-up` - Apply all migrations
- `make migrate-down` - Rollback all migrations
- `make db-reset` - Reset database and run migrations
- `make help` - Show all available commands

## Project Structure

```
├── cmd/app/               # Application entry point
├── database/              # Database connection and migrations
├── docs/                  # Swagger documentation
├── internal/              # Private application code
│   ├── config/            # Configuration
│   ├── module/            # Dependency injection modules
│   ├── server/            # Server components
│   │   ├── dto/           # Data Transfer Objects
│   │   ├── handler/       # HTTP handlers
│   │   ├── middleware/    # HTTP middleware
│   │   └── router/        # Routing
│   ├── service/           # Business logic
│   ├── test/              # Test utilities and integration tests
│   │   └── integration/   # Integration test setup
│   └── utils/             # Utility functions
├── profiles/              # Environment profiles
├── Dockerfile             # Docker configuration
├── go.mod                 # Go module definition
├── Makefile               # Build commands
└── README.md              # This file
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.