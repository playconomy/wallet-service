# Database connection string
DB_URL=postgres://postgres:postgres@localhost:5432/wallet_db?sslmode=disable

.PHONY: build run clean migrate-up migrate-down migrate-create swagger-init swagger-fmt test test-verbose test-coverage

build:
	go build -o bin/wallet-service cmd/app/main.go

run:
	go run cmd/app/main.go

clean:
	rm -rf bin/
	go clean

# Test commands
test:
	./scripts/run_tests.sh

test-verbose:
	go test -v ./internal/... -count=1

test-coverage:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out

# Install golang-migrate
install-migrate:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create a new migration file
migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir database/migrations -seq $$name

# Apply all migrations
migrate-up:
	migrate -path database/migrations -database "$(DB_URL)" up

# Rollback all migrations
migrate-down:
	migrate -path database/migrations -database "$(DB_URL)" down

# Rollback one step
migrate-rollback:
	migrate -path database/migrations -database "$(DB_URL)" down 1

# Show current migration version
migrate-version:
	migrate -path database/migrations -database "$(DB_URL)" version

# Force set migration version
migrate-force:
	@read -p "Enter version: " version; \
	migrate -path database/migrations -database "$(DB_URL)" force $$version

# Check migrations status
migrate-status:
	migrate -path database/migrations -database "$(DB_URL)" status

# Docker commands
docker-build:
	docker build -t wallet-service .

docker-run:
	docker run -p 3000:3000 \
		-e DB_HOST=host.docker.internal \
		-e DB_PORT=5432 \
		-e DB_USER=postgres \
		-e DB_PASSWORD=postgres \
		-e DB_NAME=wallet_db \
		wallet-service

# Database commands
db-create:
	PGPASSWORD=postgres psql -h localhost -U postgres -c "CREATE DATABASE wallet_db"

db-drop:
	PGPASSWORD=postgres psql -h localhost -U postgres -c "DROP DATABASE IF EXISTS wallet_db"

# Reset database and run migrations
db-reset: db-drop db-create migrate-up

# Swagger commands
swagger-init:
	swag init -g cmd/app/main.go -o docs

swagger-fmt:
	swag fmt

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make build           - Build the application"
	@echo "  make run             - Run the application"
	@echo "  make clean           - Clean build files"
	@echo "  make test            - Run tests"
	@echo "  make test-verbose    - Run tests with verbose output"
	@echo "  make test-coverage   - Run tests with coverage report"
	@echo "  make install-migrate - Install golang-migrate tool"
	@echo "  make migrate-create  - Create a new migration file"
	@echo "  make migrate-up      - Apply all migrations"
	@echo "  make migrate-down    - Rollback all migrations"
	@echo "  make migrate-rollback- Rollback one migration"
	@echo "  make migrate-version - Show current migration version"
	@echo "  make migrate-force   - Force set migration version"
	@echo "  make migrate-status  - Show migrations status"
	@echo "  make docker-build    - Build Docker image"
	@echo "  make docker-run      - Run Docker container"
	@echo "  make db-create       - Create database"
	@echo "  make db-drop         - Drop database"
	@echo "  make db-reset        - Reset database and run migrations"
	@echo "  make swagger-init    - Initialize Swagger documentation"
	@echo "  make swagger-fmt     - Format Swagger comments"