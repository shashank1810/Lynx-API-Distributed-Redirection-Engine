.PHONY: build run test lint vet fmt docker-up docker-down migrate-up migrate-down clean help

# ── Variables ────────────────────────────────────────────
BINARY_NAME    := gateway
BUILD_DIR      := ./bin
CMD_DIR        := ./cmd/gateway
COMPOSE_FILE   := deploy/docker-compose.yml
MIGRATIONS_DIR := ./migrations
DATABASE_URL   ?= postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable

# ── Build ────────────────────────────────────────────────
build: ## Build the gateway binary
	@echo "▸ Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY_NAME)"

run: build ## Build and run the gateway
	@echo "▸ Starting $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME) -config configs/app.dev.yaml

# ── Testing & Quality ───────────────────────────────────
test: ## Run all tests
	go test -v -race -count=1 ./...

test-cover: ## Run tests with coverage report
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

lint: ## Run golangci-lint
	golangci-lint run ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format all Go files
	gofmt -s -w .

# ── Docker ──────────────────────────────────────────────
docker-up: ## Start the full Docker Compose stack
	docker compose -f $(COMPOSE_FILE) up --build -d
	@echo "✓ Stack running:"
	@echo "  Gateway:    http://localhost:8080"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana:    http://localhost:3000 (admin/admin)"

docker-down: ## Tear down the Docker Compose stack
	docker compose -f $(COMPOSE_FILE) down -v

docker-logs: ## Tail logs from all containers
	docker compose -f $(COMPOSE_FILE) logs -f

# ── Database ────────────────────────────────────────────
migrate-up: ## Run database migrations up
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down: ## Rollback last migration
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

migrate-version: ## Show current migration version
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" version

# ── Load Testing ────────────────────────────────────────
load-test: ## Run k6 load test
	k6 run test/load/k6_scenario.js

# ── Cleanup ─────────────────────────────────────────────
clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "✓ Cleaned"

# ── Help ────────────────────────────────────────────────
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
