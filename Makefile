# ==============================================================================
# ICE POS API v2 - Makefile
# ==============================================================================

# Variables
APP_NAME      := ice-pos-api-v2
BINARY_NAME   := api
BIN_DIR       := bin
MAIN_SRC      := cmd/api/main.go
SWAG_DIR      := docs/swagger
GOPATH        := $(shell go env GOPATH)
SWAG_BIN      := $(shell which swag 2>/dev/null || echo $(GOPATH)/bin/swag)
AIR_BIN       := $(shell which air 2>/dev/null || echo $(GOPATH)/bin/air)
GOLANGCI_LINT := $(shell which golangci-lint 2>/dev/null || echo $(GOPATH)/bin/golangci-lint)
GO            := go

# Build flags
BUILD_TIME    := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS       := -s -w -X main.Version=2.0.0 -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

.PHONY: all help build run dev test test-coverage swagger swag tidy fmt lint clean install-tools migrate migrate-dry-run

all: help

help: ## Display this help screen with available targets
	@echo "\033[1;36m==================================================================\033[0m"
	@echo "\033[1;32m  ICE POS API v2 - Available Make Commands\033[0m"
	@echo "\033[1;36m==================================================================\033[0m"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[33m%-18s\033[0m %s\n", $$1, $$2}'
	@echo ""

build: ## Build the production binary into bin/api
	@echo "\033[1;34m==> Building $(APP_NAME) binary...\033[0m"
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_SRC)
	@echo "\033[1;32m==> Build successful: $(BIN_DIR)/$(BINARY_NAME)\033[0m"

run: ## Run the application directly with go run
	@echo "\033[1;34m==> Running $(APP_NAME)...\033[0m"
	$(GO) run $(MAIN_SRC)

run-vision-ai: ## Run the Python Vision AI embedding microservice
	@echo "\033[1;34m==> Starting Vision AI embedding service on port 5001...\033[0m"
	cd services/vision-ai && python3 -m uvicorn server:app --port 5001 --reload

migrate-dry-run: ## Validate and dry-run migration from v1 SQLite to target DB
	@echo "\033[1;34m==> Testing v1 to v2 migration in dry-run mode...\033[0m"
	$(GO) run cmd/migrate/main.go --dry-run

migrate: ## Execute data migration from v1 SQLite to target DB
	@echo "\033[1;34m==> Executing v1 to v2 data migration...\033[0m"
	$(GO) run cmd/migrate/main.go --clean --archive-stock --copy-uploads

dev: ## Run application with live reloading using Air (falls back to go run)
	@if [ -x "$(shell which air 2>/dev/null)" ] || [ -x "$(AIR_BIN)" ]; then \
		echo "\033[1;34m==> Starting Air live-reload server...\033[0m"; \
		air; \
	else \
		echo "\033[1;33m==> Air not found. Running with 'go run'... (install with 'make install-tools')\033[0m"; \
		$(GO) run $(MAIN_SRC); \
	fi

swagger: ## Generate Swagger OpenAPI 2.0 documentation files
	@echo "\033[1;34m==> Generating Swagger documentation...\033[0m"
	@if [ -x "$(shell which swag 2>/dev/null)" ] || [ -x "$(SWAG_BIN)" ]; then \
		$(SWAG_BIN) init -g $(MAIN_SRC) -o $(SWAG_DIR) --parseDependency --parseInternal; \
		echo "\033[1;32m==> Swagger docs generated at $(SWAG_DIR) (UI: http://localhost:8080/swagger)\033[0m"; \
	else \
		echo "\033[1;31m==> 'swag' CLI not found. Run 'make install-tools' to install.\033[0m"; \
		exit 1; \
	fi

swag: swagger ## Alias for swagger

test: ## Run all automated unit and integration tests
	@echo "\033[1;34m==> Running test suite...\033[0m"
	$(GO) test -v ./...

test-coverage: ## Run tests and generate HTML code coverage report
	@echo "\033[1;34m==> Generating test coverage profile...\033[0m"
	$(GO) test -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "\033[1;32m==> Coverage report generated at coverage.html\033[0m"

tidy: ## Tidy and verify Go modules
	@echo "\033[1;34m==> Tidying Go modules...\033[0m"
	$(GO) mod tidy
	$(GO) mod verify

fmt: ## Format all Go source files
	@echo "\033[1;34m==> Formatting Go files...\033[0m"
	$(GO) fmt ./...

lint: ## Run static analysis and linter
	@echo "\033[1;34m==> Running linter / vet...\033[0m"
	$(GO) vet ./...
	@if [ -x "$(shell which golangci-lint 2>/dev/null)" ] || [ -x "$(GOLANGCI_LINT)" ]; then \
		$(GOLANGCI_LINT) run ./...; \
	fi

clean: ## Remove build artifacts, coverage reports, and temp SQLite DBs
	@echo "\033[1;34m==> Cleaning artifacts...\033[0m"
	@rm -rf $(BIN_DIR) coverage.out coverage.html *.db *.db-journal
	@echo "\033[1;32m==> Clean completed.\033[0m"

install-tools: ## Install developer CLI tools (swag, air, golangci-lint)
	@echo "\033[1;34m==> Installing development tools...\033[0m"
	$(GO) install github.com/swaggo/swag/cmd/swag@latest
	$(GO) install github.com/air-verse/air@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "\033[1;32m==> Tools installed successfully to $(GOPATH)/bin\033[0m"
