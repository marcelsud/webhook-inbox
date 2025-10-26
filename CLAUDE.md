# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains a **Webhook Inbox** system - a QStash-like webhook receiver with Redis Streams backend supporting FIFO and Pub/Sub delivery modes.

**Module**: `github.com/marcelsud/webhook-inbox`
**Go Version**: 1.24.0

## Development Commands

### Webhook Inbox System

```bash
# Start Redis (required for webhook system)
make up

# Option 1: Run unified server (API + Worker in same process) - RECOMMENDED
make server

# Option 2: Run API only (separate process)
make api

# Option 3: Run worker only (separate process)
make worker

# View logs
make logs          # All services
make redis-logs    # Redis only

# Stop all services
make down

# Validate routes.yaml configuration
make validate-routes
```

### Testing

```bash
# Run all tests (unit + integration)
make tests

# Run only unit tests (fast, no Docker required)
make test-unit
# or: go test ./...

# Run only integration tests (requires Docker)
make test-integration
# or: go test -tags=integration ./...

# Run tests with coverage
go test -cover ./...
go test -tags=integration -cover ./...
```

## Architecture

### Clean Architecture / Hexagonal Architecture

The project follows a layered architecture with unidirectional dependencies flowing **downward**:

```
┌─────────────────────────────────────────┐
│  Presentation Layer (cmd/)              │
│  - cmd/server (unified API + Worker)    │
│  - cmd/api-webhook (HTTP REST API)      │
│  - cmd/worker (background worker)       │
└──────────────────┬──────────────────────┘
                   │ imports
┌──────────────────▼──────────────────────┐
│  Business Logic Layer (webhook/)        │
│  - webhook.Service (implements UseCase) │
│  - webhook.Webhook (domain entity)      │
│  - webhook.Status (custom type)         │
│  - webhook.DeliveryMode (custom type)   │
└──────────────────┬──────────────────────┘
                   │ imports
┌──────────────────▼──────────────────────┐
│  Repository Interface (webhook/)        │
│  - webhook.Reader interface             │
│  - webhook.Writer interface             │
│  - webhook.StreamConsumer interface     │
│  - webhook.Repository interface         │
└──────────────────┬──────────────────────┘
                   │ imports
┌──────────────────▼──────────────────────┐
│  Infrastructure Layer                   │
│  - webhook/redis (Redis Streams impl)   │
└─────────────────────────────────────────┘
```

### Key Architectural Patterns

1. **Small, Focused Interfaces**: `Reader`, `Writer`, and `StreamConsumer` are composed into `Repository`
2. **Interface Segregation**: Clients only depend on interfaces they need
3. **Dependency Injection**: Dependencies injected through constructors (`NewService(repo)`)
4. **Value vs Pointer Semantics**:
   - **Value semantics** for data (`Webhook`, `Status`, `DeliveryMode`) - immutable
   - **Pointer semantics** for APIs (`Service`) - methods can have side effects
5. **Context as First Parameter**: All I/O operations take `context.Context` as first parameter
6. **Error Wrapping**: Use `fmt.Errorf("context: %w", err)` to preserve error chain

### Package Structure

- **`cmd/`**: Executable entry points (main.go files)
  - `cmd/server/`: Unified server (API + Worker in same process)
  - `cmd/api-webhook/`: HTTP REST API server only
  - `cmd/worker/`: Background worker only
- **`webhook/`**: Core domain package (business logic)
  - `webhook.go`: Domain entity
  - `status.go`: Custom type with validation (Pending, Delivering, Delivered, Failed, Retrying)
  - `delivery_mode.go`: Custom type (FIFO, PubSub)
  - `service.go`: Business logic (implements `UseCase` interface)
  - `repository.go`: Interface definitions (`Reader`, `Writer`, `StreamConsumer`, `Repository`)
  - `redis/`: Redis Streams repository implementation
  - `mocks/`: Auto-generated mocks (via mockery)
- **`routes/`**: Route configuration package
  - `route.go`: Route entity (RouteID → TargetURL mapping)
  - `loader.go`: Loads and validates `routes.yaml`
- **`internal/`**: Internal packages (not importable by external projects)
  - `internal/http/chi/`: HTTP handlers and routing (Chi router)
    - `webhooks.go`: Webhook endpoint handlers
    - `webhooks_handlers.go`: Handler implementations
