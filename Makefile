.PHONY: help tests test-unit test-integration generate-mocks run-postgres stop-postgres cli-postgres migrate-up migrate-down benchmark

help:
	@echo "╔════════════════════════════════════════════════════════════╗"
	@echo "║            Talk The Go Way - Make Targets                  ║"
	@echo "╚════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Testing:"
	@echo "  make tests              - Run all tests (unit + integration)"
	@echo "  make test-unit          - Run only unit tests (fast)"
	@echo "  make test-integration   - Run only integration tests (slow, requires Docker)"
	@echo "  make benchmark          - Run benchmarks (requires Docker)"
	@echo ""
	@echo "Development:"
	@echo "  make run-postgres       - Start PostgreSQL container (Docker)"
	@echo "  make stop-postgres      - Stop PostgreSQL container"
	@echo "  make cli-postgres       - Run PostgreSQL CLI example"
	@echo ""
	@echo "Migrations:"
	@echo "  make migrate-up         - Apply database migrations"
	@echo "  make migrate-down       - Rollback database migrations"
	@echo ""
	@echo "Other:"
	@echo "  make generate-mocks     - Generate mocks from interfaces"

# Testing targets
tests: generate-mocks
	@echo "Running all tests..."
	@go test ./... && go test -tags=integration ./...

test-unit: generate-mocks
	@echo "Running unit tests only..."
	@go test ./...

test-integration: generate-mocks
	@command -v docker >/dev/null 2>&1 || { echo "❌ Docker is not installed"; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "❌ Docker is not running"; exit 1; }
	@echo "Running integration tests only (requires Docker)..."
	@go test -tags=integration ./...

benchmark: generate-mocks
	@command -v docker >/dev/null 2>&1 || { echo "❌ Docker is not installed"; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "❌ Docker is not running"; exit 1; }
	@echo "Running benchmarks (requires Docker)..."
	@go test -tags=integration -bench=. -benchmem ./book/postgres/

generate-mocks:
	@go tool mockery --output book/mocks --dir book --all

# PostgreSQL Docker Compose targets
run-postgres:
	@echo "Starting PostgreSQL container..."
	@docker-compose up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 3
	@docker-compose exec postgres pg_isready -U postgres || true
	@echo "✅ PostgreSQL is ready!"

stop-postgres:
	@echo "Stopping PostgreSQL container..."
	@docker-compose down
	@echo "✅ PostgreSQL stopped!"

# CLI targets
cli-postgres:
	@echo "Running PostgreSQL CLI..."
	@go run cmd/cli-postgres/main.go

# Migration targets
migrate-up:
	@docker-compose ps postgres | grep -q " Up " || { echo "❌ PostgreSQL container is not running. Run 'make run-postgres' first"; exit 1; }
	@echo "Applying migrations..."
	@docker-compose exec -T postgres psql -U postgres -d books -f /docker-entrypoint-initdb.d/001_create_books_table.up.sql
	@echo "✅ Migrations applied!"

migrate-down:
	@docker-compose ps postgres | grep -q " Up " || { echo "❌ PostgreSQL container is not running. Run 'make run-postgres' first"; exit 1; }
	@echo "Reverting migrations..."
	@docker-compose exec -T postgres psql -U postgres -d books -f /docker-entrypoint-initdb.d/001_create_books_table.down.sql
	@echo "✅ Migrations reverted!"