# High-Throughput API Gateway & Distributed URL Management Platform

A production-grade, enterprise-ready URL shortening and management platform built with Go, Redis, and PostgreSQL. Designed for high throughput, resilience, and observability.

---

## Architecture

```
┌─────────────┐    ┌──────────────┐    ┌────────────────┐
│   Client     │───▶│  Rate Limiter │───▶│ Circuit Breaker│
└─────────────┘    │  (Redis Lua)  │    │   (gobreaker)  │
                   └──────────────┘    └───────┬────────┘
                                               │
                   ┌──────────────┐    ┌───────▼────────┐
                   │ Bloom Filter │◀───│  URL Service   │
                   │  (in-memory) │    │  (cache-aside) │
                   └──────────────┘    └───────┬────────┘
                                           ┌───┴───┐
                                    ┌──────▼──┐  ┌─▼──────────┐
                                    │  Redis   │  │ PostgreSQL │
                                    │  Cache   │  │  (pgx)     │
                                    └─────────┘  └────────────┘
```

## Features

- **Base62 Encoding** — Collision-safe short code generation with cryptographic randomness
- **Atomic Rate Limiting** — Redis Lua token-bucket script (zero race conditions)
- **Cache-Aside Pattern** — Redis L1 cache with bloom filter anti-penetration defense
- **Null-Marker Defense** — Prevents cache stampede on nonexistent keys
- **Circuit Breaker** — gobreaker-based resilience for database failures
- **Prometheus Metrics** — HTTP latency histograms, cache hit ratios, rate limit counters
- **OpenTelemetry Tracing** — Distributed tracing via OTLP gRPC
- **Graceful Shutdown** — SIGINT/SIGTERM handling with connection draining
- **Health Probes** — Kubernetes-compatible `/healthz` and `/readyz` endpoints

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.23+ (for local development)
- k6 (for load testing)

### Run with Docker Compose

```bash
# Start the full stack (app + PostgreSQL + Redis + Prometheus + Grafana)
make docker-up

# Verify the stack
curl http://localhost:8080/healthz

# View logs
make docker-logs

# Tear down
make docker-down
```

### Run Locally

```bash
# Start dependencies
docker compose -f deploy/docker-compose.yml up -d postgres redis

# Run migrations
make migrate-up

# Build and run
make run
```

## API Reference

### Shorten URL

```bash
POST /api/v1/shorten
Content-Type: application/json

{
  "url": "https://example.com/very/long/path",
  "custom_code": "mycode",      # optional
  "expires_in": 86400           # optional, seconds
}

# Response: 201 Created
{
  "short_code": "a1B2c3D",
  "short_url": "http://localhost:8080/a1B2c3D",
  "original_url": "https://example.com/very/long/path"
}
```

### Resolve (Redirect)

```bash
GET /:code → 301 Moved Permanently
Location: https://example.com/very/long/path
```

### URL Stats

```bash
GET /api/v1/stats/:code

# Response: 200 OK
{
  "short_code": "a1B2c3D",
  "original_url": "https://example.com/very/long/path",
  "clicks": 42,
  "created_at": "2026-07-08T12:00:00Z",
  "is_active": true
}
```

### Health Probes

```bash
GET /healthz    → 200 (liveness)
GET /readyz     → 200 (readiness, checks PostgreSQL + Redis)
GET /metrics    → Prometheus metrics
```

## Load Testing

```bash
# Run the full k6 test suite (smoke → load → spike)
make load-test

# Or run with custom parameters
k6 run --env BASE_URL=http://localhost:8080 test/load/k6_scenario.js
```

### Performance Targets (SLOs)

| Metric | Target |
|--------|--------|
| p95 latency | < 200ms |
| p99 latency | < 500ms |
| Error rate | < 1% |
| Shorten p95 | < 300ms |
| Resolve p95 | < 100ms |

## Observability

| Service | URL |
|---------|-----|
| Gateway API | http://localhost:8080 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 (admin/admin) |

### Key Prometheus Metrics

- `gateway_http_requests_total` — request count by method/path/status
- `gateway_http_request_duration_seconds` — latency histograms
- `gateway_cache_hits_total` / `gateway_cache_misses_total` — cache effectiveness
- `gateway_ratelimit_allowed_total` / `gateway_ratelimit_denied_total` — rate limiter activity
- `gateway_circuit_breaker_state` — circuit breaker state (0=closed, 1=half-open, 2=open)

## Configuration

Configuration is loaded from YAML files with environment variable overrides:

```bash
# Environment variable format: GATEWAY_<SECTION>_<KEY>
GATEWAY_SERVER_PORT=9090
GATEWAY_DATABASE_HOST=my-postgres.internal
GATEWAY_REDIS_ADDR=my-redis:6379
GATEWAY_RATE_LIMIT_RATE=500
```

See `configs/app.dev.yaml` and `configs/app.prod.yaml` for all options.

## Project Structure

```
├── cmd/gateway/         # Application entrypoint
├── internal/
│   ├── config/          # Viper configuration
│   ├── server/          # HTTP server lifecycle
│   ├── router/          # Chi route registration
│   ├── handler/         # HTTP handlers
│   ├── middleware/       # Rate limiter, circuit breaker, logging
│   ├── service/         # Business logic
│   ├── repository/      # PostgreSQL data access
│   ├── cache/           # Redis cache + bloom filter
│   ├── encoding/        # Base62 encoder
│   ├── model/           # Domain types
│   └── telemetry/       # Prometheus + OpenTelemetry
├── deploy/              # Docker, Compose, Prometheus config
├── migrations/          # SQL migrations
├── scripts/             # Lua scripts, migration runner
├── test/                # Integration + load tests
└── configs/             # Environment configs
```

## License

MIT
