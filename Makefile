# JFrog CLI Evidence Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint
MOCKGEN=mockgen

# Build variables
BINARY_NAME=jfrog-evidence
MAIN_PATH=evidence/cmd/main.go
BUILD_DIR=build
COVERAGE_DIR=coverage

# Version information
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Test flags
TEST_FLAGS= -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic
TEST_TIMEOUT=10m

# Directories
EVIDENCE_DIR=evidence
MOCK_DIR=mocks
E2E_DIR=tests/e2e

# E2E Test Configuration
E2E_TEST_TIMEOUT?=30m
E2E_TEST_FLAGS?=-v -timeout $(E2E_TEST_TIMEOUT)
E2E_PARALLEL?=1

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

.PHONY: all build test clean help

help: ## Display this help message
	@echo "$(GREEN)JFrog CLI Evidence Makefile$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Categories:$(NC)"
	@grep -E '^##@' $(MAKEFILE_LIST) | awk 'BEGIN {FS = "##@ "}; {printf "  %s\n", $$2}'

##@ Development

all: test build ## Run tests and build the binary

build: ## Build the binary
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-all: ## Build for multiple platforms
	@echo "$(GREEN)Building for multiple platforms...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "$(GREEN)Multi-platform build complete$(NC)"

run: build ## Run the application
	@echo "$(GREEN)Running $(BINARY_NAME)...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME)

install: ## Install the binary to GOPATH/bin
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	$(GOCMD) install $(LDFLAGS) $(MAIN_PATH)
	@echo "$(GREEN)Installation complete$(NC)"

##@ Testing

test: ## Run all tests
	@echo "$(GREEN)Running tests...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) $(TEST_FLAGS) -timeout $(TEST_TIMEOUT) ./$(EVIDENCE_DIR)/...
	@echo "$(GREEN)Tests complete$(NC)"

test-short: ## Run short tests
	@echo "$(GREEN)Running short tests...$(NC)"
	$(GOTEST) -short ./$(EVIDENCE_DIR)/...

test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(NC)"
	$(GOTEST) -tags=integration -timeout 30m ./$(EVIDENCE_DIR)/...

test-unit: ## Run unit tests only
	@echo "$(GREEN)Running unit tests...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -short $(TEST_FLAGS) ./$(EVIDENCE_DIR)/...

coverage: test ## Generate test coverage report
	@echo "$(GREEN)Generating coverage report...$(NC)"
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(GREEN)Coverage report generated: $(COVERAGE_DIR)/coverage.html$(NC)"

##@ Code Quality

fmt: ## Format code
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GOFMT) -s -w .
	$(GOCMD) fmt ./...
	@echo "$(GREEN)Code formatting complete$(NC)"

lint: install-lint ## Run linter
	@echo "$(GREEN)Running linter...$(NC)"
	$(GOLINT) run ./...

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	$(GOCMD) vet ./...

check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)
	@echo "$(GREEN)All checks passed!$(NC)"

##@ Dependencies

deps: ## Download dependencies
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	$(GOMOD) download
	@echo "$(GREEN)Dependencies downloaded$(NC)"

deps-update: ## Update dependencies
	@echo "$(GREEN)Updating dependencies...$(NC)"
	$(GOGET) -u ./...
	$(GOMOD) tidy
	@echo "$(GREEN)Dependencies updated$(NC)"

deps-verify: ## Verify dependencies
	@echo "$(GREEN)Verifying dependencies...$(NC)"
	$(GOMOD) verify
	@echo "$(GREEN)Dependencies verified$(NC)"

tidy: ## Run go mod tidy
	@echo "$(GREEN)Running go mod tidy...$(NC)"
	$(GOMOD) tidy
	@echo "$(GREEN)go mod tidy complete$(NC)"

##@ Tools Installation

install-tools: install-mockgen install-lint ## Install all development tools
	@echo "$(GREEN)All tools installed$(NC)"

install-mockgen: ## Install mockgen tool
	@echo "$(GREEN)Installing mockgen...$(NC)"
	@which $(MOCKGEN) > /dev/null 2>&1 || $(GOCMD) install go.uber.org/mock/mockgen@latest
	@echo "$(GREEN)mockgen installed$(NC)"

install-lint: ## Install golangci-lint
	@echo "$(GREEN)Installing golangci-lint...$(NC)"
	@which $(GOLINT) > /dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin; \
	}
	@echo "$(GREEN)golangci-lint installed$(NC)"

##@ Cleanup

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -f $(BINARY_NAME)
	@echo "$(GREEN)Clean complete$(NC)"

