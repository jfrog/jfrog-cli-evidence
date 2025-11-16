#!/usr/bin/env bash

# JFrog CLI Evidence E2E Test Cleanup Script
# This script cleans up test users and permissions created by bootstrap

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/e2e-bootstrap-functions.sh"

# Configuration
JFROG_URL="${JFROG_URL:-http://localhost:8082}"
JFROG_USER="${JFROG_USER:-admin}"
JFROG_PASSWORD="${JFROG_PASSWORD:-password}"

# Test user configuration
TEST_USER="evidence-test-user"
PERMISSION_NAME="EvidenceE2EPermissions"

log_info "=========================================="
log_info "JFrog CLI Evidence E2E Cleanup"
log_info "=========================================="
log_info "URL: ${JFROG_URL}"
log_info ""

# Generate admin token (needed for API calls)
log_info "Generating admin access token..."
ADMIN_TOKEN=$(generate_admin_token)
if [[ -z "$ADMIN_TOKEN" ]]; then
    log_error "Failed to generate admin token"
    exit 1
fi
export ADMIN_TOKEN
log_success "Admin token generated"

# Delete permission
log_info "Cleaning up permission..."
delete_permission "${PERMISSION_NAME}" "${ADMIN_TOKEN}"

# Delete user
log_info "Cleaning up user..."
delete_user "${TEST_USER}" "${ADMIN_TOKEN}"

# Delete token and key files
if [[ -f "${SCRIPT_DIR}/../.access_token" ]]; then
    rm -f "${SCRIPT_DIR}/../.access_token"
    log_success "Deleted .access_token file"
fi

if [[ -f "${SCRIPT_DIR}/../.admin_token" ]]; then
    rm -f "${SCRIPT_DIR}/../.admin_token"
    log_success "Deleted .admin_token file"
fi

if [[ -f "${SCRIPT_DIR}/../.project_token" ]]; then
    rm -f "${SCRIPT_DIR}/../.project_token"
    log_success "Deleted .project_token file"
fi

if [[ -f "${SCRIPT_DIR}/../.project_key" ]]; then
    rm -f "${SCRIPT_DIR}/../.project_key"
    log_success "Deleted .project_key file"
fi

log_success "=========================================="
log_success "Bootstrap cleanup completed successfully!"
log_success "=========================================="
log_info ""
log_info "You can now run bootstrap again with:"
log_info "  make bootstrap-e2e"
log_info ""
log_info "Or restart the full environment:"
log_info "  make start-e2e-env"
log_info ""