- **`config/`**: Configuration management (Viper)
- **`webhook/signature/`**: Standard Webhooks signing and verification (HMAC-SHA256)
- **`webhook/payload/`**: Standard Webhooks payload format and validation
- **`metrics/`**: OpenTelemetry metrics collection and export

## Webhook Inbox Architecture

### Overview

A webhook receiver system with Redis Streams backend, supporting two delivery modes:

- **FIFO Mode**: Ordered delivery with parallelism=1 (guaranteed ordering)
- **Pub/Sub Mode**: High-throughput concurrent delivery with parallelism>1

### Architecture Pattern

Follows Clean Architecture:

```
┌─────────────────────────────────────────┐
│  API Layer (cmd/api-webhook)            │
│  - HTTP handlers (chi router)           │
│  - Receives webhooks, returns 202       │
└──────────────────┬──────────────────────┘
                   │ imports
┌──────────────────▼──────────────────────┐
│  Business Logic (webhook/)              │
│  - webhook.Service (UseCase)            │
│  - Domain entities (Webhook, Status)    │
└──────────────────┬──────────────────────┘
                   │ imports
┌──────────────────▼──────────────────────┐
│  Repository Interface (webhook/)        │
│  - Reader, Writer, StreamConsumer       │
└──────────────────┬──────────────────────┘
                   │ imports
┌──────────────────▼──────────────────────┐
│  Infrastructure (webhook/redis/)        │
│  - Redis Streams implementation         │
│  - FIFO: webhooks:fifo:{route_id}       │
│  - Pub/Sub: webhooks:pubsub:{route_id}  │
└─────────────────────────────────────────┘
```

### Routes Configuration

File: `routes.yaml`

```yaml
routes:
  - route_id: "user-events"
    target_url: "https://example.com/webhooks/users"
    mode: "fifo"              # ordered delivery
    max_retries: 3
    retry_backoff: "pow(2, retried) * 1000"
    parallelism: 1            # FIFO requires 1

  - route_id: "analytics"
    target_url: "https://analytics.example.com/events"
    mode: "pubsub"            # concurrent delivery
    max_retries: 3
    retry_backoff: "pow(2, retried) * 1000"
    parallelism: 5            # process 5 webhooks concurrently
```

### Redis Data Structures

1. **Streams** (message queue):
   - FIFO: `webhooks:fifo:{route_id}`
   - Pub/Sub: `webhooks:pubsub:{route_id}`
   - Consumer groups: `webhook-workers-{route_id}`
   - Uses XADD, XREADGROUP, XACK

2. **Hashes** (metadata storage):
   - Key: `webhook:{webhook_id}`
   - Stores: status, retry_count, payload, headers, timestamps

### API Endpoints

- `POST /v1/routes/{route_id}/events` - Send event to route (returns 202 with event_id)
- `GET /v1/routes` - List available routes
- `GET /health` - Health check
- `GET /metrics` - OpenTelemetry metrics in Prometheus format (requires TELEMETRY_ENABLED=true)

### Worker Behavior

1. **FIFO Routes** (parallelism=1):
   - Spawns 1 goroutine per route
   - Processes webhooks sequentially
   - Maintains order guarantee

2. **Pub/Sub Routes** (parallelism>1):
   - Spawns N goroutines per route
   - Processes webhooks concurrently
   - High throughput, no ordering

3. **Retry Logic**:
   - Exponential backoff (configurable per route)
   - Updates retry_count in Redis
   - Marks as Failed after max_retries
   - ACKs successful deliveries
   - Exponential backoff with jitter (±20%) to prevent thundering herd

## Standard Webhooks Support

**Status**: ✅ Fully Implemented (v1.0.0 spec compliance)

