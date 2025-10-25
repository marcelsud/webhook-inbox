.PHONY: help tests test-unit test-integration validate-routes up down logs redis-logs server api worker

help:
	@echo "╔════════════════════════════════════════════════════════════╗"
	@echo "║          Webhook Inbox - Make Targets                      ║"
	@echo "╚════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Testing:"
	@echo "  make tests              - Run all tests (unit + integration)"
	@echo "  make test-unit          - Run only unit tests (fast)"
	@echo "  make test-integration   - Run only integration tests (requires Docker)"
	@echo ""
	@echo "Validation:"
	@echo "  make validate-routes    - Validate routes.yaml configuration"
	@echo ""
	@echo "Development:"
	@echo "  make up                 - Start Redis (Docker Compose)"
	@echo "  make down               - Stop Redis"
	@echo "  make logs               - View Redis logs"
	@echo "  make redis-logs         - View Redis logs (alias)"
	@echo "  make server             - Run unified server (API + Worker)"
	@echo "  make api                - Run webhook API only"
	@echo "  make worker             - Run webhook worker only"
	@echo ""
	@echo "Worker Route Filtering:"
	@echo "  go run cmd/worker/main.go --routes=user-events,logs"
	@echo "  go run cmd/server/main.go --api=false --routes=user-events"

# Testing targets
tests:
	@echo "Running all tests..."
	@go test ./... && go test -tags=integration ./...

test-unit:
	@echo "Running unit tests only..."
	@go test ./...

test-integration:
	@command -v docker >/dev/null 2>&1 || { echo "❌ Docker is not installed"; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "❌ Docker is not running"; exit 1; }
	@echo "Running integration tests only (requires Docker)..."
	@go test -tags=integration ./...

# Validation targets
validate-routes:
	@go run cmd/validate-routes/main.go

# Docker Compose targets
up:
	@echo "Starting Redis..."
	@docker-compose up -d
	@echo "Waiting for Redis to be ready..."
	@sleep 3
	@docker-compose ps
	@echo "✅ Redis is up!"
	@echo ""
	@echo "Redis: localhost:6379"

down:
	@echo "Stopping Redis..."
	@docker-compose down
	@echo "✅ Redis stopped!"

logs:
	@docker-compose logs -f

redis-logs:
	@docker-compose logs -f redis

# Run webhook components locally (requires 'make up' first)

# Unified server (runs both API and worker in same process)
server:
	@echo "Starting Webhook Server (API + Worker)..."
	@echo "Make sure Redis is running: make up"
	@echo ""
	-@go run cmd/server/main.go

# Run API only
api:
	@echo "Starting Webhook API only..."
	@echo "Make sure Redis is running: make up"
	@echo ""
	-@go run cmd/server/main.go --worker=false

# Run worker only
worker:
	@echo "Starting Webhook Worker only..."
	@echo "Make sure Redis is running: make up"
	@echo ""
	-@go run cmd/server/main.go --api=false