#!/usr/bin/env bash

# Common functions for E2E test scripts

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Wait for Artifactory to be healthy
wait_for_artifactory() {
    local url="${JFROG_URL}/artifactory/api/system/ping"
    local max_retries=60  # 5 minutes max
    local retry=0
    
    while [[ $retry -lt $max_retries ]]; do
        status_code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
        
        if [[ $status_code -eq 200 ]]; then
            log_success "Artifactory is healthy!"
            return 0
        fi
        
        log_info "Waiting for Artifactory... (attempt $((retry + 1))/$max_retries, status: $status_code)"
        sleep 5
        ((retry++))
    done
    
    log_error "Artifactory failed to become healthy after $max_retries attempts"
    return 1
}

# Wait for Evidence service to be healthy
wait_for_evidence() {
    local url="${JFROG_URL}/evidence/api/v1/system/ping"
    local max_retries=60  # 5 minutes max
    local retry=0
    
    while [[ $retry -lt $max_retries ]]; do
        status_code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
        
        if [[ $status_code -eq 200 ]]; then
            log_success "Evidence service is healthy!"
            return 0
        fi
        
        log_info "Waiting for Evidence service... (attempt $((retry + 1))/$max_retries, status: $status_code)"
        sleep 5
        ((retry++))
    done
    
    log_error "Evidence service failed to become healthy after $max_retries attempts"
    return 1
}

# Configure JFrog CLI
configure_jfrog_cli() {
    # Remove existing server config if exists
    jfrog config remove e2e-test-server --quiet 2>/dev/null || true
    
    # Add new server configuration
    jfrog config add e2e-test-server \
        --url="${JFROG_URL}" \
        --user="${JFROG_USER}" \
        --password="${JFROG_PASSWORD}" \
        --interactive=false \
        --overwrite
    
    # Use this server as default
    jfrog config use e2e-test-server
    
    log_success "JFrog CLI configured successfully"
}

# Install Artifactory license
install_license() {
    local license=""
    local license_file="${SCRIPT_DIR}/../artifactory.lic"
    
    # Try to read from .lic file first (for local development)
    if [[ -f "${license_file}" ]]; then
        log_info "Reading license from: ${license_file}"
        license=$(cat "${license_file}" | tr -d '\n\r')
    # Fall back to environment variable (for CI/CD)
    elif [[ -n "${RTLIC:-}" ]]; then
        log_info "Using license from RTLIC environment variable"
        license=$(echo "${RTLIC}" | tr -d '\n\r')
    else
        # No license found - simple error message
        echo "" >&2
        log_error "Artifactory license is missing." >&2
        log_error "Please provide license file at: ${license_file}" >&2
        log_error "Or set RTLIC environment variable." >&2
        echo "" >&2
        return 1
    fi
    
    if [[ -z "${license}" ]]; then
        log_error "License is empty - cannot proceed"
        return 1
    fi
    
    log_info "Installing Artifactory license via API..."
    
    # Install license using direct curl (like evidence does)
    local response
    response=$(curl -u "${JFROG_USER}:${JFROG_PASSWORD}" \
        --retry 5 \
        --retry-max-time 60 \
        --retry-connrefused \
        -XPOST "${JFROG_URL}/artifactory/api/system/licenses" \
        --header "Content-Type: application/json" \
        --data-raw "{\"licenseKey\": \"${license}\"}" \
        -s -w "\n%{http_code}" 2>&1)
    
    # Extract HTTP status code (last line)
    local http_code=$(echo "$response" | tail -n 1)
    local body=$(echo "$response" | sed '$d')
    
    # Check if successful (2xx status codes)
    if [[ "$http_code" =~ ^2 ]]; then
        log_success "License installed successfully (HTTP $http_code)"
    else
        log_warning "License installation returned HTTP $http_code"
        if [[ -n "$body" ]]; then
            log_warning "Response: $body"
        fi
        log_warning "Continuing anyway... (some features may not work)"
    fi
}