clean-all: clean clean-mocks ## Clean everything including mocks and vendor
	@echo "$(YELLOW)Cleaning all generated files...$(NC)"
	rm -rf vendor/
	@echo "$(GREEN)Full clean complete$(NC)"


##@ Git Hooks

pre-commit: fmt vet test-short ## Run pre-commit checks
	@echo "$(GREEN)Pre-commit checks passed$(NC)"

pre-push: check ## Run pre-push checks
	@echo "$(GREEN)Pre-push checks passed$(NC)"

install-hooks: ## Install git hooks for pre-commit and pre-push
	@echo "$(GREEN)Installing git hooks...$(NC)"
	@bash scripts/setup-git-hooks.sh
	@echo "$(GREEN)Git hooks installed successfully$(NC)"

uninstall-hooks: ## Uninstall git hooks
	@echo "$(YELLOW)Uninstalling git hooks...$(NC)"
	@rm -f .git/hooks/pre-commit .git/hooks/pre-push
	@echo "$(GREEN)Git hooks uninstalled$(NC)"

##@ Utilities

version: ## Display version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

todo: ## Show TODO items in code
	@echo "$(YELLOW)TODO items in code:$(NC)"
	@grep -r "TODO" $(EVIDENCE_DIR) --include="*.go" || echo "No TODO items found"

fixme: ## Show FIXME items in code
	@echo "$(YELLOW)FIXME items in code:$(NC)"
	@grep -r "FIXME" $(EVIDENCE_DIR) --include="*.go" || echo "No FIXME items found"

# Default target
.DEFAULT_GOAL := help


##@ E2E Testing
test-e2e: build ## Run E2E tests (requires running environment)
	@echo "$(GREEN)Running E2E tests...$(NC)"
	@echo "$(YELLOW)Note: Make sure E2E environment is running (make start-e2e-env)$(NC)"
	$(GOTEST) $(E2E_TEST_FLAGS) -p $(E2E_PARALLEL) ./tests/e2e/...
	@echo "$(GREEN)E2E tests complete$(NC)"

start-e2e-env: build ## Start E2E test environment (docker-compose)
	@echo "$(GREEN)Starting E2E test environment...$(NC)"
	@bash $(E2E_DIR)/local/scripts/e2e-start-env.sh

stop-e2e-env: ## Stop E2E test environment
	@echo "$(YELLOW)Stopping E2E test environment...$(NC)"
	@bash $(E2E_DIR)/local/scripts/e2e-stop-env.sh

restart-e2e-env: stop-e2e-env start-e2e-env ## Restart E2E test environment
	@echo "$(GREEN)E2E environment restarted$(NC)"

e2e-logs: ## Show E2E environment logs
	@echo "$(BLUE)E2E Environment Logs:$(NC)"
	@cd $(E2E_DIR) && (docker compose logs -f 2>/dev/null || docker-compose logs -f)

e2e-status: ## Show E2E environment status
	@echo "$(BLUE)E2E Environment Status:$(NC)"
	@cd $(E2E_DIR) && (docker compose ps 2>/dev/null || docker-compose ps)

e2e-clean: stop-e2e-env ## Clean E2E environment (including volumes)
	@echo "$(YELLOW)Cleaning E2E environment...$(NC)"
	@cd $(E2E_DIR) && (docker compose down -v 2>/dev/null || docker-compose down -v)
	@echo "$(GREEN)E2E environment cleaned$(NC)"

e2e-bootstrap: ## Re-run bootstrap script (with environment running)
	@echo "$(GREEN)Re-bootstrapping E2E environment...$(NC)"
	@bash $(E2E_DIR)/local/scripts/e2e-bootstrap.sh

e2e-cleanup: ## Clean up E2E test data (users, permissions)
	@echo "$(YELLOW)Cleaning up E2E test data...$(NC)"
	@bash $(E2E_DIR)/local/scripts/e2e-cleanup.sh
	@echo "$(GREEN)E2E test data cleaned$(NC)"

e2e-full: clean build stop-e2e-env ## Full E2E test cycle (stop, clean volumes, build, start, test, stop)
	@echo "$(GREEN)Starting full E2E test cycle...$(NC)"
	@cd $(E2E_DIR) && (docker compose down -v 2>/dev/null || docker-compose down -v)
	@$(MAKE) start-e2e-env
	@sleep 5
	@$(MAKE) test-e2e
	@$(MAKE) stop-e2e-env
	@echo "$(GREEN)Full E2E test cycle complete!$(NC)"

