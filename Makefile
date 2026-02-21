.PHONY: help build run deps migrate-install migrate-up migrate-down migrate-status migrate-create migrate-force fmt fmt-check lint-install lint test test-coverage ci clean

# Define migrate binary path
MIGRATE := $(shell go env GOPATH)/bin/migrate

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/kiwis-worker cmd/kiwis-worker/main.go

run: ## Run the application
	@test -f .env || (echo "Error: .env file not found" && exit 1)
	go run cmd/kiwis-worker/main.go

deps: ## Download dependencies
	go mod download
	go mod tidy

migrate-install: ## Install golang-migrate CLI
	@test -f $(MIGRATE) || (echo "Installing golang-migrate..." && go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest && echo "✅ Installed to $(MIGRATE)")

migrate-up: migrate-install ## Apply all pending migrations
	@test -f .env || (echo "Error: .env file not found" && exit 1)
	@bash -c 'source .env && test -n "$$DATABASE_URL" || (echo "Error: DATABASE_URL not set in .env" && exit 1)'
	@bash -c 'source .env && $(MIGRATE) -path migrations -database "$$DATABASE_URL" up'

migrate-down: migrate-install ## Rollback last migration
	@test -f .env || (echo "Error: .env file not found" && exit 1)
	@bash -c 'source .env && test -n "$$DATABASE_URL" || (echo "Error: DATABASE_URL not set in .env" && exit 1)'
	@bash -c 'source .env && $(MIGRATE) -path migrations -database "$$DATABASE_URL" down 1'

migrate-status: migrate-install ## Show current migration version
	@test -f .env || (echo "Error: .env file not found" && exit 1)
	@bash -c 'source .env && test -n "$$DATABASE_URL" || (echo "Error: DATABASE_URL not set in .env" && exit 1)'
	@bash -c 'source .env && $(MIGRATE) -path migrations -database "$$DATABASE_URL" version'

migrate-create: migrate-install ## Create a new migration (usage: make migrate-create name=migration_name)
	$(MIGRATE) create -ext sql -dir migrations -seq $(name)

fmt: ## Format code using gofmt
	@echo "Formatting code..."
	gofmt -w -s .
	@echo "✅ Code formatted"

lint-install: ## Install golangci-lint
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && echo "✅ Installed to $(shell go env GOPATH)/bin/golangci-lint")

lint: lint-install ## Run linter using golangci-lint
	@echo "Running linter..."
	@$(shell go env GOPATH)/bin/golangci-lint run ./...
	@echo "✅ Linting complete"

fmt-check: ## Check if code is formatted (for CI)
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "❌ Code is not formatted. Run 'make fmt' to fix:"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "✅ Code is properly formatted"

test: ## Run all tests
	go test -v ./...

test-coverage: ## Run tests with coverage report
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

ci: deps fmt-check lint test build ## Run all CI checks (format, lint, test, build)
	@echo "✅ All CI checks passed"

clean: ## Clean build artifacts
	rm -rf bin/ coverage.out coverage.html