# Create a project
create_project() {
    local project_key="$1"
    local project_name="$2"
    local admin_token="$3"
    
    log_info "Creating project: ${project_name} (${project_key})..."
    
    local response=$(curl -s -w "\n%{http_code}" \
        -X POST "${JFROG_URL}/access/api/v1/projects" \
        -H "Authorization: Bearer ${admin_token}" \
        -H "Content-Type: application/json" \
        -d '{
            "project_key": "'"${project_key}"'",
            "display_name": "'"${project_name}"'",
            "soft_limit": true,
            "storage_quota_bytes": 1073741824,
            "description": "Evidence E2E Test Project",
            "storage_quota_email_notification": false,
            "admin_privileges": {
                "manageMembers": true,
                "manageResources": true,
                "indexResources": true
            }
        }')
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')
    
    if [[ "$http_code" == "201" ]]; then
        log_success "Project created: ${project_key}"
        return 0
    elif [[ "$http_code" == "409" ]]; then
        log_warning "Project ${project_key} already exists, continuing..."
        return 0
    else
        log_warning "Failed to create project. HTTP ${http_code}, continuing anyway..."
        echo "$body" >&2
        return 0
    fi
}

# Create a user
create_user() {
    local username="$1"
    local password="$2"
    local email="$3"
    local admin_token="$4"
    
    log_info "Creating user: ${username}..."
    
    local response=$(curl -s -w "\n%{http_code}" \
        -X POST "${JFROG_URL}/access/api/v2/users" \
        -H "Authorization: Bearer ${admin_token}" \
        -H "Content-Type: application/json" \
        -d '{
            "username": "'"${username}"'",
            "password": "'"${password}"'",
            "email": "'"${email}"'",
            "groups": [],
            "profile_updatable": true,
            "admin": false
        }')
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')
    
    if [[ "$http_code" == "201" ]]; then
        log_success "User created: ${username}"
        return 0
    elif [[ "$http_code" == "409" ]]; then
        log_warning "User ${username} already exists, continuing..."
        return 0
    else
        log_warning "Failed to create user. HTTP ${http_code}, continuing anyway..."
        echo "$body" >&2
        return 0
    fi
}

# Create permission (role)
create_permission() {
    local permission_name="$1"
    local username="$2"
    local admin_token="$3"
    
    log_info "Creating permission: ${permission_name} for user ${username}..."
    
    local permission_json='{
            "name": "'"${permission_name}"'",
            "resources": {
                "artifact": {
                    "actions": {
                        "users": {
                            "'"${username}"'": ["READ", "WRITE", "DELETE", "ANNOTATE"]
                        },
                        "groups": {}
                    },
                    "targets": {
                        "ANY LOCAL": {
                            "include_patterns": ["**"],
                            "exclude_patterns": []
                        },
                        "artifactory-build-info": {
                            "include_patterns": ["**"],
                            "exclude_patterns": []
                        },
                        "release-bundles-v2": {
                            "include_patterns": ["**"],
                            "exclude_patterns": []
                        }
                    }
                },
                "repository": {
                    "actions": {
                        "users": {
                            "'"${username}"'": ["READ", "WRITE", "DELETE", "ANNOTATE"]
                        },
                        "groups": {}
                    },
                    "targets": {
                        "ANY LOCAL": {
                            "include_patterns": ["**"],
                            "exclude_patterns": []
                        },
                        "artifactory-build-info": {
                            "include_patterns": ["**"],
                            "exclude_patterns": []
                        },
                        "release-bundles-v2": {
                            "include_patterns": ["**"],
                            "exclude_patterns": []
                        }
                    }
                },
                "build": {
                    "actions": {
                        "users": {
                            "'"${username}"'": ["READ", "WRITE", "DELETE", "ANNOTATE"]
                        },
                        "groups": {}
                    },
                    "targets": {
                        "ANY LOCAL": {
                            "include_patterns": ["**"],
                            "exclude_patterns": []
                        },
                        "artifactory-build-info": {
                            "include_patterns": ["**"],
                            "exclude_patterns": []
                        }
                    }
                },
                "release_bundle": {
                    "actions": {
                        "users": {
                            "'"${username}"'": ["READ", "WRITE", "DELETE", "ANNOTATE"]
                        },
                        "groups": {}
                    },
                    "targets": {
                        "ANY": {
                            "include_patterns": ["**"],
                            "exclude_patterns": []
                        },
                        "release-bundles-v2": {
                            "include_patterns": ["**"],
                            "exclude_patterns": []
                        }
                    }
                }
            }
        }'
    
    # Try to create permission first
    local response=$(curl -s -w "\n%{http_code}" \
        -X POST "${JFROG_URL}/access/api/v2/permissions" \
        -H "Authorization: Bearer ${admin_token}" \
        -H "Content-Type: application/json" \
        -d "${permission_json}")
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')
    
    if [[ "$http_code" == "201" ]]; then
        log_success "Permission created: ${permission_name}"
        return 0
    elif [[ "$http_code" == "409" ]]; then
        # Permission already exists, delete and recreate it to ensure correct permissions
        log_info "Permission ${permission_name} already exists, deleting and recreating it..."
        local delete_response=$(curl -s -w "\n%{http_code}" \
            -X DELETE "${JFROG_URL}/access/api/v2/permissions/${permission_name}" \
            -H "Authorization: Bearer ${admin_token}")
        local delete_code=$(echo "$delete_response" | tail -n1)
        
        # Wait a moment for deletion to complete
        sleep 1
        
        # Now recreate with updated permissions
        local recreate_response=$(curl -s -w "\n%{http_code}" \
            -X POST "${JFROG_URL}/access/api/v2/permissions" \
            -H "Authorization: Bearer ${admin_token}" \
            -H "Content-Type: application/json" \
            -d "${permission_json}")
        local recreate_code=$(echo "$recreate_response" | tail -n1)
        if [[ "$recreate_code" == "201" ]]; then
            log_success "Permission recreated: ${permission_name}"
            return 0
        else
            log_warning "Failed to recreate permission. HTTP ${recreate_code}, continuing anyway..."
            echo "$(echo "$recreate_response" | sed '$d')" >&2
            return 0
        fi
    else
        log_warning "Failed to create permission. HTTP ${http_code}, continuing anyway..."
        echo "$body" >&2
        return 0
    fi
}

