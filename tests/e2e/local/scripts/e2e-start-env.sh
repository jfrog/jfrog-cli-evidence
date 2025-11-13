#!/usr/bin/env bash

# Start E2E Test Environment
# This script starts the docker-compose environment and bootstraps Artifactory

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
E2E_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
PROJECT_ROOT="$(cd "${E2E_DIR}/../.." && pwd)"

source "${SCRIPT_DIR}/e2e-bootstrap-functions.sh"

log_info "=========================================="
log_info "Starting E2E Test Environment"
log_info "=========================================="
log_info "Project root: ${PROJECT_ROOT}"
log_info "E2E directory: ${E2E_DIR}"
log_info ""

# Detect Docker Compose command (V1 or V2)
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    log_error "Docker Compose is not installed"
    log_error "Please install Docker Compose: https://docs.docker.com/compose/install/"
    exit 1
fi
log_info "Using Docker Compose: ${DOCKER_COMPOSE}"

# Build the CLI binary first
log_info "Building JFrog CLI Evidence binary..."
cd "${PROJECT_ROOT}"
make build
log_success "Binary built successfully"

# Load environment variables from .env
if [[ -f "${E2E_DIR}/local/.env" ]]; then
    log_info "Loading environment from .env file..."
    set -a
    source "${E2E_DIR}/local/.env"
    set +a
else
    log_error ".env file not found at ${E2E_DIR}/local/.env"
    log_error "Please create .env file with required Docker image versions"
    exit 1
fi

# Check if environment is already running
log_info "Checking environment status..."
cd "${E2E_DIR}/local"
if ${DOCKER_COMPOSE} ps | grep -q "Up"; then
    log_warning "Environment is already running. Containers will be updated if needed."
fi

# Start Docker Compose services
log_info "Starting Docker Compose services..."
${DOCKER_COMPOSE} up -d

# Wait for services to be healthy
log_info "Waiting for services to be healthy..."
export JFROG_URL="${JFROG_URL:-http://localhost:8082}"
export JFROG_USER="${JFROG_USER:-admin}"
export JFROG_PASSWORD="${JFROG_PASSWORD:-password}"

wait_for_artifactory
wait_for_evidence

# Add delay for services to fully initialize
log_info "Waiting 30 seconds for services to fully initialize..."
sleep 30

# Always bootstrap to ensure tokens are valid for the current Artifactory instance
# (Docker volumes may persist but Artifactory signing keys change on restart)
log_info "Bootstrapping Artifactory with test data..."
"${SCRIPT_DIR}/e2e-bootstrap.sh"

log_success "=========================================="
log_success "E2E Environment Started Successfully!"
log_success "=========================================="
log_info ""
log_info "Services:"
log_info "  - Artifactory UI:  ${JFROG_URL}/ui"
log_info "  - Artifactory API: ${JFROG_URL}/artifactory"
log_info "  - Evidence API:    ${JFROG_URL}/evidence"
log_info ""
log_info "Credentials:"
log_info "  - Username: ${JFROG_USER}"
log_info "  - Password: ${JFROG_PASSWORD}"
log_info ""
log_info "Next steps:"
log_info "  1. Run E2E tests: make test-e2e"
log_info "  2. Stop environment: make stop-e2e-env"
log_info "  3. View logs: cd tests/e2e && ${DOCKER_COMPOSE} logs -f"
log_info ""