Webhook Inbox implements the [Standard Webhooks](https://www.standardwebhooks.com) specification for secure, reliable webhook delivery. See [STANDARDWEBHOOKS.md](STANDARDWEBHOOKS.md) for complete documentation.

### Payload Format (Required)

All webhooks **must** use the Standard Webhooks payload format:

```json
{
  "type": "user.created",
  "timestamp": "2024-01-01T12:00:00.123456789Z",
  "data": {
    "user_id": "123",
    "email": "user@example.com"
  }
}
```

- **`type`**: Hierarchical event type (e.g., `user.created`, `order.payment.succeeded`)
- **`timestamp`**: ISO 8601 formatted timestamp
- **`data`**: Event payload (valid JSON)

### Webhook Headers (Automatic)

Workers automatically add Standard Webhooks headers:

```
webhook-id: msg_01HQZX...           # ULID identifier
webhook-timestamp: 1674087231        # Unix timestamp (seconds)
webhook-signature: v1,K5oZfz...      # HMAC-SHA256 signature (if configured)
```

### Route Configuration (Standard Webhooks)

```yaml
routes:
  - route_id: "user-events"
    target_url: "https://app.com/webhooks/users"
    mode: "fifo"
    max_retries: 3
    retry_backoff: "pow(2, retried) * 1000"
    parallelism: 1

    # Standard Webhooks: Signing secret (optional)
    signing_secret: "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"

    # Standard Webhooks: Event type filtering (optional)
    event_types:
      - "user.created"       # Exact match
      - "user.updated"       # Exact match
      - "order.*"            # Wildcard: matches order.created, order.paid, etc.
```

### Signature Verification (Symmetric HMAC-SHA256)

**Signing Secret Format:**
- Prefix: `whsec_`
- Size: 24-64 bytes (base64 encoded)
- Algorithm: HMAC-SHA256

**Signed Content:**
```
{webhook-id}.{webhook-timestamp}.{raw_body}
```

**Consumer Implementation:**
See [STANDARDWEBHOOKS.md](STANDARDWEBHOOKS.md#consumer-implementation-guide) for complete verification examples.

### Event Type Filtering

- Empty `event_types`: Accept all events
- Exact match: `user.created` → only `user.created`
- Wildcard: `user.*` → `user.created`, `user.updated`, `user.deleted`, etc.
- Events not matching filter are skipped (acknowledged without delivery)

### HTTP Status Code Handling

Following Standard Webhooks spec:

- **2xx**: Success
- **410 Gone**: Consumer no longer interested (mark failed, don't retry)
- **429/502/504**: Server under load (retry with throttling)
- **3xx**: Redirect (treat as failure - update URL instead)
- **Other**: Failure (retry with backoff)

### Security Features

1. **Constant-time signature verification** (prevents timing attacks)
2. **Timestamp validation** (prevents replay attacks)
3. **Idempotency via webhook-id** (prevents duplicate processing)
4. **Secret rotation support** (multiple signatures in header)

### Future Enhancements

- ⏳ **Asymmetric Signatures** (ed25519, v1a)
- ⏳ **SSRF Protection** (URL filtering, proxy support)
- ⏳ **Endpoint Auto-Disable** (on 410 Gone responses)

### Configuration (.env)

```toml
PORT = "8080"
REDIS_HOST = "localhost"
REDIS_PORT = "6379"
REDIS_PASSWORD = ""
REDIS_DB = 0
ROUTES_FILE = "routes.yaml"
WEBHOOK_DELIVERED_TTL_HOURS = 1
WEBHOOK_FAILED_TTL_HOURS = 24

# Telemetry
TELEMETRY_ENABLED = true  # Enable /metrics endpoint (OpenTelemetry with Prometheus format)
```

**Telemetry Features** (when `TELEMETRY_ENABLED=true`):
- Exposes `/metrics` endpoint with Prometheus-formatted OpenTelemetry metrics
- Metrics include: queue lengths, status counts, throughput (1m/5m/15m), active workers
- Compatible with Prometheus, Grafana, Datadog, New Relic, etc.
- Workers send heartbeats (60s TTL) to track active workers per route

### Development Workflow

**Option 1: Unified Server (Recommended)**
```bash
# 1. Start Redis
make up

# 2. Start unified server (API + Worker in same process)
make server

# 3. Send test event
curl -X POST http://localhost:8080/v1/routes/user-events/events \
  -H "Content-Type: application/json" \
  -d '{"user_id": 123, "event": "created"}'
```

**Option 2: Separate Processes**
```bash
# 1. Start Redis
make up

# 2. Terminal 1: Start API only
make api

# 3. Terminal 2: Start worker only
make worker

# 4. Send webhooks (same as above)
```

### Testing

- **Unit tests**: `webhook/service_test.go`, `routes/loader_test.go`
- **Integration tests**: `webhook/redis/repository_integration_test.go` (requires Docker/Redis)
- **Mocks**: Generated with mockery for webhook interfaces
- Run: `go test ./...` (unit) or `go test -tags=integration ./...` (integration)

### Key Design Decisions

1. **Two modes with same infrastructure**: Both use Redis Streams, differ only in parallelism
2. **Separate streams per route**: Prevents head-of-line blocking
3. **Async by default**: API returns immediately, worker handles delivery
4. **Stateless worker**: Can scale horizontally by running multiple instances
5. **File-based routes**: No runtime route management (by design)

## Important Go Patterns Used

### 1. Packages "Provide" vs "Contain"
- Avoid utility packages like `models/`, `utils/`, `helpers/`
- Packages should **provide functionality**, not just **contain things**
- This prevents dependency issues

### 2. Custom Types for Type Safety
```go
type Status int

const (
    Pending Status = iota + 1
    Delivering
    Delivered
    Failed
    Retrying
)
```
Leverages compiler to catch errors at compile time.

### 3. Error Handling
```go
if err != nil {
    return Webhook{}, fmt.Errorf("inserting webhook: %w", err)
}
```
- Always wrap errors with context using `%w`
- Preserves error chain for `errors.Is()` and `errors.As()`

### 4. Graceful Shutdown
The API server handles OS signals (SIGINT, SIGTERM, etc.) and allows in-flight requests to complete before shutting down.

### 5. Subtests Organization
```go
t.Run("success", func(t *testing.T) {
    // test success case
})
t.Run("error case", func(t *testing.T) {
    // test error case
})
```

### 6. Internal Package Encapsulation
Code in `internal/` can only be imported by ancestor packages, enforcing encapsulation.

## Common Development Workflows

### Adding a New Route

1. Edit `routes.yaml`:
```yaml
routes:
  - route_id: "new-route"
    target_url: "https://example.com/new"
    mode: "fifo"
    max_retries: 3
    retry_backoff: "pow(2, retried) * 1000"
    parallelism: 1
```

2. Restart server: `make server`
3. Test: `curl -X POST http://localhost:8080/v1/routes/new-route/events -d '{}'`

### Adding Tests

- **Unit tests**: Create `*_test.go` file next to source
- **Integration tests**: Add `//go:build integration` at top
- **Use subtests**: Organize with `t.Run()`
- **Use require vs assert**: `require` stops on error, `assert` continues

### Running in Production

The unified server (cmd/server) is recommended for production:
- Single process reduces operational complexity
- Shared Redis connection pool
- Lower memory footprint
- Easier to monitor and deploy

## Tools & Dependencies

- **Router**: Chi v5.2.1
- **Config**: Viper v1.20.0
- **Testing**: Testify v1.10.0
- **Integration Tests**: Testcontainers v0.39.0 (for Redis)
- **Mocks**: Mockery v2.53.3
- **Logging**: Chi httplog v0.3.2
- **Database**: Redis (go-redis v9.14.0)
- **Telemetry**: OpenTelemetry SDK v1.34.0, Prometheus Exporter v0.56.0

## Key Files to Understand

### Webhook Inbox System

- `webhook/repository.go` - Small, focused interfaces (Reader, Writer, StreamConsumer)
- `webhook/service.go` - Business logic for webhook management
- `webhook/redis/repository.go` - Redis Streams implementation
- `webhook/redis/heartbeat.go` - Worker heartbeat tracking (for metrics)
- `routes/loader.go` - Routes.yaml loader with validation
- `cmd/server/main.go` - Unified server (API + Worker)
- `cmd/api-webhook/main.go` - Webhook API server
- `cmd/worker/main.go` - Background worker with retry logic
- `cmd/validate-routes/main.go` - Standalone routes.yaml validator CLI
- `internal/http/chi/webhooks.go` - HTTP handlers for webhook API
- `metrics/otel_exporter.go` - OpenTelemetry metrics exporter (Prometheus format)
- `metrics/redis_collector.go` - Collects metrics from Redis

## Code Quality

- Use `go vet` to catch common mistakes
- Use `go fmt` to format code
- Run tests whenever making changes
- Follow Go naming conventions
- Keep functions small and focused
- Document exported types and functions

## Important Notes

- Always validate routes.yaml on startup (done automatically)
- FIFO mode **must** have parallelism=1 (enforced by validation)
- Webhook IDs are generated using ULIDs (sortable, time-based)
- Redis Streams provide at-least-once delivery guarantee
- Worker uses consumer groups for distributed processing
- Status transitions: Pending → Delivering → Delivered/Failed/Retrying
