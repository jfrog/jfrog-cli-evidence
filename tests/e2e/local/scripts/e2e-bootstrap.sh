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
ADMIN_TOKEN=$(generate_admin_token)
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

# Wait a moment for permission to be fully applied and propagated
log_info "Waiting for permissions to propagate..."
sleep 3

# Create token for test user (must be created AFTER permission is set)
log_info "Creating access token for test user..."
USER_TOKEN=$(create_user_token "${TEST_USER}")
if [[ -z "$USER_TOKEN" ]]; then
    log_error "Failed to create user token"
    exit 1
fi

# Note: If permission was updated, we may need to regenerate token
# For now, the token is created after permission setup should work

# Save tokens to files for Go tests to read (in tests/e2e/local/)
echo "$USER_TOKEN" > "${SCRIPT_DIR}/../.access_token"
log_success "User token saved to local/.access_token file"

echo "$ADMIN_TOKEN" > "${SCRIPT_DIR}/../.admin_token"
log_success "Admin token saved to local/.admin_token file"

# ========================================
# Project Setup (for project-based tests)
# ========================================
log_info ""
log_info "=== Setting up test project for project-based tests ==="

PROJECT_KEY="evidencee2e"
PROJECT_TOKEN_FILE="${SCRIPT_DIR}/../.project_token"
PROJECT_KEY_FILE="${SCRIPT_DIR}/../.project_key"

log_info "Creating project: ${PROJECT_KEY}..."

# Create project using Access API
project_json='{
  "project_key": "'"${PROJECT_KEY}"'",
  "display_name": "evidencee2e",
  "description": "Project for E2E evidence tests with role-based permissions",
  "admin_privileges": {
    "manage_members": true,
    "manage_resources": true,
    "index_resources": true
  }
}'

# Try to create project (ignore if already exists)
create_response=$(curl -s -w "\n%{http_code}" \
    -X POST "${JFROG_URL}/access/api/v1/projects" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "${project_json}")

project_http_code=$(echo "$create_response" | tail -n1)
if [[ "$project_http_code" == "201" ]]; then
    log_success "Project ${PROJECT_KEY} created"
elif [[ "$project_http_code" == "409" ]]; then
    log_info "Project ${PROJECT_KEY} already exists (continuing)"
else
    log_warning "Project creation returned HTTP ${project_http_code} (continuing anyway)"
fi

# Assign user to project with Developer role
log_info "Assigning user ${TEST_USER} to project ${PROJECT_KEY} with Developer role..."
assign_json='{
  "name": "'"${TEST_USER}"'",
  "roles": ["Developer"]
}'

assign_response=$(curl -s -w "\n%{http_code}" \
    -X PUT "${JFROG_URL}/access/api/v1/projects/${PROJECT_KEY}/users/${TEST_USER}" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "${assign_json}")

assign_http_code=$(echo "$assign_response" | tail -n1)
if [[ "$assign_http_code" -ge 200 ]] && [[ "$assign_http_code" -lt 300 ]]; then
    log_success "User ${TEST_USER} assigned to project ${PROJECT_KEY} with Developer role"
else
    log_warning "User assignment returned HTTP ${assign_http_code} (continuing anyway)"
fi

# Create project-scoped token for the user
log_info "Creating project-scoped token for user ${TEST_USER}..."
project_token_response=$(jfrog atc "${TEST_USER}" \
    --url="${JFROG_URL}" \
    --access-token="${ADMIN_TOKEN}" \
    --project="${PROJECT_KEY}" \
    --scope="applied-permissions/user" \
    --expiry=0 2>&1)

PROJECT_TOKEN=$(echo "$project_token_response" | jq -r '.access_token' 2>/dev/null)

if [[ -n "$PROJECT_TOKEN" ]] && [[ "$PROJECT_TOKEN" != "null" ]]; then
    echo "${PROJECT_TOKEN}" > "${PROJECT_TOKEN_FILE}"
    echo "${PROJECT_KEY}" > "${PROJECT_KEY_FILE}"
    log_success "Project-scoped token saved to ${PROJECT_TOKEN_FILE}"
    log_success "Project key saved to ${PROJECT_KEY_FILE}"
else
    log_warning "Failed to create project-scoped token"
    log_warning "Project-based tests will be skipped"
    log_info "Response: ${project_token_response}"
fi

log_success "=========================================="
log_success "Bootstrap completed successfully!"
log_success "=========================================="
log_info ""
log_info "Test User Credentials:"
log_info "  Username: ${TEST_USER}"
log_info "  Password: ${TEST_PASSWORD}"
log_info "  Email: ${TEST_EMAIL}"
log_info ""
log_info "Permissions: READ, WRITE, DELETE, ANNOTATE on ANY LOCAL and artifactory-build-info"
log_info ""
log_info "Test Project:"
log_info "  Project Key: ${PROJECT_KEY}"
log_info "  User Role: Developer"
log_info "  Project Token: $([ -f "${PROJECT_TOKEN_FILE}" ] && echo "✓ Created" || echo "✗ Not created")"
log_info ""
log_info "Note: For SaaS environments:"
log_info "  1. Create project 'evidencee2e' manually"
log_info "  2. Assign user with Developer role"
log_info "  3. Create project-scoped token and save to .project_token"
log_info ""
log_info "You can now run E2E tests with:"
log_info "  make test-e2e"
log_info ""
log_info "To clean up and start fresh:"
log_info "  make e2e-full"
log_info ""