# Create token for user (admin creates it)
create_user_token() {
    local username="$1"
    
    log_info "Creating access token for user: ${username}..." >&2
    
    # Step 1: Get system token (admin)
    # FOR LOCAL TESTING ONLY - matches docker-compose.yml JF_SHARED_SECURITY_JOINKEY
    local JOIN_KEY="cc949ef041b726994a225dc20e018f23"
    
    local system_token_response
    system_token_response=$(jfrog access st \
        --url="${JFROG_URL}/artifactory" \
        --join-key="${JOIN_KEY}" \
        2>&1)
    
    local system_token
    system_token=$(echo "$system_token_response" | grep "export JF_ACCESS_ADMIN_TOKEN=eyJ" | sed 's/.*JF_ACCESS_ADMIN_TOKEN=\(eyJ.*\)/\1/')
    
    if [[ -z "$system_token" || "$system_token" == "null" ]]; then
        log_error "Failed to get system token" >&2
        return 1
    fi
    
    # Step 2: Create token for user with applied-permissions/user scope
    local json_response=$(curl -s -X POST \
        "${JFROG_URL}/access/api/v1/tokens" \
        -H "Authorization: Bearer ${system_token}" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "username=${username}&scope=applied-permissions/user&expires_in=0")
    
    local access_token
    if command -v jq >/dev/null 2>&1; then
        access_token=$(echo "$json_response" | jq -r '.access_token // empty' 2>/dev/null)
    else
        access_token=$(echo "$json_response" | grep -o '"access_token"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"\([^"]*\)".*/\1/')
    fi
    
    if [[ -z "$access_token" || "$access_token" == "null" ]]; then
        log_error "Failed to create token for user ${username}" >&2
        echo "Response: $json_response" >&2
        return 1
    fi
    
    log_success "Token created for user: ${username}" >&2
    echo "$access_token"
}

