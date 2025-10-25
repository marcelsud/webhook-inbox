# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /webhook-server ./cmd/server/main.go

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /webhook-server /app/webhook-server

# Copy routes configuration
COPY routes.yaml /app/routes.yaml

# Copy .env file
COPY .env /app/.env

# Create non-root user
RUN addgroup -g 1001 -S webhook && \
    adduser -u 1001 -S webhook -G webhook

USER webhook

EXPOSE 8080

CMD ["/app/webhook-server"]
