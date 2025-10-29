#!/usr/bin/env bash

# JFrog CLI Evidence E2E Test Bootstrap Script
# This script sets up Artifactory with test data for E2E testing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/e2e-bootstrap-functions.sh"

# Configuration
JFROG_URL="${JFROG_URL:-http://localhost:8082}"
JFROG_USER="${JFROG_USER:-admin}"
JFROG_PASSWORD="${JFROG_PASSWORD:-password}"

# Test user configuration
TEST_USER="evidence-test-user"
TEST_PASSWORD="EvidenceTest123!"
TEST_EMAIL="evidence-test@jfrog.local"
PERMISSION_NAME="EvidenceE2EPermissions"

log_info "=========================================="
log_info "JFrog CLI Evidence E2E Bootstrap"
log_info "=========================================="
log_info "URL: ${JFROG_URL}"
log_info "Admin User: ${JFROG_USER}"
log_info ""

# Wait for Artifactory to be ready
log_info "Waiting for Artifactory to be ready..."
wait_for_artifactory

# Configure JFrog CLI
log_info "Configuring JFrog CLI..."
configure_jfrog_cli

# Install license
log_info "Installing Artifactory license..."
install_license

# Generate admin token first (needed for API calls)
log_info "Generating admin access token..."
ADMIN_TOKEN=$(generate_access_token)
if [[ -z "$ADMIN_TOKEN" ]]; then
    log_error "Failed to generate admin token"
    exit 1
fi
export ADMIN_TOKEN
log_success "Admin token generated"

# Create test user (warns if exists)
log_info "Creating E2E test user..."
create_user "${TEST_USER}" "${TEST_PASSWORD}" "${TEST_EMAIL}" "${ADMIN_TOKEN}"

# Create permission and assign to user (warns if exists)
log_info "Creating E2E test permissions..."
create_permission "${PERMISSION_NAME}" "${TEST_USER}" "${ADMIN_TOKEN}"

# Create token for test user
log_info "Creating access token for test user..."
USER_TOKEN=$(create_user_token "${TEST_USER}")
if [[ -z "$USER_TOKEN" ]]; then
    log_error "Failed to create user token"
    exit 1
fi

# Save token to file for Go tests to read
echo "$USER_TOKEN" > "${SCRIPT_DIR}/../../.access_token"
log_success "User token saved to .access_token file"

log_success "=========================================="
log_success "Bootstrap completed successfully!"
log_success "=========================================="
log_info ""
log_info "Test User Credentials:"
log_info "  Username: ${TEST_USER}"
log_info "  Password: ${TEST_PASSWORD}"
log_info "  Email: ${TEST_EMAIL}"
log_info ""
log_info "Permissions: READ, WRITE, DELETE, ANNOTATE on ANY LOCAL"
log_info ""
log_info "You can now run E2E tests with:"
log_info "  make test-e2e"
log_info ""
log_info "To clean up test data:"
log_info "  make e2e-cleanup"
log_info ""