# Delete user
delete_user() {
    local username="$1"
    local admin_token="$2"
    
    log_info "Deleting user: ${username}..."
    
    local response=$(curl -s -w "\n%{http_code}" \
        -X DELETE "${JFROG_URL}/access/api/v2/users/${username}" \
        -H "Authorization: Bearer ${admin_token}")
    
    local http_code=$(echo "$response" | tail -n1)
    
    if [[ "$http_code" == "204" ]] || [[ "$http_code" == "404" ]]; then
        log_success "User deleted: ${username}"
        return 0
    else
        log_warning "Failed to delete user. HTTP ${http_code}"
        return 0
    fi
}

# Delete permission
delete_permission() {
    local permission_name="$1"
    local admin_token="$2"
    
    log_info "Deleting permission: ${permission_name}..."
    
    local response=$(curl -s -w "\n%{http_code}" \
        -X DELETE "${JFROG_URL}/access/api/v2/permissions/${permission_name}" \
        -H "Authorization: Bearer ${admin_token}")
    
    local http_code=$(echo "$response" | tail -n1)
    
    if [[ "$http_code" == "204" ]] || [[ "$http_code" == "404" ]]; then
        log_success "Permission deleted: ${permission_name}"
        return 0
    else
        log_warning "Failed to delete permission. HTTP ${http_code}"
        return 0
    fi
}

# Delete project
delete_project() {
    local project_key="$1"
    local admin_token="$2"
    
    log_info "Deleting project: ${project_key}..."
    
    local response=$(curl -s -w "\n%{http_code}" \
        -X DELETE "${JFROG_URL}/access/api/v1/projects/${project_key}" \
        -H "Authorization: Bearer ${admin_token}")
    
    local http_code=$(echo "$response" | tail -n1)
    
    if [[ "$http_code" == "204" ]] || [[ "$http_code" == "404" ]]; then
        log_success "Project deleted: ${project_key}"
        return 0
    else
        log_warning "Failed to delete project. HTTP ${http_code}"
        return 0
    fi
}

# Generate admin access token for testing
generate_admin_token() {
    log_info "Generating admin access token with applied-permissions/admin scope..." >&2
    
    # Use the join-key approach (same as Evidence project's tokengen.sh)
    # Step 1: Get a system token using the join-key
    # FOR LOCAL TESTING ONLY - matches docker-compose.yml JF_SHARED_SECURITY_JOINKEY
    local JOIN_KEY="cc949ef041b726994a225dc20e018f23"
    
    log_info "Getting system token using join-key..." >&2
    local system_token_response
    system_token_response=$(jfrog access st \
        --url="${JFROG_URL}/artifactory" \
        --join-key="${JOIN_KEY}" \
        2>&1)
    
    # Extract the system token from the export command
    # Output format: "export JF_ACCESS_ADMIN_TOKEN=eyJ..."
    local system_token
    system_token=$(echo "$system_token_response" | grep "export JF_ACCESS_ADMIN_TOKEN=eyJ" | sed 's/.*JF_ACCESS_ADMIN_TOKEN=\(eyJ.*\)/\1/')
    
    if [ -z "$system_token" ]; then
        log_error "Failed to get system token" >&2
        log_error "Response: $system_token_response" >&2
        return 1
    fi
    
    log_info "Got system token, creating admin token with applied-permissions/admin..." >&2
    
    # Step 2: Use system token to create admin token via Access API
    local admin_response
    admin_response=$(curl -s -X POST "${JFROG_URL}/access/api/v1/tokens" \
        -H "Authorization: Bearer ${system_token}" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "username=admin&scope=applied-permissions/admin&expires_in=0" \
        2>&1)
    
    # Extract access token from JSON response
    local json_response
    json_response=$(echo "$admin_response" | grep -E '^\s*[{"}]')
    
    local access_token
    if command -v jq >/dev/null 2>&1; then
        access_token=$(echo "$json_response" | jq -r '.access_token // empty' 2>/dev/null)
    else
        access_token=$(echo "$json_response" | grep -o '"access_token"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"\([^"]*\)".*/\1/')
    fi
    
    if [ -z "$access_token" ] || [ "$access_token" = "null" ]; then
        log_error "Failed to generate admin access token" >&2
        log_error "Response: $admin_response" >&2
        return 1
    fi
    
    log_success "Admin access token with applied-permissions/admin scope generated successfully" >&2
    
    # Output only the token (to stdout for capture)
    echo "$access_token"
}
