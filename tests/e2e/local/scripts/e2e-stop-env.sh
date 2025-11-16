#!/usr/bin/env bash

# Stop E2E Test Environment
# This script stops and removes the docker-compose environment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
E2E_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

source "${SCRIPT_DIR}/e2e-bootstrap-functions.sh"

log_info "=========================================="
log_info "Stopping E2E Test Environment"
log_info "=========================================="

# Detect Docker Compose command (V1 or V2)
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    log_error "Docker Compose is not installed"
    exit 1
fi

# Stop Docker Compose services
log_info "Stopping Docker Compose services..."
cd "${E2E_DIR}/local"
${DOCKER_COMPOSE} down

# Optionally remove volumes
if [[ "${CLEAN_VOLUMES}" == "true" ]]; then
    log_warning "Removing Docker volumes..."
    ${DOCKER_COMPOSE} down -v
    log_success "Volumes removed"
fi

log_success "=========================================="
log_success "E2E Environment Stopped Successfully!"
log_success "=========================================="
log_info ""
log_info "To clean up completely (including volumes):"
log_info "  CLEAN_VOLUMES=true make stop-e2e-env"
log_info ""
log_info "To start again:"
log_info "  make start-e2e-env"
log_info ""

