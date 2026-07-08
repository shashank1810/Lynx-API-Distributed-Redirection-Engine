# ============================================================
# Stage 1: Build
# ============================================================
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache dependency downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /bin/gateway ./cmd/gateway

# ============================================================
# Stage 2: Runtime (distroless-style minimal image)
# ============================================================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Non-root user for security.
RUN addgroup -S gateway && adduser -S -G gateway gateway

WORKDIR /app

# Copy binary and assets.
COPY --from=builder /bin/gateway /app/gateway
COPY configs/ /app/configs/
COPY migrations/ /app/migrations/
COPY scripts/ /app/scripts/

# Own all files as non-root.
RUN chown -R gateway:gateway /app

USER gateway

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/gateway"]
CMD ["-config", "/app/configs/app.prod.yaml"]
