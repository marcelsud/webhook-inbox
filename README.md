# Webhook Inbox 📬

A high-performance webhook receiver system built with Go, featuring Redis Streams backend and support for both FIFO (ordered) and Pub/Sub (concurrent) delivery modes.

## 🎯 Overview

Webhook Inbox is a QStash-like webhook receiver that:
- Receives events via HTTP POST (fire-and-forget pattern)
- Stores them reliably in Redis Streams
- Delivers them to configured target URLs
- Supports two delivery modes: **FIFO** (guaranteed ordering) and **Pub/Sub** (high throughput)
- Provides automatic retries with exponential backoff
- Returns immediately with event ID for correlation

**Key Features:**
- 🚀 High-performance async delivery
- 🔄 Automatic retries with configurable backoff
- 📊 Redis Streams for reliable message queuing
- 🎛️ Flexible delivery modes (FIFO vs Pub/Sub)
- 🔥 Fire-and-forget API pattern (202 Accepted)
- 📈 OpenTelemetry metrics for monitoring
- 🐳 Docker-ready with docker-compose
- ✅ Comprehensive testing (unit + integration)

---

## 📚 Table of Contents

- [Architecture](#-architecture)
- [Quick Start](#-quick-start)
- [Configuration](#-configuration)
- [Delivery Modes](#-delivery-modes)
- [API Reference](#-api-reference)
- [Development](#-development)
- [Testing](#-testing)
- [Docker Deployment](#-docker-deployment)

---

## 🏗️ Architecture

### System Components

```
┌─────────────────────────────────────────┐
│  HTTP API (cmd/server)                  │
│  - Receives events (POST)               │
│  - Returns 202 Accepted immediately     │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│  Business Logic (webhook/)              │
│  - Domain entities (Webhook, Status)    │
│  - Service layer (UseCase)              │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│  Repository Interface (webhook/)        │
│  - Reader, Writer, StreamConsumer       │
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│  Infrastructure (webhook/redis/)        │
│  - Redis Streams implementation         │
│  - FIFO: webhooks:fifo:{route_id}       │
│  - Pub/Sub: webhooks:pubsub:{route_id}  │
└─────────────────────────────────────────┘
```

### Data Flow

1. **Receive**: HTTP POST → API validates → Store in Redis Streams → Return 202
2. **Process**: Worker polls Redis → Read webhook → Forward to target URL
3. **Retry**: On failure → Update retry count → Exponential backoff → Retry
4. **Complete**: On success → ACK message → Update status to Delivered

### Directory Structure

```
.
├── cmd/                          # Executable entry points
│   └── server/main.go           # Unified server (API + Worker)
├── webhook/                     # Domain package
│   ├── webhook.go               # Webhook entity
│   ├── status.go                # Status type (Pending, Delivered, etc.)
│   ├── delivery_mode.go         # Delivery mode (FIFO, PubSub)
│   ├── service.go               # Business logic
│   ├── repository.go            # Interfaces
│   ├── redis/                   # Redis implementation
│   │   └── repository.go
│   └── mocks/                   # Generated mocks
├── routes/                      # Route configuration
│   ├── route.go                 # Route entity
│   └── loader.go                # Loads routes.yaml
├── internal/http/chi/           # HTTP handlers
│   ├── webhooks.go              # Webhook endpoints
│   └── webhooks_handlers.go    # Handler implementations
├── config/                      # Configuration
│   └── config.go
├── routes.yaml                  # Route definitions
├── docker-compose.yml           # Docker services
└── Makefile                     # Build commands
```

---

## ⚡ Quick Start

Get Webhook Inbox running in 3 minutes!

### Prerequisites

- **Go 1.24.0+** ([download](https://go.dev/dl/))
- **Docker & Docker Compose** ([download](https://docs.docker.com/get-docker/))
- **Make** (optional, for convenience commands)

### Step 1: Clone the Repository

```bash
git clone <your-repo-url>
cd webhook-inbox
go mod download
```

### Step 2: Start Redis

```bash
make up
# Output: ✅ Redis is up! (localhost:6379)
```

> **Note:** The project includes a `docker-compose.yml` that starts Redis automatically.

### Step 3: Configure Routes (Optional)

The project includes example routes in `routes.yaml`. You can use them as-is or customize:

```yaml
routes:
  - route_id: "user-events"
    target_url: "https://echo.free.beeceptor.com/webhooks/users"
    mode: "fifo"              # Ordered delivery
    max_retries: 3
    retry_backoff: "pow(2, retried) * 1000"
    parallelism: 1

  - route_id: "analytics-events"
    target_url: "https://echo.free.beeceptor.com/webhooks/analytics"
    mode: "pubsub"            # Concurrent delivery
    max_retries: 3
    retry_backoff: "pow(2, retried) * 1000"
    parallelism: 5
```

> **Tip:** Use [Beeceptor](https://beeceptor.com) or [RequestBin](https://requestbin.com) for testing webhooks.

### Step 4: Start the Server

```bash
make server
```

You should see:

```
✓ Loaded and validated 4 routes from routes.yaml
✓ Connected to Redis at localhost:6379
✓ Worker started for 4 routes
✓ API Server listening on port 8080
  POST /v1/routes/{route_id}/events - Send event to route
  GET  /v1/routes - List available routes
  GET  /health - Health check
  GET  /metrics - OpenTelemetry metrics
```

### Step 5: Send Your First Event

```bash
curl -X POST http://localhost:8080/v1/routes/user-events/events \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 123,
    "event": "user.created",
    "timestamp": "2025-10-19T12:00:00Z"
  }'
```

**Response (202 Accepted):**

```json
{
  "event_id": "6245a52e-dfb1-42b6-b60c-bec1da862ce1",
  "route_id": "user-events"
}
```

The event is now queued for delivery! Check your target URL (or Beeceptor) to see the delivered webhook.

### Next Steps

- 📖 Read the [API Reference](#-api-reference) to learn about all endpoints
- ⚙️ Customize [Configuration](#-configuration) for your environment
- 🎛️ Learn about [Delivery Modes](#-delivery-modes) (FIFO vs Pub/Sub)
- 📈 Enable [OpenTelemetry Metrics](#opentelemetry-metrics) for monitoring

---

## ⚙️ Configuration

### Environment Variables (.env)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | 8080 | HTTP server port |
| `REDIS_HOST` | Yes | - | Redis hostname |
| `REDIS_PORT` | No | 6379 | Redis port |
| `REDIS_PASSWORD` | No | "" | Redis password |
| `REDIS_DB` | No | 0 | Redis database number |
| `ROUTES_FILE` | No | routes.yaml | Path to routes configuration |
| `WEBHOOK_DELIVERED_TTL_HOURS` | No | 1 | TTL for delivered webhooks |
| `WEBHOOK_FAILED_TTL_HOURS` | No | 24 | TTL for failed webhooks |
| `TELEMETRY_ENABLED` | No | false | Enable OpenTelemetry metrics export |

### Routes Configuration (routes.yaml)

```yaml
routes:
  - route_id: "unique-route-id"                   # Unique identifier
    target_url: "https://..."                     # Destination URL
    mode: "fifo"                                  # "fifo" or "pubsub"
    max_retries: 3                                # Max retry attempts
    retry_backoff: "pow(2, retried) * 1000"       # Backoff formula (ms)
    parallelism: 1                                # Concurrent workers (FIFO must be 1)
    expected_status: 200                          # Expected HTTP status for success (default: 200)
```

**Field Descriptions:**

| Field | Required | Description |
|-------|----------|-------------|
| `route_id` | Yes | Unique identifier for the route |
| `target_url` | Yes | Destination URL where events will be delivered |
| `mode` | Yes | Delivery mode: `"fifo"` (ordered) or `"pubsub"` (concurrent) |
| `max_retries` | Yes | Maximum number of retry attempts on failure |
| `retry_backoff` | Yes | Backoff formula in milliseconds (supports expressions) |
| `parallelism` | Yes | Number of concurrent workers (must be 1 for FIFO) |
| `expected_status` | No | Expected HTTP status code for successful delivery (default: 200) |

**Validation Rules:**
- `route_id` must be unique across all routes
- `mode` must be either `"fifo"` or `"pubsub"`
- FIFO mode **requires** `parallelism: 1` (ordering guarantee)
- Pub/Sub mode allows `parallelism > 1` (concurrent delivery)
- `retry_backoff` supports expressions like `pow(2, retried) * 1000` or `min(pow(2, retried) * 1000, 60000)`

**Validate Configuration:**
```bash
# Validate routes.yaml before running server (fails fast with exit code 1 on errors)
make validate-routes

# Or specify a custom file
go run cmd/validate-routes/main.go path/to/routes.yaml
```

---

## 🎛️ Delivery Modes

### FIFO Mode (Ordered Delivery)

**Characteristics:**
- ✅ Guarantees message ordering
- ✅ Processes one webhook at a time
- ✅ Suitable for workflows requiring strict order
- ⏱️ Lower throughput

**Use Cases:**
- User state changes (create → update → delete)
- Financial transactions
- Sequential workflows

**Configuration:**

```yaml
mode: "fifo"
parallelism: 1  # MUST be 1
```

### Pub/Sub Mode (High Throughput)

**Characteristics:**
- ✅ High concurrent processing
- ✅ Maximum throughput
- ❌ No ordering guarantee
- ⚡ Parallel workers

**Use Cases:**
- Analytics events
- Logging/metrics
- Independent notifications

**Configuration:**

```yaml
mode: "pubsub"
parallelism: 10  # Can be > 1
```

---

## 📡 API Reference

### Send Event to Route

Send an event to a configured route for async delivery.

```http
POST /v1/routes/{route_id}/events
Content-Type: application/json

{
  "any": "json",
  "payload": "here"
}
```

**Path Parameters:**
- `route_id` - The route ID configured in `routes.yaml`

**Request Body:**
- Any valid JSON payload (will be forwarded to the target URL as-is)

**Response (202 Accepted):**

```json
{
  "event_id": "01JAXXX...",
  "route_id": "user-events"
}
```

**Fire-and-Forget Pattern:**

Once you receive `202 Accepted`, the event is queued for delivery. The API does not provide a way to query event status - this is intentional:

- ✅ **Use the `event_id`** for correlation in logs and monitoring
- ✅ **Use `/metrics` endpoint** to monitor queue length and delivery rates
- ✅ **Target URL receives the event** - your application logic handles success/failure
- ❌ **No status query API** - keeps the system simple and scalable

**Why Fire-and-Forget?**

1. **Simplicity** - Publishers send and continue, no polling required
2. **Performance** - No database lookups for status queries
3. **Scalability** - Stateless API, easy to scale horizontally
4. **Monitoring** - Use OpenTelemetry metrics for operational visibility

If you need event tracking, implement it at the target URL (your webhook endpoint).

### List Available Routes

```http
GET /v1/routes
```

**Response (200 OK):**

```json
[
  {
    "route_id": "user-events",
    "target_url": "https://example.com/webhooks/users",
    "mode": "fifo",
    "max_retries": 3,
    "retry_backoff": "pow(2, retried) * 1000",
    "parallelism": 1,
    "expected_status": 200
  },
  {
    "route_id": "analytics",
    "target_url": "https://analytics.example.com/events",
    "mode": "pubsub",
    "max_retries": 5,
    "retry_backoff": "pow(2, retried) * 1000",
    "parallelism": 10,
    "expected_status": 200
  }
]
```

### Health Check

```http
GET /health
```

**Response (200 OK):**

```json
{
  "status": "healthy"
}
```

### OpenTelemetry Metrics

When `TELEMETRY_ENABLED=true` in `.env`, the server exposes Prometheus-formatted metrics:

```http
GET /metrics
```

**Available Metrics:**

- `webhook_queue_length{route_id}` - Number of pending webhooks per route
- `webhook_status_count{webhook_status}` - Webhook count by status (pending, delivered, failed, etc.)
- `webhook_throughput{time_window}` - Delivery rate for 1m, 5m, 15m windows
- `webhook_workers_active{route_id}` - Active workers per route

**Example Response:**

```
# HELP webhook_queue_length Number of webhooks in queue per route
# TYPE webhook_queue_length gauge
webhook_queue_length{route_id="user-events"} 5
webhook_queue_length{route_id="analytics"} 12

# HELP webhook_status_count Webhook count by status
# TYPE webhook_status_count gauge
webhook_status_count{webhook_status="delivered"} 150
webhook_status_count{webhook_status="pending"} 17
webhook_status_count{webhook_status="failed"} 3

# HELP webhook_throughput Webhook delivery throughput
# TYPE webhook_throughput gauge
webhook_throughput{time_window="1m"} 45
webhook_throughput{time_window="5m"} 38
webhook_throughput{time_window="15m"} 42

# HELP webhook_workers_active Active workers per route
# TYPE webhook_workers_active gauge
webhook_workers_active{route_id="user-events"} 1
webhook_workers_active{route_id="analytics"} 10
```

**Integration with Monitoring Tools:**

The `/metrics` endpoint can be scraped by:
- Prometheus
- Grafana
- Datadog
- New Relic
- Any OpenTelemetry-compatible monitoring tool

---

## 🛠️ Development

### Available Make Commands

```bash
make help              # Show all commands
make up                # Start Redis
make down              # Stop all services
make server            # Run unified server (API + Worker)
make api               # Run API only
make worker            # Run worker only
make tests             # Run all tests
make test-unit         # Run unit tests
make test-integration  # Run integration tests (requires Docker)
```

### Running Locally

**Unified Server (Easiest):**

```bash
make up      # Start Redis
make server  # Start API + Worker
```

**Separate Processes:**

```bash
# Terminal 1
make up

# Terminal 2
make api

# Terminal 3
make worker
```

### Project Structure

- `cmd/server/` - Unified server entrypoint
- `webhook/` - Core domain logic
- `webhook/redis/` - Redis Streams implementation
- `routes/` - Route loading and validation
- `internal/http/chi/` - HTTP handlers

---

## ✅ Testing

### Run All Tests

```bash
make tests
```

### Unit Tests Only

```bash
make test-unit
# or
go test ./...
```

### Integration Tests (Requires Docker)

```bash
make test-integration
# or
go test -tags=integration ./...
```

**What's Tested:**
- ✅ Webhook creation and status updates
- ✅ Redis Streams integration
- ✅ Route loading and validation
- ✅ HTTP API endpoints
- ✅ Retry logic
- ✅ FIFO vs Pub/Sub behavior

### Test Coverage

```bash
go test -cover ./...
go test -tags=integration -cover ./...
```

---

## 🐳 Docker Deployment

### Using Docker Compose

The project includes a complete docker-compose setup:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

**Services:**
- **redis**: Redis 7 Alpine (message queue)
- **webhook-server**: Unified API + Worker
- **dummy-server**: Test target server (for testing)

### Production Deployment

**Environment variables for container:**

```env
PORT=8080
REDIS_HOST=your-redis-host
REDIS_PORT=6379
REDIS_PASSWORD=your-password
REDIS_DB=0
ROUTES_FILE=/app/routes.yaml
```

**Docker run:**

```bash
docker build -f Dockerfile -t webhook-inbox .
docker run -p 8080:8080 \
  -e REDIS_HOST=redis \
  -v $(pwd)/routes.yaml:/app/routes.yaml \
  webhook-inbox
```

---

## 🔧 Troubleshooting

### Redis Connection Failed

**Problem:** Worker can't connect to Redis

**Solution:**

```bash
# Check if Redis is running
docker-compose ps

# Start Redis
make up

# Check Redis logs
make redis-logs
```

### Route Not Found

**Problem:** `POST /v1/routes/unknown-route/events` returns 404

**Solution:** Check `routes.yaml` and ensure the route_id exists:

```yaml
routes:
  - route_id: "your-route"  # Must match route_id in POST path
    target_url: "..."
    mode: "fifo"
    max_retries: 3
    parallelism: 1
```

### Events Not Being Delivered

**Problem:** Events are accepted but not delivered to target

**Solutions:**
1. Check if worker is running: `docker-compose ps` or `make worker`
2. Check Redis: `docker-compose logs redis`
3. Verify target URL is reachable
4. Check worker logs for delivery errors

---

## 📊 Redis Data Structures

### Streams (Message Queue)

**FIFO:**
```
Key: webhooks:fifo:{route_id}
Consumer Group: webhook-workers-{route_id}
```

**Pub/Sub:**
```
Key: webhooks:pubsub:{route_id}
Consumer Group: webhook-workers-{route_id}
```

### Hashes (Event Metadata)

```
Key: webhook:{event_id}
Fields:
  - id (event_id)
  - route_id
  - status
  - retry_count
  - max_retries
  - delivery_mode
  - payload
  - headers
  - created_at
  - updated_at
```

---

## 🎯 Key Design Patterns

### Clean Architecture
- **Domain layer** (`webhook/`): Business logic, entities
- **Infrastructure layer** (`webhook/redis/`): Redis implementation
- **Presentation layer** (`cmd/`, `internal/http/`): HTTP/CLI

### Dependency Injection

```go
repo := redis.NewRepository(redisClient)
service := webhook.NewService(repo)
handler := chi.NewWebhookHandler(service)
```

### Interface Segregation

```go
type Reader interface { ... }
type Writer interface { ... }
type StreamConsumer interface { ... }

// Compose small interfaces
type Repository interface {
    Reader
    Writer
    StreamConsumer
}
```

---

## 📝 License

This project is provided as educational material.

---

## 🤝 Contributing

Contributions welcome! Please open issues and PRs.

---

**Built with ❤️ using Go, Redis Streams, and Clean Architecture principles**
