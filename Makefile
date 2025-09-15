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
TEST_FLAGS=-v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic
TEST_TIMEOUT=10m

# Directories
EVIDENCE_DIR=evidence
MOCK_DIR=mocks

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
	@mkdir -p $(COVERAGE_DIR) test-results
	$(GOTEST) $(TEST_FLAGS) -timeout $(TEST_TIMEOUT) ./$(EVIDENCE_DIR)/... 2>&1 | tee test-results/test.log
	@echo "$(GREEN)Tests complete$(NC)"

test-short: ## Run short tests
	@echo "$(GREEN)Running short tests...$(NC)"
	@mkdir -p test-results
	$(GOTEST) -short -v ./$(EVIDENCE_DIR)/... 2>&1 | tee test-results/test-short.log

test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(NC)"
	@mkdir -p test-results
	$(GOTEST) -v -tags=integration -timeout 30m ./$(EVIDENCE_DIR)/... 2>&1 | tee test-results/test-integration.log

test-unit: ## Run unit tests only
	@echo "$(GREEN)Running unit tests...$(NC)"
	@mkdir -p $(COVERAGE_DIR) test-results
	$(GOTEST) -short $(TEST_FLAGS) ./$(EVIDENCE_DIR)/... 2>&1 | tee test-results/test-unit.log

test-junit: ## Run tests with JUnit XML output
	@echo "$(GREEN)Running tests with JUnit output...$(NC)"
	@mkdir -p $(COVERAGE_DIR) test-results
	@which gotestsum > /dev/null 2>&1 || go install github.com/gotestyourself/gotestsum@latest
	gotestsum --junitfile test-results/junit.xml \
		--format testname \
		--jsonfile test-results/test.json \
		-- $(TEST_FLAGS) -timeout $(TEST_TIMEOUT) ./$(EVIDENCE_DIR)/...
	@echo "$(GREEN)JUnit report generated: test-results/junit.xml$(NC)"

test-json: ## Run tests with JSON output
	@echo "$(GREEN)Running tests with JSON output...$(NC)"
	@mkdir -p $(COVERAGE_DIR) test-results
	$(GOTEST) -json $(TEST_FLAGS) -timeout $(TEST_TIMEOUT) ./$(EVIDENCE_DIR)/... > test-results/test.json
	@echo "$(GREEN)JSON report generated: test-results/test.json$(NC)"

coverage: test ## Generate test coverage report
	@echo "$(GREEN)Generating coverage report...$(NC)"
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out > $(COVERAGE_DIR)/coverage-summary.txt
	@echo "$(GREEN)Coverage report generated: $(COVERAGE_DIR)/coverage.html$(NC)"
	@echo "$(GREEN)Coverage summary:$(NC)"
	@tail -n 1 $(COVERAGE_DIR)/coverage-summary.txt

coverage-xml: coverage ## Generate XML coverage report
	@echo "$(GREEN)Generating XML coverage report...$(NC)"
	@which gocov > /dev/null 2>&1 || go install github.com/axw/gocov/gocov@latest
	@which gocov-xml > /dev/null 2>&1 || go install github.com/AlekSi/gocov-xml@latest
	gocov convert $(COVERAGE_DIR)/coverage.out > $(COVERAGE_DIR)/coverage.json
	gocov-xml < $(COVERAGE_DIR)/coverage.json > $(COVERAGE_DIR)/coverage.xml
	@echo "$(GREEN)XML coverage report generated: $(COVERAGE_DIR)/coverage.xml$(NC)"


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
